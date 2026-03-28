// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// CreditAccount represents a user's credit balance.
type CreditAccount struct {
	ID          int64              `xorm:"pk autoincr"`
	UserID      int64              `xorm:"UNIQUE NOT NULL"`
	Balance     int64              `xorm:"NOT NULL DEFAULT 0"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(CreditAccount))
}

// ErrInsufficientCredits represents a "InsufficientCredits" kind of error.
type ErrInsufficientCredits struct {
	UserID    int64
	Balance   int64
	Requested int64
}

// IsErrInsufficientCredits checks if an error is a ErrInsufficientCredits.
func IsErrInsufficientCredits(err error) bool {
	_, ok := err.(ErrInsufficientCredits)
	return ok
}

func (err ErrInsufficientCredits) Error() string {
	return fmt.Sprintf("insufficient credits [user_id: %d, balance: %d, requested: %d]", err.UserID, err.Balance, err.Requested)
}

func (err ErrInsufficientCredits) Unwrap() error {
	return util.ErrInvalidArgument
}
