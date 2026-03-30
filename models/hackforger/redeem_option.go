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

// ErrRedeemOptionNotExist represents a "RedeemOptionNotExist" kind of error.
type ErrRedeemOptionNotExist struct {
	ID int64
}

// IsErrRedeemOptionNotExist checks if an error is a ErrRedeemOptionNotExist.
func IsErrRedeemOptionNotExist(err error) bool {
	_, ok := err.(ErrRedeemOptionNotExist)
	return ok
}

func (err ErrRedeemOptionNotExist) Error() string {
	return fmt.Sprintf("redeem option does not exist [id: %d]", err.ID)
}

func (err ErrRedeemOptionNotExist) Unwrap() error {
	return util.ErrNotExist
}

// CreateRedeemOption creates a new redeem option.
func CreateRedeemOption(ctx context.Context, opt *RedeemOption) error {
	_, err := db.GetEngine(ctx).Insert(opt)
	return err
}

// GetRedeemOptionByID returns a redeem option by its ID.
func GetRedeemOptionByID(ctx context.Context, id int64) (*RedeemOption, error) {
	opt := new(RedeemOption)
	has, err := db.GetEngine(ctx).ID(id).Get(opt)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrRedeemOptionNotExist{ID: id}
	}
	return opt, nil
}

// UpdateRedeemOption updates an existing redeem option.
func UpdateRedeemOption(ctx context.Context, opt *RedeemOption) error {
	_, err := db.GetEngine(ctx).ID(opt.ID).AllCols().Update(opt)
	return err
}

// ListAllRedeemOptions returns all redeem options (including inactive), for admin use.
func ListAllRedeemOptions(ctx context.Context) ([]*RedeemOption, error) {
	var options []*RedeemOption
	err := db.GetEngine(ctx).OrderBy("id DESC").Find(&options)
	return options, err
}
