// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	issue_service "forgejo.org/services/issue"
	notify_service "forgejo.org/services/notify"
)

// postBountyIssueComment adds a timeline comment to the bounty's linked Issue
// when a bounty event occurs (created, applied, accepted, completed, etc.).
// Errors are logged but not propagated — issue comments are best-effort and
// should not block bounty operations.
func postBountyIssueComment(ctx context.Context, bounty *hackforger_model.Bounty, doerID int64, message string) {
	doer, err := user_model.GetUserByID(ctx, doerID)
	if err != nil {
		log.Error("postBountyIssueComment: GetUserByID(%d): %v", doerID, err)
		return
	}
	issue, err := issues_model.GetIssueByID(ctx, bounty.IssueID)
	if err != nil {
		log.Error("postBountyIssueComment: GetIssueByID(%d): %v", bounty.IssueID, err)
		return
	}
	if err := issue.LoadRepo(ctx); err != nil {
		log.Error("postBountyIssueComment: LoadRepo for issue %d: %v", bounty.IssueID, err)
		return
	}
	if _, err := issue_service.CreateIssueComment(ctx, doer, issue.Repo, issue, message, nil); err != nil {
		log.Error("postBountyIssueComment: CreateIssueComment for issue %d: %v", bounty.IssueID, err)
	}
}

// closeBountyIssue closes the Issue linked to a bounty when it completes.
func closeBountyIssue(ctx context.Context, bounty *hackforger_model.Bounty, doerID int64) {
	issue, err := issues_model.GetIssueByID(ctx, bounty.IssueID)
	if err != nil {
		log.Error("closeBountyIssue: GetIssueByID(%d): %v", bounty.IssueID, err)
		return
	}
	if issue.IsClosed {
		return // already closed (e.g., by PR merge)
	}
	doer, err := user_model.GetUserByID(ctx, doerID)
	if err != nil {
		log.Error("closeBountyIssue: GetUserByID(%d): %v", doerID, err)
		return
	}
	if err := issue_service.ChangeStatus(ctx, issue, doer, "", true); err != nil {
		log.Error("closeBountyIssue: ChangeStatus(%d): %v", issue.ID, err)
	}
}

// ErrInvalidBountyStatus is returned when a state transition is not allowed.
type ErrInvalidBountyStatus struct {
	BountyID int64
	Current  hackforger_model.BountyStatus
	Expected string
}

func (err ErrInvalidBountyStatus) Error() string {
	return fmt.Sprintf("invalid bounty status [bounty_id: %d, current: %d, expected: %s]",
		err.BountyID, err.Current, err.Expected)
}

// IsErrInvalidBountyStatus checks if an error is a ErrInvalidBountyStatus.
func IsErrInvalidBountyStatus(err error) bool {
	_, ok := err.(ErrInvalidBountyStatus)
	return ok
}

// ErrNotPublisher is returned when the doer is not the bounty publisher.
type ErrNotPublisher struct {
	BountyID    int64
	DoerID      int64
	PublisherID int64
}

func (err ErrNotPublisher) Error() string {
	return fmt.Sprintf("user is not the bounty publisher [bounty_id: %d, doer: %d, publisher: %d]",
		err.BountyID, err.DoerID, err.PublisherID)
}

// IsErrNotPublisher checks if an error is a ErrNotPublisher.
func IsErrNotPublisher(err error) bool {
	_, ok := err.(ErrNotPublisher)
	return ok
}

// ErrBountyHasApplications is returned when trying to delete a bounty that has applications.
type ErrBountyHasApplications struct {
	BountyID int64
}

func (err ErrBountyHasApplications) Error() string {
	return fmt.Sprintf("bounty has applications and cannot be deleted [bounty_id: %d]", err.BountyID)
}

// IsErrBountyHasApplications checks if an error is a ErrBountyHasApplications.
func IsErrBountyHasApplications(err error) bool {
	_, ok := err.(ErrBountyHasApplications)
	return ok
}

// WinnerInput holds the data needed to record a bounty winner.
type WinnerInput struct {
	UserID int64
	Rank   int
}

