// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	organization_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
	actions_service "forgejo.org/services/actions"
	notify_service "forgejo.org/services/notify"
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

// ErrNoRegistrationPhase means hackathon has no registration phase (cannot publish).
type ErrNoRegistrationPhase struct{ HackathonID int64 }

func (e ErrNoRegistrationPhase) Error() string {
	return fmt.Sprintf("hackathon has no registration phase [id: %d]", e.HackathonID)
}

// IsErrNoRegistrationPhase checks if err is ErrNoRegistrationPhase.
func IsErrNoRegistrationPhase(err error) bool { _, ok := err.(ErrNoRegistrationPhase); return ok }

// ErrNoDevelopmentPhase means hackathon has no development phase (cannot publish).
type ErrNoDevelopmentPhase struct{ HackathonID int64 }

func (e ErrNoDevelopmentPhase) Error() string {
	return fmt.Sprintf("hackathon has no development phase [id: %d]", e.HackathonID)
}

// IsErrNoDevelopmentPhase checks if err is ErrNoDevelopmentPhase.
func IsErrNoDevelopmentPhase(err error) bool { _, ok := err.(ErrNoDevelopmentPhase); return ok }

// ErrNoCriteria means a hackathon or track has no scoring criteria — either
// no criteria row exists for the hackathon (TrackID == 0) or every criteria
// is disabled via hackathon_track_criteria for the given track (TrackID != 0).
type ErrNoCriteria struct {
	HackathonID int64
	TrackID     int64 // 0 = global; non-zero = specific track has no enabled criteria
}

func (e ErrNoCriteria) Error() string {
	if e.TrackID == 0 {
		return fmt.Sprintf("hackathon has no scoring criteria [id: %d]", e.HackathonID)
	}
	return fmt.Sprintf("track has no enabled scoring criteria [hackathon: %d, track: %d]", e.HackathonID, e.TrackID)
}

// Unwrap lets errors.Is detect invalid-argument and route HTTP 400 — aligns
// with the dominant pattern in services/hackforger/hackathon_criteria.go.
func (e ErrNoCriteria) Unwrap() error { return util.ErrInvalidArgument }

// IsErrNoCriteria checks if err is ErrNoCriteria.
func IsErrNoCriteria(err error) bool { _, ok := err.(ErrNoCriteria); return ok }

// ErrDuplicateSubmission means a user has already submitted to this track.
type ErrDuplicateSubmission struct {
	UserID  int64
	TrackID int64
}

func (e ErrDuplicateSubmission) Error() string {
	return fmt.Sprintf("duplicate submission [user: %d, track: %d]", e.UserID, e.TrackID)
}

// IsErrDuplicateSubmission checks if err is ErrDuplicateSubmission.
func IsErrDuplicateSubmission(err error) bool { _, ok := err.(ErrDuplicateSubmission); return ok }

// CreateHackathon creates a Forgejo Organization for the hackathon,
// then creates the hackathon record and publishes a creation event.
// ErrDuplicateHackathonName indicates a hackathon with the same name already exists.
type ErrDuplicateHackathonName struct{ Name string }

func (e ErrDuplicateHackathonName) Error() string {
	return fmt.Sprintf("hackathon name already exists: %s", e.Name)
}

func IsErrDuplicateHackathonName(err error) bool { _, ok := err.(ErrDuplicateHackathonName); return ok }

func CreateHackathon(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon) error {
	// Check for duplicate hackathon name
	existing, err := hackforger_model.GetHackathonBySlug(ctx, h.Slug)
	if err == nil && existing != nil {
		return ErrDuplicateHackathonName{Name: h.Name}
	}

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

	notify_service.HackforgerEntityCreated(ctx, doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionHackathonCreated,
		EntityType:   "hackathon",
		EntityID:     h.ID,
		EntityName:   h.Name,
		EntitySlug:   h.Slug,
		AudienceType: notify_service.AudienceGlobal,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
		},
	})
	return nil
}

