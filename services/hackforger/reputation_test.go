// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveTier(t *testing.T) {
	tiers := []ReputationTier{
		{Name: "Bronze", Min: 0},
		{Name: "Silver", Min: 50},
		{Name: "Gold", Min: 200},
		{Name: "Diamond", Min: 500},
	}
	assert.Equal(t, "Bronze", deriveTier(0, tiers))
	assert.Equal(t, "Bronze", deriveTier(49, tiers))
	assert.Equal(t, "Silver", deriveTier(50, tiers))
	assert.Equal(t, "Gold", deriveTier(200, tiers))
	assert.Equal(t, "Diamond", deriveTier(500, tiers))
	assert.Equal(t, "Diamond", deriveTier(9999, tiers))
}

func TestRecalculateReputation(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := RecalculateReputation(db.DefaultContext, 1)
	require.NoError(t, err)

	rep, err := hackforger_model.GetOrCreateReputation(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.True(t, rep.Score >= 0)
	assert.NotEmpty(t, rep.Tier)
}
