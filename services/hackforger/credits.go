// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
)

// GetOrCreateCreditAccount returns the credit account for a user,
// creating one with zero balance if it doesn't exist.
func GetOrCreateCreditAccount(ctx context.Context, userID int64) (*hackforger_model.CreditAccount, error) {
	acct := &hackforger_model.CreditAccount{UserID: userID}
	has, err := db.GetEngine(ctx).Where("user_id = ?", userID).Get(acct)
	if err != nil {
		return nil, err
	}
	if !has {
		acct = &hackforger_model.CreditAccount{UserID: userID, Balance: 0}
		if _, err := db.GetEngine(ctx).Insert(acct); err != nil {
			return nil, err
		}
	}
	return acct, nil
}

// Deposit adds credits to a user's account within a transaction.
func Deposit(ctx context.Context, userID int64, amount int64, reference, note string) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		acct, err := GetOrCreateCreditAccount(ctx, userID)
		if err != nil {
			return err
		}

		acct.Balance += amount
		if _, err := db.GetEngine(ctx).ID(acct.ID).Cols("balance").Update(acct); err != nil {
			return err
		}

		tx := &hackforger_model.CreditTransaction{
			UserID:    userID,
			Type:      hackforger_model.TransactionTypeDeposit,
			Amount:    amount,
			Balance:   acct.Balance,
			Reference: reference,
			Note:      note,
		}
		_, err = db.GetEngine(ctx).Insert(tx)
		return err
	})
}

// Redeem deducts credits and creates an order within a transaction.
func Redeem(ctx context.Context, userID int64, optionID int64) (*hackforger_model.RedeemOrder, error) {
	var order *hackforger_model.RedeemOrder

	err := db.WithTx(ctx, func(ctx context.Context) error {
		// Get option
		option := &hackforger_model.RedeemOption{ID: optionID}
		has, err := db.GetEngine(ctx).Get(option)
		if err != nil {
			return err
		}
		if !has {
			return hackforger_model.ErrOutOfStock{OptionID: optionID}
		}

		// Check stock
		if option.Stock == 0 {
			return hackforger_model.ErrOutOfStock{OptionID: optionID}
		}

		// Get account
		acct, err := GetOrCreateCreditAccount(ctx, userID)
		if err != nil {
			return err
		}

		// Check balance
		if acct.Balance < option.Cost {
			return hackforger_model.ErrInsufficientCredits{
				UserID:    userID,
				Balance:   acct.Balance,
				Requested: option.Cost,
			}
		}

		// Deduct balance
		acct.Balance -= option.Cost
		if _, err := db.GetEngine(ctx).ID(acct.ID).Cols("balance").Update(acct); err != nil {
			return err
		}

		// Deduct stock (if not unlimited)
		if option.Stock > 0 {
			option.Stock--
			if _, err := db.GetEngine(ctx).ID(option.ID).Cols("stock").Update(option); err != nil {
				return err
			}
		}

		// Create transaction
		tx := &hackforger_model.CreditTransaction{
			UserID:    userID,
			Type:      hackforger_model.TransactionTypeRedeem,
			Amount:    -option.Cost,
			Balance:   acct.Balance,
			Reference: option.Name,
		}
		if _, err := db.GetEngine(ctx).Insert(tx); err != nil {
			return err
		}

		// Create order
		order = &hackforger_model.RedeemOrder{
			UserID:   userID,
			OptionID: optionID,
			Cost:     option.Cost,
			Status:   hackforger_model.OrderStatusPending,
		}
		_, err = db.GetEngine(ctx).Insert(order)
		return err
	})

	return order, err
}

// ListTransactions returns credit transactions for a user.
func ListTransactions(ctx context.Context, userID int64, opts db.ListOptions) ([]*hackforger_model.CreditTransaction, int64, error) {
	sess := db.GetEngine(ctx).Where("user_id = ?", userID)
	var txns []*hackforger_model.CreditTransaction
	count, err := sess.OrderBy("created_unix DESC").
		Limit(opts.PageSize, (opts.Page-1)*opts.PageSize).
		FindAndCount(&txns)
	return txns, count, err
}

// ListRedeemOptions returns all active redeem options.
func ListRedeemOptions(ctx context.Context) ([]*hackforger_model.RedeemOption, error) {
	var options []*hackforger_model.RedeemOption
	err := db.GetEngine(ctx).Where("is_active = ?", true).Find(&options)
	return options, err
}

// ListOrders returns redeem orders for a user.
func ListOrders(ctx context.Context, userID int64, opts db.ListOptions) ([]*hackforger_model.RedeemOrder, int64, error) {
	sess := db.GetEngine(ctx).Where("user_id = ?", userID)
	var orders []*hackforger_model.RedeemOrder
	count, err := sess.OrderBy("created_unix DESC").
		Limit(opts.PageSize, (opts.Page-1)*opts.PageSize).
		FindAndCount(&orders)
	return orders, count, err
}
