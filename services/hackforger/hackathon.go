// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	organization_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"
	pull_service "forgejo.org/services/pull"
	release_service "forgejo.org/services/release"
	repo_service "forgejo.org/services/repository"
	files_service "forgejo.org/services/repository/files"
)

// ErrNoTracks means hackathon has no tracks (cannot publish).
type ErrNoTracks struct{ HackathonID int64 }

func (e ErrNoTracks) Error() string {
	return fmt.Sprintf("hackathon has no tracks [id: %d]", e.HackathonID)
}

// IsErrNoTracks checks if err is ErrNoTracks.
func IsErrNoTracks(err error) bool { _, ok := err.(ErrNoTracks); return ok }

// ErrNoSubmissions means hackathon has no submissions (cannot start judging).
type ErrNoSubmissions struct{ HackathonID int64 }

func (e ErrNoSubmissions) Error() string {
	return fmt.Sprintf("hackathon has no submissions [id: %d]", e.HackathonID)
}

// IsErrNoSubmissions checks if err is ErrNoSubmissions.
func IsErrNoSubmissions(err error) bool { _, ok := err.(ErrNoSubmissions); return ok }

// CreateHackathon creates a Forgejo Organization for the hackathon,
// then creates the hackathon record and publishes a creation event.
func CreateHackathon(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon) error {
	// Create a Forgejo Organization linked to this hackathon
	org := &organization_model.Organization{
		Name:       h.Slug,
		FullName:   h.Name,
		Visibility: structs.VisibleTypePublic,
	}
	if err := organization_model.CreateOrganization(ctx, org, doer); err != nil {
		if user_model.IsErrUserAlreadyExist(err) {
			org.Name = h.Slug + "-hackathon"
			if err := organization_model.CreateOrganization(ctx, org, doer); err != nil {
				return fmt.Errorf("create hackathon org: %w", err)
			}
		} else {
			return fmt.Errorf("create hackathon org: %w", err)
		}
	}
	h.LinkedOrgID = org.ID
	h.OrgID = org.ID
	h.OwnerID = doer.ID

	if err := hackforger_model.CreateHackathon(ctx, h); err != nil {
		return err
	}

	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doer.ID,
		OpType:       hackforger_model.ActionHackathonCreated,
		EntityType:   "hackathon",
		EntityID:     h.ID,
		EntityName:   h.Name,
		EntitySlug:   h.Slug,
		AudienceType: AudienceGlobal,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
		},
	})
	return nil
}

// CreateTrackWithRepo creates a Forgejo Repository in the hackathon's linked
// Organization for the track, then inserts the track record.
func CreateTrackWithRepo(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, track *hackforger_model.HackathonTrack) error {
	if h.LinkedOrgID == 0 {
		// No linked org — just create track record without repo
		return hackforger_model.CreateTrack(ctx, track)
	}

	orgUser, err := user_model.GetUserByID(ctx, h.LinkedOrgID)
	if err != nil {
		return fmt.Errorf("get hackathon org user: %w", err)
	}

	repoName := strings.ToLower(strings.ReplaceAll(track.Name, " ", "-"))
	repo, err := repo_service.CreateRepository(ctx, doer, orgUser, repo_service.CreateRepoOptions{
		Name:          repoName,
		Description:   track.Description,
		Readme:        "Default",
		AutoInit:      true,
		DefaultBranch: "main",
	})
	if err != nil {
		return fmt.Errorf("create track repo: %w", err)
	}

	track.RepoID = repo.ID
	if err := hackforger_model.CreateTrack(ctx, track); err != nil {
		return err
	}

	// Initialize submission index files in the track repo
	_, closer, err := gitrepo.RepositoryFromContextOrOpen(ctx, repo)
	if err == nil {
		defer closer.Close()
		_, initErr := files_service.ChangeRepoFiles(ctx, repo, doer, &files_service.ChangeRepoFilesOptions{
			OldBranch: repo.DefaultBranch,
			NewBranch: repo.DefaultBranch,
			Message:   "Initialize submission index",
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "SUBMISSIONS.md",
					ContentReader: strings.NewReader(fmt.Sprintf("# %s — Submissions\n\nNo submissions yet.\n", track.Name)),
				},
				{
					Operation:     "create",
					TreePath:      "submissions.json",
					ContentReader: strings.NewReader("[]\n"),
				},
			},
		})
		if initErr != nil {
			log.Warn("CreateTrackWithRepo: failed to initialize index files: %v", initErr)
		}
	}

	// Auto-seed track criteria for any existing hackathon criteria
	criteria, err := hackforger_model.ListCriteriaByHackathon(ctx, h.ID)
	if err == nil && len(criteria) > 0 {
		_ = hackforger_model.SeedTrackCriteria(ctx, track.ID, criteria)
	}

	// Auto-create phase milestones on the track repo
	milestones := []struct {
		Name     string
		Deadline timeutil.TimeStamp
	}{
		{"Registration", h.RegistrationEnd},
		{"Hacking", h.HackingEnd},
		{"Judging", h.JudgingEnd},
		{"Results", 0}, // closed when finalized
	}
	for _, ms := range milestones {
		_ = issues_model.NewMilestone(ctx, &issues_model.Milestone{
			RepoID:       repo.ID,
			Name:         ms.Name,
			DeadlineUnix: ms.Deadline,
		})
	}
	return nil
}

