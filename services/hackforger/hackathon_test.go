// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"errors"
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/util"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrNoCriteria_Error_Global(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 42}
	assert.Equal(t, "hackathon has no scoring criteria [id: 42]", err.Error())
}

func TestErrNoCriteria_Error_TrackScoped(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 42, TrackID: 7}
	assert.Equal(t, "track has no enabled scoring criteria [hackathon: 42, track: 7]", err.Error())
}

func TestErrNoCriteria_IsInvalidArgument(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 42}
	assert.True(t, errors.Is(err, util.ErrInvalidArgument),
		"ErrNoCriteria should unwrap to util.ErrInvalidArgument so middleware maps to 400")
}

func TestIsErrNoCriteria(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 1}
	assert.True(t, hackforger_service.IsErrNoCriteria(err))
	assert.False(t, hackforger_service.IsErrNoCriteria(errors.New("unrelated")))
}

func TestPublishHackathon_NoCriteria(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Insert phase types programmatically (they are seeded by migration, not fixtures).
	regType := &hackforger_model.PhaseType{
		ActivityKind:    "hackathon",
		Key:             "registration",
		DisplayNameI18n: "hackforger.phase.registration",
		IsUnique:        true,
		AllowedActions:  "[]",
		DefaultOrder:    1,
	}
	devType := &hackforger_model.PhaseType{
		ActivityKind:    "hackathon",
		Key:             "development",
		DisplayNameI18n: "hackforger.phase.development",
		IsUnique:        true,
		AllowedActions:  "[]",
		DefaultOrder:    2,
	}
	require.NoError(t, db.Insert(db.DefaultContext, regType))
	require.NoError(t, db.Insert(db.DefaultContext, devType))

	// Seed two phases for hackathon 7 so publish gets past the phase checks.
	require.NoError(t, db.Insert(db.DefaultContext, &hackforger_model.Phase{
		ActivityKind: "hackathon", ActivityID: 7,
		PhaseTypeID: regType.ID,
		StartTime:   1672578000, EndTime: 1672664400,
	}))
	require.NoError(t, db.Insert(db.DefaultContext, &hackforger_model.Phase{
		ActivityKind: "hackathon", ActivityID: 7,
		PhaseTypeID: devType.ID,
		StartTime:   1672664400, EndTime: 1672750800,
	}))

	h, err := hackforger_model.GetHackathonByID(db.DefaultContext, 7)
	require.NoError(t, err)

	err = hackforger_service.PublishHackathon(db.DefaultContext, 2, h)
	require.Error(t, err)
	assert.True(t, hackforger_service.IsErrNoCriteria(err),
		"expected ErrNoCriteria, got: %T (%v)", err, err)

	typed, ok := err.(hackforger_service.ErrNoCriteria)
	require.True(t, ok)
	assert.Equal(t, int64(7), typed.HackathonID)
	assert.Equal(t, int64(0), typed.TrackID, "global check, not track-scoped")
}
