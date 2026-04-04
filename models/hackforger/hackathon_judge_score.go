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

// HackathonJudgeScore represents a judge's score for a submission on a specific criteria.
type HackathonJudgeScore struct {
	ID           int64              `xorm:"pk autoincr"`
	HackathonID  int64             `xorm:"INDEX NOT NULL"`
	SubmissionID int64             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	JudgeID      int64             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CriteriaID   int64             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Score        float64           `xorm:"NOT NULL DEFAULT 0"`
	Comment      string            `xorm:"TEXT"`
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
	CriteriaID   int64
}

// IsErrDuplicateScore checks if an error is a ErrDuplicateScore.
func IsErrDuplicateScore(err error) bool {
	_, ok := err.(ErrDuplicateScore)
	return ok
}

func (err ErrDuplicateScore) Error() string {
	return fmt.Sprintf("judge has already scored this submission criteria [judge_id: %d, submission_id: %d, criteria_id: %d]", err.JudgeID, err.SubmissionID, err.CriteriaID)
}

func (err ErrDuplicateScore) Unwrap() error {
	return util.ErrAlreadyExist
}

// GetScore returns the score record for a given judge, submission, and criteria, or nil if none exists.
func GetScore(ctx context.Context, judgeID, submissionID, criteriaID int64) (*HackathonJudgeScore, error) {
	s := &HackathonJudgeScore{}
	has, err := db.GetEngine(ctx).Where("judge_id = ? AND submission_id = ? AND criteria_id = ?", judgeID, submissionID, criteriaID).Get(s)
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

// ListScoresByJudgeAndSubmission returns all criteria scores for a given judge and submission.
func ListScoresByJudgeAndSubmission(ctx context.Context, judgeID, submissionID int64) ([]*HackathonJudgeScore, error) {
	var scores []*HackathonJudgeScore
	err := db.GetEngine(ctx).Where("judge_id = ? AND submission_id = ?", judgeID, submissionID).Find(&scores)
	return scores, err
}

// CreateScore inserts a new judge score, returning ErrDuplicateScore if one already exists
// for the same judge + submission + criteria combination.
func CreateScore(ctx context.Context, s *HackathonJudgeScore) error {
	exists, err := db.GetEngine(ctx).Where("judge_id = ? AND submission_id = ? AND criteria_id = ?", s.JudgeID, s.SubmissionID, s.CriteriaID).Exist(new(HackathonJudgeScore))
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateScore{JudgeID: s.JudgeID, SubmissionID: s.SubmissionID, CriteriaID: s.CriteriaID}
	}
	return db.Insert(ctx, s)
}

// UpdateScore updates the score and comment fields of an existing judge score.
func UpdateScore(ctx context.Context, s *HackathonJudgeScore) error {
	_, err := db.GetEngine(ctx).ID(s.ID).Cols("score", "comment").Update(s)
	return err
}

// HasAllJudgesScored checks if every assigned judge for a track has scored a given submission
// on all enabled criteria. Cross-references the HackathonJudge and HackathonTrackCriteria tables.
func HasAllJudgesScored(ctx context.Context, hackathonID, trackID, submissionID int64) (bool, error) {
	// 1. Get judges for this track.
	judges, err := ListJudges(ctx, hackathonID, trackID)
	if err != nil {
		return false, err
	}
	if len(judges) == 0 {
		return false, nil
	}

	// 2. Get enabled criteria IDs for this track.
	var enabledLinks []*HackathonTrackCriteria
	err = db.GetEngine(ctx).Where("track_id = ? AND enabled = ?", trackID, true).Find(&enabledLinks)
	if err != nil {
		return false, err
	}
	if len(enabledLinks) == 0 {
		return false, nil
	}

	criteriaIDs := make([]int64, 0, len(enabledLinks))
	for _, link := range enabledLinks {
		criteriaIDs = append(criteriaIDs, link.CriteriaID)
	}

	// 3. For each judge, count scores that match the enabled criteria.
	expectedCount := int64(len(criteriaIDs))
	for _, judge := range judges {
		count, err := db.GetEngine(ctx).
			Where("judge_id = ? AND submission_id = ?", judge.UserID, submissionID).
			In("criteria_id", criteriaIDs).
			Count(new(HackathonJudgeScore))
		if err != nil {
			return false, err
		}
		if count < expectedCount {
			return false, nil
		}
	}

	return true, nil
}