// PublishHackathon transitions Draft → Open. Requires >= 1 track.
func PublishHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status != hackforger_model.HackathonStatusDraft {
		return fmt.Errorf("hackathon must be in Draft status to publish [id: %d, status: %d]", h.ID, h.Status)
	}
	trackCount, err := hackforger_model.CountTracksByHackathon(ctx, h.ID)
	if err != nil {
		return err
	}
	if trackCount == 0 {
		return ErrNoTracks{HackathonID: h.ID}
	}
	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusOpen); err != nil {
		return err
	}
	publishPhaseChange(ctx, doerID, h, hackforger_model.HackathonStatusDraft, hackforger_model.HackathonStatusOpen)
	return nil
}

// StartHacking transitions Open → Hacking.
func StartHacking(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status != hackforger_model.HackathonStatusOpen {
		return hackforger_model.ErrInvalidHackathonPhase{HackathonID: h.ID, Current: h.Status, Expected: hackforger_model.HackathonStatusOpen}
	}
	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusHacking); err != nil {
		return err
	}
	// Tag track repos with v0-kickoff baseline
	tagTrackRepos(ctx, doerID, h.ID, "v0-kickoff", "Hackathon kickoff baseline")
	publishPhaseChange(ctx, doerID, h, hackforger_model.HackathonStatusOpen, hackforger_model.HackathonStatusHacking)
	return nil
}

// StartJudging transitions Hacking → Judging. Requires >= 1 submission.
func StartJudging(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status != hackforger_model.HackathonStatusHacking {
		return hackforger_model.ErrInvalidHackathonPhase{HackathonID: h.ID, Current: h.Status, Expected: hackforger_model.HackathonStatusHacking}
	}
	criteriaCount, err := hackforger_model.CountCriteriaByHackathon(ctx, h.ID)
	if err != nil {
		return err
	}
	if criteriaCount == 0 {
		return hackforger_model.ErrNoCriteria{HackathonID: h.ID}
	}
	subCount, err := hackforger_model.CountSubmissions(ctx, h.ID)
	if err != nil {
		return err
	}
	if subCount == 0 {
		return ErrNoSubmissions{HackathonID: h.ID}
	}
	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusJudging); err != nil {
		return err
	}
	publishPhaseChange(ctx, doerID, h, hackforger_model.HackathonStatusHacking, hackforger_model.HackathonStatusJudging)
	return nil
}

