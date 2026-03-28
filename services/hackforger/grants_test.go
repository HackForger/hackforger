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

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestGrantRoundStatusTransition_Valid(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 1 is Draft, doer 2 is org 3 owner
	err := hackforger_service.OpenRound(db.DefaultContext, 2, 1)
	require.NoError(t, err)

	round, err := hackforger_model.GetGrantRoundByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.GrantRoundStatusOpen, round.Status)
}

func TestGrantRoundStatusTransition_Invalid(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Draft -> Finalized is not allowed (must go Draft -> Open -> Review -> Finalized)
	// We use transitionRound indirectly via FinalizeRound
	err := hackforger_service.FinalizeRound(db.DefaultContext, 2, 1)
	require.Error(t, err)
	assert.True(t, hackforger_service.IsErrInvalidTransition(err))
}

func TestSubmitProject_RoundNotOpen(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 1 is Draft, so submission should fail
	_, err := hackforger_service.SubmitProject(db.DefaultContext, 4, 1, hackforger_service.SubmitProjectOpts{
		Title:       "My Project",
		Description: "Description",
		RepoID:      1,
	})
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrGrantRoundNotOpen(err))
}

func TestSubmitProject_Duplicate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User 4 already has a project in round 2 (fixture project 1)
	_, err := hackforger_service.SubmitProject(db.DefaultContext, 4, 2, hackforger_service.SubmitProjectOpts{
		Title:       "Duplicate Project",
		Description: "Should fail",
		RepoID:      1,
	})
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrGrantProjectAlreadyExists(err))
}

func TestAllocateAward_ExceedsBudget(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 3: budget=15000, budget_credits=8000
	// Project 2 (round 3): award_amount=5000, award_credits=2000
	// Project 3 (round 3): award_amount=3000, award_credits=1500
	// Used: 8000 amount, 3500 credits
	// Remaining: 7000 amount, 4500 credits
	// Allocating 8000 to project 2 should exceed (existing 5000 is subtracted: used=3000, remaining=12000... wait)
	// Actually: used_total=5000+3000=8000, subtract project 2's existing 5000 = 3000 used by others
	// So remaining = 15000 - 3000 = 12000. Allocating 8000 fits in amount.
	// For credits: used_total=2000+1500=3500, subtract project 2's existing 2000 = 1500 used by others
	// remaining = 8000 - 1500 = 6500. We need to test exceeds.
	// Let's allocate a large credits value to exceed.
	err := hackforger_service.AllocateAward(db.DefaultContext, 2, 2, 13000, 0)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrExceedsBudget(err))
}

func TestFinalizeRound_UnallocatedProjects(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create a new round in Review status with one Approved project that has no allocation
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Finalize Test", Slug: "finalize-test",
		Status: hackforger_model.GrantRoundStatusReview,
		Budget: 10000, Currency: "USD", BudgetCredits: 5000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	project := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 4, RepoID: 1,
		Title: "Unallocated", Description: "No allocation",
		Status: hackforger_model.GrantProjectStatusApproved,
	}
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, project))

	err := hackforger_service.FinalizeRound(db.DefaultContext, 2, round.ID)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrUnallocatedProjects(err))
}

func TestDistributeProject_CreditsDeposit(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create a finalized round with one approved project
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Distribute Test", Slug: "distribute-test",
		Status: hackforger_model.GrantRoundStatusFinalized,
		Budget: 10000, Currency: "USD", BudgetCredits: 5000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	project := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 4, RepoID: 1,
		Title: "Funded Project", Description: "Will be funded",
		Status: hackforger_model.GrantProjectStatusApproved,
		AwardAmount: 2000, AwardCredits: 500,
	}
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, project))

	// User 4 starts with balance 1000 (from fixture)
	acctBefore, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	balanceBefore := acctBefore.Balance

	err = hackforger_service.DistributeProject(db.DefaultContext, 2, project.ID)
	require.NoError(t, err)

	// Check balance increased
	acctAfter, err := hackforger_service.GetOrCreateCreditAccount(db.DefaultContext, 4)
	require.NoError(t, err)
	assert.Equal(t, balanceBefore+500, acctAfter.Balance)

	// Check project is now Funded
	updated, err := hackforger_model.GetGrantProjectByID(db.DefaultContext, project.ID)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.GrantProjectStatusFunded, updated.Status)
}

