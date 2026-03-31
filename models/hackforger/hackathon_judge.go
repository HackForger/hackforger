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

// HackathonJudge represents a judge assignment for a hackathon track.
type HackathonJudge struct {
	ID          int64              `xorm:"pk autoincr"`
	HackathonID int64             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	TrackID     int64             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	UserID      int64             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(HackathonJudge))
}

// ErrDuplicateJudge represents a "judge already assigned" kind of error.
type ErrDuplicateJudge struct {
	HackathonID int64
	TrackID     int64
	UserID      int64
}

// IsErrDuplicateJudge checks if an error is a ErrDuplicateJudge.
func IsErrDuplicateJudge(err error) bool {
	_, ok := err.(ErrDuplicateJudge)
	return ok
}

func (err ErrDuplicateJudge) Error() string {
	return fmt.Sprintf("judge already assigned [hackathon_id: %d, track_id: %d, user_id: %d]", err.HackathonID, err.TrackID, err.UserID)
}

func (err ErrDuplicateJudge) Unwrap() error {
	return util.ErrAlreadyExist
}

// AddJudge assigns a user as a judge for a hackathon track.
func AddJudge(ctx context.Context, hackathonID, trackID, userID int64) error {
	exists, err := db.GetEngine(ctx).Where("hackathon_id = ? AND track_id = ? AND user_id = ?", hackathonID, trackID, userID).Exist(new(HackathonJudge))
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateJudge{HackathonID: hackathonID, TrackID: trackID, UserID: userID}
	}
	return db.Insert(ctx, &HackathonJudge{HackathonID: hackathonID, TrackID: trackID, UserID: userID})
}

// RemoveJudge removes a judge assignment from a hackathon track.
func RemoveJudge(ctx context.Context, hackathonID, trackID, userID int64) error {
	_, err := db.GetEngine(ctx).Where("hackathon_id = ? AND track_id = ? AND user_id = ?", hackathonID, trackID, userID).Delete(new(HackathonJudge))
	return err
}

// ListJudges returns judge assignments for a hackathon.
// If trackID is provided and > 0, results are filtered to that track.
func ListJudges(ctx context.Context, hackathonID int64, trackID ...int64) ([]*HackathonJudge, error) {
	var judges []*HackathonJudge
	sess := db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID)
	if len(trackID) > 0 && trackID[0] > 0 {
		sess = sess.And("track_id = ?", trackID[0])
	}
	err := sess.Find(&judges)
	return judges, err
}

// IsJudge checks if a user is a judge for a specific hackathon track.
func IsJudge(ctx context.Context, hackathonID, trackID, userID int64) (bool, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ? AND track_id = ? AND user_id = ?", hackathonID, trackID, userID).Exist(new(HackathonJudge))
}

// IsJudgeForAnyTrack checks if a user is a judge for any track within a hackathon.
func IsJudgeForAnyTrack(ctx context.Context, hackathonID, userID int64) (bool, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ? AND user_id = ?", hackathonID, userID).Exist(new(HackathonJudge))
}

// CountJudges returns the number of judges assigned to a hackathon.
// If trackID is provided and > 0, the count is filtered to that track.
func CountJudges(ctx context.Context, hackathonID int64, trackID ...int64) (int64, error) {
	sess := db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID)
	if len(trackID) > 0 && trackID[0] > 0 {
		sess = sess.And("track_id = ?", trackID[0])
	}
	return sess.Count(new(HackathonJudge))
}