// PreviewFinalize calculates rankings for all tracks without changing status.
// Idempotent — can be called multiple times.
func PreviewFinalize(ctx context.Context, hackathonID int64) (map[int64][]RankedSubmission, error) {
	h, err := hackforger_model.GetHackathonByID(ctx, hackathonID)
	if err != nil {
		return nil, err
	}
	if h.Status != hackforger_model.HackathonStatusJudging {
		return nil, hackforger_model.ErrInvalidHackathonPhase{
			HackathonID: h.ID, Current: h.Status, Expected: hackforger_model.HackathonStatusJudging,
		}
	}
	return CalculateRanks(ctx, hackathonID)
}

// ConfirmFinalize locks status to Finished, persists ranks, creates releases.
func ConfirmFinalize(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status != hackforger_model.HackathonStatusJudging {
		return hackforger_model.ErrInvalidHackathonPhase{
			HackathonID: h.ID, Current: h.Status, Expected: hackforger_model.HackathonStatusJudging,
		}
	}
	// Tag track repos (preserves existing behavior)
	tagTrackRepos(ctx, doerID, h.ID, "submission-deadline", "Submission deadline snapshot")

	// Calculate and persist ranks
	rankings, err := CalculateRanks(ctx, h.ID)
	if err != nil {
		return err
	}
	if err := PersistRanks(ctx, rankings); err != nil {
		return err
	}

	// Update status
	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusFinished); err != nil {
		return err
	}

	// Create releases (preserves existing behavior)
	createTrackReleases(ctx, doerID, h)

	// Build winners summary
	trackWinners := make([]map[string]any, 0)
	for trackID, subs := range rankings {
		if len(subs) > 0 {
			trackWinners = append(trackWinners, map[string]any{
				"track_id": trackID, "winner_submission_id": subs[0].SubmissionID, "winner_user_id": subs[0].UserID,
			})
		}
	}

	// Publish feed event
	// NOTE: Deliberately using HackforgerActionContent (not HackforgerPhaseContent)
	// because ActionHackathonFinalized now carries results summary.
	// TODO: update to new feed API format after feed refactor merges
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doerID,
		OpType:       hackforger_model.ActionHackathonFinalized,
		EntityType:   "hackathon",
		EntityID:     h.ID,
		EntityName:   h.Name,
		EntitySlug:   h.Slug,
		AudienceType: AudienceGlobal,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
			Extra: map[string]any{"tracks": trackWinners},
		},
	})
	return nil
}

// CancelHackathon transitions to Cancelled. Any non-Finished status.
func CancelHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status == hackforger_model.HackathonStatusFinished {
		return hackforger_model.ErrInvalidHackathonPhase{HackathonID: h.ID, Current: h.Status, Expected: 0}
	}
	if h.Status == hackforger_model.HackathonStatusCancelled {
		return hackforger_model.ErrInvalidHackathonPhase{HackathonID: h.ID, Current: h.Status, Expected: 0}
	}
	oldStatus := h.Status
	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusCancelled); err != nil {
		return err
	}
	publishPhaseChange(ctx, doerID, h, oldStatus, hackforger_model.HackathonStatusCancelled)
	return nil
}

// HackathonStatusLabel returns a human-readable status label.
func HackathonStatusLabel(status hackforger_model.HackathonStatus) string {
	labels := map[hackforger_model.HackathonStatus]string{
		hackforger_model.HackathonStatusDraft:     "Draft",
		hackforger_model.HackathonStatusOpen:      "Open (Registration)",
		hackforger_model.HackathonStatusHacking:   "Hacking",
		hackforger_model.HackathonStatusJudging:   "Judging",
		hackforger_model.HackathonStatusFinished:  "Finished",
		hackforger_model.HackathonStatusCancelled: "Cancelled",
	}
	if l, ok := labels[status]; ok {
		return l
	}
	return "Unknown"
}

