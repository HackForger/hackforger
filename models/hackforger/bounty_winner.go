// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

// BountyWinner represents a winner of a bounty.
type BountyWinner struct {
	ID          int64              `xorm:"pk autoincr"`
	BountyID    int64              `xorm:"INDEX NOT NULL"`
	UserID      int64              `xorm:"INDEX NOT NULL"`
	Rank        int                `xorm:"NOT NULL DEFAULT 1"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
}

func init() {
	db.RegisterModel(new(BountyWinner))
}
