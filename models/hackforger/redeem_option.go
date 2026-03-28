// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// RedeemOption represents an item that can be redeemed with credits.
type RedeemOption struct {
	ID          int64              `xorm:"pk autoincr"`
	Name        string             `xorm:"NOT NULL"`
	Description string             `xorm:"TEXT"`
	Cost        int64              `xorm:"NOT NULL"`
	Stock       int                `xorm:"NOT NULL DEFAULT -1"`
	IsActive    bool               `xorm:"NOT NULL DEFAULT true"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(RedeemOption))
}

// ErrOutOfStock represents a "OutOfStock" kind of error.
type ErrOutOfStock struct {
	OptionID int64
}

// IsErrOutOfStock checks if an error is a ErrOutOfStock.
func IsErrOutOfStock(err error) bool {
	_, ok := err.(ErrOutOfStock)
	return ok
}

func (err ErrOutOfStock) Error() string {
	return fmt.Sprintf("redeem option is out of stock [option_id: %d]", err.OptionID)
}

func (err ErrOutOfStock) Unwrap() error {
	return util.ErrInvalidArgument
}
