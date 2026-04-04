// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

// CreditAccount represents a user's credit balance.
type CreditAccount struct {
	ID          int64              `xorm:"pk autoincr"`
	UserID      int64              `xorm:"UNIQUE NOT NULL"`
	Balance     int64              `xorm:"NOT NULL DEFAULT 0"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(CreditAccount))
}

// ErrInsufficientCredits represents a "InsufficientCredits" kind of error.
type ErrInsufficientCredits struct {
	UserID    int64
	Balance   int64
	Requested int64
}

// IsErrInsufficientCredits checks if an error is a ErrInsufficientCredits.
func IsErrInsufficientCredits(err error) bool {
	_, ok := err.(ErrInsufficientCredits)
	return ok
}

func (err ErrInsufficientCredits) Error() string {
	return fmt.Sprintf("insufficient credits [user_id: %d, balance: %d, requested: %d]", err.UserID, err.Balance, err.Requested)
}

func (err ErrInsufficientCredits) Unwrap() error {
	return util.ErrInvalidArgument
}

// GetCreditAccount returns a user's credit account, or nil if not found.
func GetCreditAccount(ctx context.Context, userID int64) (*CreditAccount, error) {
	a := new(CreditAccount)
	has, err := db.GetEngine(ctx).Where("user_id = ?", userID).Get(a)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return a, nil
}

// ListCreditAccountsOptions holds options for listing credit accounts.
type ListCreditAccountsOptions struct {
	db.ListOptions
}

func (opts ListCreditAccountsOptions) ToConds() builder.Cond {
	return builder.NewCond()
}

// ListCreditAccounts returns credit accounts matching the given options.
func ListCreditAccounts(ctx context.Context, opts ListCreditAccountsOptions) ([]*CreditAccount, int64, error) {
	return db.FindAndCount[CreditAccount](ctx, opts)
}

// LeaderboardEntry represents a single row in the credit leaderboard.
type LeaderboardEntry struct {
	Rank     int
	UserID   int64
	Username string
	FullName string
	Balance  int64
}

// GetCreditLeaderboard returns users ordered by credit balance descending,
// with pagination. It joins with the user table for display names.
func GetCreditLeaderboard(ctx context.Context, page, pageSize int) ([]*LeaderboardEntry, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	e := db.GetEngine(ctx)

	total, err := e.Table("credit_account").
		Join("INNER", "`user`", "`credit_account`.user_id = `user`.id").
		Where("`user`.type = 0 AND `user`.is_active = ?", true).
		Where("`credit_account`.balance > 0").
		Count(new(CreditAccount))
	if err != nil {
		return nil, 0, err
	}

	type row struct {
		UserID   int64  `xorm:"user_id"`
		Name     string `xorm:"name"`
		FullName string `xorm:"full_name"`
		Balance  int64  `xorm:"balance"`
	}

	var rows []row
	err = e.Table("credit_account").
		Select("`credit_account`.user_id, `user`.name, `user`.full_name, `credit_account`.balance").
		Join("INNER", "`user`", "`credit_account`.user_id = `user`.id").
		Where("`user`.type = 0 AND `user`.is_active = ?", true).
		Where("`credit_account`.balance > 0").
		OrderBy("`credit_account`.balance DESC").
		Limit(pageSize, (page-1)*pageSize).
		Find(&rows)
	if err != nil {
		return nil, 0, err
	}

	entries := make([]*LeaderboardEntry, 0, len(rows))
	for i, r := range rows {
		entries = append(entries, &LeaderboardEntry{
			Rank:     (page-1)*pageSize + i + 1,
			UserID:   r.UserID,
			Username: r.Name,
			FullName: r.FullName,
			Balance:  r.Balance,
		})
	}
	return entries, total, nil
}
