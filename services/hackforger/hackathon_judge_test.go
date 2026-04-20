// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"errors"
	"testing"
	"time"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/util"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrNoRubricConfigured_Error(t *testing.T) {
	err := hackforger_service.ErrNoRubricConfigured{TrackID: 99}
	assert.Equal(t, "no scoring rubric configured for track [track: 99]", err.Error())
}

func TestErrNoRubricConfigured_IsInvalidArgument(t *testing.T) {
	err := hackforger_service.ErrNoRubricConfigured{TrackID: 1}
	assert.True(t, errors.Is(err, util.ErrInvalidArgument))
}

func TestIsErrNoRubricConfigured(t *testing.T) {
	err := hackforger_service.ErrNoRubricConfigured{TrackID: 1}
	assert.True(t, hackforger_service.IsErrNoRubricConfigured(err))
	assert.False(t, hackforger_service.IsErrNoRubricConfigured(errors.New("x")))
}

func TestSubmitScores_EmptyRubric(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Seed a judging PhaseType for hackathon 8 so AllowsAction("score") returns true.
	judgingType := &hackforger_model.PhaseType{
		ActivityKind:    "hackathon",
		Key:             "judging",
		DisplayNameI18n: "hackforger.phase.hackathon.judging",
		IsUnique:        true,
		AllowedActions:  `["score"]`,
		DefaultOrder:    3,
	}
	require.NoError(t, db.Insert(db.DefaultContext, judgingType))

	// Seed a judging Phase for hackathon 8, currently active.
	now := time.Now().Unix()
	phase := &hackforger_model.Phase{
		ActivityKind: "hackathon",
		ActivityID:   8,
		PhaseTypeID:  judgingType.ID,
		StartTime:    now - 3600,
		EndTime:      now + 3600,
	}
	require.NoError(t, db.Insert(db.DefaultContext, phase))

	// Submission 100 (track 9 of hackathon 8) has no criteria; expect early reject.
	err := hackforger_service.SubmitScores(db.DefaultContext, 2, 100, []hackforger_service.CriteriaScore{})
	require.Error(t, err)
	assert.True(t, hackforger_service.IsErrNoRubricConfigured(err),
		"expected ErrNoRubricConfigured, got %T (%v)", err, err)
}
