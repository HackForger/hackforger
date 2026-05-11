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

// TestCancelHackathon_DoesNotTouchIsPublished pins the minimal-scope decision
// for issue #167: cancel must flip status_cache to Cancelled but must NOT
// modify is_published. Keeping is_published=true preserves two existing
// guards: PublishHackathon's "already published" rejection (prevents
// re-publishing a cancelled hackathon) and manage.tmpl's `{{if not
// .Hackathon.IsPublished}}` block that hides the Publish button.
//
// The actual "block new registrations on cancelled" enforcement lives in the
// Register handlers (API + web), not in the service. We verify the status
// transition the handlers key off of.
func TestCancelHackathon_DoesNotTouchIsPublished(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Fixture id=3 is "Open Hackathon": status_cache=1 (Open), is_published=true
	h, err := hackforger_model.GetHackathonByID(db.DefaultContext, 3)
	require.NoError(t, err)
	require.True(t, h.IsPublished, "precondition: hackathon must start published")
	require.Equal(t, hackforger_model.HackathonStatusOpen, h.StatusCache, "precondition: hackathon must start in Open state")

	require.NoError(t, hackforger_service.CancelHackathon(db.DefaultContext, 2, h))

	h2, err := hackforger_model.GetHackathonByID(db.DefaultContext, 3)
	require.NoError(t, err)

	assert.Equal(t, hackforger_model.HackathonStatusCancelled, h2.StatusCache,
		"cancel must set status_cache to Cancelled (5) — register handlers key off of this")
	assert.True(t, h2.IsPublished,
		"minimal scope: cancel must NOT modify is_published (keeps re-publish guard and manage-page UI consistent)")
}
