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
