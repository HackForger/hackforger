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

func TestAddKeys(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	keys := []string{"KEY-AAA", "KEY-BBB", "KEY-CCC"}
	err := hackforger_model.AddKeys(db.DefaultContext, 1, keys)
	require.NoError(t, err)

	total, err := hackforger_model.CountTotalKeys(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	available, err := hackforger_model.CountAvailableKeys(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(3), available)

	// Verify listing returns all keys in order
	listed, err := hackforger_model.ListKeysByOption(db.DefaultContext, 1)
	require.NoError(t, err)
	require.Len(t, listed, 3)
	assert.Equal(t, "KEY-AAA", listed[0].KeyValue)
	assert.Equal(t, "KEY-BBB", listed[1].KeyValue)
	assert.Equal(t, "KEY-CCC", listed[2].KeyValue)
}

func TestClaimKey(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Add keys first
	keys := []string{"CLAIM-1", "CLAIM-2"}
	err := hackforger_model.AddKeys(db.DefaultContext, 1, keys)
	require.NoError(t, err)

	// Claim first key
	key1, err := hackforger_model.ClaimKey(db.DefaultContext, 1, 100)
	require.NoError(t, err)
	assert.Equal(t, "CLAIM-1", key1.KeyValue)
	assert.True(t, key1.IsUsed)
	assert.Equal(t, int64(100), key1.OrderID)

	// Claim second key
	key2, err := hackforger_model.ClaimKey(db.DefaultContext, 1, 101)
	require.NoError(t, err)
	assert.Equal(t, "CLAIM-2", key2.KeyValue)
	assert.True(t, key2.IsUsed)
	assert.Equal(t, int64(101), key2.OrderID)

	// Verify counts
	available, err := hackforger_model.CountAvailableKeys(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(0), available)

	total, err := hackforger_model.CountTotalKeys(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
}

func TestClaimKey_Empty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// No keys added — pool is empty
	_, err := hackforger_model.ClaimKey(db.DefaultContext, 999, 1)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrOutOfStock(err))
}
