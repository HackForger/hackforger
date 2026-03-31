// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

// SubmissionStatus represents the status of a hackathon submission.
type SubmissionStatus int

const (
	SubmissionStatusDraft     SubmissionStatus = iota // 0
	SubmissionStatusSubmitted                         // 1
	SubmissionStatusJudged                            // 2
)

// HackathonSubmission represents a project submission to a hackathon.
type HackathonSubmission struct {
	ID             int64              `xorm:"pk autoincr"`
	HackathonID    int64              `xorm:"INDEX NOT NULL"`
	RegistrationID int64              `xorm:"INDEX NOT NULL"`
	UserID         int64              `xorm:"INDEX NOT NULL"`
	TrackID        int64              `xorm:"INDEX"`
	RepoID         int64              `xorm:"INDEX"`                // user's own project repo
	PRID           int64              `xorm:"INDEX"` // pull request ID
	PullIndex      int64              `xorm:""`      // pull request index within the track repo
	Title          string             `xorm:"NOT NULL"`
	Description    string             `xorm:"TEXT"`
	DemoURL        string             `xorm:"VARCHAR(2048)"`
	Status         SubmissionStatus   `xorm:"NOT NULL DEFAULT 0"`
	TotalScore     float64            `xorm:"NOT NULL DEFAULT 0"`
	Rank           int                `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix    timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix    timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(HackathonSubmission))
}

// ErrSubmissionNotExist represents a "SubmissionNotExist" kind of error.
type ErrSubmissionNotExist struct {
	ID int64
}

// IsErrSubmissionNotExist checks if an error is a ErrSubmissionNotExist.
func IsErrSubmissionNotExist(err error) bool {
	_, ok := err.(ErrSubmissionNotExist)
	return ok
}

func (err ErrSubmissionNotExist) Error() string {
	return fmt.Sprintf("hackathon submission does not exist [id: %d]", err.ID)
}

func (err ErrSubmissionNotExist) Unwrap() error {
	return util.ErrNotExist
}

// ListSubmissionsOptions holds options for listing submissions.
type ListSubmissionsOptions struct {
	db.ListOptions
	HackathonID int64
	TrackID     int64
	Status      *SubmissionStatus
}

func (opts ListSubmissionsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.HackathonID != 0 {
		cond = cond.And(builder.Eq{"hackathon_submission.hackathon_id": opts.HackathonID})
	}
	if opts.TrackID != 0 {
		cond = cond.And(builder.Eq{"hackathon_submission.track_id": opts.TrackID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"hackathon_submission.status": *opts.Status})
	}
	return cond
}

// GetSubmissionByID returns a submission by its ID.
func GetSubmissionByID(ctx context.Context, id int64) (*HackathonSubmission, error) {
	s, exists, err := db.GetByID[HackathonSubmission](ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrSubmissionNotExist{ID: id}
	}
	return s, nil
}

// ListSubmissions returns a paginated list of submissions matching the given options.
func ListSubmissions(ctx context.Context, opts ListSubmissionsOptions) ([]*HackathonSubmission, int64, error) {
	return db.FindAndCount[HackathonSubmission](ctx, opts)
}

// CreateSubmission inserts a new submission.
func CreateSubmission(ctx context.Context, s *HackathonSubmission) error {
	return db.Insert(ctx, s)
}

// UpdateSubmission updates all columns of a submission.
func UpdateSubmission(ctx context.Context, s *HackathonSubmission) error {
	_, err := db.GetEngine(ctx).ID(s.ID).AllCols().Update(s)
	return err
}

// CountSubmissions returns the number of submissions for a given hackathon.
func CountSubmissions(ctx context.Context, hackathonID int64) (int64, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonSubmission))
}

// SubmissionRanking holds a submission ID with its computed score and rank.
type SubmissionRanking struct {
	SubmissionID int64
	TotalScore   float64
	Rank         int
}

// UpdateSubmissionRanks bulk-updates total_score and rank for a list of submissions.
func UpdateSubmissionRanks(ctx context.Context, rankings []SubmissionRanking) error {
	for _, r := range rankings {
		if _, err := db.GetEngine(ctx).ID(r.SubmissionID).
			Cols("total_score", "rank").
			Update(&HackathonSubmission{TotalScore: r.TotalScore, Rank: r.Rank}); err != nil {
			return err
		}
	}
	return nil
}
