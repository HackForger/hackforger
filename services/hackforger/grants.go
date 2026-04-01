// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
	notify_service "forgejo.org/services/notify"
)

// SubmitProjectOpts holds the parameters for submitting a project to a grant round.
type SubmitProjectOpts struct {
	Title       string
	Description string
	RepoID      int64
}

// CreateGrantRoundOpts holds the parameters for creating a grant round.
type CreateGrantRoundOpts struct {
	Name          string
	Slug          string
	Description   string
	Budget        float64
	Currency      string
	BudgetCredits int64
	Deadline      timeutil.TimeStamp
}

// ErrAccessDenied is returned when the user lacks permission for the operation.
type ErrAccessDenied struct {
	UserID int64
	OrgID  int64
}

func (err ErrAccessDenied) Error() string {
	return fmt.Sprintf("access denied [user_id: %d, org_id: %d]", err.UserID, err.OrgID)
}

func (err ErrAccessDenied) Unwrap() error {
	return util.ErrPermissionDenied
}

// IsErrAccessDenied checks if an error is ErrAccessDenied.
func IsErrAccessDenied(err error) bool {
	_, ok := err.(ErrAccessDenied)
	return ok
}

// ErrInvalidTransition is returned when a state transition is not allowed.
type ErrInvalidTransition struct {
	RoundID int64
	From    hackforger_model.GrantRoundStatus
	To      hackforger_model.GrantRoundStatus
}

func (err ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid status transition [round_id: %d, from: %d, to: %d]", err.RoundID, err.From, err.To)
}

func (err ErrInvalidTransition) Unwrap() error {
	return util.ErrInvalidArgument
}

// IsErrInvalidTransition checks if an error is ErrInvalidTransition.
func IsErrInvalidTransition(err error) bool {
	_, ok := err.(ErrInvalidTransition)
	return ok
}

// ErrRoundNotFinalized is returned when distributing from a non-finalized round.
type ErrRoundNotFinalized struct {
	RoundID int64
	Status  hackforger_model.GrantRoundStatus
}

func (err ErrRoundNotFinalized) Error() string {
	return fmt.Sprintf("grant round is not finalized [round_id: %d, status: %d]", err.RoundID, err.Status)
}

func (err ErrRoundNotFinalized) Unwrap() error {
	return util.ErrInvalidArgument
}

// IsErrRoundNotFinalized checks if an error is ErrRoundNotFinalized.
func IsErrRoundNotFinalized(err error) bool {
	_, ok := err.(ErrRoundNotFinalized)
	return ok
}

// validTransitions defines the allowed state transitions for grant rounds.
var validTransitions = map[hackforger_model.GrantRoundStatus][]hackforger_model.GrantRoundStatus{
	hackforger_model.GrantRoundStatusDraft:     {hackforger_model.GrantRoundStatusOpen, hackforger_model.GrantRoundStatusCancelled},
	hackforger_model.GrantRoundStatusOpen:      {hackforger_model.GrantRoundStatusReview, hackforger_model.GrantRoundStatusCancelled},
	hackforger_model.GrantRoundStatusReview:    {hackforger_model.GrantRoundStatusFinalized, hackforger_model.GrantRoundStatusCancelled},
	hackforger_model.GrantRoundStatusFinalized: {hackforger_model.GrantRoundStatusDistributed, hackforger_model.GrantRoundStatusCancelled},
}

// checkGrantRoundAccess verifies that doerID is an owner or admin of the round's org.
func checkGrantRoundAccess(ctx context.Context, doerID int64, round *hackforger_model.GrantRound) error {
	isOwner, err := org_model.IsOrganizationOwner(ctx, round.OrgID, doerID)
	if err != nil {
		return err
	}
	if isOwner {
		return nil
	}
	isAdmin, err := org_model.IsOrganizationAdmin(ctx, round.OrgID, doerID)
	if err != nil {
		return err
	}
	if isAdmin {
		return nil
	}
	return ErrAccessDenied{UserID: doerID, OrgID: round.OrgID}
}

