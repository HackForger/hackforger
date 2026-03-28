// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

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

// CreateBountyWinner creates a new winner record.
func CreateBountyWinner(ctx context.Context, w *BountyWinner) error {
	_, err := db.GetEngine(ctx).Insert(w)
	return err
}

// ListBountyWinners returns all winners for a bounty, ordered by rank.
func ListBountyWinners(ctx context.Context, bountyID int64) ([]*BountyWinner, error) {
	var winners []*BountyWinner
	err := db.GetEngine(ctx).Where("bounty_id = ?", bountyID).OrderBy("rank ASC").Find(&winners)
	return winners, err
}
