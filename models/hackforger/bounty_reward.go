// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

// RewardType represents the type of a bounty reward.
type RewardType string

const (
	RewardTypeMoney   RewardType = "money"
	RewardTypeCredits RewardType = "credits"
	RewardTypeOther   RewardType = "other"
)

// BountyReward represents a reward tier for a bounty.
type BountyReward struct {
	ID         int64              `xorm:"pk autoincr"`
	BountyID   int64              `xorm:"INDEX NOT NULL"`
	Type       RewardType         `xorm:"VARCHAR(16) NOT NULL"`
	Amount     float64            `xorm:"NOT NULL DEFAULT 0"`
	Currency   string             `xorm:"VARCHAR(16)"`
	Credits    int64              `xorm:"NOT NULL DEFAULT 0"`
	Rank       int                `xorm:"NOT NULL DEFAULT 1"`
	Note       string             `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
}

func init() {
	db.RegisterModel(new(BountyReward))
}