// isValidTransition checks if the transition from current to target is allowed.
func isValidTransition(from, to hackforger_model.GrantRoundStatus) bool {
	targets, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// transitionRound gets a round, checks access, validates the transition, and updates the status.
func transitionRound(ctx context.Context, doerID, roundID int64, target hackforger_model.GrantRoundStatus) (*hackforger_model.GrantRound, error) {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return nil, err
	}

	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return nil, err
	}

	if !isValidTransition(round.Status, target) {
		return nil, ErrInvalidTransition{RoundID: roundID, From: round.Status, To: target}
	}

	round.Status = target
	if err := hackforger_model.UpdateGrantRound(ctx, round); err != nil {
		return nil, err
	}

	return round, nil
}

// publishGrantEvent builds a HackforgerEventOpts and dispatches a feed event.
func publishGrantEvent(ctx context.Context, actUserID int64, opType activities_model.ActionType, round *hackforger_model.GrantRound, audience notify_service.HackforgerAudienceType) error {
	doer, err := user_model.GetUserByID(ctx, actUserID)
	if err != nil {
		return err
	}
	content := &hackforger_model.HackforgerActionContent{
		EntityType: "grant_round",
		EntityID:   round.ID,
		EntityName: round.Name,
		EntitySlug: round.Slug,
	}
	opts := &notify_service.HackforgerEventOpts{
		OpType:       opType,
		EntityType:   "grant_round",
		EntityID:     round.ID,
		EntityName:   round.Name,
		EntitySlug:   round.Slug,
		Content:      content,
		AudienceType: audience,
		OrgID:        round.OrgID,
	}
	if opType == hackforger_model.ActionGrantRoundCreated {
		notify_service.HackforgerEntityCreated(ctx, doer, opts)
	} else {
		notify_service.HackforgerEntityStatusChanged(ctx, doer, opts)
	}
	return nil
}

// CreateGrantRound creates a new grant round and publishes a feed event.
func CreateGrantRound(ctx context.Context, doerID, orgID int64, opts CreateGrantRoundOpts) (*hackforger_model.GrantRound, error) {
	round := &hackforger_model.GrantRound{
		OrgID:         orgID,
		OwnerID:       doerID,
		Name:          opts.Name,
		Slug:          opts.Slug,
		Description:   opts.Description,
		Status:        hackforger_model.GrantRoundStatusDraft,
		Budget:        opts.Budget,
		Currency:      opts.Currency,
		BudgetCredits: opts.BudgetCredits,
		Deadline:      opts.Deadline,
	}

	if err := hackforger_model.CreateGrantRound(ctx, round); err != nil {
		return nil, err
	}

	if err := publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundCreated, round, notify_service.AudienceGlobal); err != nil {
		return nil, err
	}

	return round, nil
}

// UpdateGrantRoundService updates a grant round. Only Draft or Open rounds can be updated.
func UpdateGrantRoundService(ctx context.Context, doerID int64, round *hackforger_model.GrantRound) error {
	existing, err := hackforger_model.GetGrantRoundByID(ctx, round.ID)
	if err != nil {
		return err
	}

	if err := checkGrantRoundAccess(ctx, doerID, existing); err != nil {
		return err
	}

	if existing.Status != hackforger_model.GrantRoundStatusDraft && existing.Status != hackforger_model.GrantRoundStatusOpen {
		return ErrInvalidTransition{RoundID: round.ID, From: existing.Status, To: existing.Status}
	}

	return hackforger_model.UpdateGrantRound(ctx, round)
}

// DeleteGrantRoundService deletes a grant round after checking access.
func DeleteGrantRoundService(ctx context.Context, doerID, roundID int64) error {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return err
	}

	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}

	return hackforger_model.DeleteGrantRound(ctx, roundID)
}

// OpenRound transitions a round from Draft to Open.
func OpenRound(ctx context.Context, doerID, roundID int64) error {
	round, err := transitionRound(ctx, doerID, roundID, hackforger_model.GrantRoundStatusOpen)
	if err != nil {
		return err
	}
	return publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundOpened, round, notify_service.AudienceGlobal)
}