// requirePublisher returns ErrNotPublisher if doerID != bounty.PublisherID.
func requirePublisher(bounty *hackforger_model.Bounty, doerID int64) error {
	if bounty.PublisherID != doerID {
		return ErrNotPublisher{
			BountyID:    bounty.ID,
			DoerID:      doerID,
			PublisherID: bounty.PublisherID,
		}
	}
	return nil
}

// ApplyForBounty creates a new application for a bounty.
// The bounty must be in Open status.
func ApplyForBounty(ctx context.Context, bountyID, userID int64, message string) (*hackforger_model.BountyApplication, error) {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return nil, err
	}

	// Phase gating: check if apply action is allowed in current phase (no-op if no phases configured)
	if allowed, _ := AllowsAction(ctx, "bounty", bountyID, "apply"); !allowed {
		return nil, ErrInvalidBountyStatus{BountyID: bountyID, Current: bounty.Status, Expected: "Open (phase)"}
	}

	if bounty.Status != hackforger_model.BountyStatusOpen {
		return nil, ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Open",
		}
	}

	app := &hackforger_model.BountyApplication{
		BountyID: bountyID,
		UserID:   userID,
		Status:   hackforger_model.ApplicationStatusPending,
		Message:  message,
	}
	if err := hackforger_model.CreateBountyApplication(ctx, app); err != nil {
		return nil, err
	}
	// Auto-watch: applicant receives milestone events for this bounty's repo
	_ = repo_model.WatchRepo(ctx, userID, bounty.RepoID, true)
	postBountyIssueComment(ctx, bounty, userID, fmt.Sprintf("📋 **Bounty Application** — user #%d applied", userID))
	return app, nil
}

// AcceptApplication accepts an application. For Exclusive bounties, this also
// rejects all other pending applications, sets the claimer, and transitions
// the bounty to Claimed. For Competitive bounties, it just accepts the application.
func AcceptApplication(ctx context.Context, applicationID, doerID int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		app, err := hackforger_model.GetBountyApplicationByID(ctx, applicationID)
		if err != nil {
			return err
		}

		bounty, err := hackforger_model.GetBountyByID(ctx, app.BountyID)
		if err != nil {
			return err
		}

		if err := requirePublisher(bounty, doerID); err != nil {
			return err
		}

		if bounty.Status != hackforger_model.BountyStatusOpen {
			return ErrInvalidBountyStatus{
				BountyID: bounty.ID,
				Current:  bounty.Status,
				Expected: "Open",
			}
		}

		// Accept this application.
		app.Status = hackforger_model.ApplicationStatusAccepted
		if err := hackforger_model.UpdateBountyApplication(ctx, app); err != nil {
			return err
		}

		if bounty.Mode == hackforger_model.BountyModeExclusive {
			// Reject all other pending applications.
			pendingStatus := hackforger_model.ApplicationStatusPending
			others, _, err := hackforger_model.ListBountyApplications(ctx, hackforger_model.ListBountyApplicationsOptions{
				BountyID: bounty.ID,
				Status:   &pendingStatus,
			})
			if err != nil {
				return err
			}
			for _, other := range others {
				if other.ID == applicationID {
					continue
				}
				other.Status = hackforger_model.ApplicationStatusRejected
				if err := hackforger_model.UpdateBountyApplication(ctx, other); err != nil {
					return err
				}
			}

			// Transition bounty to Claimed.
			bounty.Status = hackforger_model.BountyStatusClaimed
			bounty.ClaimerID = app.UserID
			if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
				return err
			}
			// Auto-watch: claimer receives milestone events for this bounty's repo
			_ = repo_model.WatchRepo(ctx, app.UserID, bounty.RepoID, true)

			// Publish feed event.
			if doer, err := user_model.GetUserByID(ctx, doerID); err == nil {
				notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
					OpType:       hackforger_model.ActionBountyClaimed,
					EntityType:   "bounty",
					EntityID:     bounty.ID,
					EntityName:   bounty.Title,
					RepoID:       bounty.RepoID,
					AudienceType: notify_service.AudienceFollowers | notify_service.AudienceRepoWatchers,
					Content: &hackforger_model.HackforgerPhaseContent{
						HackforgerActionContent: hackforger_model.HackforgerActionContent{
							EntityType: "bounty",
							EntityID:   bounty.ID,
							EntityName: bounty.Title,
						},
						OldStatus: "open",
						NewStatus: "claimed",
					},
				})
			}
		}
		postBountyIssueComment(ctx, bounty, doerID, fmt.Sprintf("🏷 **Bounty Accepted** — claimed by user #%d", app.UserID))

		return nil
	})
}

