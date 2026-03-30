// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	"forgejo.org/models/db"
)

// HackathonTrackCriteria links a judge criteria to a track with optional weight override.
type HackathonTrackCriteria struct {
	ID         int64   `xorm:"pk autoincr"`
	TrackID    int64   `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CriteriaID int64  `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Enabled    bool    `xorm:"NOT NULL DEFAULT true"`
	Weight     float64 `xorm:"NOT NULL DEFAULT 0"`
}

func init() {
	db.RegisterModel(new(HackathonTrackCriteria))
}

// CreateTrackCriteria inserts a new track-criteria link.
func CreateTrackCriteria(ctx context.Context, tc *HackathonTrackCriteria) error {
	return db.Insert(ctx, tc)
}

// UpdateTrackCriteria updates the enabled and weight columns of a track-criteria link.
func UpdateTrackCriteria(ctx context.Context, tc *HackathonTrackCriteria) error {
	_, err := db.GetEngine(ctx).ID(tc.ID).Cols("enabled", "weight").Update(tc)
	return err
}

// ListTrackCriteria returns all criteria links for a given track.
func ListTrackCriteria(ctx context.Context, trackID int64) ([]*HackathonTrackCriteria, error) {
	var links []*HackathonTrackCriteria
	err := db.GetEngine(ctx).Where("track_id = ?", trackID).Find(&links)
	return links, err
}

// GetTrackCriteria returns a track-criteria link by track and criteria IDs.
// Returns nil (not an error) if the link does not exist.
func GetTrackCriteria(ctx context.Context, trackID, criteriaID int64) (*HackathonTrackCriteria, error) {
	tc := &HackathonTrackCriteria{
		TrackID:    trackID,
		CriteriaID: criteriaID,
	}
	has, err := db.GetEngine(ctx).Where("track_id = ? AND criteria_id = ?", trackID, criteriaID).Get(tc)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return tc, nil
}

// SeedTrackCriteria idempotently ensures that every criterion is linked to the track.
// Existing links are left untouched; missing ones are inserted with Enabled=true, Weight=0.
func SeedTrackCriteria(ctx context.Context, trackID int64, criteria []*HackathonJudgeCriteria) error {
	for _, c := range criteria {
		existing, err := GetTrackCriteria(ctx, trackID, c.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		if err := CreateTrackCriteria(ctx, &HackathonTrackCriteria{
			TrackID:    trackID,
			CriteriaID: c.ID,
			Enabled:    true,
			Weight:     0,
		}); err != nil {
			return err
		}
	}
	return nil
}
