// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
)

// SearchResult represents a single item in search results.
type SearchResult struct {
	Type   string `json:"type"`
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Slug   string `json:"slug,omitempty"`
}

// SearchOptions holds parameters for the search query.
type SearchOptions struct {
	Keyword string
	Scope   string // "all" | "hackathons" | "bounties" | "grants"
	Page    int
	Limit   int
}

// Search performs a cross-entity keyword search across hackathons, bounties,
// and grant rounds. Results are collected in-memory and paginated.
func Search(ctx context.Context, opts *SearchOptions) ([]*SearchResult, int64, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}

	var results []*SearchResult
	keyword := "%" + opts.Keyword + "%"
	e := db.GetEngine(ctx)

	// Search hackathons
	if opts.Scope == "all" || opts.Scope == "hackathons" {
		var hackathons []*hackforger_model.Hackathon
		err := e.Where("name LIKE ? OR description LIKE ?", keyword, keyword).
			OrderBy("created_unix DESC").Find(&hackathons)
		if err != nil {
			return nil, 0, err
		}
		for _, h := range hackathons {
			results = append(results, &SearchResult{
				Type:   "hackathon",
				ID:     h.ID,
				Title:  h.Name,
				Status: hackforger_model.HackathonStatusNames[h.Status],
				Slug:   h.Slug,
			})
		}
	}

	// Search bounties
	if opts.Scope == "all" || opts.Scope == "bounties" {
		var bounties []*hackforger_model.Bounty
		err := e.Where("title LIKE ?", keyword).
			OrderBy("created_unix DESC").Find(&bounties)
		if err != nil {
			return nil, 0, err
		}
		for _, b := range bounties {
			results = append(results, &SearchResult{
				Type:   "bounty",
				ID:     b.ID,
				Title:  b.Title,
				Status: hackforger_model.BountyStatusNames[b.Status],
			})
		}
	}

	// Search grant rounds
	if opts.Scope == "all" || opts.Scope == "grants" {
		var rounds []*hackforger_model.GrantRound
		err := e.Where("name LIKE ? OR description LIKE ?", keyword, keyword).
			OrderBy("created_unix DESC").Find(&rounds)
		if err != nil {
			return nil, 0, err
		}
		for _, r := range rounds {
			results = append(results, &SearchResult{
				Type:   "grant",
				ID:     r.ID,
				Title:  r.Name,
				Status: hackforger_model.GrantRoundStatusNames[r.Status],
				Slug:   r.Slug,
			})
		}
	}

	total := int64(len(results))

	// Apply pagination
	start := (opts.Page - 1) * opts.Limit
	if start >= len(results) {
		return []*SearchResult{}, total, nil
	}
	end := start + opts.Limit
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], total, nil
}