// CreateSubmission creates a hackathon submission. When the user provides a
// repo_id, it is validated to belong to the user (or an org they are a member
// of). If the submission targets a track with a linked repository, a PR is
// created on the track repo updating SUBMISSIONS.md and submissions.json.
func CreateSubmission(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, sub *hackforger_model.HackathonSubmission) error {
	// 1. If user provided a repo_id, validate it exists and belongs to the doer
	if sub.RepoID > 0 {
		repo, err := repo_model.GetRepositoryByID(ctx, sub.RepoID)
		if err != nil {
			return fmt.Errorf("get user repo: %w", err)
		}
		if repo.OwnerID != doer.ID {
			// Also allow repos owned by orgs where user is member
			isMember, _ := organization_model.IsOrganizationMember(ctx, repo.OwnerID, doer.ID)
			if !isMember {
				return fmt.Errorf("repo does not belong to user")
			}
		}
	}

	// 2. Save the submission record first (so we have an ID)
	if err := hackforger_model.CreateSubmission(ctx, sub); err != nil {
		return err
	}

	// 3. If track has a repo, create a PR updating the submission index.
	//    Use the hackathon org owner (not the submitter) to create the branch/PR,
	//    because submitters may not have write access to the track repo.
	if sub.TrackID > 0 {
		track, err := hackforger_model.GetTrackByID(ctx, sub.TrackID)
		if err == nil && track.RepoID > 0 {
			orgOwner, ownerErr := user_model.GetUserByID(ctx, h.OwnerID)
			if ownerErr != nil {
				log.Warn("CreateSubmission: failed to load hackathon owner: %v", ownerErr)
			} else {
				if err := updateTrackSubmissionIndex(ctx, orgOwner, doer, h, track, sub); err != nil {
					// Log but don't fail the submission — the DB record is already saved
					log.Warn("CreateSubmission: failed to update track index: %v", err)
				}
			}
		}
	}

	publishSubmissionEvent(ctx, doer, h, sub)
	return nil
}