// RejectApplication rejects an application. The doer must be the bounty publisher.
func RejectApplication(ctx context.Context, applicationID, doerID int64) error {
	app, err := hackforger_model.GetBountyApplicationByID(ctx, applicationID)
	if err != nil {
		return err
	}

	bounty, err := hackforger_model.GetBountyByID(ctx, app.BountyID)
	if err != nil {
		return err
	}

	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}

	app.Status = hackforger_model.ApplicationStatusRejected
	return hackforger_model.UpdateBountyApplication(ctx, app)
}

// StartReview transitions a Competitive bounty from Open to InReview.
func StartReview(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}

	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}

	if bounty.Mode != hackforger_model.BountyModeCompetitive {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Competitive mode only",
		}
	}

	if bounty.Status != hackforger_model.BountyStatusOpen {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Open",
		}
	}

	bounty.Status = hackforger_model.BountyStatusInReview
	return hackforger_model.UpdateBounty(ctx, bounty)
}

// SubmitDelivery transitions an Exclusive bounty from Claimed to InReview.
// Called when the claimer submits their work for review.
func SubmitDelivery(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}

	if bounty.Mode != hackforger_model.BountyModeExclusive {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Exclusive mode only",
		}
	}

	if bounty.Status != hackforger_model.BountyStatusClaimed {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Claimed",
		}
	}

	bounty.Status = hackforger_model.BountyStatusInReview
	return hackforger_model.UpdateBounty(ctx, bounty)
}

// CompleteBounty transitions an Exclusive bounty from InReview to Completed.
// If credit-type rewards exist, they are deposited to the claimer.
func CompleteBounty(ctx context.Context, bountyID, doerID int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
		if err != nil {
			return err
		}

		if err := requirePublisher(bounty, doerID); err != nil {
			return err
		}

		if bounty.Status != hackforger_model.BountyStatusInReview {
			return ErrInvalidBountyStatus{
				BountyID: bountyID,
				Current:  bounty.Status,
				Expected: "InReview",
			}
		}

		bounty.Status = hackforger_model.BountyStatusCompleted
		if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
			return err
		}

		// Deposit credits-type rewards to the claimer.
		if bounty.ClaimerID > 0 {
			rewards, err := hackforger_model.ListBountyRewards(ctx, bountyID)
			if err != nil {
				return err
			}
			for _, r := range rewards {
				if r.Type == hackforger_model.RewardTypeCredits && r.Credits > 0 {
					ref := fmt.Sprintf("bounty:%d", bountyID)
					note := fmt.Sprintf("Bounty reward: %s", bounty.Title)
					if err := Deposit(ctx, bounty.ClaimerID, r.Credits, ref, note); err != nil {
						return err
					}
				}
			}
		}

		// Publish feed event.
		if doer, err := user_model.GetUserByID(ctx, doerID); err == nil {
			notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
				OpType:       hackforger_model.ActionBountyCompleted,
				EntityType:   "bounty",
				EntityID:     bounty.ID,
				EntityName:   bounty.Title,
				RepoID:       bounty.RepoID,
				AudienceType: notify_service.AudienceFollowers | notify_service.AudienceRepoWatchers,
				Content: &hackforger_model.HackforgerPhaseContent{
					HackforgerActionContent: hackforger_model.HackforgerActionContent{
						EntityType: "bounty",
						EntityID:   bounty.ID,
						EntityName: bounty.Title,
					},
					OldStatus: "in_review",
					NewStatus: "completed",
				},
			})
		}

		postBountyIssueComment(ctx, bounty, doerID, "✅ **Bounty Completed** — delivery accepted and bounty fulfilled.")
		// Close the linked Issue — bounty completion means task is done.
		closeBountyIssue(ctx, bounty, doerID)

		return nil
	})
}

// RejectDelivery transitions an Exclusive bounty from InReview back to Claimed.
func RejectDelivery(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}

	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}

	if bounty.Mode != hackforger_model.BountyModeExclusive {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Exclusive mode only",
		}
	}

	if bounty.Status != hackforger_model.BountyStatusInReview {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "InReview",
		}
	}

	bounty.Status = hackforger_model.BountyStatusClaimed
	return hackforger_model.UpdateBounty(ctx, bounty)
}

