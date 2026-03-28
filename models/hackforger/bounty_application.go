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

// ApplicationStatus represents the status of a bounty application.
type ApplicationStatus int

const (
	ApplicationStatusPending  ApplicationStatus = iota // 0
	ApplicationStatusAccepted                          // 1
	ApplicationStatusRejected                          // 2
)

// BountyApplication represents a user's application to work on a bounty.
type BountyApplication struct {
	ID          int64              `xorm:"pk autoincr"`
	BountyID    int64              `xorm:"INDEX NOT NULL"`
	UserID      int64              `xorm:"INDEX NOT NULL"`
	Status      ApplicationStatus  `xorm:"NOT NULL DEFAULT 0"`
	Message     string             `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(BountyApplication))
}

// ErrAlreadyApplied represents a "AlreadyApplied" kind of error.
type ErrAlreadyApplied struct {
	BountyID int64
	UserID   int64
}

// IsErrAlreadyApplied checks if an error is a ErrAlreadyApplied.
func IsErrAlreadyApplied(err error) bool {
	_, ok := err.(ErrAlreadyApplied)
	return ok
}

func (err ErrAlreadyApplied) Error() string {
	return fmt.Sprintf("user already applied to bounty [bounty_id: %d, user_id: %d]", err.BountyID, err.UserID)
}

func (err ErrAlreadyApplied) Unwrap() error {
	return util.ErrAlreadyExist
}

// CreateBountyApplication creates a new application.
func CreateBountyApplication(ctx context.Context, app *BountyApplication) error {
	has, err := db.GetEngine(ctx).
		Where("bounty_id = ? AND user_id = ?", app.BountyID, app.UserID).
		Exist(new(BountyApplication))
	if err != nil {
		return err
	}
	if has {
		return ErrAlreadyApplied{BountyID: app.BountyID, UserID: app.UserID}
	}
	_, err = db.GetEngine(ctx).Insert(app)
	return err
}

// GetBountyApplicationByID returns an application by ID.
func GetBountyApplicationByID(ctx context.Context, id int64) (*BountyApplication, error) {
	app := new(BountyApplication)
	has, err := db.GetEngine(ctx).ID(id).Get(app)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("bounty application does not exist [id: %d]", id)
	}
	return app, nil
}

// ListBountyApplicationsOptions holds options for listing applications.
type ListBountyApplicationsOptions struct {
	db.ListOptions
	BountyID int64
	UserID   int64
	Status   *ApplicationStatus
}

func (opts ListBountyApplicationsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.BountyID > 0 {
		cond = cond.And(builder.Eq{"bounty_id": opts.BountyID})
	}
	if opts.UserID > 0 {
		cond = cond.And(builder.Eq{"user_id": opts.UserID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"status": *opts.Status})
	}
	return cond
}

// ListBountyApplications returns applications matching the given options.
func ListBountyApplications(ctx context.Context, opts ListBountyApplicationsOptions) ([]*BountyApplication, int64, error) {
	return db.FindAndCount[BountyApplication](ctx, opts)
}

// UpdateBountyApplication updates an existing application.
func UpdateBountyApplication(ctx context.Context, app *BountyApplication) error {
	_, err := db.GetEngine(ctx).ID(app.ID).AllCols().Update(app)
	return err
}