// updateTrackSubmissionIndex creates a PR to the track repo that updates
// SUBMISSIONS.md and submissions.json with the new submission entry.
// actor: the user performing git operations (hackathon org owner, has write access).
// submitter: the actual submission author (used for PR content attribution).
func updateTrackSubmissionIndex(ctx context.Context, actor, submitter *user_model.User, h *hackforger_model.Hackathon, track *hackforger_model.HackathonTrack, sub *hackforger_model.HackathonSubmission) error {
	baseRepo, err := repo_model.GetRepositoryByID(ctx, track.RepoID)
	if err != nil {
		return fmt.Errorf("get track repo: %w", err)
	}
	if err := baseRepo.LoadOwner(ctx); err != nil {
		return fmt.Errorf("load track repo owner: %w", err)
	}

	// Build submission repo URL for the PR body
	var repoURL string
	if sub.RepoID > 0 {
		if userRepo, err := repo_model.GetRepositoryByID(ctx, sub.RepoID); err == nil {
			repoURL = setting.AppURL + userRepo.FullName()
		}
	}

	// Load ALL current submissions for this track to rebuild the full index
	allSubs, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
		HackathonID: sub.HackathonID,
		TrackID:     track.ID,
	})
	if err != nil {
		return fmt.Errorf("list submissions: %w", err)
	}

	// Generate SUBMISSIONS.md content
	var mdBuf strings.Builder
	mdBuf.WriteString(fmt.Sprintf("# %s — Submissions\n\n", track.Name))
	mdBuf.WriteString(fmt.Sprintf("Track for [%s](%shackathon/%s)\n\n", h.Name, setting.AppURL, h.Slug))
	mdBuf.WriteString("| # | Project | Author | Repo | Demo |\n")
	mdBuf.WriteString("|---|---------|--------|------|------|\n")
	for i, s := range allSubs {
		repoLink := "-"
		if s.RepoID > 0 {
			if r, err := repo_model.GetRepositoryByID(ctx, s.RepoID); err == nil {
				repoLink = fmt.Sprintf("[%s](%s%s)", r.FullName(), setting.AppURL, r.FullName())
			}
		}
		demoLink := "-"
		if s.DemoURL != "" {
			demoLink = fmt.Sprintf("[Demo](%s)", s.DemoURL)
		}
		authorName := fmt.Sprintf("User #%d", s.UserID)
		if u, err := user_model.GetUserByID(ctx, s.UserID); err == nil {
			authorName = fmt.Sprintf("[%s](%s%s)", u.Name, setting.AppURL, u.Name)
		}
		mdBuf.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s |\n", i+1, s.Title, authorName, repoLink, demoLink))
	}

	// Generate submissions.json content
	type jsonEntry struct {
		ID          int64  `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		UserID      int64  `json:"user_id"`
		RepoURL     string `json:"repo_url,omitempty"`
		DemoURL     string `json:"demo_url,omitempty"`
	}
	jsonEntries := make([]jsonEntry, 0, len(allSubs))
	for _, s := range allSubs {
		entry := jsonEntry{
			ID: s.ID, Title: s.Title, Description: s.Description,
			UserID: s.UserID, DemoURL: s.DemoURL,
		}
		if s.RepoID > 0 {
			if r, err := repo_model.GetRepositoryByID(ctx, s.RepoID); err == nil {
				entry.RepoURL = setting.AppURL + r.FullName()
			}
		}
		jsonEntries = append(jsonEntries, entry)
	}
	jsonBytes, _ := json.MarshalIndent(jsonEntries, "", "  ")

	// Create a new branch for this submission's PR (actor = org owner with write access)
	branchName := fmt.Sprintf("submission/%d-%s", sub.ID, submitter.LowerName)

	gitRepo, closer, err := gitrepo.RepositoryFromContextOrOpen(ctx, baseRepo)
	if err != nil {
		return fmt.Errorf("open git repo: %w", err)
	}
	defer closer.Close()

	if err := repo_service.CreateNewBranch(ctx, actor, baseRepo, gitRepo, baseRepo.DefaultBranch, branchName); err != nil {
		return fmt.Errorf("create branch: %w", err)
	}

	// Commit both files to the new branch (actor = org owner)
	mdContent := mdBuf.String()
	_, err = files_service.ChangeRepoFiles(ctx, baseRepo, actor, &files_service.ChangeRepoFilesOptions{
		OldBranch: branchName,
		NewBranch: branchName,
		Message:   fmt.Sprintf("Add submission: %s by %s", sub.Title, submitter.Name),
		Files: []*files_service.ChangeRepoFile{
			{
				Operation:     "update",
				TreePath:      "SUBMISSIONS.md",
				ContentReader: strings.NewReader(mdContent),
			},
			{
				Operation:     "update",
				TreePath:      "submissions.json",
				ContentReader: strings.NewReader(string(jsonBytes)),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("commit files: %w", err)
	}

	// Create PR from the branch to the default branch (poster = actor, content credits submitter)
	issue := &issues_model.Issue{
		RepoID:   baseRepo.ID,
		Title:    fmt.Sprintf("Submission: %s", sub.Title),
		Content:  fmt.Sprintf("**Author:** @%s\n**Project:** %s\n\n%s", submitter.Name, sub.Title, sub.Description),
		PosterID: actor.ID,
	}
	if repoURL != "" {
		issue.Content += fmt.Sprintf("\n\n**Repository:** %s", repoURL)
	}
	if sub.DemoURL != "" {
		issue.Content += fmt.Sprintf("\n**Demo:** %s", sub.DemoURL)
	}

	pr := &issues_model.PullRequest{
		HeadRepoID: baseRepo.ID,
		BaseRepoID: baseRepo.ID,
		HeadBranch: branchName,
		BaseBranch: baseRepo.DefaultBranch,
		Type:       issues_model.PullRequestGitea,
	}
	if err := pull_service.NewPullRequest(ctx, baseRepo, issue, nil, nil, pr, nil); err != nil {
		log.Warn("updateTrackSubmissionIndex: PR creation failed: %v", err)
	} else {
		// Save PR reference back to submission
		sub.PRID = pr.ID
		sub.PullIndex = issue.Index
		_ = hackforger_model.UpdateSubmission(ctx, sub)
	}

	return nil
}

func publishSubmissionEvent(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, sub *hackforger_model.HackathonSubmission) {
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doer.ID,
		OpType:       hackforger_model.ActionHackathonSubmitted,
		EntityType:   "hackathon",
		EntityID:     h.ID,
		EntityName:   h.Name,
		EntitySlug:   h.Slug,
		AudienceType: AudienceFollowers,
		RepoID:       sub.RepoID,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
			Extra: map[string]any{"submission_id": sub.ID, "title": sub.Title},
		},
	})
}

var hackathonStatusName = map[hackforger_model.HackathonStatus]string{
	hackforger_model.HackathonStatusDraft:     "draft",
	hackforger_model.HackathonStatusOpen:      "open",
	hackforger_model.HackathonStatusHacking:   "hacking",
	hackforger_model.HackathonStatusJudging:   "judging",
	hackforger_model.HackathonStatusFinished:  "finished",
	hackforger_model.HackathonStatusCancelled: "cancelled",
}

// tagTrackRepos tags all track repos for a hackathon. Errors are logged but do not block the caller.
func tagTrackRepos(ctx context.Context, doerID, hackathonID int64, tagName, msg string) {
	doer, err := user_model.GetUserByID(ctx, doerID)
	if err != nil {
		log.Error("tagTrackRepos: get doer: %v", err)
		return
	}
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, hackathonID)
	if err != nil {
		log.Error("tagTrackRepos: list tracks: %v", err)
		return
	}
	for _, track := range tracks {
		if track.RepoID == 0 {
			continue
		}
		repo, err := repo_model.GetRepositoryByID(ctx, track.RepoID)
		if err != nil {
			continue
		}
		if err := release_service.CreateNewTag(ctx, doer, repo, repo.DefaultBranch, tagName, msg); err != nil {
			log.Warn("tagTrackRepos: tag %s on repo %d: %v", tagName, repo.ID, err)
		}
	}
}

// createTrackReleases creates a v1-results release on each track repo with a leaderboard summary.
func createTrackReleases(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) {
	doer, err := user_model.GetUserByID(ctx, doerID)
	if err != nil {
		return
	}
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	if err != nil {
		return
	}

	// Build leaderboard text
	subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
		HackathonID: h.ID,
	})
	var body strings.Builder
	body.WriteString("## Results\n\n")
	body.WriteString("| Rank | Project | Score |\n|------|---------|-------|\n")
	for _, s := range subs {
		if s.Rank > 0 {
			body.WriteString(fmt.Sprintf("| #%d | %s | %.1f |\n", s.Rank, s.Title, s.TotalScore))
		}
	}

	for _, track := range tracks {
		if track.RepoID == 0 {
			continue
		}
		repo, err := repo_model.GetRepositoryByID(ctx, track.RepoID)
		if err != nil {
			continue
		}
		gitRepo, closer, err := gitrepo.RepositoryFromContextOrOpen(ctx, repo)
		if err != nil {
			continue
		}
		rel := &repo_model.Release{
			RepoID:       repo.ID,
			PublisherID:  doer.ID,
			TagName:      "v1-results",
			Target:       repo.DefaultBranch,
			Title:        fmt.Sprintf("%s — %s Results", h.Name, track.Name),
			Note:         body.String(),
			IsDraft:      false,
			IsPrerelease: false,
		}
		if err := release_service.CreateRelease(gitRepo, rel, "", nil); err != nil {
			log.Warn("createTrackReleases: release on repo %d: %v", repo.ID, err)
		}
		closer.Close()
	}
}

func publishPhaseChange(ctx context.Context, doerID int64, h *hackforger_model.Hackathon, oldStatus, newStatus hackforger_model.HackathonStatus) {
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doerID,
		OpType:       hackforger_model.ActionHackathonPhaseChanged,
		EntityType:   "hackathon",
		EntityID:     h.ID,
		EntityName:   h.Name,
		EntitySlug:   h.Slug,
		AudienceType: AudienceOrgMembers,
		OrgID:        h.OrgID,
		Content: hackforger_model.HackforgerPhaseContent{
			HackforgerActionContent: hackforger_model.HackforgerActionContent{
				EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
			},
			OldStatus: hackathonStatusName[oldStatus],
			NewStatus: hackathonStatusName[newStatus],
		},
	})
}
