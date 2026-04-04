// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"encoding/json"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

// PhaseType defines a catalog entry for phase types that an admin can configure.
// Each activity kind (hackathon, bounty, grant) has its own set of phase types.
type PhaseType struct {
	ID              int64              `xorm:"pk autoincr"`
	ActivityKind    string             `xorm:"VARCHAR(20) NOT NULL INDEX"`
	Key             string             `xorm:"VARCHAR(50) NOT NULL"`
	DisplayNameI18n string             `xorm:"VARCHAR(100) NOT NULL"`
	IsUnique        bool               `xorm:"NOT NULL DEFAULT true"`
	AllowedActions  string             `xorm:"TEXT NOT NULL"` // JSON array of action strings
	DefaultOrder    int                `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix     timeutil.TimeStamp `xorm:"created"`
}

func init() {
	db.RegisterModel(new(PhaseType))
}

// GetAllowedActions parses the JSON array of allowed action strings.
func (pt *PhaseType) GetAllowedActions() ([]string, error) {
	var actions []string
	err := json.Unmarshal([]byte(pt.AllowedActions), &actions)
	return actions, err
}

// HasAction checks if the given action is in this phase type's allowed actions.
func (pt *PhaseType) HasAction(action string) bool {
	actions, err := pt.GetAllowedActions()
	if err != nil {
		return false
	}
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

// GetPhaseTypesByActivityKind returns all phase types for a given activity kind.
func GetPhaseTypesByActivityKind(ctx context.Context, activityKind string) ([]*PhaseType, error) {
	types := make([]*PhaseType, 0)
	return types, db.GetEngine(ctx).
		Where("activity_kind = ?", activityKind).
		OrderBy("default_order ASC").
		Find(&types)
}

// GetPhaseTypeByID returns a phase type by ID.
func GetPhaseTypeByID(ctx context.Context, id int64) (*PhaseType, error) {
	pt := &PhaseType{}
	has, err := db.GetEngine(ctx).ID(id).Get(pt)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return pt, nil
}

// CreatePhaseType creates a new phase type.
func CreatePhaseType(ctx context.Context, pt *PhaseType) error {
	return db.Insert(ctx, pt)
}

// UpdatePhaseType updates an existing phase type.
func UpdatePhaseType(ctx context.Context, pt *PhaseType) error {
	_, err := db.GetEngine(ctx).ID(pt.ID).AllCols().Update(pt)
	return err
}

// DeletePhaseType deletes a phase type by ID.
func DeletePhaseType(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(&PhaseType{})
	return err
}