// SelectWinners transitions a Competitive bounty from InReview to Completed.
// It records winners and deposits credits rewards matched by rank.
func SelectWinners(ctx context.Context, bountyID, doerID int64, winners []WinnerInput) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
		if err != nil {
			return err
		}

		if err := requirePublisher(bounty, doerID); err != nil {
			return err
		}

		if bounty.Mode != hackforger_model.BountyModeCompetitive {
			return ErrInvalidBountyStatus{
				BountyID: bountyID,
				Current:  bounty.Status,
				Expected: "Competitive mode only",
			}
		}

		if bounty.Status != hackforger_model.BountyStatusInReview {
			return ErrInvalidBountyStatus{
				BountyID: bountyID,
				Current:  bounty.Status,
				Expected: "InReview",
			}
		}

		// Load rewards indexed by rank for credit deposit lookup.
		rewards, err := hackforger_model.ListBountyRewards(ctx, bountyID)
		if err != nil {
			return err
		}
		rewardByRank := make(map[int]*hackforger_model.BountyReward)
		for _, r := range rewards {
			rewardByRank[r.Rank] = r
		}

		// Record each winner and deposit credits if applicable.
		for _, w := range winners {
			winner := &hackforger_model.BountyWinner{
				BountyID: bountyID,
				UserID:   w.UserID,
				Rank:     w.Rank,
			}
			if err := hackforger_model.CreateBountyWinner(ctx, winner); err != nil {
				return err
			}

			if reward, ok := rewardByRank[w.Rank]; ok {
				if reward.Type == hackforger_model.RewardTypeCredits && reward.Credits > 0 {
					ref := fmt.Sprintf("bounty:%d:rank:%d", bountyID, w.Rank)
					note := fmt.Sprintf("Bounty winner rank %d: %s", w.Rank, bounty.Title)
					if err := Deposit(ctx, w.UserID, reward.Credits, ref, note); err != nil {
						return err
					}
				}
			}
		}

		bounty.Status = hackforger_model.BountyStatusCompleted
		if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
			return err
		}

		// Publish feed event.
		if doer, err := user_model.GetUserByID(ctx, doerID); err == nil {
			notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
				OpType:       hackforger_model.ActionBountyWinnersSelected,
				EntityType:   "bounty",
				EntityID:     bounty.ID,
				EntityName:   bounty.Title,
				RepoID:       bounty.RepoID,
				AudienceType: notify_service.AudienceGlobal,
				Content: &hackforger_model.HackforgerPhaseContent{
					HackforgerActionContent: hackforger_model.HackforgerActionContent{
						EntityType: "bounty",
						EntityID:   bounty.ID,
						EntityName: bounty.Title,
					},
					OldStatus: "in_review",
					NewStatus: "completed",
				},
			})
		}

		// Close the linked Issue — bounty completion means task is done.
		closeBountyIssue(ctx, bounty, doerID)

		return nil
	})
}

// MarkPaid transitions a bounty from Completed to Paid.
func MarkPaid(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}

	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}

	if bounty.Status != hackforger_model.BountyStatusCompleted {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Completed",
		}
	}

	bounty.Status = hackforger_model.BountyStatusPaid
	if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
		return err
	}

	if doer, err := user_model.GetUserByID(ctx, doerID); err == nil {
		notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
			OpType:       hackforger_model.ActionBountyPaid,
			EntityType:   "bounty",
			EntityID:     bounty.ID,
			EntityName:   bounty.Title,
			RepoID:       bounty.RepoID,
			AudienceType: notify_service.AudienceDirectUser | notify_service.AudienceRepoWatchers,
			TargetUserID: bounty.ClaimerID,
			Content: &hackforger_model.HackforgerPhaseContent{
				HackforgerActionContent: hackforger_model.HackforgerActionContent{
					EntityType: "bounty",
					EntityID:   bounty.ID,
					EntityName: bounty.Title,
				},
				OldStatus: "completed",
				NewStatus: "paid",
			},
		})
	}

	return nil
}

