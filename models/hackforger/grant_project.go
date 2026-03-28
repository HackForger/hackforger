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

// GrantProjectStatus represents the status of a grant project.
type GrantProjectStatus int

const (
	GrantProjectStatusPending  GrantProjectStatus = iota // 0
	GrantProjectStatusApproved                           // 1
	GrantProjectStatusRejected                           // 2
	GrantProjectStatusFunded                             // 3
)

// GrantProject represents a project submitted to a grant round.
type GrantProject struct {
	ID           int64              `xorm:"pk autoincr"`
	RoundID      int64              `xorm:"INDEX NOT NULL"`
	UserID       int64              `xorm:"INDEX NOT NULL"`
	RepoID       int64              `xorm:"INDEX"`
	Title        string             `xorm:"NOT NULL"`
	Description  string             `xorm:"TEXT"`
	Status       GrantProjectStatus `xorm:"NOT NULL DEFAULT 0"`
	AwardAmount  float64            `xorm:"NOT NULL DEFAULT 0"`
	AwardCredits int64              `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix  timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(GrantProject))
}

// ErrGrantProjectNotExist represents a "GrantProjectNotExist" kind of error.
type ErrGrantProjectNotExist struct {
	ID int64
}

// IsErrGrantProjectNotExist checks if an error is a ErrGrantProjectNotExist.
func IsErrGrantProjectNotExist(err error) bool {
	_, ok := err.(ErrGrantProjectNotExist)
	return ok
}

func (err ErrGrantProjectNotExist) Error() string {
	return fmt.Sprintf("grant project does not exist [id: %d]", err.ID)
}

func (err ErrGrantProjectNotExist) Unwrap() error {
	return util.ErrNotExist
}

// CreateGrantProject creates a new grant project.
func CreateGrantProject(ctx context.Context, p *GrantProject) error {
	_, err := db.GetEngine(ctx).Insert(p)
	return err
}

// GetGrantProjectByID returns a grant project by its ID.
func GetGrantProjectByID(ctx context.Context, id int64) (*GrantProject, error) {
	p := new(GrantProject)
	has, err := db.GetEngine(ctx).ID(id).Get(p)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrGrantProjectNotExist{ID: id}
	}
	return p, nil
}

// ListGrantProjectsByRoundOptions holds options for listing grant projects in a round.
type ListGrantProjectsByRoundOptions struct {
	db.ListOptions
	RoundID int64
	Status  *GrantProjectStatus
}

func (opts ListGrantProjectsByRoundOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.RoundID > 0 {
		cond = cond.And(builder.Eq{"round_id": opts.RoundID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"status": *opts.Status})
	}
	return cond
}

// ListGrantProjectsByRound returns grant projects matching the given options.
func ListGrantProjectsByRound(ctx context.Context, opts ListGrantProjectsByRoundOptions) ([]*GrantProject, int64, error) {
	return db.FindAndCount[GrantProject](ctx, opts)
}

// UpdateGrantProject updates an existing grant project.
func UpdateGrantProject(ctx context.Context, p *GrantProject) error {
	_, err := db.GetEngine(ctx).ID(p.ID).AllCols().Update(p)
	return err
}
