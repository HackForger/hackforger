// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

// Phase represents an instance of a phase for a specific activity.
// Lock state is computed (not stored): a phase is locked when EndTime < now().
type Phase struct {
	ID           int64              `xorm:"pk autoincr" json:"id"`
	PhaseTypeID  int64              `xorm:"NOT NULL INDEX" json:"phase_type_id"`
	ActivityKind string             `xorm:"VARCHAR(20) NOT NULL" json:"activity_kind"`
	ActivityID   int64              `xorm:"NOT NULL" json:"activity_id"`
	SortOrder    int                `xorm:"NOT NULL" json:"sort_order"`
	CustomName   string             `xorm:"VARCHAR(100) NOT NULL DEFAULT ''" json:"custom_name"`
	StartTime    int64              `xorm:"NOT NULL" json:"start_time"`
	EndTime      int64              `xorm:"NOT NULL" json:"end_time"`
	CreatedUnix  timeutil.TimeStamp `xorm:"created" json:"created_unix"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated" json:"updated_unix"`

	// Loaded via join, not stored
	PhaseType *PhaseType `xorm:"-" json:"phase_type,omitempty"`
}

func init() {
	db.RegisterModel(new(Phase))
}

// IsLocked returns true if this phase has ended.
func (p *Phase) IsLocked() bool {
	return p.EndTime < time.Now().Unix()
}

// IsActive returns true if this phase is currently active.
func (p *Phase) IsActive() bool {
	now := time.Now().Unix()
	return p.StartTime <= now && p.EndTime > now
}

// IsFuture returns true if this phase hasn't started yet.
func (p *Phase) IsFuture() bool {
	return p.StartTime > time.Now().Unix()
}

// GetPhasesByActivity returns all phases for an activity, ordered by sort_order.
func GetPhasesByActivity(ctx context.Context, activityKind string, activityID int64) ([]*Phase, error) {
	phases := make([]*Phase, 0)
	err := db.GetEngine(ctx).
		Where("activity_kind = ? AND activity_id = ?", activityKind, activityID).
		OrderBy("sort_order ASC").
		Find(&phases)
	if err != nil {
		return nil, err
	}
	for _, p := range phases {
		pt, err := GetPhaseTypeByID(ctx, p.PhaseTypeID)
		if err != nil {
			return nil, err
		}
		p.PhaseType = pt
	}
	return phases, nil
}

// GetCurrentPhase returns the currently active phase for an activity.
// Returns nil if no phase is active (between phases or no phases defined).
func GetCurrentPhase(ctx context.Context, activityKind string, activityID int64) (*Phase, error) {
	now := time.Now().Unix()
	phase := &Phase{}
	has, err := db.GetEngine(ctx).
		Where("activity_kind = ? AND activity_id = ? AND start_time <= ? AND end_time > ?",
			activityKind, activityID, now, now).
		Get(phase)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	pt, err := GetPhaseTypeByID(ctx, phase.PhaseTypeID)
	if err != nil {
		return nil, err
	}
	phase.PhaseType = pt
	return phase, nil
}

// CreatePhase creates a new phase instance.
func CreatePhase(ctx context.Context, p *Phase) error {
	return db.Insert(ctx, p)
}

// UpdatePhase updates an existing phase.
func UpdatePhase(ctx context.Context, p *Phase) error {
	_, err := db.GetEngine(ctx).ID(p.ID).Cols("start_time", "end_time", "sort_order", "custom_name", "updated_unix").Update(p)
	return err
}

// DeletePhase deletes a phase by ID.
func DeletePhase(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(&Phase{})
	return err
}

// PhaseSortOrder holds an ID and its new sort order for batch updates.
type PhaseSortOrder struct {
	ID        int64 `json:"id"`
	SortOrder int   `json:"sort_order"`
}

// BatchUpdatePhaseSortOrder updates sort_order for multiple phases.
func BatchUpdatePhaseSortOrder(ctx context.Context, orders []PhaseSortOrder) error {
	sess := db.GetEngine(ctx)
	for _, o := range orders {
		if _, err := sess.ID(o.ID).Cols("sort_order").Update(&Phase{SortOrder: o.SortOrder}); err != nil {
			return err
		}
	}
	return nil
}

// CountPhasesByActivity returns the count of phases for an activity.
func CountPhasesByActivity(ctx context.Context, activityKind string, activityID int64) (int64, error) {
	return db.GetEngine(ctx).Where("activity_kind = ? AND activity_id = ?", activityKind, activityID).Count(&Phase{})
}

// GetFuturePhases returns all phases where end_time > the given timestamp.
func GetFuturePhases(ctx context.Context, afterUnix int64) ([]*Phase, error) {
	phases := make([]*Phase, 0)
	return phases, db.GetEngine(ctx).Where("end_time > ?", afterUnix).Find(&phases)
}

// GetPhaseByID returns a phase by ID with its PhaseType loaded.
func GetPhaseByID(ctx context.Context, id int64) (*Phase, error) {
	p := &Phase{}
	has, err := db.GetEngine(ctx).ID(id).Get(p)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	pt, err := GetPhaseTypeByID(ctx, p.PhaseTypeID)
	if err != nil {
		return nil, err
	}
	p.PhaseType = pt
	return p, nil
}