// CancelBounty transitions a bounty from Open or Claimed to Cancelled.
func CancelBounty(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}

	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}

	if bounty.Status != hackforger_model.BountyStatusOpen && bounty.Status != hackforger_model.BountyStatusClaimed {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Open or Claimed",
		}
	}

	bounty.Status = hackforger_model.BountyStatusCancelled
	if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
		return err
	}

	if doer, err := user_model.GetUserByID(ctx, doerID); err == nil {
		notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
			OpType:       hackforger_model.ActionBountyCancelled,
			EntityType:   "bounty",
			EntityID:     bounty.ID,
			EntityName:   bounty.Title,
			RepoID:       bounty.RepoID,
			AudienceType: notify_service.AudienceRepoWatchers,
			Content: &hackforger_model.HackforgerPhaseContent{
				HackforgerActionContent: hackforger_model.HackforgerActionContent{
					EntityType: "bounty",
					EntityID:   bounty.ID,
					EntityName: bounty.Title,
				},
				OldStatus: "open",
				NewStatus: "cancelled",
			},
		})
	}
	postBountyIssueComment(ctx, bounty, doerID, "❌ **Bounty Cancelled**")

	return nil
}

// CheckExpiredBounties finds all Open or Claimed bounties past their deadline
// and transitions them to Expired. Intended to be called from a cron task.
func CheckExpiredBounties(ctx context.Context) error {
	now := timeutil.TimeStampNow()

	var bounties []*hackforger_model.Bounty
	err := db.GetEngine(ctx).
		Where("(status = ? OR status = ?) AND deadline > 0 AND deadline < ?",
			hackforger_model.BountyStatusOpen,
			hackforger_model.BountyStatusClaimed,
			now).
		Find(&bounties)
	if err != nil {
		return err
	}

	for _, bounty := range bounties {
		oldStatus := bounty.Status
		bounty.Status = hackforger_model.BountyStatusExpired
		if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
			log.Error("CheckExpiredBounties: UpdateBounty(%d): %v", bounty.ID, err)
			continue
		}

		oldStatusStr := "open"
		if oldStatus == hackforger_model.BountyStatusClaimed {
			oldStatusStr = "claimed"
		}

		if doer, err := user_model.GetUserByID(ctx, bounty.PublisherID); err == nil {
			notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
				OpType:       hackforger_model.ActionBountyExpired,
				EntityType:   "bounty",
				EntityID:     bounty.ID,
				EntityName:   bounty.Title,
				RepoID:       bounty.RepoID,
				AudienceType: notify_service.AudienceRepoWatchers,
				Content: &hackforger_model.HackforgerPhaseContent{
					HackforgerActionContent: hackforger_model.HackforgerActionContent{
						EntityType: "bounty",
						EntityID:   bounty.ID,
						EntityName: bounty.Title,
					},
					OldStatus: oldStatusStr,
					NewStatus: "expired",
				},
			})
		}
	}

	return nil
}

// UpdateBountyMeta updates a bounty's title and deadline. Only allowed when Open.
func UpdateBountyMeta(ctx context.Context, bountyID, doerID int64, title string, deadline timeutil.TimeStamp) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}

	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}

	if bounty.Status != hackforger_model.BountyStatusOpen {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Open",
		}
	}

	bounty.Title = title
	bounty.Deadline = deadline
	return hackforger_model.UpdateBounty(ctx, bounty)
}

// DeleteBounty deletes a bounty. Only allowed when Open and no applications exist.
func DeleteBounty(ctx context.Context, bountyID, doerID int64) error {
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		return err
	}

	if err := requirePublisher(bounty, doerID); err != nil {
		return err
	}

	if bounty.Status != hackforger_model.BountyStatusOpen {
		return ErrInvalidBountyStatus{
			BountyID: bountyID,
			Current:  bounty.Status,
			Expected: "Open",
		}
	}

	// Check for existing applications.
	apps, count, err := hackforger_model.ListBountyApplications(ctx, hackforger_model.ListBountyApplicationsOptions{
		BountyID: bountyID,
	})
	if err != nil {
		return err
	}
	_ = apps
	if count > 0 {
		return ErrBountyHasApplications{BountyID: bountyID}
	}

	return hackforger_model.DeleteBounty(ctx, bountyID)
}

