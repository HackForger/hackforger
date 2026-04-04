// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "add credits fulfill + hackathon prize fields",
		Upgrade:     addCreditsFulfillHackathonPrizes,
	})
}

type v14gHackathonTrack struct {
	PrizeDistMode   string `xorm:"VARCHAR(20) NOT NULL DEFAULT 'winner_takes_all'"`
	PrizeDistRatios string `xorm:"TEXT NOT NULL DEFAULT ''"`
}

func (v14gHackathonTrack) TableName() string { return "hackathon_track" }

type v14gRedeemOption struct {
	FulfillMode string `xorm:"VARCHAR(16) NOT NULL DEFAULT 'manual'"`
}

func (v14gRedeemOption) TableName() string { return "redeem_option" }

type v14gRedeemOrder struct {
	DeliveryType  string `xorm:"VARCHAR(32) NOT NULL DEFAULT ''"`
	DeliveryValue string `xorm:"TEXT NOT NULL DEFAULT ''"`
}

func (v14gRedeemOrder) TableName() string { return "redeem_order" }

type v14gRedeemOptionKey struct {
	ID        int64  `xorm:"pk autoincr"`
	OptionID  int64  `xorm:"INDEX NOT NULL"`
	KeyValue  string `xorm:"TEXT NOT NULL"`
	IsUsed    bool   `xorm:"NOT NULL DEFAULT false"`
	OrderID   int64  `xorm:"INDEX"`
	CreatedUnix int64 `xorm:"created"`
}

func (v14gRedeemOptionKey) TableName() string { return "redeem_option_key" }

func addCreditsFulfillHackathonPrizes(x *xorm.Engine) error {
	if err := x.Sync(new(v14gHackathonTrack)); err != nil {
		return err
	}
	if err := x.Sync(new(v14gRedeemOption)); err != nil {
		return err
	}
	if err := x.Sync(new(v14gRedeemOrder)); err != nil {
		return err
	}
	return x.Sync(new(v14gRedeemOptionKey))
}
