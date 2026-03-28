// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// HackathonJudgeScore represents a judge's score for a submission.
type HackathonJudgeScore struct {
	ID           int64              `xorm:"pk autoincr"`
	HackathonID  int64              `xorm:"INDEX NOT NULL"`
	SubmissionID int64              `xorm:"INDEX NOT NULL"`
	JudgeID      int64              `xorm:"INDEX NOT NULL"`
	Score        float64            `xorm:"NOT NULL DEFAULT 0"`
	Comment      string             `xorm:"TEXT"`
	CreatedUnix  timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(HackathonJudgeScore))
}

// TableName returns the XORM table name for HackathonJudgeScore.
func (s *HackathonJudgeScore) TableName() string {
	return "hackforger_hackathon_judge_score"
}

// ErrDuplicateScore represents a "DuplicateScore" kind of error.
type ErrDuplicateScore struct {
	JudgeID      int64
	SubmissionID int64
}

// IsErrDuplicateScore checks if an error is a ErrDuplicateScore.
func IsErrDuplicateScore(err error) bool {
	_, ok := err.(ErrDuplicateScore)
	return ok
}

func (err ErrDuplicateScore) Error() string {
	return fmt.Sprintf("judge has already scored this submission [judge_id: %d, submission_id: %d]", err.JudgeID, err.SubmissionID)
}

func (err ErrDuplicateScore) Unwrap() error {
	return util.ErrAlreadyExist
}
