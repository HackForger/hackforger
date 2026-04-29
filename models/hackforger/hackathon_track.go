// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"encoding/json"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// HackathonTrack represents a prize track within a hackathon.
type HackathonTrack struct {
	ID            int64              `xorm:"pk autoincr"`
	HackathonID   int64              `xorm:"INDEX NOT NULL"`
	Name          string             `xorm:"NOT NULL"`
	Description   string             `xorm:"TEXT"`
	PrizeAmount   float64            `xorm:""`
	PrizeCurrency string             `xorm:"VARCHAR(16)"`
	RepoID        int64              `xorm:"INDEX"` // auto-created Forgejo Repository for this track
	PrizeCredits    int64              `xorm:""`
	PrizeDistMode   string             `xorm:"VARCHAR(20) NOT NULL DEFAULT 'winner_takes_all'"`
	PrizeDistRatios string             `xorm:"TEXT NOT NULL DEFAULT ''"`
	CreatedUnix     timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(HackathonTrack))
}

// ErrTrackNotExist represents a "TrackNotExist" kind of error.
type ErrTrackNotExist struct {
	ID int64
}

// IsErrTrackNotExist checks if an error is a ErrTrackNotExist.
func IsErrTrackNotExist(err error) bool {
	_, ok := err.(ErrTrackNotExist)
	return ok
}

func (err ErrTrackNotExist) Error() string {
	return fmt.Sprintf("hackathon track does not exist [id: %d]", err.ID)
}

func (err ErrTrackNotExist) Unwrap() error {
	return util.ErrNotExist
}

// GetTrackByID returns a hackathon track by its ID.
func GetTrackByID(ctx context.Context, id int64) (*HackathonTrack, error) {
	t, exists, err := db.GetByID[HackathonTrack](ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrTrackNotExist{ID: id}
	}
	return t, nil
}

// ListTracksByHackathon returns all tracks for a given hackathon.
func ListTracksByHackathon(ctx context.Context, hackathonID int64) ([]*HackathonTrack, error) {
	var tracks []*HackathonTrack
	err := db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).OrderBy("id ASC").Find(&tracks)
	return tracks, err
}

// CreateTrack inserts a new hackathon track.
func CreateTrack(ctx context.Context, t *HackathonTrack) error {
	return db.Insert(ctx, t)
}

// UpdateTrack updates all columns of a hackathon track.
func UpdateTrack(ctx context.Context, t *HackathonTrack) error {
	_, err := db.GetEngine(ctx).ID(t.ID).AllCols().Update(t)
	return err
}

// DeleteTrack deletes a hackathon track by ID.
func DeleteTrack(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(new(HackathonTrack))
	return err
}

// CountTracksByHackathon returns the number of tracks for a given hackathon.
func CountTracksByHackathon(ctx context.Context, hackathonID int64) (int64, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonTrack))
}

// PrizeDistRatio represents the prize distribution percentage for a given rank.
type PrizeDistRatio struct {
	Rank int `json:"rank"`
	Pct  int `json:"pct"`
}

// ParsePrizeDistRatios parses a JSON string into a slice of PrizeDistRatio.
// An empty string returns nil (no ratios defined).
func ParsePrizeDistRatios(s string) ([]PrizeDistRatio, error) {
	if s == "" {
		return nil, nil
	}
	var ratios []PrizeDistRatio
	if err := json.Unmarshal([]byte(s), &ratios); err != nil {
		return nil, fmt.Errorf("invalid prize distribution ratios JSON: %w", err)
	}
	return ratios, nil
}

// ErrInvalidDistRatios represents a validation error for prize distribution ratios.
type ErrInvalidDistRatios struct {
	Reason string
}

// IsErrInvalidDistRatios checks if an error is a ErrInvalidDistRatios.
func IsErrInvalidDistRatios(err error) bool {
	_, ok := err.(ErrInvalidDistRatios)
	return ok
}

func (err ErrInvalidDistRatios) Error() string {
	return fmt.Sprintf("invalid prize distribution ratios: %s", err.Reason)
}

func (err ErrInvalidDistRatios) Unwrap() error {
	return util.ErrInvalidArgument
}

// ValidatePrizeDistRatios validates that the ratios have positive percentages,
// sum to exactly 100, and have contiguous ranks starting from 1.
func ValidatePrizeDistRatios(ratios []PrizeDistRatio) error {
	if len(ratios) == 0 {
		return ErrInvalidDistRatios{Reason: "ratios must not be empty"}
	}
	sum := 0
	for i, r := range ratios {
		if r.Pct <= 0 {
			return ErrInvalidDistRatios{Reason: fmt.Sprintf("pct must be > 0 for rank %d", r.Rank)}
		}
		if r.Rank != i+1 {
			return ErrInvalidDistRatios{Reason: fmt.Sprintf("ranks must be contiguous 1..N, got rank %d at position %d", r.Rank, i+1)}
		}
		sum += r.Pct
	}
	if sum != 100 {
		return ErrInvalidDistRatios{Reason: fmt.Sprintf("pct sum must be 100, got %d", sum)}
	}
	return nil
}

// ErrInvalidPrizeDistMode represents a validation error for prize distribution mode.
type ErrInvalidPrizeDistMode struct {
	Mode string
}

// IsErrInvalidPrizeDistMode checks if an error is a ErrInvalidPrizeDistMode.
func IsErrInvalidPrizeDistMode(err error) bool {
	_, ok := err.(ErrInvalidPrizeDistMode)
	return ok
}

func (err ErrInvalidPrizeDistMode) Error() string {
	return fmt.Sprintf("invalid prize_dist_mode %q (must be winner_takes_all, tiered, or equal)", err.Mode)
}

func (err ErrInvalidPrizeDistMode) Unwrap() error {
	return util.ErrInvalidArgument
}

// ValidatePrizeDistConfig validates a prize distribution configuration.
// mode must be one of: winner_takes_all, tiered, equal.
// For tiered mode, ratiosJSON must parse to a valid PrizeDistRatio slice
// (non-empty, contiguous ranks, percentages summing to 100).
// For non-tiered modes, ratiosJSON is not inspected.
func ValidatePrizeDistConfig(mode, ratiosJSON string) error {
	switch mode {
	case "winner_takes_all", "equal":
		return nil
	case "tiered":
		ratios, err := ParsePrizeDistRatios(ratiosJSON)
		if err != nil {
			return err
		}
		return ValidatePrizeDistRatios(ratios)
	default:
		return ErrInvalidPrizeDistMode{Mode: mode}
	}
}
