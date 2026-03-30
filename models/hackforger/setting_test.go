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

func TestSetting_GetSetDefault(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Key does not exist yet — default must be returned.
	val := hackforger_model.GetSettingWithDefault(db.DefaultContext, "test.key", "fallback")
	assert.Equal(t, "fallback", val)

	// Insert a new key.
	require.NoError(t, hackforger_model.SetSetting(db.DefaultContext, "test.key", "hello"))
	val, err := hackforger_model.GetSetting(db.DefaultContext, "test.key")
	require.NoError(t, err)
	assert.Equal(t, "hello", val)

	// Update the existing key.
	require.NoError(t, hackforger_model.SetSetting(db.DefaultContext, "test.key", "world"))
	val, err = hackforger_model.GetSetting(db.DefaultContext, "test.key")
	require.NoError(t, err)
	assert.Equal(t, "world", val)
}
