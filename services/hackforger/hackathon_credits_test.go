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

// findTxByRef is a test helper that finds a credit transaction by reference string.
func findTxByRef(t *testing.T, ref string) *hackforger_model.CreditTransaction {
	t.Helper()
	tx := &hackforger_model.CreditTransaction{}
	has, err := db.GetEngine(db.DefaultContext).Where("`reference` = ?", ref).Get(tx)
	require.NoError(t, err)
	if !has {
		return nil
	}
	return tx
}

func TestDistributeHackathonCredits_WinnerTakesAll(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Track 1: winner_takes_all, 1000 credits
	// Submissions: user 2 (score 95.0, rank 1), user 4 (score 80.0, rank 2)
	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// User 2 (rank 1) gets all 1000 credits
	tx := findTxByRef(t, "hackathon:1/track:1:rank:1")
	require.NotNil(t, tx, "expected transaction for rank 1")
	assert.Equal(t, int64(2), tx.UserID)
	assert.Equal(t, int64(1000), tx.Amount)

	// User 4 (rank 2) gets nothing from track 1
	tx2 := findTxByRef(t, "hackathon:1/track:1:rank:2")
	assert.Nil(t, tx2, "rank 2 should not receive credits in winner_takes_all")
}

func TestDistributeHackathonCredits_Tiered(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Track 2: tiered 60/30/10, 1000 credits
	// Submissions: user 2 (score 90.0), user 4 (score 85.0), user 5 (score 70.0)
	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// Rank 1 (user 2): 1000 * 60 / 100 = 600
	tx1 := findTxByRef(t, "hackathon:1/track:2:rank:1")
	require.NotNil(t, tx1, "expected transaction for rank 1")
	assert.Equal(t, int64(2), tx1.UserID)
	assert.Equal(t, int64(600), tx1.Amount)

	// Rank 2 (user 4): 1000 * 30 / 100 = 300
	tx2 := findTxByRef(t, "hackathon:1/track:2:rank:2")
	require.NotNil(t, tx2, "expected transaction for rank 2")
	assert.Equal(t, int64(4), tx2.UserID)
	assert.Equal(t, int64(300), tx2.Amount)

	// Rank 3 (user 5): 1000 * 10 / 100 = 100
	tx3 := findTxByRef(t, "hackathon:1/track:2:rank:3")
	require.NotNil(t, tx3, "expected transaction for rank 3")
	assert.Equal(t, int64(5), tx3.UserID)
	assert.Equal(t, int64(100), tx3.Amount)

	// No remainder (600 + 300 + 100 = 1000)
	txR := findTxByRef(t, "hackathon:1/track:2:rank:1:remainder")
	assert.Nil(t, txR, "no remainder expected when ratios divide evenly")
}

func TestDistributeHackathonCredits_Equal(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Track 3: equal, 1000 credits
	// Submissions: user 2 (score 88.0), user 4 (score 88.0)
	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// 1000 / 2 = 500 each, no remainder
	tx1 := findTxByRef(t, "hackathon:1/track:3:rank:1")
	require.NotNil(t, tx1, "expected transaction for rank 1")
	assert.Equal(t, int64(2), tx1.UserID)
	assert.Equal(t, int64(500), tx1.Amount)

	tx2 := findTxByRef(t, "hackathon:1/track:3:rank:2")
	require.NotNil(t, tx2, "expected transaction for rank 2")
	assert.Equal(t, int64(4), tx2.UserID)
	assert.Equal(t, int64(500), tx2.Amount)

	txR := findTxByRef(t, "hackathon:1/track:3:rank:1:remainder")
	assert.Nil(t, txR, "no remainder expected when credits divide evenly")
}

func TestDistributeHackathonCredits_EqualRemainder(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Modify track 3 to have 999 credits (odd number for 2 users)
	track := unittest.AssertExistsAndLoadBean(t, &hackforger_model.HackathonTrack{ID: 3})
	track.PrizeCredits = 999
	_, err := db.GetEngine(db.DefaultContext).ID(track.ID).Cols("prize_credits").Update(track)
	require.NoError(t, err)

	err = distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// 999 / 2 = 499 each, remainder 1 goes to rank 1
	tx1 := findTxByRef(t, "hackathon:1/track:3:rank:1")
	require.NotNil(t, tx1, "expected transaction for rank 1")
	assert.Equal(t, int64(2), tx1.UserID)
	assert.Equal(t, int64(499), tx1.Amount)

	tx2 := findTxByRef(t, "hackathon:1/track:3:rank:2")
	require.NotNil(t, tx2, "expected transaction for rank 2")
	assert.Equal(t, int64(4), tx2.UserID)
	assert.Equal(t, int64(499), tx2.Amount)

	txR := findTxByRef(t, "hackathon:1/track:3:rank:1:remainder")
	require.NotNil(t, txR, "expected remainder transaction for rank 1")
	assert.Equal(t, int64(2), txR.UserID)
	assert.Equal(t, int64(1), txR.Amount)
}

func TestDistributeHackathonCredits_ZeroPrize(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Track 4: 0 credits, should be skipped silently
	err := distributeHackathonCredits(db.DefaultContext, 1)
	require.NoError(t, err)

	// No transactions should exist for track 4
	tx := findTxByRef(t, "hackathon:1/track:4:rank:1")
	assert.Nil(t, tx, "zero-prize track should produce no transactions")
}

func TestDistributeHackathonCredits_RejectsUnknownMode(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	tainted := &hackforger_model.HackathonTrack{
		HackathonID:     1,
		Name:            "Tainted Track",
		PrizeCredits:    100,
		PrizeDistMode:   "",
		PrizeDistRatios: "",
	}
	_, err := db.GetEngine(db.DefaultContext).Insert(tainted)
	require.NoError(t, err)
	require.NotZero(t, tainted.ID)

	sub := &hackforger_model.HackathonSubmission{
		HackathonID: 1,
		TrackID:     tainted.ID,
		UserID:      2,
		TotalScore:  90.0,
	}
	_, err = db.GetEngine(db.DefaultContext).Insert(sub)
	require.NoError(t, err)

	err = distributeHackathonCredits(db.DefaultContext, 1)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrInvalidPrizeDistMode(err),
		"expected ErrInvalidPrizeDistMode, got %T: %v", err, err)
}
