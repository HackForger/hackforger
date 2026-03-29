// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
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


