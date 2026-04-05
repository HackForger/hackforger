// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
)

// Phase System errors
type ErrPhaseActionNotAllowed struct{ ActivityKind string; ActivityID int64; Action string }
func (e ErrPhaseActionNotAllowed) Error() string { return fmt.Sprintf("action %q not allowed in current phase [%s/%d]", e.Action, e.ActivityKind, e.ActivityID) }
func IsErrPhaseActionNotAllowed(err error) bool { _, ok := err.(ErrPhaseActionNotAllowed); return ok }

type ErrPhaseLocked struct{ PhaseID int64 }
func (e ErrPhaseLocked) Error() string { return fmt.Sprintf("phase is locked [id: %d]", e.PhaseID) }
func IsErrPhaseLocked(err error) bool { _, ok := err.(ErrPhaseLocked); return ok }

type ErrPhaseOverlap struct{ ActivityKind string; ActivityID int64 }
func (e ErrPhaseOverlap) Error() string { return fmt.Sprintf("phase overlaps with existing phase [%s/%d]", e.ActivityKind, e.ActivityID) }
func IsErrPhaseOverlap(err error) bool { _, ok := err.(ErrPhaseOverlap); return ok }

type ErrPhaseUniqueViolation struct{ PhaseTypeID int64 }
func (e ErrPhaseUniqueViolation) Error() string { return fmt.Sprintf("unique phase type already exists [type: %d]", e.PhaseTypeID) }
func IsErrPhaseUniqueViolation(err error) bool { _, ok := err.(ErrPhaseUniqueViolation); return ok }

type ErrPhaseNotFuture struct{ PhaseID int64 }
func (e ErrPhaseNotFuture) Error() string { return fmt.Sprintf("can only remove future phases [id: %d]", e.PhaseID) }
func IsErrPhaseNotFuture(err error) bool { _, ok := err.(ErrPhaseNotFuture); return ok }

type ErrActivePhaseStartLocked struct{ PhaseID int64 }
func (e ErrActivePhaseStartLocked) Error() string { return fmt.Sprintf("cannot change start time of active phase [id: %d]", e.PhaseID) }
func IsErrActivePhaseStartLocked(err error) bool { _, ok := err.(ErrActivePhaseStartLocked); return ok }

// CurrentPhase returns the currently active phase for an activity.
func CurrentPhase(ctx context.Context, activityKind string, activityID int64) (*hackforger_model.Phase, error) {
	return hackforger_model.GetCurrentPhase(ctx, activityKind, activityID)
}

// AllowsAction checks if the given action is permitted in the current phase.
// Returns true if no phases are configured (backward compatible — unconstrained).
func AllowsAction(ctx context.Context, activityKind string, activityID int64, action string) (bool, error) {
	count, err := hackforger_model.CountPhasesByActivity(ctx, activityKind, activityID)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return true, nil // no phases configured = unconstrained (backward compatible)
	}
	phase, err := CurrentPhase(ctx, activityKind, activityID)
	if err != nil {
		return false, err
	}
	if phase == nil {
		return false, nil // phases exist but none active = no actions allowed
	}
	return phase.PhaseType.HasAction(action), nil
}

// UpdatePhaseTime updates the time window of a phase. Rejects if phase is locked.
func UpdatePhaseTime(ctx context.Context, phaseID int64, startTime, endTime int64) error {
	phase, err := hackforger_model.GetPhaseByID(ctx, phaseID)
	if err != nil {
		return err
	}
	if phase == nil {
		return fmt.Errorf("phase not found: %d", phaseID)
	}
	if phase.IsLocked() {
		return ErrPhaseLocked{PhaseID: phaseID}
	}
	if phase.IsActive() && startTime != phase.StartTime {
		return ErrActivePhaseStartLocked{PhaseID: phaseID}
	}
	if endTime <= startTime {
		return fmt.Errorf("end_time must be after start_time")
	}
	if err := checkOverlap(ctx, phase.ActivityKind, phase.ActivityID, phaseID, startTime, endTime); err != nil {
		return err
	}

	phase.StartTime = startTime
	phase.EndTime = endTime
	if err := hackforger_model.UpdatePhase(ctx, phase); err != nil {
		return err
	}
	UnschedulePhaseJobs(phaseID)
	SchedulePhaseJobs(phase)
	return nil
}

// AddPhase creates a new phase for an activity.
func AddPhase(ctx context.Context, activityKind string, activityID int64, phaseTypeID int64, sortOrder int, startTime, endTime int64) (*hackforger_model.Phase, error) {
	if endTime <= startTime {
		return nil, fmt.Errorf("end_time must be after start_time")
	}

	pt, err := hackforger_model.GetPhaseTypeByID(ctx, phaseTypeID)
	if err != nil {
		return nil, err
	}
	if pt == nil {
		return nil, fmt.Errorf("phase type not found: %d", phaseTypeID)
	}

	// Check uniqueness constraint
	if pt.IsUnique {
		phases, err := hackforger_model.GetPhasesByActivity(ctx, activityKind, activityID)
		if err != nil {
			return nil, err
		}
		for _, p := range phases {
			if p.PhaseTypeID == phaseTypeID {
				return nil, ErrPhaseUniqueViolation{PhaseTypeID: phaseTypeID}
			}
		}
	}

	if err := checkOverlap(ctx, activityKind, activityID, 0, startTime, endTime); err != nil {
		return nil, err
	}

	phase := &hackforger_model.Phase{
		PhaseTypeID:  phaseTypeID,
		ActivityKind: activityKind,
		ActivityID:   activityID,
		SortOrder:    sortOrder,
		StartTime:    startTime,
		EndTime:      endTime,
	}
	if err := hackforger_model.CreatePhase(ctx, phase); err != nil {
		return nil, err
	}
	SchedulePhaseJobs(phase)
	return phase, nil
}

// RemovePhase deletes a future phase. Active and locked phases cannot be removed.
func RemovePhase(ctx context.Context, phaseID int64) error {
	phase, err := hackforger_model.GetPhaseByID(ctx, phaseID)
	if err != nil {
		return err
	}
	if phase == nil {
		return fmt.Errorf("phase not found: %d", phaseID)
	}
	if !phase.IsFuture() {
		return ErrPhaseNotFuture{PhaseID: phaseID}
	}
	UnschedulePhaseJobs(phaseID)
	return hackforger_model.DeletePhase(ctx, phaseID)
}

// GetPhases returns all phases for an activity with their types loaded.
func GetPhases(ctx context.Context, activityKind string, activityID int64) ([]*hackforger_model.Phase, error) {
	return hackforger_model.GetPhasesByActivity(ctx, activityKind, activityID)
}

// checkOverlap verifies that a proposed time window doesn't overlap with existing phases.
func checkOverlap(ctx context.Context, activityKind string, activityID int64, excludePhaseID int64, startTime, endTime int64) error {
	builder := db.GetEngine(ctx).
		Where("activity_kind = ? AND activity_id = ?", activityKind, activityID).
		And("((start_time < ? AND end_time > ?) OR (start_time < ? AND end_time > ?) OR (start_time >= ? AND end_time <= ?))",
			endTime, startTime, endTime, startTime, startTime, endTime)
	if excludePhaseID > 0 {
		builder = builder.And("id != ?", excludePhaseID)
	}
	count, err := builder.Count(&hackforger_model.Phase{})
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrPhaseOverlap{ActivityKind: activityKind, ActivityID: activityID}
	}
	return nil
}
