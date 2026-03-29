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

// GetScore returns the score record for a given judge and submission, or nil if none exists.
func GetScore(ctx context.Context, judgeID, submissionID int64) (*HackathonJudgeScore, error) {
	s := &HackathonJudgeScore{}
	has, err := db.GetEngine(ctx).Where("judge_id = ? AND submission_id = ?", judgeID, submissionID).Get(s)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return s, nil
}

// ListScoresBySubmission returns all scores for a given submission.
func ListScoresBySubmission(ctx context.Context, submissionID int64) ([]*HackathonJudgeScore, error) {
	var scores []*HackathonJudgeScore
	err := db.GetEngine(ctx).Where("submission_id = ?", submissionID).Find(&scores)
	return scores, err
}

// CreateScore inserts a new judge score, returning ErrDuplicateScore if one already exists.
func CreateScore(ctx context.Context, s *HackathonJudgeScore) error {
	exists, err := db.GetEngine(ctx).Where("judge_id = ? AND submission_id = ?", s.JudgeID, s.SubmissionID).Exist(new(HackathonJudgeScore))
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateScore{JudgeID: s.JudgeID, SubmissionID: s.SubmissionID}
	}
	return db.Insert(ctx, s)
}

// UpdateScore updates the score and comment fields of an existing judge score.
func UpdateScore(ctx context.Context, s *HackathonJudgeScore) error {
	_, err := db.GetEngine(ctx).ID(s.ID).Cols("score", "comment").Update(s)
	return err
}

// HasAllJudgesScored checks if every assigned judge has scored a given submission.
// Cross-references the HackathonJudge table.
func HasAllJudgesScored(ctx context.Context, hackathonID, submissionID int64) (bool, error) {
	judgeCount, err := db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonJudge))
	if err != nil {
		return false, err
	}
	if judgeCount == 0 {
		return false, nil
	}
	scoreCount, err := db.GetEngine(ctx).Where("submission_id = ?", submissionID).Count(new(HackathonJudgeScore))
	if err != nil {
		return false, err
	}
	return scoreCount >= judgeCount, nil
}
