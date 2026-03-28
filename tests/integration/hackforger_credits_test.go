// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

// TestAPICreditsBalance tests getting the credit balance for a user with existing credits.
func TestAPICreditsBalance(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 4 has a credit account with balance=1000 from fixtures.
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session := loginUser(t, user4.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "GET", "/api/v1/hackforger/credits/balance").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]int64
	DecodeJSON(t, resp, &result)
	assert.Equal(t, int64(1000), result["balance"])
}

// TestAPICreditsBalanceNewUser tests that a user with no existing account gets balance=0.
func TestAPICreditsBalanceNewUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 5 has no credit_account fixture, so GetOrCreateCreditAccount creates one with 0.
	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	session := loginUser(t, user5.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "GET", "/api/v1/hackforger/credits/balance").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]int64
	DecodeJSON(t, resp, &result)
	assert.Equal(t, int64(0), result["balance"])
}

// TestAPICreditsRedeem tests redeeming credits for an option.
func TestAPICreditsRedeem(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 4 has balance=1000, option 1 costs 500.
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session := loginUser(t, user4.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/redeem", map[string]any{
		"option_id": 1,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var order hackforger_model.RedeemOrder
	DecodeJSON(t, resp, &order)
	assert.Equal(t, int64(4), order.UserID)
	assert.Equal(t, int64(1), order.OptionID)
	assert.Equal(t, int64(500), order.Cost)
	assert.Equal(t, hackforger_model.OrderStatusPending, order.Status)

	// Verify balance decreased.
	req = NewRequest(t, "GET", "/api/v1/hackforger/credits/balance").AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var result map[string]int64
	DecodeJSON(t, resp, &result)
	assert.Equal(t, int64(500), result["balance"])
}

// TestAPICreditsRedeemInsufficientBalance tests redeeming with insufficient balance.
func TestAPICreditsRedeemInsufficientBalance(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 4 has balance=1000, option 2 costs 2000.
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session := loginUser(t, user4.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/redeem", map[string]any{
		"option_id": 2,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

// TestAPICreditsAdminDeposit tests admin depositing credits to a user.
func TestAPICreditsAdminDeposit(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 1 is admin.
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	// User 4 has balance=1000 from fixtures.
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	user4Session := loginUser(t, user4.Name)
	user4Token := getTokenForLoggedInUser(t, user4Session, auth_model.AccessTokenScopeAll)

	// Deposit 500 to user 4.
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/admin/deposit", map[string]any{
		"user_id":   4,
		"amount":    500,
		"reference": "test_deposit",
		"note":      "Integration test deposit",
	}).AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify balance increased.
	req = NewRequest(t, "GET", "/api/v1/hackforger/credits/balance").AddTokenAuth(user4Token)
	resp := MakeRequest(t, req, http.StatusOK)
	var result map[string]int64
	DecodeJSON(t, resp, &result)
	assert.Equal(t, int64(1500), result["balance"])
}

// TestAPICreditsAdminDeduct tests admin deducting credits from a user.
func TestAPICreditsAdminDeduct(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	user4Session := loginUser(t, user4.Name)
	user4Token := getTokenForLoggedInUser(t, user4Session, auth_model.AccessTokenScopeAll)

	// Deduct 300 from user 4 (balance=1000).
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/admin/deduct", map[string]any{
		"user_id":   4,
		"amount":    300,
		"reference": "test_deduct",
		"note":      "Integration test deduct",
	}).AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify balance decreased.
	req = NewRequest(t, "GET", "/api/v1/hackforger/credits/balance").AddTokenAuth(user4Token)
	resp := MakeRequest(t, req, http.StatusOK)
	var result map[string]int64
	DecodeJSON(t, resp, &result)
	assert.Equal(t, int64(700), result["balance"])
}

// TestAPICreditsAdminDeductInsufficientBalance tests admin deduct with insufficient balance.
func TestAPICreditsAdminDeductInsufficientBalance(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	// User 4 has balance=1000. Try to deduct 2000.
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/admin/deduct", map[string]any{
		"user_id":   4,
		"amount":    2000,
		"reference": "test_deduct_fail",
	}).AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

// TestAPICreditsNonAdminDeposit tests that a non-admin cannot deposit.
func TestAPICreditsNonAdminDeposit(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 2 is not a site admin.
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/admin/deposit", map[string]any{
		"user_id": 4,
		"amount":  100,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)
}

// TestAPICreditsAdminFulfillOrder tests admin fulfilling a pending order.
func TestAPICreditsAdminFulfillOrder(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	// Fixture order 1 is pending.
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/redeem/orders/1/fulfill", map[string]any{
		"note": "Fulfilled via integration test",
	}).AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Try to fulfill again (already fulfilled) -> 409.
	req = NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/redeem/orders/1/fulfill", map[string]any{
		"note": "Again",
	}).AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusConflict)
}