// CloseRound transitions a round from Open to Review.
func CloseRound(ctx context.Context, doerID, roundID int64) error {
	round, err := transitionRound(ctx, doerID, roundID, hackforger_model.GrantRoundStatusReview)
	if err != nil {
		return err
	}
	return publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundClosed, round, notify_service.AudienceGlobal)
}

// FinalizeRound transitions a round from Review to Finalized.
// All Approved projects must have non-zero allocations.
func FinalizeRound(ctx context.Context, doerID, roundID int64) error {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return err
	}

	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}

	if !isValidTransition(round.Status, hackforger_model.GrantRoundStatusFinalized) {
		return ErrInvalidTransition{RoundID: roundID, From: round.Status, To: hackforger_model.GrantRoundStatusFinalized}
	}

	// Check all Approved projects have allocations
	approvedStatus := hackforger_model.GrantProjectStatusApproved
	projects, _, err := hackforger_model.ListGrantProjectsByRound(ctx, hackforger_model.ListGrantProjectsByRoundOptions{
		ListOptions: db.ListOptions{Page: 1, PageSize: 1000},
		RoundID:     roundID,
		Status:      &approvedStatus,
	})
	if err != nil {
		return err
	}

	unallocated := int64(0)
	for _, p := range projects {
		if p.AwardAmount == 0 && p.AwardCredits == 0 {
			unallocated++
		}
	}
	if unallocated > 0 {
		return hackforger_model.ErrUnallocatedProjects{RoundID: roundID, Count: unallocated}
	}

	round.Status = hackforger_model.GrantRoundStatusFinalized
	if err := hackforger_model.UpdateGrantRound(ctx, round); err != nil {
		return err
	}

	return publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundFinalized, round, notify_service.AudienceGlobal)
}

// CancelRound transitions a round to Cancelled from any non-terminal status.
func CancelRound(ctx context.Context, doerID, roundID int64) error {
	round, err := transitionRound(ctx, doerID, roundID, hackforger_model.GrantRoundStatusCancelled)
	if err != nil {
		return err
	}
	return publishGrantEvent(ctx, doerID, hackforger_model.ActionGrantRoundCancelled, round, notify_service.AudienceGlobal)
}

// SubmitProject submits a project to an open grant round.
func SubmitProject(ctx context.Context, doerID, roundID int64, opts SubmitProjectOpts) (*hackforger_model.GrantProject, error) {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return nil, err
	}

	if round.Status != hackforger_model.GrantRoundStatusOpen {
		return nil, hackforger_model.ErrGrantRoundNotOpen{RoundID: roundID, Status: round.Status}
	}

	// Check for duplicate submission
	_, err = hackforger_model.GetGrantProjectByUserAndRound(ctx, doerID, roundID)
	if err == nil {
		return nil, hackforger_model.ErrGrantProjectAlreadyExists{UserID: doerID, RoundID: roundID}
	}
	if !hackforger_model.IsErrGrantProjectNotExist(err) {
		return nil, err
	}

	project := &hackforger_model.GrantProject{
		RoundID:     roundID,
		UserID:      doerID,
		RepoID:      opts.RepoID,
		Title:       opts.Title,
		Description: opts.Description,
		Status:      hackforger_model.GrantProjectStatusPending,
	}

	if err := hackforger_model.CreateGrantProject(ctx, project); err != nil {
		return nil, err
	}

	if doer, err := user_model.GetUserByID(ctx, doerID); err == nil {
		notify_service.HackforgerEntityCreated(ctx, doer, &notify_service.HackforgerEventOpts{
			OpType:       hackforger_model.ActionGrantProjectSubmitted,
			EntityType:   "grant_project",
			EntityID:     project.ID,
			EntityName:   project.Title,
			AudienceType: notify_service.AudienceFollowers,
			Content: &hackforger_model.HackforgerActionContent{
				EntityType: "grant_project",
				EntityID:   project.ID,
				EntityName: project.Title,
			},
		})
	}

	return project, nil
}

