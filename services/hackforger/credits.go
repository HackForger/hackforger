// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/util"
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
// If the option's FulfillMode is "auto", a key is claimed and the order
// is immediately fulfilled inside the same transaction.
func Redeem(ctx context.Context, userID int64, optionID int64) (*hackforger_model.RedeemOrder, error) {
	var order *hackforger_model.RedeemOrder
	var optionName string
	var capturedOption *hackforger_model.RedeemOption

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

		optionName = option.Name
		capturedOption = option

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
		if _, err := db.GetEngine(ctx).Insert(order); err != nil {
			return err
		}

		// Auto-fulfill if option is configured for it
		if option.FulfillMode == "auto" {
			key, err := hackforger_model.ClaimKey(ctx, optionID, order.ID)
			if err != nil {
				return err // rolls back entire tx
			}
			order.Status = hackforger_model.OrderStatusFulfilled
			order.DeliveryType = "license_key"
			order.DeliveryValue = key.KeyValue
			if _, err := db.GetEngine(ctx).ID(order.ID).
				Cols("status", "delivery_type", "delivery_value").Update(order); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sync stock from key pool after auto-fulfill
	if capturedOption != nil && capturedOption.FulfillMode == "auto" {
		_ = syncOptionStock(ctx, optionID)
	}

	// Publish feed event for the redemption
	if err := PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    userID,
		OpType:       hackforger_model.ActionCreditsRedeemed,
		EntityType:   "credits",
		EntityName:   optionName,
		AudienceType: AudienceFollowers,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "credits",
			EntityName: optionName,
		},
	}); err != nil {
		return order, err
	}

	return order, nil
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

// ErrNotAdmin is returned when a non-admin user attempts an admin operation.
type ErrNotAdmin struct {
	UserID int64
}

func (err ErrNotAdmin) Error() string {
	return fmt.Sprintf("user is not admin [user_id: %d]", err.UserID)
}

func (err ErrNotAdmin) Unwrap() error {
	return util.ErrPermissionDenied
}

// IsErrNotAdmin checks if an error is ErrNotAdmin.
func IsErrNotAdmin(err error) bool {
	_, ok := err.(ErrNotAdmin)
	return ok
}

// ErrOrderNotPending is returned when an order operation requires Pending status.
type ErrOrderNotPending struct {
	OrderID int64
	Status  hackforger_model.OrderStatus
}

func (err ErrOrderNotPending) Error() string {
	return fmt.Sprintf("order is not pending [order_id: %d, status: %s]", err.OrderID, err.Status)
}

func (err ErrOrderNotPending) Unwrap() error {
	return util.ErrInvalidArgument
}

// IsErrOrderNotPending checks if an error is ErrOrderNotPending.
func IsErrOrderNotPending(err error) bool {
	_, ok := err.(ErrOrderNotPending)
	return ok
}

// AdminDeposit adds credits to a user's account. Only site admins can call this.
func AdminDeposit(ctx context.Context, admin *user_model.User, userID, amount int64, ref, note string) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}
	return Deposit(ctx, userID, amount, ref, note)
}

// AdminDeduct deducts credits from a user's account. Only site admins can call this.
func AdminDeduct(ctx context.Context, admin *user_model.User, userID, amount int64, ref, note string) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}
	return db.WithTx(ctx, func(ctx context.Context) error {
		acct, err := GetOrCreateCreditAccount(ctx, userID)
		if err != nil {
			return err
		}

		if acct.Balance < amount {
			return hackforger_model.ErrInsufficientCredits{
				UserID:    userID,
				Balance:   acct.Balance,
				Requested: amount,
			}
		}

		acct.Balance -= amount
		if _, err := db.GetEngine(ctx).ID(acct.ID).Cols("balance").Update(acct); err != nil {
			return err
		}

		tx := &hackforger_model.CreditTransaction{
			UserID:    userID,
			Type:      hackforger_model.TransactionTypeWithdraw,
			Amount:    -amount,
			Balance:   acct.Balance,
			Reference: ref,
			Note:      note,
		}
		_, err = db.GetEngine(ctx).Insert(tx)
		return err
	})
}

// FulfillOrder marks a pending order as fulfilled. Only site admins can call this.
func FulfillOrder(ctx context.Context, admin *user_model.User, orderID int64, note, deliveryType, deliveryValue string) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}

	order, err := hackforger_model.GetRedeemOrderByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status != hackforger_model.OrderStatusPending {
		return ErrOrderNotPending{OrderID: orderID, Status: order.Status}
	}

	order.Status = hackforger_model.OrderStatusFulfilled
	order.FulfillNote = note
	order.DeliveryType = deliveryType
	order.DeliveryValue = deliveryValue
	if err := hackforger_model.UpdateRedeemOrder(ctx, order); err != nil {
		return err
	}

	var optName string
	if opt, _ := hackforger_model.GetRedeemOptionByID(ctx, order.OptionID); opt != nil {
		optName = opt.Name
	}
	notifyOrderStatusChange(ctx, order, admin.ID, hackforger_model.ActionOrderFulfilled, optName)
	return nil
}

