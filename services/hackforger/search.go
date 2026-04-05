// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	hackforger_model "forgejo.org/models/hackforger"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	hackforger_indexer "forgejo.org/modules/indexer/hackforger"
	issue_indexer "forgejo.org/modules/indexer/issues"
	"forgejo.org/modules/log"
)

const defaultGroupLimit = 5

// SearchItem represents a single item in grouped search results.
type SearchItem struct {
	Type   string `json:"type"`
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Desc   string `json:"desc,omitempty"`
	Status string `json:"status,omitempty"`
	URL    string `json:"url"`
	Icon   string `json:"icon"`
}

// SearchGroup holds items for one entity type group.
type SearchGroup struct {
	Key   string        `json:"key"`
	Title string        `json:"title"`
	Items []*SearchItem `json:"items"`
}

// GroupedSearchResult contains search results grouped by entity type.
type GroupedSearchResult struct {
	Groups []*SearchGroup `json:"groups"`
}

// UnifiedSearchOptions holds parameters for the unified search query.
type UnifiedSearchOptions struct {
	Keyword string
	Doer    *user_model.User
}

// UnifiedSearch performs a cross-entity keyword search in parallel across
// HackForger entities (hackathons, bounties, grants, submissions),
// repositories, users, and issues. Returns grouped results with top 5 per group.
func UnifiedSearch(ctx context.Context, opts *UnifiedSearchOptions) (*GroupedSearchResult, error) {
	if opts.Keyword == "" {
		return &GroupedSearchResult{}, nil
	}

	type groupResult struct {
		key   string
		items []*SearchItem
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []groupResult
	)

	collect := func(key string, items []*SearchItem) {
		if len(items) == 0 {
			return
		}
		mu.Lock()
		results = append(results, groupResult{key: key, items: items})
		mu.Unlock()
	}

	// 1. HackForger indexer — hackathons, bounties, grants, submissions
	wg.Add(1)
	go func() {
		defer wg.Done()
		searchHackforgerEntities(ctx, opts.Keyword, collect)
	}()

	// 2. Repositories
	wg.Add(1)
	go func() {
		defer wg.Done()
		items := searchRepos(ctx, opts)
		collect("repos", items)
	}()

	// 3. Users
	wg.Add(1)
	go func() {
		defer wg.Done()
		items := searchUsers(ctx, opts)
		collect("users", items)
	}()

	// 4. Issues
	wg.Add(1)
	go func() {
		defer wg.Done()
		items := searchIssues(ctx, opts)
		collect("issues", items)
	}()

	wg.Wait()

	// Build groups in fixed display order
	groupOrder := []string{"hackathons", "bounties", "grants", "submissions", "repos", "users", "issues"}
	resultMap := make(map[string][]*SearchItem, len(results))
	for _, r := range results {
		resultMap[r.key] = r.items
	}

	var groups []*SearchGroup
	for _, key := range groupOrder {
		items, ok := resultMap[key]
		if !ok || len(items) == 0 {
			continue
		}
		groups = append(groups, &SearchGroup{
			Key:   key,
			Items: items,
		})
	}

	return &GroupedSearchResult{Groups: groups}, nil
}

// searchHackforgerEntities queries the HackForger indexer for all 4 entity types
// and populates items by loading full entity data from the database.
func searchHackforgerEntities(ctx context.Context, keyword string, collect func(string, []*SearchItem)) {
	entityTypes := []string{"hackathon", "bounty", "grant", "submission"}
	groupKeys := map[string]string{
		"hackathon":  "hackathons",
		"bounty":     "bounties",
		"grant":      "grants",
		"submission": "submissions",
	}

	for _, et := range entityTypes {
		result, err := hackforger_indexer.SearchHackforger(ctx, &hackforger_indexer.SearchOptions{
			Keyword:    keyword,
			EntityType: et,
			Paginator:  &db.ListOptions{Page: 1, PageSize: defaultGroupLimit},
		})
		if err != nil {
			log.Error("UnifiedSearch: hackforger indexer error for %s: %v", et, err)
			continue
		}

		items := make([]*SearchItem, 0, len(result.Hits))
		for _, hit := range result.Hits {
			item := loadHackforgerEntity(ctx, hit.EntityType, hit.ID)
			if item != nil {
				items = append(items, item)
			}
		}
		collect(groupKeys[et], items)
	}
}

// loadHackforgerEntity loads a HackForger entity from the database and converts it to a SearchItem.
func loadHackforgerEntity(ctx context.Context, entityType string, id int64) *SearchItem {
	switch entityType {
	case "hackathon":
		h, err := hackforger_model.GetHackathonByID(ctx, id)
		if err != nil {
			log.Error("UnifiedSearch: load hackathon %d: %v", id, err)
			return nil
		}
		return &SearchItem{
			Type:   "hackathon",
			ID:     h.ID,
			Title:  h.Name,
			Desc:   truncateDesc(h.Description),
			Status: hackforger_model.HackathonStatusNames[h.StatusCache],
			URL:    fmt.Sprintf("/hackathons/%s", h.Slug),
			Icon:   "octicon-rocket",
		}
	case "bounty":
		b, err := hackforger_model.GetBountyByID(ctx, id)
		if err != nil {
			log.Error("UnifiedSearch: load bounty %d: %v", id, err)
			return nil
		}
		return &SearchItem{
			Type:   "bounty",
			ID:     b.ID,
			Title:  b.Title,
			Status: hackforger_model.BountyStatusNames[b.Status],
			URL:    "/explore/bounties",
			Icon:   "octicon-gift",
		}
	case "grant":
		r, err := hackforger_model.GetGrantRoundByID(ctx, id)
		if err != nil {
			log.Error("UnifiedSearch: load grant round %d: %v", id, err)
			return nil
		}
		return &SearchItem{
			Type:   "grant",
			ID:     r.ID,
			Title:  r.Name,
			Desc:   truncateDesc(r.Description),
			Status: hackforger_model.GrantRoundStatusNames[r.Status],
			URL:    fmt.Sprintf("/grants/%s", r.Slug),
			Icon:   "octicon-heart",
		}
	case "submission":
		s, err := hackforger_model.GetSubmissionByID(ctx, id)
		if err != nil {
			log.Error("UnifiedSearch: load submission %d: %v", id, err)
			return nil
		}
		return &SearchItem{
			Type:   "submission",
			ID:     s.ID,
			Title:  s.Title,
			Desc:   truncateDesc(s.Description),
			URL:    fmt.Sprintf("/hackathons/submissions/%d", s.ID),
			Icon:   "octicon-file-code",
		}
	default:
		return nil
	}
}

