// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

// TransactionType represents the type of a credit transaction.
type TransactionType string

const (
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypeWithdraw TransactionType = "withdraw"
	TransactionTypeReward   TransactionType = "reward"
	TransactionTypeRedeem   TransactionType = "redeem"
	TransactionTypeRefund   TransactionType = "refund"
)

// CreditTransaction represents a single credit ledger entry.
type CreditTransaction struct {
	ID          int64              `xorm:"pk autoincr"`
	UserID      int64              `xorm:"INDEX NOT NULL"`
	Type        TransactionType    `xorm:"VARCHAR(16) NOT NULL"`
	Amount      int64              `xorm:"NOT NULL"`
	Balance     int64              `xorm:"NOT NULL"`
	Reference   string             `xorm:"VARCHAR(255)"`
	Note        string             `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(CreditTransaction))
}

// ListAllTransactionsOptions holds options for listing credit transactions.
type ListAllTransactionsOptions struct {
	db.ListOptions
	UserID int64
}

func (opts ListAllTransactionsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.UserID > 0 {
		cond = cond.And(builder.Eq{"user_id": opts.UserID})
	}
	return cond
}

// ListAllTransactions returns credit transactions matching the given options.
func ListAllTransactions(ctx context.Context, opts ListAllTransactionsOptions) ([]*CreditTransaction, int64, error) {
	return db.FindAndCount[CreditTransaction](ctx, opts)
}