func TestDistributeProject_AutoDistributeRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create a finalized round with a single approved project
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Auto Distribute", Slug: "auto-distribute",
		Status: hackforger_model.GrantRoundStatusFinalized,
		Budget: 5000, Currency: "USD", BudgetCredits: 2000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	project := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 4, RepoID: 1,
		Title: "Only Project", Description: "Only one",
		Status: hackforger_model.GrantProjectStatusApproved,
		AwardAmount: 1000, AwardCredits: 200,
	}
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, project))

	err := hackforger_service.DistributeProject(db.DefaultContext, 2, project.ID)
	require.NoError(t, err)

	// Round should auto-transition to Distributed
	updatedRound, err := hackforger_model.GetGrantRoundByID(db.DefaultContext, round.ID)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.GrantRoundStatusDistributed, updatedRound.Status)
}

func TestCancelRound(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 2 is Open, cancel it
	err := hackforger_service.CancelRound(db.DefaultContext, 2, 2)
	require.NoError(t, err)

	round, err := hackforger_model.GetGrantRoundByID(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.GrantRoundStatusCancelled, round.Status)
}

func TestCancelRound_AlreadyDistributed_Rejected(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create a distributed round
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Distributed Round", Slug: "distributed-round",
		Status: hackforger_model.GrantRoundStatusDistributed,
		Budget: 5000, Currency: "USD", BudgetCredits: 2000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	err := hackforger_service.CancelRound(db.DefaultContext, 2, round.ID)
	require.Error(t, err)
	assert.True(t, hackforger_service.IsErrInvalidTransition(err))
}

func TestCancelRound_AlreadyCancelled_Rejected(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create a cancelled round
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Cancelled Round", Slug: "cancelled-round",
		Status: hackforger_model.GrantRoundStatusCancelled,
		Budget: 5000, Currency: "USD", BudgetCredits: 2000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	err := hackforger_service.CancelRound(db.DefaultContext, 2, round.ID)
	require.Error(t, err)
	assert.True(t, hackforger_service.IsErrInvalidTransition(err))
}

func TestDeleteGrantRound_NonDraft_Rejected(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Round 2 is Open, delete should fail
	err := hackforger_service.DeleteGrantRoundService(db.DefaultContext, 2, 2)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrGrantRoundNotDraft(err))
}

func TestDistributeRound_BatchConvenience(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create a finalized round with 2 approved projects
	round := &hackforger_model.GrantRound{
		OrgID: 3, OwnerID: 2, Name: "Batch Distribute", Slug: "batch-distribute",
		Status: hackforger_model.GrantRoundStatusFinalized,
		Budget: 10000, Currency: "USD", BudgetCredits: 5000,
	}
	require.NoError(t, hackforger_model.CreateGrantRound(db.DefaultContext, round))

	p1 := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 4, RepoID: 1,
		Title: "Batch P1", Description: "First",
		Status: hackforger_model.GrantProjectStatusApproved,
		AwardAmount: 2000, AwardCredits: 300,
	}
	p2 := &hackforger_model.GrantProject{
		RoundID: round.ID, UserID: 5, RepoID: 2,
		Title: "Batch P2", Description: "Second",
		Status: hackforger_model.GrantProjectStatusApproved,
		AwardAmount: 3000, AwardCredits: 400,
	}
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, p1))
	require.NoError(t, hackforger_model.CreateGrantProject(db.DefaultContext, p2))

	err := hackforger_service.DistributeRound(db.DefaultContext, 2, round.ID)
	require.NoError(t, err)

	// Both should be funded
	up1, err := hackforger_model.GetGrantProjectByID(db.DefaultContext, p1.ID)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.GrantProjectStatusFunded, up1.Status)

	up2, err := hackforger_model.GetGrantProjectByID(db.DefaultContext, p2.ID)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.GrantProjectStatusFunded, up2.Status)

	// Round should be Distributed
	updatedRound, err := hackforger_model.GetGrantRoundByID(db.DefaultContext, round.ID)
	require.NoError(t, err)
	assert.Equal(t, hackforger_model.GrantRoundStatusDistributed, updatedRound.Status)
}

func TestCreateGrantRound_Service(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	round, err := hackforger_service.CreateGrantRound(db.DefaultContext, 2, 3, hackforger_service.CreateGrantRoundOpts{
		Name:          "Service Created Round",
		Slug:          "service-created",
		Description:   "Created via service",
		Budget:        50000,
		Currency:      "USD",
		BudgetCredits: 25000,
	})
	require.NoError(t, err)
	assert.Greater(t, round.ID, int64(0))
	assert.Equal(t, "Service Created Round", round.Name)
	assert.Equal(t, hackforger_model.GrantRoundStatusDraft, round.Status)
	assert.Equal(t, int64(3), round.OrgID)
	assert.Equal(t, int64(2), round.OwnerID)
}