// searchRepos queries Forgejo's repository search.
func searchRepos(ctx context.Context, opts *UnifiedSearchOptions) []*SearchItem {
	repos, _, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{Page: 1, PageSize: defaultGroupLimit},
		Keyword:     opts.Keyword,
		Actor:       opts.Doer,
		AllPublic:   true,
		OrderBy:     db.SearchOrderByNewest,
	})
	if err != nil {
		log.Error("UnifiedSearch: repo search error: %v", err)
		return nil
	}

	items := make([]*SearchItem, 0, len(repos))
	for _, r := range repos {
		items = append(items, &SearchItem{
			Type:  "repo",
			ID:    r.ID,
			Title: r.FullName(),
			Desc:  truncateDesc(r.Description),
			URL:   r.Link(),
			Icon:  "octicon-repo",
		})
	}
	return items
}

// searchUsers queries Forgejo's user search.
func searchUsers(ctx context.Context, opts *UnifiedSearchOptions) []*SearchItem {
	users, _, err := user_model.SearchUsers(ctx, &user_model.SearchUserOptions{
		ListOptions: db.ListOptions{Page: 1, PageSize: defaultGroupLimit},
		Keyword:     opts.Keyword,
		Type:        user_model.UserTypeIndividual,
		Actor:       opts.Doer,
	})
	if err != nil {
		log.Error("UnifiedSearch: user search error: %v", err)
		return nil
	}

	items := make([]*SearchItem, 0, len(users))
	for _, u := range users {
		desc := u.FullName
		if desc == "" {
			desc = u.Name
		}
		items = append(items, &SearchItem{
			Type:  "user",
			ID:    u.ID,
			Title: u.Name,
			Desc:  desc,
			URL:   fmt.Sprintf("/%s", u.Name),
			Icon:  "octicon-person",
		})
	}
	return items
}

// searchIssues queries Forgejo's issue indexer.
func searchIssues(ctx context.Context, opts *UnifiedSearchOptions) []*SearchItem {
	searchOpts := &issue_indexer.SearchOptions{
		Paginator: &db.ListOptions{Page: 1, PageSize: defaultGroupLimit},
		AllPublic: true,
		SortBy:    issue_indexer.SortByScore,
	}
	_ = searchOpts.WithKeyword(ctx, opts.Keyword)

	issueIDs, _, err := issue_indexer.SearchIssues(ctx, searchOpts)
	if err != nil {
		log.Error("UnifiedSearch: issue search error: %v", err)
		return nil
	}
	if len(issueIDs) == 0 {
		return nil
	}

	issues, err := issues_model.GetIssuesByIDs(ctx, issueIDs, true)
	if err != nil {
		log.Error("UnifiedSearch: load issues error: %v", err)
		return nil
	}

	items := make([]*SearchItem, 0, len(issues))
	for _, iss := range issues {
		icon := "octicon-issue-opened"
		if iss.IsClosed {
			icon = "octicon-issue-closed"
		}
		if iss.IsPull {
			icon = "octicon-git-pull-request"
		}
		// Build URL — need repo loaded for full path
		url := fmt.Sprintf("/issues/%d", iss.ID)
		if iss.Repo != nil {
			url = fmt.Sprintf("/%s/issues/%d", iss.Repo.FullName(), iss.Index)
		}
		items = append(items, &SearchItem{
			Type:  "issue",
			ID:    iss.ID,
			Title: iss.Title,
			URL:   url,
			Icon:  icon,
		})
	}
	return items
}

// truncateDesc truncates a description to maxLen characters.
// stripMarkdown removes common Markdown syntax to produce plain text for search result previews.
func stripMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "---" {
			continue
		}
		// Remove headings (## ...)
		for strings.HasPrefix(line, "#") {
			line = strings.TrimLeft(line, "#")
			line = strings.TrimSpace(line)
		}
		// Remove list markers (- ..., * ..., 1. ...)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			line = line[2:]
		}
		// Remove bold/italic (**text**, *text*, __text__, _text_)
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		// Remove image/link syntax ![alt](url) → alt, [text](url) → text
		for {
			idx := strings.Index(line, "](")
			if idx < 0 {
				break
			}
			// Find opening [ or ![
			start := strings.LastIndex(line[:idx], "[")
			end := strings.Index(line[idx:], ")")
			if start < 0 || end < 0 {
				break
			}
			text := line[start+1 : idx]
			if start > 0 && line[start-1] == '!' {
				start--
			}
			line = line[:start] + text + line[idx+end+1:]
		}
		// Remove backticks
		line = strings.ReplaceAll(line, "`", "")
		// Remove blockquote markers
		line = strings.TrimPrefix(line, "> ")
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, " ")
}

func truncateDesc(s string) string {
	const maxLen = 120
	s = stripMarkdown(s)
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
