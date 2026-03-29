// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"
	"strings"

	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	organization_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"
	pull_service "forgejo.org/services/pull"
	release_service "forgejo.org/services/release"
	repo_service "forgejo.org/services/repository"
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
		return fmt.Errorf("hackathon must be in Open status to start hacking [id: %d, status: %d]", h.ID, h.Status)
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
		return fmt.Errorf("hackathon must be in Hacking status to start judging [id: %d, status: %d]", h.ID, h.Status)
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

// FinalizeHackathon transitions Judging → Finished. Calculates ranks.
func FinalizeHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status != hackforger_model.HackathonStatusJudging {
		return fmt.Errorf("hackathon must be in Judging status to finalize [id: %d, status: %d]", h.ID, h.Status)
	}
	// Tag track repos with submission-deadline snapshot before rank calculation
	tagTrackRepos(ctx, doerID, h.ID, "submission-deadline", "Submission deadline snapshot")
	if err := CalculateRanks(ctx, h.ID); err != nil {
		return err
	}
	if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusFinished); err != nil {
		return err
	}
	// Create v1-results release (also creates the tag) on each track repo
	createTrackReleases(ctx, doerID, h)
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doerID,
		OpType:       hackforger_model.ActionHackathonFinalized,
		AudienceType: AudienceGlobal,
		Content: hackforger_model.HackforgerPhaseContent{
			HackforgerActionContent: hackforger_model.HackforgerActionContent{
				EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
			},
			OldStatus: "judging", NewStatus: "finished",
		},
	})
	return nil
}

// CancelHackathon transitions to Cancelled. Any non-Finished status.
func CancelHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.Status == hackforger_model.HackathonStatusFinished {
		return fmt.Errorf("cannot cancel a finished hackathon [id: %d]", h.ID)
	}
	if h.Status == hackforger_model.HackathonStatusCancelled {
		return fmt.Errorf("hackathon is already cancelled [id: %d]", h.ID)
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

// CreateSubmission creates a hackathon submission. When the submission targets
// a track that has a linked repository, the track repo is forked to the user's
// space and a pull request is opened from the fork back to the track repo.
func CreateSubmission(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, sub *hackforger_model.HackathonSubmission) error {
	// If no track selected, just create a basic submission record.
	if sub.TrackID == 0 {
		if err := hackforger_model.CreateSubmission(ctx, sub); err != nil {
			return err
		}
		publishSubmissionEvent(ctx, doer, h, sub)
		return nil
	}

	track, err := hackforger_model.GetTrackByID(ctx, sub.TrackID)
	if err != nil || track.RepoID == 0 {
		// Track doesn't exist or has no repo — save without fork/PR.
		if err := hackforger_model.CreateSubmission(ctx, sub); err != nil {
			return err
		}
		publishSubmissionEvent(ctx, doer, h, sub)
		return nil
	}

	baseRepo, err := repo_model.GetRepositoryByID(ctx, track.RepoID)
	if err != nil {
		return fmt.Errorf("get track repo: %w", err)
	}
	if err := baseRepo.LoadOwner(ctx); err != nil {
		return fmt.Errorf("load track repo owner: %w", err)
	}

	// Fork the track repo to the participant's personal space.
	fork, err := repo_service.ForkRepositoryIfNotExists(ctx, doer, doer, repo_service.ForkRepoOptions{
		BaseRepo:    baseRepo,
		Name:        baseRepo.Name + "-" + doer.LowerName,
		Description: sub.Title,
	})
	if err != nil {
		if repo_service.IsErrForkAlreadyExist(err) {
			fork, err = repo_model.GetUserFork(ctx, baseRepo.ID, doer.ID)
			if err != nil {
				return fmt.Errorf("get existing fork: %w", err)
			}
		} else {
			return fmt.Errorf("fork track repo: %w", err)
		}
	}
	sub.ForkRepoID = fork.ID
	sub.RepoID = fork.ID

	// Create a pull request from the fork back to the track repo.
	issue := &issues_model.Issue{
		RepoID:   baseRepo.ID,
		Title:    sub.Title,
		Content:  sub.Description,
		PosterID: doer.ID,
	}
	pr := &issues_model.PullRequest{
		HeadRepoID: fork.ID,
		BaseRepoID: baseRepo.ID,
		HeadBranch: fork.DefaultBranch,
		BaseBranch: baseRepo.DefaultBranch,
		Type:       issues_model.PullRequestGitea,
	}
	if err := pull_service.NewPullRequest(ctx, baseRepo, issue, nil, nil, pr, nil); err != nil {
		// PR creation may fail if fork has no diff yet — that is acceptable.
		log.Warn("CreateSubmission: PR creation failed (may be no diff): %v", err)
	} else {
		sub.PRID = pr.ID
		sub.PullIndex = issue.Index
	}

	if err := hackforger_model.CreateSubmission(ctx, sub); err != nil {
		return err
	}

	publishSubmissionEvent(ctx, doer, h, sub)
	return nil
}

func publishSubmissionEvent(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, sub *hackforger_model.HackathonSubmission) {
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    doer.ID,
		OpType:       hackforger_model.ActionHackathonSubmitted,
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