// ExpireBounty expires a single bounty. It must be in Open or Claimed status.
func ExpireBounty(ctx context.Context, bounty *hackforger_model.Bounty) error {
	if bounty.Status != hackforger_model.BountyStatusOpen && bounty.Status != hackforger_model.BountyStatusClaimed {
		return ErrInvalidBountyStatus{
			BountyID: bounty.ID,
			Current:  bounty.Status,
			Expected: "Open or Claimed",
		}
	}

	oldStatusStr := "open"
	if bounty.Status == hackforger_model.BountyStatusClaimed {
		oldStatusStr = "claimed"
	}

	bounty.Status = hackforger_model.BountyStatusExpired
	if err := hackforger_model.UpdateBounty(ctx, bounty); err != nil {
		return err
	}

	if doer, err := user_model.GetUserByID(ctx, bounty.PublisherID); err == nil {
		notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
			OpType:       hackforger_model.ActionBountyExpired,
			EntityType:   "bounty",
			EntityID:     bounty.ID,
			EntityName:   bounty.Title,
			RepoID:       bounty.RepoID,
			AudienceType: notify_service.AudienceRepoWatchers,
			Content: &hackforger_model.HackforgerPhaseContent{
				HackforgerActionContent: hackforger_model.HackforgerActionContent{
					EntityType: "bounty",
					EntityID:   bounty.ID,
					EntityName: bounty.Title,
				},
				OldStatus: oldStatusStr,
				NewStatus: "expired",
			},
		})
	}

	return nil
}

// BountyStatsResponse holds aggregate statistics for bounties.
type BountyStatsResponse struct {
	Total     int64 `json:"total"`
	Open      int64 `json:"open"`
	Claimed   int64 `json:"claimed"`
	InReview  int64 `json:"in_review"`
	Completed int64 `json:"completed"`
	Paid      int64 `json:"paid"`
	Expired   int64 `json:"expired"`
	Cancelled int64 `json:"cancelled"`
}

// GetBountyStats returns aggregate bounty statistics.
func GetBountyStats(ctx context.Context) (*BountyStatsResponse, error) {
	stats := &BountyStatsResponse{}

	statuses := []hackforger_model.BountyStatus{
		hackforger_model.BountyStatusOpen,
		hackforger_model.BountyStatusClaimed,
		hackforger_model.BountyStatusInReview,
		hackforger_model.BountyStatusCompleted,
		hackforger_model.BountyStatusPaid,
		hackforger_model.BountyStatusExpired,
		hackforger_model.BountyStatusCancelled,
	}

	for _, s := range statuses {
		status := s
		_, count, err := hackforger_model.ListBounties(ctx, hackforger_model.ListBountiesOptions{
			ListOptions: db.ListOptions{PageSize: 1},
			Status:      &status,
		})
		if err != nil {
			return nil, err
		}

		switch s {
		case hackforger_model.BountyStatusOpen:
			stats.Open = count
		case hackforger_model.BountyStatusClaimed:
			stats.Claimed = count
		case hackforger_model.BountyStatusInReview:
			stats.InReview = count
		case hackforger_model.BountyStatusCompleted:
			stats.Completed = count
		case hackforger_model.BountyStatusPaid:
			stats.Paid = count
		case hackforger_model.BountyStatusExpired:
			stats.Expired = count
		case hackforger_model.BountyStatusCancelled:
			stats.Cancelled = count
		}
		stats.Total += count
	}

	return stats, nil
}

// LeaderboardEntry represents one row in the bounty hunter leaderboard.
type LeaderboardEntry struct {
	UserID       int64 `json:"user_id"`
	WinCount     int64 `json:"win_count"`
	TotalCredits int64 `json:"total_credits"`
}

// GetBountyLeaderboard returns the top bounty hunters by win count.
func GetBountyLeaderboard(ctx context.Context, limit int) ([]*LeaderboardEntry, error) {
	var entries []*LeaderboardEntry
	err := db.GetEngine(ctx).
		Table("bounty_winner").
		Select("user_id, COUNT(*) AS win_count, 0 AS total_credits").
		GroupBy("user_id").
		OrderBy("win_count DESC").
		Limit(limit).
		Find(&entries)
	if err != nil {
		return nil, err
	}
	return entries, nil
}