// notifyOrderStatusChange publishes a feed event when an order status changes.
func notifyOrderStatusChange(ctx context.Context, order *hackforger_model.RedeemOrder, adminID int64, actionType activities_model.ActionType, optionName string) {
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    adminID,
		OpType:       actionType,
		AudienceType: AudienceDirectUser,
		TargetUserID: order.UserID,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "credits",
			EntityName: optionName,
			Extra:      map[string]any{"order_id": order.ID},
		},
	})
}

// CancelOrder cancels a pending order and refunds the credits. Only site admins can call this.
func CancelOrder(ctx context.Context, admin *user_model.User, orderID int64) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}

	var order *hackforger_model.RedeemOrder
	var optName string

	err := db.WithTx(ctx, func(ctx context.Context) error {
		var err error
		order, err = hackforger_model.GetRedeemOrderByID(ctx, orderID)
		if err != nil {
			return err
		}

		if order.Status != hackforger_model.OrderStatusPending {
			return ErrOrderNotPending{OrderID: orderID, Status: order.Status}
		}

		// Cancel the order
		order.Status = hackforger_model.OrderStatusCancelled
		if err := hackforger_model.UpdateRedeemOrder(ctx, order); err != nil {
			return err
		}

		// Refund balance
		acct, err := GetOrCreateCreditAccount(ctx, order.UserID)
		if err != nil {
			return err
		}

		acct.Balance += order.Cost
		if _, err := db.GetEngine(ctx).ID(acct.ID).Cols("balance").Update(acct); err != nil {
			return err
		}

		// Record refund transaction
		tx := &hackforger_model.CreditTransaction{
			UserID:    order.UserID,
			Type:      hackforger_model.TransactionTypeRefund,
			Amount:    order.Cost,
			Balance:   acct.Balance,
			Reference: fmt.Sprintf("order_%d_refund", orderID),
			Note:      "Order cancelled and refunded",
		}
		_, err = db.GetEngine(ctx).Insert(tx)
		if opt, _ := hackforger_model.GetRedeemOptionByID(ctx, order.OptionID); opt != nil {
			optName = opt.Name
		}
		return err
	})
	if err != nil {
		return err
	}

	notifyOrderStatusChange(ctx, order, admin.ID, hackforger_model.ActionOrderCancelled, optName)
	return nil
}

// BatchFulfillOrders fulfills multiple orders at once. Returns the count of
// successes and a list of order IDs that failed. Only site admins can call this.
func BatchFulfillOrders(ctx context.Context, admin *user_model.User, orderIDs []int64, note, deliveryType, deliveryValue string) (int, []int64, error) {
	if !admin.IsAdmin {
		return 0, nil, ErrNotAdmin{UserID: admin.ID}
	}
	var success int
	var failed []int64
	for _, oid := range orderIDs {
		if err := FulfillOrder(ctx, admin, oid, note, deliveryType, deliveryValue); err != nil {
			failed = append(failed, oid)
		} else {
			success++
		}
	}
	return success, failed, nil
}

// syncOptionStock updates the option's Stock and IsActive based on available keys.
func syncOptionStock(ctx context.Context, optionID int64) error {
	avail, err := hackforger_model.CountAvailableKeys(ctx, optionID)
	if err != nil {
		return err
	}
	opt, err := hackforger_model.GetRedeemOptionByID(ctx, optionID)
	if err != nil {
		return err
	}
	opt.Stock = int(avail)
	if avail == 0 {
		opt.IsActive = false
	}
	return hackforger_model.UpdateRedeemOption(ctx, opt)
}

// AddKeysToOption bulk-inserts keys into the pool for a given option, then
// syncs the option's stock to reflect available keys. Only site admins can call this.
func AddKeysToOption(ctx context.Context, admin *user_model.User, optionID int64, keys []string) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}
	if err := hackforger_model.AddKeys(ctx, optionID, keys); err != nil {
		return err
	}
	return syncOptionStock(ctx, optionID)
}

// GetKeyPoolStatus returns the total and available key counts for a given option.
func GetKeyPoolStatus(ctx context.Context, optionID int64) (total, available int64, err error) {
	total, err = hackforger_model.CountTotalKeys(ctx, optionID)
	if err != nil {
		return
	}
	available, err = hackforger_model.CountAvailableKeys(ctx, optionID)
	return
}

// CreateRedeemOptionAsAdmin creates a new redeem option. Only site admins can call this.
func CreateRedeemOptionAsAdmin(ctx context.Context, admin *user_model.User, opt *hackforger_model.RedeemOption) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}
	return hackforger_model.CreateRedeemOption(ctx, opt)
}

// UpdateRedeemOptionAsAdmin updates an existing redeem option. Only site admins can call this.
func UpdateRedeemOptionAsAdmin(ctx context.Context, admin *user_model.User, opt *hackforger_model.RedeemOption) error {
	if !admin.IsAdmin {
		return ErrNotAdmin{UserID: admin.ID}
	}
	return hackforger_model.UpdateRedeemOption(ctx, opt)
}