// TestAPICreditsAdminCancelOrder tests admin cancelling a pending order and receiving a refund.
func TestAPICreditsAdminCancelOrder(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	user4Session := loginUser(t, user4.Name)
	user4Token := getTokenForLoggedInUser(t, user4Session, auth_model.AccessTokenScopeAll)

	// First redeem to create a new pending order (fixture order 1 is used by fulfill test).
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/redeem", map[string]any{
		"option_id": 1,
	}).AddTokenAuth(user4Token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var order hackforger_model.RedeemOrder
	DecodeJSON(t, resp, &order)
	orderID := order.ID

	// Check balance after redeem (1000 - 500 = 500).
	req = NewRequest(t, "GET", "/api/v1/hackforger/credits/balance").AddTokenAuth(user4Token)
	resp = MakeRequest(t, req, http.StatusOK)
	var result map[string]int64
	DecodeJSON(t, resp, &result)
	assert.Equal(t, int64(500), result["balance"])

	// Cancel the order.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/credits/redeem/orders/%d/cancel", orderID).AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify balance refunded (500 + 500 = 1000).
	req = NewRequest(t, "GET", "/api/v1/hackforger/credits/balance").AddTokenAuth(user4Token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &result)
	assert.Equal(t, int64(1000), result["balance"])
}

// TestAPICreditsListTransactions tests listing credit transactions.
func TestAPICreditsListTransactions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 4 has 1 transaction in fixtures.
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session := loginUser(t, user4.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "GET", "/api/v1/hackforger/credits/transactions").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var txns []hackforger_model.CreditTransaction
	DecodeJSON(t, resp, &txns)
	assert.GreaterOrEqual(t, len(txns), 1)
	assert.Equal(t, int64(4), txns[0].UserID)
}

// TestAPICreditsListRedeemOptions tests listing active redeem options.
func TestAPICreditsListRedeemOptions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Public endpoint, no auth needed.
	req := NewRequest(t, "GET", "/api/v1/hackforger/credits/redeem/options")
	resp := MakeRequest(t, req, http.StatusOK)

	var options []hackforger_model.RedeemOption
	DecodeJSON(t, resp, &options)
	assert.GreaterOrEqual(t, len(options), 2) // 2 active options in fixtures
}

// TestAPICreditsListRedeemOrders tests listing the current user's redeem orders.
func TestAPICreditsListRedeemOrders(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 4 has 1 order in fixtures.
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session := loginUser(t, user4.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "GET", "/api/v1/hackforger/credits/redeem/orders").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var orders []hackforger_model.RedeemOrder
	DecodeJSON(t, resp, &orders)
	assert.GreaterOrEqual(t, len(orders), 1)
	assert.Equal(t, int64(4), orders[0].UserID)
}

// TestAPICreditsCreateRedeemOption tests admin creating a redeem option.
func TestAPICreditsCreateRedeemOption(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/redeem/options", map[string]any{
		"name":        "Test Swag",
		"description": "A test swag item",
		"cost":        100,
		"stock":       5,
		"is_active":   true,
	}).AddTokenAuth(adminToken)
	resp := MakeRequest(t, req, http.StatusCreated)

	var opt hackforger_model.RedeemOption
	DecodeJSON(t, resp, &opt)
	assert.Equal(t, "Test Swag", opt.Name)
	assert.Equal(t, int64(100), opt.Cost)
	assert.Equal(t, 5, opt.Stock)
	assert.True(t, opt.IsActive)
}

// TestAPICreditsUpdateRedeemOption tests admin updating a redeem option.
func TestAPICreditsUpdateRedeemOption(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	// Update fixture option 1.
	req := NewRequestWithJSON(t, "PUT", "/api/v1/hackforger/credits/redeem/options/1", map[string]any{
		"name": "Updated Cloud Credits",
		"cost": 600,
	}).AddTokenAuth(adminToken)
	resp := MakeRequest(t, req, http.StatusOK)

	var opt hackforger_model.RedeemOption
	DecodeJSON(t, resp, &opt)
	assert.Equal(t, "Updated Cloud Credits", opt.Name)
	assert.Equal(t, int64(600), opt.Cost)
}

// TestAPICreditsUnauthenticated tests that balance and transaction endpoints require auth.
func TestAPICreditsUnauthenticated(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/credits/balance")
	MakeRequest(t, req, http.StatusUnauthorized)

	req = NewRequest(t, "GET", "/api/v1/hackforger/credits/transactions")
	MakeRequest(t, req, http.StatusUnauthorized)

	req = NewRequestWithJSON(t, "POST", "/api/v1/hackforger/credits/redeem", map[string]any{
		"option_id": 1,
	})
	MakeRequest(t, req, http.StatusUnauthorized)
}