// submissionIndexWorkflow is the Forgejo Actions workflow committed into every
// track repo. It is dispatched via workflow_dispatch whenever a new submission
// is created, and regenerates SUBMISSIONS.md + submissions.json from the API.
//
// IMPORTANT: editing this string only affects tracks created AFTER the edit.
// Existing tracks keep the YAML they were initialised with, because the file
// lives in their git history. Read docs/notes/forgejo-actions-yaml-drift.md
// before making behavior-changing edits — it covers when a backfill is
// required and how to write one.
const submissionIndexWorkflow = `name: Update Submission Index
on:
  workflow_dispatch:
    inputs:
      hackathon_id:
        description: 'Hackathon ID'
        required: true
        type: string
      track_id:
        description: 'Track ID'
        required: true
        type: string
      hackathon_slug:
        description: 'Hackathon slug for URLs'
        required: true
        type: string
      track_name:
        description: 'Track name for index header'
        required: true
        type: string

jobs:
  update-index:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repo
        run: |
          # Disable any system credential helper (e.g. macOS osxkeychain on
          # host-mode runners) — it can't store creds in a non-interactive
          # session and emits a noisy "failed to store: -25308" line. The
          # token is embedded in the push URL below, so no helper is needed.
          git config --global credential.helper ""
          git clone "${{ github.server_url }}/${{ github.repository }}.git" repo
          cd repo
          git config user.name "${{ github.actor }}"
          git config user.email "${{ github.actor }}@noreply.localhost"

      - name: Fetch submissions and generate index
        working-directory: repo
        env:
          API_BASE: ${{ github.server_url }}/api/v1/hackforger
          HACKATHON_ID: ${{ inputs.hackathon_id }}
          TRACK_ID: ${{ inputs.track_id }}
          HACKATHON_SLUG: ${{ inputs.hackathon_slug }}
          TRACK_NAME: ${{ inputs.track_name }}
          INSTANCE_URL: ${{ github.server_url }}
        run: |
          SUBS=$(curl -sf "${API_BASE}/hackathons/${HACKATHON_ID}/submissions" 2>/dev/null || echo '[]')
          TRACK_SUBS=$(echo "$SUBS" | jq --argjson tid "$TRACK_ID" '[.[] | select(.TrackID == $tid)]')

          # Generate SUBMISSIONS.md (API returns PascalCase field names)
          {
            echo "# ${TRACK_NAME} — Submissions"
            echo ""
            echo "Track for [hackathon](${INSTANCE_URL}/hackathon/${HACKATHON_SLUG})"
            echo ""
            echo "| # | Project | Author | Repo | Demo |"
            echo "|---|---------|--------|------|------|"
            echo "$TRACK_SUBS" | jq -r 'to_entries[] | "| \(.key + 1) | \(.value.Title) | User #\(.value.UserID) | \(if .value.RepoID > 0 then "repo" else "-" end) | \(if .value.DemoURL != "" then "[Demo](\(.value.DemoURL))" else "-" end) |"'
          } > SUBMISSIONS.md

          echo "$TRACK_SUBS" | jq '[.[] | {id: .ID, title: .Title, description: .Description, user_id: .UserID, repo_id: .RepoID, demo_url: .DemoURL}]' > submissions.json

      - name: Commit and push to main
        working-directory: repo
        env:
          GITHUB_TOKEN: ${{ github.token }}
          API_BASE: ${{ github.server_url }}/api/v1
          REPO: ${{ github.repository }}
        run: |
          git add SUBMISSIONS.md submissions.json
          if git diff --cached --quiet; then
            echo "No changes to commit"
            exit 0
          fi
          git commit -m "Update submission index"

          # Push directly to main (action token has push access).
          # No PR flow needed — this is an automated index update.
          SERVER_HOST="${{ github.server_url }}"
          SERVER_HOST="${SERVER_HOST#http://}"
          SERVER_HOST="${SERVER_HOST#https://}"
          PUSH_URL="http://x-access-token:${GITHUB_TOKEN}@${SERVER_HOST}/${{ github.repository }}.git"
          git push "$PUSH_URL" main
          echo "Pushed index update to main"
`

