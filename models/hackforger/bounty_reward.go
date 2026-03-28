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

// ErrBountyRewardNotExist represents a "BountyRewardNotExist" kind of error.
type ErrBountyRewardNotExist struct {
	ID int64
}

// IsErrBountyRewardNotExist checks if an error is a ErrBountyRewardNotExist.
func IsErrBountyRewardNotExist(err error) bool {
	_, ok := err.(ErrBountyRewardNotExist)
	return ok
}

func (err ErrBountyRewardNotExist) Error() string {
	return fmt.Sprintf("bounty reward does not exist [id: %d]", err.ID)
}

func (err ErrBountyRewardNotExist) Unwrap() error {
	return util.ErrNotExist
}

// CreateBountyReward creates a new reward.
func CreateBountyReward(ctx context.Context, r *BountyReward) error {
	_, err := db.GetEngine(ctx).Insert(r)
	return err
}

// ListBountyRewards returns all rewards for a bounty, ordered by rank.
func ListBountyRewards(ctx context.Context, bountyID int64) ([]*BountyReward, error) {
	var rewards []*BountyReward
	err := db.GetEngine(ctx).Where("bounty_id = ?", bountyID).OrderBy("rank ASC").Find(&rewards)
	return rewards, err
}

// DeleteBountyReward deletes a reward by ID.
func DeleteBountyReward(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(new(BountyReward))
	return err
}
