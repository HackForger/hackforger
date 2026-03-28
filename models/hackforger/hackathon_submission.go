// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
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
	RepoID         int64              `xorm:"INDEX"`
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

// TableName returns the XORM table name for HackathonSubmission.
func (s *HackathonSubmission) TableName() string {
	return "hackforger_hackathon_submission"
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
