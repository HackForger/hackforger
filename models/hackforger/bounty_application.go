// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
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