// ApproveProject transitions a project from Pending to Approved.
func ApproveProject(ctx context.Context, doerID, projectID int64) error {
	project, err := hackforger_model.GetGrantProjectByID(ctx, projectID)
	if err != nil {
		return err
	}

	round, err := hackforger_model.GetGrantRoundByID(ctx, project.RoundID)
	if err != nil {
		return err
	}

	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}

	if project.Status != hackforger_model.GrantProjectStatusPending {
		return fmt.Errorf("project is not pending [id: %d, status: %d]: %w", projectID, project.Status, util.ErrInvalidArgument)
	}

	project.Status = hackforger_model.GrantProjectStatusApproved
	return hackforger_model.UpdateGrantProject(ctx, project)
}

// RejectProject transitions a project from Pending to Rejected.
func RejectProject(ctx context.Context, doerID, projectID int64) error {
	project, err := hackforger_model.GetGrantProjectByID(ctx, projectID)
	if err != nil {
		return err
	}

	round, err := hackforger_model.GetGrantRoundByID(ctx, project.RoundID)
	if err != nil {
		return err
	}

	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}

	if project.Status != hackforger_model.GrantProjectStatusPending {
		return fmt.Errorf("project is not pending [id: %d, status: %d]: %w", projectID, project.Status, util.ErrInvalidArgument)
	}

	project.Status = hackforger_model.GrantProjectStatusRejected
	return hackforger_model.UpdateGrantProject(ctx, project)
}

// AllocateAward sets the award amount and credits for a project,
// validating against the round's budget (subtracting the project's existing allocation first).
func AllocateAward(ctx context.Context, doerID, projectID int64, amount float64, credits int64) error {
	project, err := hackforger_model.GetGrantProjectByID(ctx, projectID)
	if err != nil {
		return err
	}

	round, err := hackforger_model.GetGrantRoundByID(ctx, project.RoundID)
	if err != nil {
		return err
	}

	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}

	// Get current budget usage
	usedAmount, usedCredits, err := hackforger_model.GetGrantRoundBudgetUsage(ctx, round.ID)
	if err != nil {
		return err
	}

	// Subtract existing allocation of this project (so re-allocation is possible)
	usedAmount -= project.AwardAmount
	usedCredits -= project.AwardCredits

	// Validate budget
	if usedAmount+amount > round.Budget {
		return hackforger_model.ErrExceedsBudget{
			RoundID:     round.ID,
			BudgetField: "amount",
			Budget:      round.Budget,
			Used:        usedAmount,
			Requested:   amount,
		}
	}
	if usedCredits+credits > round.BudgetCredits {
		return hackforger_model.ErrExceedsBudget{
			RoundID:     round.ID,
			BudgetField: "credits",
			Budget:      float64(round.BudgetCredits),
			Used:        float64(usedCredits),
			Requested:   float64(credits),
		}
	}

	project.AwardAmount = amount
	project.AwardCredits = credits
	return hackforger_model.UpdateGrantProject(ctx, project)
}

