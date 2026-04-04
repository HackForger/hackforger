// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCreditAccount(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Existing account for user 4
	acct, err := hackforger_model.GetCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	require.NotNil(t, acct)
	assert.Equal(t, int64(1000), acct.Balance)
	assert.Equal(t, int64(4), acct.UserID)

	// Non-existent account returns nil, no error
	acct, err = hackforger_model.GetCreditAccount(db.DefaultContext, 999)
	require.NoError(t, err)
	assert.Nil(t, acct)
}

func TestCreateRedeemOption(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	opt := &hackforger_model.RedeemOption{
		Name:        "New Option",
		Description: "A new redeem option",
		Cost:        100,
		Stock:       5,
		IsActive:    true,
	}
	err := hackforger_model.CreateRedeemOption(db.DefaultContext, opt)
	require.NoError(t, err)
	assert.Greater(t, opt.ID, int64(0))

	// Verify it can be fetched
	fetched, err := hackforger_model.GetRedeemOptionByID(db.DefaultContext, opt.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Option", fetched.Name)
	assert.Equal(t, int64(100), fetched.Cost)
	assert.Equal(t, 5, fetched.Stock)

	// Non-existent option
	_, err = hackforger_model.GetRedeemOptionByID(db.DefaultContext, 999)
	assert.True(t, hackforger_model.IsErrRedeemOptionNotExist(err))
}

func TestGetRedeemOrderByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Existing order
	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(4), order.UserID)
	assert.Equal(t, int64(1), order.OptionID)
	assert.Equal(t, int64(500), order.Cost)
	assert.Equal(t, hackforger_model.OrderStatusPending, order.Status)

	// Non-existent order
	_, err = hackforger_model.GetRedeemOrderByID(db.DefaultContext, 999)
	assert.True(t, hackforger_model.IsErrRedeemOrderNotExist(err))
}

func TestUpdateRedeemOrder(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	order, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)

	order.Status = hackforger_model.OrderStatusFulfilled
	order.FulfillNote = "Fulfilled via API"
	err = hackforger_model.UpdateRedeemOrder(db.DefaultContext, order)
	require.NoError(t, err)

	// Verify the update
	updated, err := hackforger_model.GetRedeemOrderByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.OrderStatusFulfilled, updated.Status)
	assert.Equal(t, "Fulfilled via API", updated.FulfillNote)
}
