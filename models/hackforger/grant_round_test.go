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

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestCreateGrantRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "New Round", Slug: "new-round",
		Description: "test", Status: hackforger_model.GrantRoundStatusDraft,
		Budget: 5000, Currency: "USD", BudgetCredits: 2000,
	}
	err := hackforger_model.CreateGrantRound(db.DefaultContext, round)
	require.NoError(t, err)
	assert.Greater(t, round.ID, int64(0))
	assert.Equal(t, hackforger_model.GrantRoundStatusDraft, round.Status)
}

func TestGetGrantRoundBySlug(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	round, err := hackforger_model.GetGrantRoundBySlug(db.DefaultContext, "test-grant-draft")
	require.NoError(t, err)
	assert.Equal(t, int64(1), round.ID)
	assert.Equal(t, "Test Grant Round Draft", round.Name)
	assert.Equal(t, hackforger_model.GrantRoundStatusDraft, round.Status)

	_, err = hackforger_model.GetGrantRoundBySlug(db.DefaultContext, "non-existent")
	assert.True(t, hackforger_model.IsErrGrantRoundNotExist(err))
}

func TestDeleteGrantRound_OnlyDraft(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	err := hackforger_model.DeleteGrantRound(db.DefaultContext, 1)
	require.NoError(t, err)
	_, err = hackforger_model.GetGrantRoundByID(db.DefaultContext, 1)
	assert.True(t, hackforger_model.IsErrGrantRoundNotExist(err))

	err = hackforger_model.DeleteGrantRound(db.DefaultContext, 2)
	assert.Error(t, err)
	assert.True(t, hackforger_model.IsErrGrantRoundNotDraft(err))
}

func TestGetGrantRoundBudgetUsage(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	usedAmount, usedCredits, err := hackforger_model.GetGrantRoundBudgetUsage(db.DefaultContext, 3)
	require.NoError(t, err)
	assert.Equal(t, float64(0), usedAmount)
	assert.Equal(t, int64(0), usedCredits)
}

func TestListGrantRounds_FilterByStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	status := hackforger_model.GrantRoundStatusOpen
	rounds, count, err := hackforger_model.ListGrantRounds(db.DefaultContext, hackforger_model.ListGrantRoundsOptions{
		Status: &status,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, hackforger_model.GrantRoundStatusOpen, rounds[0].Status)
}
