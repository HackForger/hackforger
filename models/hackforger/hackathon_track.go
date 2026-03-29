// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
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
	PrizeCredits  int64              `xorm:""`
	CreatedUnix   timeutil.TimeStamp `xorm:"INDEX created"`
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