// trackRepoContributingTemplate is bilingual (zh-CN + en-US) so it works
// regardless of the platform's default locale. We write to CONTRIBUTING.md
// (NOT README.md) because the organizer may have provided their own
// description that auto-init wrote into README.md — clobbering it would lose
// their content, and trying to also create README.md would atomically fail
// the entire ChangeRepoFiles batch (also losing SUBMISSIONS.md + workflow).
// Forgejo highlights CONTRIBUTING.md prominently in the file list and on the
// "open Pull Request" flow, so contestants will discover it.
const trackRepoContributingTemplate = `# How to submit / 如何提交作品 — %s

## 中文

这是当前 HackForger 实例上 **%s** 活动的赛道仓库（track repository）。

**用途**：本仓库用来收集本赛道的所有参赛作品，并作为评委评审的代码基线。

**怎么提交作品**：
1. 在 web 页面上 **fork 本仓库**
2. 在你 fork 的版本里开发你的作品
3. 完成后，向 **本仓库** 发起 Pull Request
4. **回到活动页面，提交你的作品并填上 PR 链接**
   — 这一步会创建 submission 记录并触发索引更新
5. 之后你的提交会自动出现在 ` + "`SUBMISSIONS.md`" + ` 索引中

**仓库内容**：
- ` + "`SUBMISSIONS.md`" + `——所有提交的索引（自动维护，请勿手动编辑）
- ` + "`submissions.json`" + `——提交元数据（机器可读）
- ` + "`.forgejo/workflows/`" + `——自动化工作流定义

---

## English

This is the track repository for the **%s** event on this HackForger instance.

**Purpose**: This repo collects all submissions for this track and serves as
the code baseline for judges to review.

**How to submit**:
1. **Fork this repo** in the web UI
2. Develop your project in your fork
3. When done, open a **Pull Request back to this repo**
4. **Go back to the event page and submit your work, pasting
   the PR URL** — this creates a submission record and triggers an index update
5. Your submission will then appear automatically in ` + "`SUBMISSIONS.md`" + `

**What's in this repo**:
- ` + "`SUBMISSIONS.md`" + ` — auto-maintained index of all submissions (do not edit by hand)
- ` + "`submissions.json`" + ` — machine-readable submission metadata
- ` + "`.forgejo/workflows/`" + ` — automation workflow definitions
`

// phasesToMilestones converts hackathon phases into Milestone records for the
// track repo. Phases with EndTime == 0 (unscheduled) produce no milestone —
// a milestone without a deadline has no semantic meaning in HackForger's
// phase-marker model (unlike Forgejo's user-authored milestones, which are
// meaningful as issue-grouping buckets even without deadlines).
func phasesToMilestones(phases []*hackforger_model.Phase, repoID int64) []*issues_model.Milestone {
	result := make([]*issues_model.Milestone, 0, len(phases))
	for _, p := range phases {
		if p.EndTime == 0 {
			continue
		}
		name := p.CustomName
		if name == "" && p.PhaseType != nil {
			name = p.PhaseType.Key // "registration", "development", "judging", "results"
		}
		result = append(result, &issues_model.Milestone{
			RepoID:       repoID,
			Name:         name,
			DeadlineUnix: timeutil.TimeStamp(p.EndTime),
		})
	}
	return result
}