// DistributeProject deposits credits for a single project and marks it as Funded.
func DistributeProject(ctx context.Context, doerID, projectID int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		project, err := hackforger_model.GetGrantProjectByID(ctx, projectID)
		if err != nil {
			return err
		}

		round, err := hackforger_model.GetGrantRoundByID(ctx, project.RoundID)
		if err != nil {
			return err
		}

		if round.Status != hackforger_model.GrantRoundStatusFinalized {
			return ErrRoundNotFinalized{RoundID: round.ID, Status: round.Status}
		}

		if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
			return err
		}

		if project.Status != hackforger_model.GrantProjectStatusApproved {
			return fmt.Errorf("project is not approved [id: %d, status: %d]: %w", projectID, project.Status, util.ErrInvalidArgument)
		}

		// Deposit credits
		if project.AwardCredits > 0 {
			if err := Deposit(ctx, project.UserID, project.AwardCredits,
				fmt.Sprintf("grant_round:%d/project:%d", round.ID, project.ID),
				fmt.Sprintf("Grant award for project: %s", project.Title),
			); err != nil {
				return err
			}
		}

		// Mark project as Funded
		project.Status = hackforger_model.GrantProjectStatusFunded
		if err := hackforger_model.UpdateGrantProject(ctx, project); err != nil {
			return err
		}

		// Publish award event
		if doer, err := user_model.GetUserByID(ctx, doerID); err == nil {
			notify_service.HackforgerEntityStatusChanged(ctx, doer, &notify_service.HackforgerEventOpts{
				OpType:       hackforger_model.ActionGrantAwarded,
				EntityType:   "grant_project",
				EntityID:     project.ID,
				EntityName:   project.Title,
				AudienceType: notify_service.AudienceGlobal,
				OrgID:        round.OrgID,
				Content: &hackforger_model.HackforgerActionContent{
					EntityType: "grant_project",
					EntityID:   project.ID,
					EntityName: project.Title,
					Extra: map[string]any{
						"round_id":      round.ID,
						"round_name":    round.Name,
						"award_amount":  project.AwardAmount,
						"award_credits": project.AwardCredits,
					},
				},
			})
		}

		// Check if all approved projects are now funded; if so, auto-transition round to Distributed
		approvedStatus := hackforger_model.GrantProjectStatusApproved
		remaining, err := hackforger_model.CountGrantProjectsByRound(ctx, round.ID, &approvedStatus)
		if err != nil {
			return err
		}

		if remaining == 0 {
			round.Status = hackforger_model.GrantRoundStatusDistributed
			if err := hackforger_model.UpdateGrantRound(ctx, round); err != nil {
				return err
			}
		}

		return nil
	})
}

// DistributeRound distributes all approved projects in a finalized round.
func DistributeRound(ctx context.Context, doerID, roundID int64) error {
	round, err := hackforger_model.GetGrantRoundByID(ctx, roundID)
	if err != nil {
		return err
	}

	if round.Status != hackforger_model.GrantRoundStatusFinalized {
		return ErrRoundNotFinalized{RoundID: roundID, Status: round.Status}
	}

	if err := checkGrantRoundAccess(ctx, doerID, round); err != nil {
		return err
	}

	approvedStatus := hackforger_model.GrantProjectStatusApproved
	projects, _, err := hackforger_model.ListGrantProjectsByRound(ctx, hackforger_model.ListGrantProjectsByRoundOptions{
		ListOptions: db.ListOptions{Page: 1, PageSize: 1000},
		RoundID:     roundID,
		Status:      &approvedStatus,
	})
	if err != nil {
		return err
	}

	for _, p := range projects {
		if err := DistributeProject(ctx, doerID, p.ID); err != nil {
			return err
		}
	}

	return nil
}

// grantProjectStatusName converts a GrantProjectStatus to its human-readable string.
func grantProjectStatusName(s hackforger_model.GrantProjectStatus) string {
	switch s {
	case hackforger_model.GrantProjectStatusPending:
		return "pending"
	case hackforger_model.GrantProjectStatusApproved:
		return "approved"
	case hackforger_model.GrantProjectStatusRejected:
		return "rejected"
	case hackforger_model.GrantProjectStatusFunded:
		return "funded"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// ExportRoundCSV generates a CSV export of all projects in a round.
func ExportRoundCSV(ctx context.Context, roundID int64) ([]byte, error) {
	projects, _, err := hackforger_model.ListGrantProjectsByRound(ctx, hackforger_model.ListGrantProjectsByRoundOptions{
		ListOptions: db.ListOptions{Page: 1, PageSize: 10000},
		RoundID:     roundID,
	})
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header
	if err := w.Write([]string{"ID", "Title", "UserID", "RepoID", "Status", "AwardAmount", "AwardCredits"}); err != nil {
		return nil, err
	}

	for _, p := range projects {
		record := []string{
			strconv.FormatInt(p.ID, 10),
			p.Title,
			strconv.FormatInt(p.UserID, 10),
			strconv.FormatInt(p.RepoID, 10),
			grantProjectStatusName(p.Status),
			strconv.FormatFloat(p.AwardAmount, 'f', 2, 64),
			strconv.FormatInt(p.AwardCredits, 10),
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}

	w.Flush()
	return buf.Bytes(), w.Error()
}
