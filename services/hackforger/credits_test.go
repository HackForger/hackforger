// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminDeposit(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	require.True(t, admin.IsAdmin)

	// User 4 starts with balance 1000
	acctBefore, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), acctBefore.Balance)

	err = hackforger_service.AdminDeposit(db.DefaultContext, admin, 4, 500, "admin_grant", "Admin deposit test")
	require.NoError(t, err)

	acctAfter, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), acctAfter.Balance)
}

func TestAdminDeposit_NotAdmin(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	nonAdmin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	require.False(t, nonAdmin.IsAdmin)

	err := hackforger_service.AdminDeposit(db.DefaultContext, nonAdmin, 4, 500, "ref", "note")
	require.Error(t, err)
	assert.True(t, hackforger_service.IsErrNotAdmin(err))
}

func TestAdminDeduct(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// User 4 starts with balance 1000, deduct 200
	err := hackforger_service.AdminDeduct(db.DefaultContext, admin, 4, 200, "admin_deduct", "Deduction test")
	require.NoError(t, err)

	acctAfter, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(800), acctAfter.Balance)
}

func TestAdminDeduct_InsufficientBalance(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// User 4 has 1000, deducting 2000 should fail
	err := hackforger_service.AdminDeduct(db.DefaultContext, admin, 4, 2000, "ref", "note")
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrInsufficientCredits(err))
}

func TestFulfillOrder(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// Order 1 is Pending
	err := hackforger_service.FulfillOrder(db.DefaultContext, admin, 1, "Fulfilled by admin", "", "")
	require.NoError(t, err)

	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order.Status)
	assert.Equal(t, "Fulfilled by admin", order.FulfillNote)
}

func TestFulfillOrder_WithDelivery(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// Order 3 is pending (user 5, cost 500)
	err := hackforger_service.FulfillOrder(db.DefaultContext, admin, 3, "License delivered", "license_key", "ABCD-1234")
	require.NoError(t, err)

	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 3)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order.Status)
	assert.Equal(t, "License delivered", order.FulfillNote)
	assert.Equal(t, "license_key", order.DeliveryType)
	assert.Equal(t, "ABCD-1234", order.DeliveryValue)
}

func TestCancelOrder_RefundsBalance(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// User 4 balance=1000, order 1 cost=500
	acctBefore, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), acctBefore.Balance)

	err = hackforger_service.CancelOrder(db.DefaultContext, admin, 1)
	require.NoError(t, err)

	// Balance should be 1000 + 500 = 1500
	acctAfter, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(1500), acctAfter.Balance)

	// Order should be cancelled
	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusCancelled, order.Status)
}

func TestRedeem_AutoFulfill(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Option 2 is auto-fulfill with cost=2000. User 4 has balance=1000.
	// Deposit 1500 extra so user 4 has 2500 (enough for the 2000 cost).
	err := hackforger_service.Deposit(db.DefaultContext, 4, 1500, "test_topup", "Top up for auto-fulfill test")
	require.NoError(t, err)

	acct, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(2500), acct.Balance)

	// Add keys to option 2 (auto-fulfill pool)
	err = hackforger_model.AddKeys(db.DefaultContext, 2, []string{"AUTO-KEY-001", "AUTO-KEY-002"})
	require.NoError(t, err)

	// Redeem option 2 — should auto-fulfill
	order, err := hackforger_service.Redeem(db.DefaultContext, 4, 2)
	require.NoError(t, err)
	require.NotNil(t, order)

	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order.Status)
	assert.Equal(t, "license_key", order.DeliveryType)
	assert.Equal(t, "AUTO-KEY-001", order.DeliveryValue)

	// Verify balance was deducted
	acctAfter, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(500), acctAfter.Balance)

	// Verify key was claimed
	avail, err := hackforger_model.CountAvailableKeys(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), avail)
}

func TestRedeem_AutoFulfill_NoKeys(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Option 2 is auto-fulfill with cost=2000. User 4 has balance=1000.
	// Deposit so user has enough.
	err := hackforger_service.Deposit(db.DefaultContext, 4, 1500, "test_topup", "Top up")
	require.NoError(t, err)

	// Do NOT add any keys to option 2 — pool is empty

	// Redeem should fail with ErrOutOfStock because no keys available
	order, err := hackforger_service.Redeem(db.DefaultContext, 4, 2)
	require.Error(t, err)
	assert.Nil(t, order)
	assert.True(t, hackforger_model.IsErrOutOfStock(err))

	// Verify balance is unchanged (tx rolled back)
	acctAfter, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(2500), acctAfter.Balance)
}

func TestBatchFulfillOrders(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// Orders 1 and 3 are both pending
	success, failed, err := hackforger_service.BatchFulfillOrders(db.DefaultContext, admin, []int64{1, 3}, "batch note", "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, success)
	assert.Empty(t, failed)

	// Verify both orders are fulfilled
	order1, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order1.Status)
	assert.Equal(t, "batch note", order1.FulfillNote)

	order3, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 3)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order3.Status)
	assert.Equal(t, "batch note", order3.FulfillNote)
}

func TestBatchFulfillOrders_PartialFailure(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	// Fulfill order 1 first so it's no longer pending
	err := hackforger_service.FulfillOrder(db.DefaultContext, admin, 1, "pre-fulfilled", "", "")
	require.NoError(t, err)

	// Now batch fulfill orders 1 (already fulfilled) and 3 (still pending)
	success, failed, err := hackforger_service.BatchFulfillOrders(db.DefaultContext, admin, []int64{1, 3}, "batch note", "", "")
	require.NoError(t, err)
	assert.Equal(t, 1, success)
	assert.Equal(t, []int64{1}, failed)

	// Order 3 should be fulfilled
	order3, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 3)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, order3.Status)
}
