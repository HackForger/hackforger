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

// RedeemOptionKey represents a single key in a pool for auto-fulfill redeem options.
type RedeemOptionKey struct {
	ID          int64              `xorm:"pk autoincr"`
	OptionID    int64              `xorm:"INDEX NOT NULL"`
	KeyValue    string             `xorm:"TEXT NOT NULL"`
	IsUsed      bool               `xorm:"NOT NULL DEFAULT false"`
	OrderID     int64              `xorm:"INDEX"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
}

func init() {
	db.RegisterModel(new(RedeemOptionKey))
}

// ErrKeyPoolEmpty represents an empty key pool error.
type ErrKeyPoolEmpty struct {
	OptionID int64
}

// IsErrKeyPoolEmpty checks if an error is a ErrKeyPoolEmpty.
func IsErrKeyPoolEmpty(err error) bool {
	_, ok := err.(ErrKeyPoolEmpty)
	return ok
}

func (err ErrKeyPoolEmpty) Error() string {
	return fmt.Sprintf("key pool is empty [option_id: %d]", err.OptionID)
}

func (err ErrKeyPoolEmpty) Unwrap() error {
	return util.ErrNotExist
}

// AddKeys bulk-inserts keys into the pool for a given option.
func AddKeys(ctx context.Context, optionID int64, keys []string) error {
	beans := make([]RedeemOptionKey, len(keys))
	for i, k := range keys {
		beans[i] = RedeemOptionKey{
			OptionID: optionID,
			KeyValue: k,
		}
	}
	return db.Insert(ctx, &beans)
}

// ClaimKey atomically claims an available key for the given option and order.
// It uses a transaction with SELECT ... WHERE ... followed by UPDATE for SQLite safety.
// Returns ErrOutOfStock if no available keys exist.
func ClaimKey(ctx context.Context, optionID, orderID int64) (*RedeemOptionKey, error) {
	var key *RedeemOptionKey
	err := db.WithTx(ctx, func(txCtx context.Context) error {
		k := new(RedeemOptionKey)
		has, err := db.GetEngine(txCtx).
			Where("option_id = ? AND is_used = ?", optionID, false).
			OrderBy("id ASC").
			Limit(1).
			Get(k)
		if err != nil {
			return err
		}
		if !has {
			return ErrOutOfStock{OptionID: optionID}
		}

		k.IsUsed = true
		k.OrderID = orderID
		if _, err := db.GetEngine(txCtx).ID(k.ID).Cols("is_used", "order_id").Update(k); err != nil {
			return err
		}

		key = k
		return nil
	})
	return key, err
}

// CountAvailableKeys returns the number of unclaimed keys for a given option.
func CountAvailableKeys(ctx context.Context, optionID int64) (int64, error) {
	return db.GetEngine(ctx).Where("option_id = ? AND is_used = ?", optionID, false).Count(new(RedeemOptionKey))
}

// CountTotalKeys returns the total number of keys for a given option.
func CountTotalKeys(ctx context.Context, optionID int64) (int64, error) {
	return db.GetEngine(ctx).Where("option_id = ?", optionID).Count(new(RedeemOptionKey))
}

// ListKeysByOption returns all keys for a given option.
func ListKeysByOption(ctx context.Context, optionID int64) ([]*RedeemOptionKey, error) {
	var keys []*RedeemOptionKey
	err := db.GetEngine(ctx).Where("option_id = ?", optionID).OrderBy("id ASC").Find(&keys)
	return keys, err
}
