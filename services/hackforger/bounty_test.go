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

func TestApplyForBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Bounty 1 is Open, exclusive, publisher=2. Apply as user 5.
	app, err := ApplyForBounty(db.DefaultContext, 1, 5, "I want to work on this")
	require.NoError(t, err)
	assert.Equal(t, int64(1), app.BountyID)
	assert.Equal(t, int64(5), app.UserID)
	assert.Equal(t, hackforger_model.ApplicationStatusPending, app.Status)
	assert.True(t, app.ID > 0, "application should have an ID after creation")
}

func TestApplyForBounty_NotOpen(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Transition bounty 2 to Claimed first so it's no longer Open.
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 2)
	require.NoError(t, err)
	bounty.Status = hackforger_model.BountyStatusClaimed
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, bounty))

	_, err = ApplyForBounty(db.DefaultContext, 2, 5, "too late")
	require.Error(t, err)
	assert.True(t, IsErrInvalidBountyStatus(err))
}

func TestAcceptApplication_Exclusive(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Application 1: bounty_id=1, user_id=4, pending.
	// Bounty 1: publisher=2, exclusive, Open.
	err := AcceptApplication(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	// Verify bounty is now Claimed with ClaimerID=4.
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, bounty.Status)
	assert.Equal(t, int64(4), bounty.ClaimerID)

	// Verify the application is accepted.
	app, err := hackforger_model.GetBountyApplicationByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.ApplicationStatusAccepted, app.Status)
}

func TestAcceptApplication_NotPublisher(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Application 1: bounty_id=1, publisher=2. Try as user 5.
	err := AcceptApplication(db.DefaultContext, 1, 5)
	require.Error(t, err)
	assert.True(t, IsErrNotPublisher(err))
}

func TestCancelBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Bounty 1: publisher=2, Open.
	err := CancelBounty(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusCancelled, bounty.Status)
}

func TestCancelBounty_NotPublisher(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := CancelBounty(db.DefaultContext, 1, 5)
	require.Error(t, err)
	assert.True(t, IsErrNotPublisher(err))
}

func TestCompleteBounty(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 to InReview with ClaimerID=4.
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	bounty.Status = hackforger_model.BountyStatusInReview
	bounty.ClaimerID = 4
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, bounty))

	err = CompleteBounty(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusCompleted, bounty.Status)
}

func TestRejectDelivery(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 to InReview (Exclusive mode).
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	bounty.Status = hackforger_model.BountyStatusInReview
	bounty.ClaimerID = 4
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, bounty))

	err = RejectDelivery(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, bounty.Status)
}

func TestMarkPaid(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Set bounty 1 to Completed.
	bounty, err := hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	bounty.Status = hackforger_model.BountyStatusCompleted
	require.NoError(t, hackforger_model.UpdateBounty(db.DefaultContext, bounty))

	err = MarkPaid(db.DefaultContext, 1, 2)
	require.NoError(t, err)

	bounty, err = hackforger_model.GetBountyByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.BountyStatusPaid, bounty.Status)
}
