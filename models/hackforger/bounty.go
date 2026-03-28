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

// BountyStatus represents the status of a bounty.
type BountyStatus int

const (
	BountyStatusOpen      BountyStatus = 0
	BountyStatusClaimed   BountyStatus = 1
	BountyStatusInReview  BountyStatus = 2
	BountyStatusCompleted BountyStatus = 3
	BountyStatusPaid      BountyStatus = 4
	BountyStatusExpired   BountyStatus = 5
	BountyStatusCancelled BountyStatus = 6
)

// BountyMode represents how a bounty is assigned.
type BountyMode int

const (
	BountyModeExclusive   BountyMode = 0
	BountyModeCompetitive BountyMode = 1
)

// Bounty represents a bounty attached to a repository issue.
type Bounty struct {
	ID          int64              `xorm:"pk autoincr"`
	RepoID      int64              `xorm:"INDEX NOT NULL"`
	IssueID     int64              `xorm:"UNIQUE NOT NULL"`
	PublisherID int64              `xorm:"INDEX NOT NULL"`
	ClaimerID   int64              `xorm:"INDEX"`
	Title       string             `xorm:"NOT NULL"`
	Status      BountyStatus       `xorm:"NOT NULL DEFAULT 0"`
	Mode        BountyMode         `xorm:"NOT NULL DEFAULT 0"`
	Deadline    timeutil.TimeStamp `xorm:""`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(Bounty))
}

// ErrBountyNotExist represents a "BountyNotExist" kind of error.
type ErrBountyNotExist struct {
	ID      int64
	IssueID int64
}

// IsErrBountyNotExist checks if an error is a ErrBountyNotExist.
func IsErrBountyNotExist(err error) bool {
	_, ok := err.(ErrBountyNotExist)
	return ok
}

func (err ErrBountyNotExist) Error() string {
	return fmt.Sprintf("bounty does not exist [id: %d, issue_id: %d]", err.ID, err.IssueID)
}

func (err ErrBountyNotExist) Unwrap() error {
	return util.ErrNotExist
}

// ErrBountyAlreadyExists represents a "BountyAlreadyExists" kind of error.
type ErrBountyAlreadyExists struct {
	IssueID int64
}

// IsErrBountyAlreadyExists checks if an error is a ErrBountyAlreadyExists.
func IsErrBountyAlreadyExists(err error) bool {
	_, ok := err.(ErrBountyAlreadyExists)
	return ok
}

func (err ErrBountyAlreadyExists) Error() string {
	return fmt.Sprintf("bounty already exists for issue [issue_id: %d]", err.IssueID)
}

func (err ErrBountyAlreadyExists) Unwrap() error {
	return util.ErrAlreadyExist
}

// CreateBounty creates a new bounty. Returns ErrBountyAlreadyExists if a
// bounty already exists for the same issue.
func CreateBounty(ctx context.Context, b *Bounty) error {
	has, err := db.GetEngine(ctx).Where("issue_id = ?", b.IssueID).Exist(new(Bounty))
	if err != nil {
		return err
	}
	if has {
		return ErrBountyAlreadyExists{IssueID: b.IssueID}
	}
	_, err = db.GetEngine(ctx).Insert(b)
	return err
}

// GetBountyByID returns a bounty by its ID.
func GetBountyByID(ctx context.Context, id int64) (*Bounty, error) {
	b := new(Bounty)
	has, err := db.GetEngine(ctx).ID(id).Get(b)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrBountyNotExist{ID: id}
	}
	return b, nil
}

// GetBountyByIssueID returns a bounty by its issue ID.
func GetBountyByIssueID(ctx context.Context, issueID int64) (*Bounty, error) {
	b := new(Bounty)
	has, err := db.GetEngine(ctx).Where("issue_id = ?", issueID).Get(b)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrBountyNotExist{IssueID: issueID}
	}
	return b, nil
}

// ListBountiesOptions holds options for listing bounties.
type ListBountiesOptions struct {
	db.ListOptions
	RepoID      int64
	PublisherID int64
	Status      *BountyStatus
}

func (opts ListBountiesOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.RepoID > 0 {
		cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	}
	if opts.PublisherID > 0 {
		cond = cond.And(builder.Eq{"publisher_id": opts.PublisherID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"status": *opts.Status})
	}
	return cond
}

// ListBounties returns bounties matching the given options.
func ListBounties(ctx context.Context, opts ListBountiesOptions) ([]*Bounty, int64, error) {
	return db.FindAndCount[Bounty](ctx, opts)
}

// UpdateBounty updates an existing bounty.
func UpdateBounty(ctx context.Context, b *Bounty) error {
	_, err := db.GetEngine(ctx).ID(b.ID).AllCols().Update(b)
	return err
}

// DeleteBounty deletes a bounty by its ID.
func DeleteBounty(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(new(Bounty))
	return err
}