// CreateTrackWithRepo creates a Forgejo Repository in the hackathon's linked
// Organization for the track, then inserts the track record.
// The repo is named "track-{ID}" to avoid issues with Unicode/Chinese track names.
func CreateTrackWithRepo(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, track *hackforger_model.HackathonTrack) error {
	if h.LinkedOrgID == 0 {
		// No linked org — just create track record without repo
		return hackforger_model.CreateTrack(ctx, track)
	}

	// Insert track first to get auto-increment ID for a safe repo name
	if err := hackforger_model.CreateTrack(ctx, track); err != nil {
		return err
	}

	orgUser, err := user_model.GetUserByID(ctx, h.LinkedOrgID)
	if err != nil {
		// Clean up: delete the track record we just created
		_ = hackforger_model.DeleteTrack(ctx, track.ID)
		return fmt.Errorf("get hackathon org user: %w", err)
	}

	repoName := fmt.Sprintf("track-%d", track.ID)
	repo, err := repo_service.CreateRepository(ctx, doer, orgUser, repo_service.CreateRepoOptions{
		Name:          repoName,
		Description:   track.Description,
		Readme:        "Default",
		AutoInit:      true,
		DefaultBranch: "main",
	})
	if err != nil {
		// Clean up: delete the track record we just created
		_ = hackforger_model.DeleteTrack(ctx, track.ID)
		return fmt.Errorf("create track repo: %w", err)
	}

	track.RepoID = repo.ID
	if err := hackforger_model.UpdateTrack(ctx, track); err != nil {
		return err
	}

	// Initialize submission index files in the track repo
	_, closer, err := gitrepo.RepositoryFromContextOrOpen(ctx, repo)
	if err == nil {
		defer closer.Close()
		_, initErr := files_service.ChangeRepoFiles(ctx, repo, doer, &files_service.ChangeRepoFilesOptions{
			OldBranch: repo.DefaultBranch,
			NewBranch: repo.DefaultBranch,
			Message:   "Initialize track repository",
			Files: []*files_service.ChangeRepoFile{
				{
					// CONTRIBUTING.md (not README.md) — see comment on
					// trackRepoContributingTemplate for why.
					Operation:     "create",
					TreePath:      "CONTRIBUTING.md",
					ContentReader: strings.NewReader(fmt.Sprintf(trackRepoContributingTemplate, track.Name, h.Name, h.Name)),
				},
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
				{
					Operation:     "create",
					TreePath:      ".forgejo/workflows/update-submission-index.yml",
					ContentReader: strings.NewReader(submissionIndexWorkflow),
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

	// Auto-create phase milestones on the track repo from Phase records.
	// Phases without EndTime are skipped; see phasesToMilestones for rationale.
	phases, phaseErr := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
	if phaseErr == nil {
		for _, m := range phasesToMilestones(phases, repo.ID) {
			_ = issues_model.NewMilestone(ctx, m)
		}
	}
	return nil
}

// PublishHackathon sets IsPublished=true after validating phases and tracks.
// StatusCache will be updated by SyncStatusCache when the first phase's StartTime arrives.
func PublishHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.IsPublished {
		return fmt.Errorf("hackathon is already published [id: %d]", h.ID)
	}

	// Validate tracks
	trackCount, err := hackforger_model.CountTracksByHackathon(ctx, h.ID)
	if err != nil {
		return err
	}
	if trackCount == 0 {
		return ErrNoTracks{HackathonID: h.ID}
	}

	// Validate phases
	phases, err := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
	if err != nil {
		return err
	}
	hasRegistration := false
	hasDevelopment := false
	for _, p := range phases {
		if p.PhaseType == nil {
			continue
		}
		switch p.PhaseType.Key {
		case "registration":
			hasRegistration = true
		case "development":
			hasDevelopment = true
		}
	}
	if !hasRegistration {
		return ErrNoRegistrationPhase{HackathonID: h.ID}
	}
	if !hasDevelopment {
		return ErrNoDevelopmentPhase{HackathonID: h.ID}
	}

	// Validate scoring criteria — at least one criterion required globally.
	criteriaList, err := hackforger_model.ListCriteriaByHackathon(ctx, h.ID)
	if err != nil {
		return err
	}
	if len(criteriaList) == 0 {
		return ErrNoCriteria{HackathonID: h.ID}
	}

	// Each track must have at least one enabled criterion (via effective rubric).
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	if err != nil {
		return err
	}
	for _, t := range tracks {
		rubric, err := GetEffectiveRubric(ctx, t.ID)
		if err != nil {
			return err
		}
		if len(rubric) == 0 {
			return ErrNoCriteria{HackathonID: h.ID, TrackID: t.ID}
		}
	}

	// Set published
	if err := hackforger_model.SetIsPublished(ctx, h.ID, true); err != nil {
		return err
	}
	h.IsPublished = true

	// Sync status (may transition if first phase already started)
	return SyncStatusCache(ctx, h.ID, doerID)
}

// PreviewFinalize calculates rankings for all tracks without changing status.
// Idempotent — can be called multiple times.
func PreviewFinalize(ctx context.Context, hackathonID int64) (map[int64][]RankedSubmission, error) {
	h, err := hackforger_model.GetHackathonByID(ctx, hackathonID)
	if err != nil {
		return nil, err
	}
	if h.StatusCache != hackforger_model.HackathonStatusJudging {
		return nil, hackforger_model.ErrInvalidHackathonPhase{
			HackathonID: h.ID, Current: h.StatusCache, Expected: hackforger_model.HackathonStatusJudging,
		}
	}
	return CalculateRanks(ctx, hackathonID)
}

// ConfirmFinalize locks status to Finished, persists ranks, creates releases.
func ConfirmFinalize(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.StatusCache != hackforger_model.HackathonStatusJudging {
		return hackforger_model.ErrInvalidHackathonPhase{
			HackathonID: h.ID, Current: h.StatusCache, Expected: hackforger_model.HackathonStatusJudging,
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
	if err := distributeHackathonCredits(ctx, h.ID); err != nil {
		return err
	}

	// Update status
	if err := hackforger_model.UpdateHackathonStatusCache(ctx, h.ID, hackforger_model.HackathonStatusFinished); err != nil {
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
	if doer, err := user_model.GetUserByID(ctx, doerID); err == nil {
		notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
			OpType:       hackforger_model.ActionHackathonFinalized,
			EntityType:   "hackathon",
			EntityID:     h.ID,
			EntityName:   h.Name,
			EntitySlug:   h.Slug,
			AudienceType: notify_service.AudienceGlobal,
			Content: hackforger_model.HackforgerActionContent{
				EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
				Extra: map[string]any{"tracks": trackWinners},
			},
		})
	}
	return nil
}

// CancelHackathon transitions to Cancelled. Any non-Finished status.
func CancelHackathon(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
	if h.StatusCache == hackforger_model.HackathonStatusFinished {
		return hackforger_model.ErrInvalidHackathonPhase{HackathonID: h.ID, Current: h.StatusCache, Expected: 0}
	}
	if h.StatusCache == hackforger_model.HackathonStatusCancelled {
		return hackforger_model.ErrInvalidHackathonPhase{HackathonID: h.ID, Current: h.StatusCache, Expected: 0}
	}
	oldStatus := h.StatusCache
	if err := hackforger_model.UpdateHackathonStatusCache(ctx, h.ID, hackforger_model.HackathonStatusCancelled); err != nil {
		return err
	}
	// Delete all future phases
	phases, _ := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
	for _, p := range phases {
		if p.IsFuture() {
			_ = hackforger_model.DeletePhase(ctx, p.ID)
		}
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
// of). If the submission targets a track with a linked repository, a Forgejo
// Actions workflow is dispatched to update SUBMISSIONS.md and submissions.json.
func CreateSubmission(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, sub *hackforger_model.HackathonSubmission) error {
	// 0. Check for duplicate submission per user+track
	if sub.TrackID > 0 {
		exists, err := hackforger_model.SubmissionExistsByUserAndTrack(ctx, doer.ID, sub.TrackID)
		if err != nil {
			return err
		}
		if exists {
			return ErrDuplicateSubmission{UserID: doer.ID, TrackID: sub.TrackID}
		}
	}

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

	// 3. If track has a repo, auto-watch and trigger workflow
	if sub.TrackID > 0 {
		track, err := hackforger_model.GetTrackByID(ctx, sub.TrackID)
		if err == nil && track.RepoID > 0 {
			_ = repo_model.WatchRepo(ctx, doer.ID, track.RepoID, true)
			triggerSubmissionIndexUpdate(ctx, doer, h, track)
		}
	}

	publishSubmissionEvent(ctx, doer, h, sub)
	return nil
}

// DeleteSubmission deletes a submission and triggers index re-sync on the track repo.
func DeleteSubmission(ctx context.Context, doer *user_model.User, submissionID int64) error {
	sub, err := hackforger_model.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return err
	}
	h, err := hackforger_model.GetHackathonByID(ctx, sub.HackathonID)
	if err != nil {
		return err
	}

	// Delete associated judge scores first
	if _, err := db.GetEngine(ctx).Where("submission_id = ?", sub.ID).Delete(new(hackforger_model.HackathonJudgeScore)); err != nil {
		return fmt.Errorf("delete submission scores: %w", err)
	}

	if err := hackforger_model.DeleteSubmission(ctx, sub.ID); err != nil {
		return err
	}

	// Re-sync track index (the workflow will regenerate without the deleted submission)
	if sub.TrackID > 0 {
		track, err := hackforger_model.GetTrackByID(ctx, sub.TrackID)
		if err == nil && track.RepoID > 0 {
			triggerSubmissionIndexUpdate(ctx, doer, h, track)
		}
	}

	return nil
}

// triggerSubmissionIndexUpdate dispatches the update-submission-index workflow
// in the track repo via the Forgejo Actions internal API. The workflow handles
// fetching submissions, regenerating SUBMISSIONS.md + submissions.json, and
// pushing the update directly to main -- all within the Actions runner, not
// in Go code. Existing track repos still carry the old workflow YAML; the new
// version only ships with tracks created after this change.
func triggerSubmissionIndexUpdate(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, track *hackforger_model.HackathonTrack) {
	if track.RepoID == 0 {
		return
	}
	baseRepo, err := repo_model.GetRepositoryByID(ctx, track.RepoID)
	if err != nil {
		log.Warn("triggerSubmissionIndexUpdate: get track repo: %v", err)
		return
	}
	if err := baseRepo.LoadOwner(ctx); err != nil {
		log.Warn("triggerSubmissionIndexUpdate: load repo owner: %v", err)
		return
	}

	// Open the git repo to locate the workflow file on the default branch
	gitRepo, closer, err := gitrepo.RepositoryFromContextOrOpen(ctx, baseRepo)
	if err != nil {
		log.Warn("triggerSubmissionIndexUpdate: open git repo: %v", err)
		return
	}
	defer closer.Close()

	workflow, err := actions_service.GetWorkflowFromCommit(gitRepo, baseRepo.DefaultBranch, "update-submission-index.yml")
	if err != nil {
		log.Warn("triggerSubmissionIndexUpdate: get workflow: %v", err)
		return
	}

	// Build the inputs map for the workflow_dispatch event
	inputs := map[string]string{
		"hackathon_id":   strconv.FormatInt(h.ID, 10),
		"track_id":       strconv.FormatInt(track.ID, 10),
		"hackathon_slug": h.Slug,
		"track_name":     track.Name,
	}
	inputGetter := func(key string) string {
		return inputs[key]
	}

	// Dispatch as the hackathon owner (a human user with admin privileges on the org).
	// baseRepo.Owner is the org itself (type=1), which cannot merge PRs.
	// Using the submitter (doer) would also fail because hackers lack write access.
	hackathonOwner, err := user_model.GetUserByID(ctx, h.OwnerID)
	if err != nil {
		log.Warn("triggerSubmissionIndexUpdate: get hackathon owner: %v", err)
		return
	}
	_, _, err = workflow.Dispatch(ctx, inputGetter, baseRepo, hackathonOwner)
	if err != nil {
		log.Warn("triggerSubmissionIndexUpdate: dispatch workflow: %v", err)
		return
	}

	log.Info("triggerSubmissionIndexUpdate: dispatched workflow for hackathon %d track %d", h.ID, track.ID)
}

func publishSubmissionEvent(ctx context.Context, doer *user_model.User, h *hackforger_model.Hackathon, sub *hackforger_model.HackathonSubmission) {
	notify_service.HackforgerEntityCreated(ctx, doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionHackathonSubmitted,
		EntityType:   "hackathon",
		EntityID:     h.ID,
		EntityName:   h.Name,
		EntitySlug:   h.Slug,
		AudienceType: notify_service.AudienceFollowers,
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

// distributeHackathonCredits awards prize credits to hackathon winners per track.
// It supports three distribution modes:
//   - winner_takes_all: rank 1 gets all PrizeCredits
//   - tiered: each rank gets PrizeCredits * pct / 100, remainder to rank 1
//   - equal: PrizeCredits / N per winner, remainder to rank 1
func distributeHackathonCredits(ctx context.Context, hackathonID int64) error {
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, hackathonID)
	if err != nil {
		return err
	}

	for _, track := range tracks {
		if track.PrizeCredits <= 0 {
			continue
		}

		subs, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
			ListOptions: db.ListOptions{ListAll: true},
			TrackID:     track.ID,
		})
		if err != nil {
			return err
		}

		// Sort by TotalScore DESC, ID ASC (per-track ranking)
		sort.Slice(subs, func(i, j int) bool {
			if subs[i].TotalScore != subs[j].TotalScore {
				return subs[i].TotalScore > subs[j].TotalScore
			}
			return subs[i].ID < subs[j].ID
		})

		// Filter out unscored submissions (TotalScore <= 0)
		var scored []*hackforger_model.HackathonSubmission
		for _, s := range subs {
			if s.TotalScore > 0 {
				scored = append(scored, s)
			}
		}
		if len(scored) == 0 {
			continue
		}

		switch track.PrizeDistMode {
		case "winner_takes_all":
			ref := fmt.Sprintf("hackathon:%d/track:%d:rank:1", hackathonID, track.ID)
			note := fmt.Sprintf("Hackathon prize: %s (winner takes all)", track.Name)
			if err := Deposit(ctx, scored[0].UserID, track.PrizeCredits, ref, note); err != nil {
				return err
			}

		case "tiered":
			ratios, err := hackforger_model.ParsePrizeDistRatios(track.PrizeDistRatios)
			if err != nil {
				return err
			}
			var distributed int64
			for i, ratio := range ratios {
				if i >= len(scored) {
					break
				}
				amount := track.PrizeCredits * int64(ratio.Pct) / 100
				distributed += amount
				ref := fmt.Sprintf("hackathon:%d/track:%d:rank:%d", hackathonID, track.ID, ratio.Rank)
				note := fmt.Sprintf("Hackathon prize: %s (rank %d, %d%%)", track.Name, ratio.Rank, ratio.Pct)
				if err := Deposit(ctx, scored[i].UserID, amount, ref, note); err != nil {
					return err
				}
			}
			// Remainder (rounding residue) goes to rank 1
			remainder := track.PrizeCredits - distributed
			if remainder > 0 {
				ref := fmt.Sprintf("hackathon:%d/track:%d:rank:1:remainder", hackathonID, track.ID)
				note := fmt.Sprintf("Hackathon prize: %s (rounding remainder)", track.Name)
				if err := Deposit(ctx, scored[0].UserID, remainder, ref, note); err != nil {
					return err
				}
			}

		case "equal":
			n := int64(len(scored))
			perUser := track.PrizeCredits / n
			distributed := perUser * n
			for i, s := range scored {
				ref := fmt.Sprintf("hackathon:%d/track:%d:rank:%d", hackathonID, track.ID, i+1)
				note := fmt.Sprintf("Hackathon prize: %s (equal split, rank %d)", track.Name, i+1)
				if err := Deposit(ctx, s.UserID, perUser, ref, note); err != nil {
					return err
				}
			}
			// Remainder goes to rank 1
			remainder := track.PrizeCredits - distributed
			if remainder > 0 {
				ref := fmt.Sprintf("hackathon:%d/track:%d:rank:1:remainder", hackathonID, track.ID)
				note := fmt.Sprintf("Hackathon prize: %s (equal split remainder)", track.Name)
				if err := Deposit(ctx, scored[0].UserID, remainder, ref, note); err != nil {
					return err
				}
			}

		default:
			return hackforger_model.ErrInvalidPrizeDistMode{Mode: track.PrizeDistMode}
		}
	}

	return nil
}

func publishPhaseChange(ctx context.Context, doerID int64, h *hackforger_model.Hackathon, oldStatus, newStatus hackforger_model.HackathonStatus) {
	if doer, err := user_model.GetUserByID(ctx, doerID); err == nil {
		notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
			OpType:       hackforger_model.ActionHackathonPhaseChanged,
			EntityType:   "hackathon",
			EntityID:     h.ID,
			EntityName:   h.Name,
			EntitySlug:   h.Slug,
			AudienceType: notify_service.AudienceOrgMembers,
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
}
