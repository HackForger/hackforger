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

// GrantRoundStatus represents the status of a grant round.
type GrantRoundStatus int

const (
	GrantRoundStatusDraft       GrantRoundStatus = iota // 0
	GrantRoundStatusOpen                                // 1
	GrantRoundStatusReview                              // 2
	GrantRoundStatusFinalized                           // 3
	GrantRoundStatusDistributed                         // 4
	GrantRoundStatusCancelled                           // 5
)

// GrantRoundStatusNames maps status values to human-readable names.
var GrantRoundStatusNames = map[GrantRoundStatus]string{
	GrantRoundStatusDraft:       "draft",
	GrantRoundStatusOpen:        "open",
	GrantRoundStatusReview:      "review",
	GrantRoundStatusFinalized:   "finalized",
	GrantRoundStatusDistributed: "distributed",
	GrantRoundStatusCancelled:   "cancelled",
}

// GrantRound represents a grant funding round.
type GrantRound struct {
	ID            int64              `xorm:"pk autoincr"`
	OrgID         int64              `xorm:"INDEX NOT NULL"`
	OwnerID       int64              `xorm:"INDEX NOT NULL"`
	Name          string             `xorm:"NOT NULL"`
	Slug          string             `xorm:"UNIQUE NOT NULL"`
	Description   string             `xorm:"TEXT"`
	Status        GrantRoundStatus   `xorm:"NOT NULL DEFAULT 0"`
	Budget        float64            `xorm:"NOT NULL DEFAULT 0"`
	Currency      string             `xorm:"VARCHAR(16) NOT NULL DEFAULT 'USD'"`
	BudgetCredits int64              `xorm:"NOT NULL DEFAULT 0"`
	Deadline      timeutil.TimeStamp `xorm:""`
	CreatedUnix   timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix   timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(GrantRound))
}

// ErrGrantRoundNotExist represents a "GrantRoundNotExist" kind of error.
type ErrGrantRoundNotExist struct {
	ID int64
}

// IsErrGrantRoundNotExist checks if an error is a ErrGrantRoundNotExist.
func IsErrGrantRoundNotExist(err error) bool {
	_, ok := err.(ErrGrantRoundNotExist)
	return ok
}

func (err ErrGrantRoundNotExist) Error() string {
	return fmt.Sprintf("grant round does not exist [id: %d]", err.ID)
}

func (err ErrGrantRoundNotExist) Unwrap() error {
	return util.ErrNotExist
}

// CreateGrantRound creates a new grant round.
func CreateGrantRound(ctx context.Context, r *GrantRound) error {
	_, err := db.GetEngine(ctx).Insert(r)
	return err
}

// GetGrantRoundByID returns a grant round by its ID.
func GetGrantRoundByID(ctx context.Context, id int64) (*GrantRound, error) {
	r := new(GrantRound)
	has, err := db.GetEngine(ctx).ID(id).Get(r)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrGrantRoundNotExist{ID: id}
	}
	return r, nil
}

// ListGrantRoundsOptions holds options for listing grant rounds.
type ListGrantRoundsOptions struct {
	db.ListOptions
	OrgID  int64
	Status *GrantRoundStatus
}

func (opts ListGrantRoundsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.OrgID > 0 {
		cond = cond.And(builder.Eq{"org_id": opts.OrgID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"status": *opts.Status})
	}
	return cond
}

// ListGrantRounds returns grant rounds matching the given options.
func ListGrantRounds(ctx context.Context, opts ListGrantRoundsOptions) ([]*GrantRound, int64, error) {
	return db.FindAndCount[GrantRound](ctx, opts)
}

// UpdateGrantRound updates an existing grant round.
func UpdateGrantRound(ctx context.Context, r *GrantRound) error {
	_, err := db.GetEngine(ctx).ID(r.ID).AllCols().Update(r)
	return err
}
