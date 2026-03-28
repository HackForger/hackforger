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

// OrderStatus represents the status of a redeem order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusFulfilled OrderStatus = "fulfilled"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// RedeemOrder represents a user's order to redeem credits for an option.
type RedeemOrder struct {
	ID          int64              `xorm:"pk autoincr"`
	UserID      int64              `xorm:"INDEX NOT NULL"`
	OptionID    int64              `xorm:"INDEX NOT NULL"`
	Cost        int64              `xorm:"NOT NULL"`
	Status      OrderStatus        `xorm:"VARCHAR(16) NOT NULL DEFAULT 'pending'"`
	FulfillNote string             `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(RedeemOrder))
}

// ErrRedeemOrderNotExist represents a "RedeemOrderNotExist" kind of error.
type ErrRedeemOrderNotExist struct {
	ID int64
}

// IsErrRedeemOrderNotExist checks if an error is a ErrRedeemOrderNotExist.
func IsErrRedeemOrderNotExist(err error) bool {
	_, ok := err.(ErrRedeemOrderNotExist)
	return ok
}

func (err ErrRedeemOrderNotExist) Error() string {
	return fmt.Sprintf("redeem order does not exist [id: %d]", err.ID)
}

func (err ErrRedeemOrderNotExist) Unwrap() error {
	return util.ErrNotExist
}

// GetRedeemOrderByID returns a redeem order by its ID.
func GetRedeemOrderByID(ctx context.Context, id int64) (*RedeemOrder, error) {
	order := new(RedeemOrder)
	has, err := db.GetEngine(ctx).ID(id).Get(order)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrRedeemOrderNotExist{ID: id}
	}
	return order, nil
}

// UpdateRedeemOrder updates an existing redeem order.
func UpdateRedeemOrder(ctx context.Context, order *RedeemOrder) error {
	_, err := db.GetEngine(ctx).ID(order.ID).AllCols().Update(order)
	return err
}
