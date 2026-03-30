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

// HackathonJudgeCriteria represents a scoring criterion for a hackathon.
type HackathonJudgeCriteria struct {
	ID          int64              `xorm:"pk autoincr"`
	HackathonID int64             `xorm:"INDEX NOT NULL"`
	Name        string             `xorm:"NOT NULL"`
	Description string             `xorm:"TEXT"`
	MaxScore    float64            `xorm:"NOT NULL DEFAULT 10"`
	Weight      float64            `xorm:"NOT NULL DEFAULT 25"`
	SortOrder   int                `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(HackathonJudgeCriteria))
}

// ErrCriteriaNotExist represents a "CriteriaNotExist" kind of error.
type ErrCriteriaNotExist struct {
	ID int64
}

// IsErrCriteriaNotExist checks if an error is a ErrCriteriaNotExist.
func IsErrCriteriaNotExist(err error) bool {
	_, ok := err.(ErrCriteriaNotExist)
	return ok
}

func (err ErrCriteriaNotExist) Error() string {
	return fmt.Sprintf("hackathon judge criteria does not exist [id: %d]", err.ID)
}

func (err ErrCriteriaNotExist) Unwrap() error {
	return util.ErrNotExist
}

// CreateCriteria inserts a new hackathon judge criteria.
func CreateCriteria(ctx context.Context, c *HackathonJudgeCriteria) error {
	return db.Insert(ctx, c)
}

// UpdateCriteria updates all columns of a hackathon judge criteria.
func UpdateCriteria(ctx context.Context, c *HackathonJudgeCriteria) error {
	_, err := db.GetEngine(ctx).ID(c.ID).AllCols().Update(c)
	return err
}

// DeleteCriteria deletes a hackathon judge criteria by ID.
func DeleteCriteria(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(new(HackathonJudgeCriteria))
	return err
}

// ListCriteriaByHackathon returns all criteria for a given hackathon, ordered by sort_order.
func ListCriteriaByHackathon(ctx context.Context, hackathonID int64) ([]*HackathonJudgeCriteria, error) {
	var criteria []*HackathonJudgeCriteria
	err := db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).OrderBy("sort_order ASC").Find(&criteria)
	return criteria, err
}

// GetCriteriaByID returns a hackathon judge criteria by its ID.
func GetCriteriaByID(ctx context.Context, id int64) (*HackathonJudgeCriteria, error) {
	c, exists, err := db.GetByID[HackathonJudgeCriteria](ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCriteriaNotExist{ID: id}
	}
	return c, nil
}

// CountCriteriaByHackathon returns the number of criteria for a given hackathon.
func CountCriteriaByHackathon(ctx context.Context, hackathonID int64) (int64, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonJudgeCriteria))
}
