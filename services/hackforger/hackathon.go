// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"
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
	actions_service "forgejo.org/services/actions"
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

// submissionIndexWorkflow is the Forgejo Actions workflow committed into every
// track repo. It is dispatched via workflow_dispatch whenever a new submission
// is created, and regenerates SUBMISSIONS.md + submissions.json from the API.
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
          git clone "${{ github.server_url }}/${{ github.repository }}.git" repo
          cd repo
          git config user.name "HackForger Bot"
          git config user.email "noreply@hackforger"

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
          TRACK_SUBS=$(echo "$SUBS" | jq --argjson tid "$TRACK_ID" '[.[] | select(.track_id == $tid)]')

          # Generate SUBMISSIONS.md
          {
            echo "# ${TRACK_NAME} — Submissions"
            echo ""
            echo "Track for [hackathon](${INSTANCE_URL}/hackathon/${HACKATHON_SLUG})"
            echo ""
            echo "| # | Project | Author | Repo | Demo |"
            echo "|---|---------|--------|------|------|"
            echo "$TRACK_SUBS" | jq -r 'to_entries[] | "| \(.key + 1) | \(.value.title) | User #\(.value.user_id) | \(if .value.repo_id > 0 then \"repo\" else \"-\" end) | \(if .value.demo_url != \"\" then \"[Demo](\(.value.demo_url))\" else \"-\" end) |"'
          } > SUBMISSIONS.md

          echo "$TRACK_SUBS" | jq '[.[] | {id, title, description, user_id, repo_id, demo_url}]' > submissions.json

      - name: Commit, push, create PR, and auto-merge
        working-directory: repo
        env:
          GITHUB_TOKEN: ${{ github.token }}
          API_BASE: ${{ github.server_url }}/api/v1
          REPO: ${{ github.repository }}
        run: |
          BRANCH="submission-index-update"
          git checkout -b "$BRANCH" 2>/dev/null || git checkout "$BRANCH"
          git add SUBMISSIONS.md submissions.json
          if git diff --cached --quiet; then
            echo "No changes to commit"
            exit 0
          fi
          git commit -m "Update submission index"
          git push "http://x-access-token:${GITHUB_TOKEN}@$(echo '${{ github.server_url }}' | sed 's|https\?://||')/${{ github.repository }}.git" "$BRANCH" --force

          # Create or update PR
          EXISTING=$(curl -sf "${API_BASE}/repos/${REPO}/pulls?state=open&head=${BRANCH}&base=main" \
            -H "Authorization: token ${GITHUB_TOKEN}" 2>/dev/null || echo '[]')
          PR_COUNT=$(echo "$EXISTING" | jq 'length')

          if [ "$PR_COUNT" -gt "0" ]; then
            PR_NUMBER=$(echo "$EXISTING" | jq '.[0].number')
            echo "Existing PR #${PR_NUMBER} updated"
          else
            PR_RESPONSE=$(curl -sf -X POST "${API_BASE}/repos/${REPO}/pulls" \
              -H "Authorization: token ${GITHUB_TOKEN}" \
              -H "Content-Type: application/json" \
              -d "{\"title\":\"Update submission index\",\"head\":\"${BRANCH}\",\"base\":\"main\",\"body\":\"Automated update\"}" 2>/dev/null || echo '{}')
            PR_NUMBER=$(echo "$PR_RESPONSE" | jq '.number // empty')
            if [ -z "$PR_NUMBER" ]; then
              echo "PR creation failed"
              exit 0
            fi
            echo "Created PR #${PR_NUMBER}"
          fi

          # Auto-merge
          curl -sf -X POST "${API_BASE}/repos/${REPO}/pulls/${PR_NUMBER}/merge" \
            -H "Authorization: token ${GITHUB_TOKEN}" \
            -H "Content-Type: application/json" \
            -d '{"Do":"merge","merge_message_field":"Auto-merge submission index update"}' || echo "Merge skipped"
`

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
// of). If the submission targets a track with a linked repository, a Forgejo
// Actions workflow is dispatched to update SUBMISSIONS.md and submissions.json.
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

	// 3. If track has a repo, trigger workflow to update submission index
	if sub.TrackID > 0 {
		track, err := hackforger_model.GetTrackByID(ctx, sub.TrackID)
		if err == nil && track.RepoID > 0 {
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
// creating a PR -- all within the Actions runner, not in Go code.
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

	_, _, err = workflow.Dispatch(ctx, inputGetter, baseRepo, doer)
	if err != nil {
		log.Warn("triggerSubmissionIndexUpdate: dispatch workflow: %v", err)
		return
	}

	log.Info("triggerSubmissionIndexUpdate: dispatched workflow for hackathon %d track %d", h.ID, track.ID)
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
