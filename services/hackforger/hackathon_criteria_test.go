// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixture status_cache: hackathon 1=Judging, 2=Draft, 5=Finalized.
// Criterion 1 belongs to hackathon 1.

func TestAddCriteria_AllowedInJudging(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := hackforger_service.AddCriteria(db.DefaultContext, 1,
		"Late Added Criterion", "Added during Judging", 10.0, 50.0, 0)
	require.NoError(t, err, "Judging-phase add must succeed after refactor")
}

func TestAddCriteria_AllowedInDraft(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := hackforger_service.AddCriteria(db.DefaultContext, 2,
		"Draft Criterion", "", 10.0, 25.0, 0)
	require.NoError(t, err)
}

func TestAddCriteria_BlockedInFinalized(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := hackforger_service.AddCriteria(db.DefaultContext, 5, "x", "", 10.0, 25.0, 0)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrInvalidHackathonPhase(err))
}

func TestUpdateCriteria_BlockedInJudging(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	c, err := hackforger_model.GetCriteriaByID(db.DefaultContext, 1)
	require.NoError(t, err)
	c.Name = "Renamed During Judging"

	err = hackforger_service.UpdateCriteria(db.DefaultContext, c)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrInvalidHackathonPhase(err),
		"Update during Judging must still be rejected")
}

func TestDeleteCriteria_BlockedInJudging(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := hackforger_service.RemoveCriteria(db.DefaultContext, 1)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrInvalidHackathonPhase(err))
}
