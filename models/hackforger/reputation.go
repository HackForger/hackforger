// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

// Reputation tracks a user's overall reputation score.
type Reputation struct {
	ID                 int64              `xorm:"pk autoincr"`
	UserID             int64              `xorm:"UNIQUE NOT NULL"`
	Score              int64              `xorm:"NOT NULL DEFAULT 0"`
	BountiesCompleted  int                `xorm:"NOT NULL DEFAULT 0"`
	HackathonWins      int                `xorm:"NOT NULL DEFAULT 0"`
	GrantsReceived     int                `xorm:"NOT NULL DEFAULT 0"`
	TotalStars         int64              `xorm:"NOT NULL DEFAULT 0"`
	TotalCreditsEarned int64              `xorm:"NOT NULL DEFAULT 0"`
	Tier               string             `xorm:"VARCHAR(20) NOT NULL DEFAULT 'Bronze'"`
	UpdatedUnix        timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(Reputation))
}

// GetOrCreateReputation returns the reputation record for a user,
// creating one if it does not exist.
func GetOrCreateReputation(ctx context.Context, userID int64) (*Reputation, error) {
	r := new(Reputation)
	has, err := db.GetEngine(ctx).Where("user_id = ?", userID).Get(r)
	if err != nil {
		return nil, err
	}
	if has {
		return r, nil
	}

	r = &Reputation{UserID: userID}
	if _, err := db.GetEngine(ctx).Insert(r); err != nil {
		return nil, err
	}
	return r, nil
}

// UpdateReputation updates an existing reputation record.
func UpdateReputation(ctx context.Context, r *Reputation) error {
	_, err := db.GetEngine(ctx).ID(r.ID).AllCols().Update(r)
	return err
}

// ReputationLeaderboard returns the top N users by reputation score.
func ReputationLeaderboard(ctx context.Context, limit int) ([]*Reputation, error) {
	records := make([]*Reputation, 0, limit)
	err := db.GetEngine(ctx).OrderBy("score DESC").Limit(limit).Find(&records)
	if err != nil {
		return nil, err
	}
	return records, nil
}
