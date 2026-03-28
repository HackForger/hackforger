// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIGrantRoundCRUD tests the full CRUD lifecycle for grant rounds.
func TestAPIGrantRoundCRUD(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 2 is org owner for org 3 (via team 1 "Owners").
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// --- Create ---
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/grant-rounds", map[string]any{
		"org_id":         3,
		"name":           "Integration CRUD Round",
		"slug":           "integration-crud-round",
		"description":    "A round created via integration test",
		"budget":         10000,
		"currency":       "USD",
		"budget_credits": 5000,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var round hackforger_model.GrantRound
	DecodeJSON(t, resp, &round)
	assert.Equal(t, "Integration CRUD Round", round.Name)
	assert.Equal(t, "integration-crud-round", round.Slug)
	assert.Equal(t, hackforger_model.GrantRoundStatusDraft, round.Status)
	assert.Equal(t, float64(10000), round.Budget)
	assert.Equal(t, int64(5000), round.BudgetCredits)
	roundID := round.ID
	require.Greater(t, roundID, int64(0))

	// --- Get by ID ---
	req = NewRequestf(t, "GET", "/api/v1/hackforger/grant-rounds/%d", roundID).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var fetched hackforger_model.GrantRound
	DecodeJSON(t, resp, &fetched)
	assert.Equal(t, roundID, fetched.ID)
	assert.Equal(t, "Integration CRUD Round", fetched.Name)

	// --- List (unauthenticated) ---
	req = NewRequest(t, "GET", "/api/v1/hackforger/grant-rounds")
	resp = MakeRequest(t, req, http.StatusOK)
	// Should include fixtures plus our new round.
	assert.NotEmpty(t, resp.Header().Get("X-Total-Count"))

	// --- List with org_id filter ---
	req = NewRequest(t, "GET", "/api/v1/hackforger/grant-rounds?org_id=3")
	resp = MakeRequest(t, req, http.StatusOK)
	var rounds []hackforger_model.GrantRound
	DecodeJSON(t, resp, &rounds)
	assert.GreaterOrEqual(t, len(rounds), 4) // 3 fixtures + 1 new

	// --- Update ---
	req = NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/hackforger/grant-rounds/%d", roundID), map[string]any{
		"name": "Updated CRUD Round",
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var updated hackforger_model.GrantRound
	DecodeJSON(t, resp, &updated)
	assert.Equal(t, "Updated CRUD Round", updated.Name)

	// --- Delete (Draft status) ---
	req = NewRequestf(t, "DELETE", "/api/v1/hackforger/grant-rounds/%d", roundID).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// --- Verify deleted ---
	req = NewRequestf(t, "GET", "/api/v1/hackforger/grant-rounds/%d", roundID).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

// TestAPIGrantRoundStatusFlow tests the complete lifecycle:
// Draft -> Open -> Review -> Finalized -> Distributed.
func TestAPIGrantRoundStatusFlow(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Owner of org 3.
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	ownerSession := loginUser(t, owner.Name)
	ownerToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeAll)

	// Regular user for project submission.
	submitter := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	subSession := loginUser(t, submitter.Name)
	subToken := getTokenForLoggedInUser(t, subSession, auth_model.AccessTokenScopeAll)

	// 1. Create round.
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/grant-rounds", map[string]any{
		"org_id":         3,
		"name":           "Lifecycle Round",
		"slug":           "lifecycle-round",
		"description":    "Full lifecycle test",
		"budget":         5000,
		"currency":       "USD",
		"budget_credits": 3000,
	}).AddTokenAuth(ownerToken)
	resp := MakeRequest(t, req, http.StatusCreated)
	var round hackforger_model.GrantRound
	DecodeJSON(t, resp, &round)
	roundID := round.ID

	// 2. Open round.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/open", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify status is Open.
	req = NewRequestf(t, "GET", "/api/v1/hackforger/grant-rounds/%d", roundID).AddTokenAuth(ownerToken)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &round)
	assert.Equal(t, hackforger_model.GrantRoundStatusOpen, round.Status)

	// 3. Submit a project.
	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/hackforger/grant-rounds/%d/projects", roundID), map[string]any{
		"title":       "Submitter's Project",
		"description": "Test project for lifecycle",
	}).AddTokenAuth(subToken)
	resp = MakeRequest(t, req, http.StatusCreated)
	var project hackforger_model.GrantProject
	DecodeJSON(t, resp, &project)
	projectID := project.ID
	assert.Equal(t, hackforger_model.GrantProjectStatusPending, project.Status)

	// 4. Close round (move to Review).
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/close", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// 5. Approve the project.
	req = NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/hackforger/grant-rounds/%d/projects/%d", roundID, projectID), map[string]any{
		"status": "approved",
	}).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// 6. Allocate award.
	req = NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/hackforger/grant-rounds/%d/projects/%d/award", roundID, projectID), map[string]any{
		"amount":  1000,
		"credits": 500,
	}).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// 7. Finalize round.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/finalize", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// 8. Distribute round (batch distribute all projects).
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/distribute", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify round is Distributed.
	req = NewRequestf(t, "GET", "/api/v1/hackforger/grant-rounds/%d", roundID).AddTokenAuth(ownerToken)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &round)
	assert.Equal(t, hackforger_model.GrantRoundStatusDistributed, round.Status)

	// Verify credits were deposited to submitter.
	req = NewRequest(t, "GET", "/api/v1/hackforger/credits/balance").AddTokenAuth(subToken)
	resp = MakeRequest(t, req, http.StatusOK)
	var balanceResp map[string]int64
	DecodeJSON(t, resp, &balanceResp)
	assert.Equal(t, int64(500), balanceResp["balance"])
}

// TestAPIGrantProjectSubmitAndApprove tests project submission and approval.
func TestAPIGrantProjectSubmitAndApprove(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Owner of org 3.
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	ownerSession := loginUser(t, owner.Name)
	ownerToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeAll)

	// User 5 submits to the Open round (fixture ID=2).
	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	user5Session := loginUser(t, user5.Name)
	user5Token := getTokenForLoggedInUser(t, user5Session, auth_model.AccessTokenScopeAll)

	// User 4 already has a project in round 2 (fixture).
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	user4Session := loginUser(t, user4.Name)
	user4Token := getTokenForLoggedInUser(t, user4Session, auth_model.AccessTokenScopeAll)

	// Submit as user 5 (should succeed).
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/grant-rounds/2/projects", map[string]any{
		"title":       "User5 Project",
		"description": "Project by user 5",
	}).AddTokenAuth(user5Token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var project hackforger_model.GrantProject
	DecodeJSON(t, resp, &project)
	assert.Equal(t, int64(2), project.RoundID)
	assert.Equal(t, int64(5), project.UserID)

	// Duplicate submit as user 4 (already has project in round 2) -> 409.
	req = NewRequestWithJSON(t, "POST", "/api/v1/hackforger/grant-rounds/2/projects", map[string]any{
		"title":       "Duplicate Project",
		"description": "Should fail",
	}).AddTokenAuth(user4Token)
	MakeRequest(t, req, http.StatusConflict)

	// List projects in round 2.
	req = NewRequest(t, "GET", "/api/v1/hackforger/grant-rounds/2/projects")
	resp = MakeRequest(t, req, http.StatusOK)
	var projects []hackforger_model.GrantProject
	DecodeJSON(t, resp, &projects)
	assert.GreaterOrEqual(t, len(projects), 2) // fixture + newly submitted

	// Owner approves user 5's project.
	req = NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/hackforger/grant-rounds/2/projects/%d", project.ID), map[string]any{
		"status": "approved",
	}).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify project is approved.
	req = NewRequestf(t, "GET", "/api/v1/hackforger/grant-rounds/2/projects/%d", project.ID)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &project)
	assert.Equal(t, hackforger_model.GrantProjectStatusApproved, project.Status)
}

// TestAPIGrantAllocateOverBudget tests that allocating beyond the budget returns 422.
func TestAPIGrantAllocateOverBudget(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Fixture round 3 (Review, budget=15000, budget_credits=8000).
	// Fixture projects 2 and 3 already have allocations: 5000+3000=8000 amount, 2000+1500=3500 credits.
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	ownerSession := loginUser(t, owner.Name)
	ownerToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeAll)

	// Try to allocate amount that exceeds remaining budget (15000-8000=7000 remaining).
	req := NewRequestWithJSON(t, "PUT", "/api/v1/hackforger/grant-rounds/3/projects/2/award", map[string]any{
		"amount":  13000, // existing 5000 subtracted, net new is 13000, used=3000+13000=16000 > 15000
		"credits": 100,
	}).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

// TestAPIGrantFinalizeWithUnallocated tests that finalization fails when projects lack allocations.
func TestAPIGrantFinalizeWithUnallocated(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	ownerSession := loginUser(t, owner.Name)
	ownerToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeAll)

	// Create a round, open it, submit a project, close it, approve it, but do NOT allocate.
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/grant-rounds", map[string]any{
		"org_id":         3,
		"name":           "Unallocated Finalize Test",
		"slug":           "unallocated-finalize-test",
		"budget":         5000,
		"currency":       "USD",
		"budget_credits": 2000,
	}).AddTokenAuth(ownerToken)
	resp := MakeRequest(t, req, http.StatusCreated)
	var round hackforger_model.GrantRound
	DecodeJSON(t, resp, &round)
	roundID := round.ID

	// Open.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/open", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Submit project as user 5.
	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	user5Session := loginUser(t, user5.Name)
	user5Token := getTokenForLoggedInUser(t, user5Session, auth_model.AccessTokenScopeAll)
	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/hackforger/grant-rounds/%d/projects", roundID), map[string]any{
		"title": "Unallocated Project",
	}).AddTokenAuth(user5Token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var project hackforger_model.GrantProject
	DecodeJSON(t, resp, &project)

	// Close.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/close", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Approve project.
	req = NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/hackforger/grant-rounds/%d/projects/%d", roundID, project.ID), map[string]any{
		"status": "approved",
	}).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Finalize should fail (unallocated approved project).
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/finalize", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

// TestAPIGrantCancelRound tests cancelling a round from Open state.
func TestAPIGrantCancelRound(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	ownerSession := loginUser(t, owner.Name)
	ownerToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeAll)

	// Cancel from Open (fixture round 2).
	req := NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/2/cancel").AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify status is Cancelled.
	req = NewRequestf(t, "GET", "/api/v1/hackforger/grant-rounds/2").AddTokenAuth(ownerToken)
	resp := MakeRequest(t, req, http.StatusOK)
	var round hackforger_model.GrantRound
	DecodeJSON(t, resp, &round)
	assert.Equal(t, hackforger_model.GrantRoundStatusCancelled, round.Status)

	// Try cancel again (already cancelled, no valid transition) -> 409.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/2/cancel").AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusConflict)
}

// TestAPIGrantExportCSV tests the CSV export endpoint.
func TestAPIGrantExportCSV(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Round 3 has projects in fixtures.
	req := NewRequest(t, "GET", "/api/v1/hackforger/grant-rounds/3/export")
	resp := MakeRequest(t, req, http.StatusOK)
	assert.Equal(t, "text/csv", resp.Header().Get("Content-Type"))
	assert.Contains(t, resp.Body.String(), "ID,Title,UserID,RepoID,Status,AwardAmount,AwardCredits")
	assert.Contains(t, resp.Body.String(), "Eve's Approved Project")
	assert.Contains(t, resp.Body.String(), "Frank's Approved Project")
}

// TestAPIGrantDistributeProject tests distributing a single project and verifying credits.
func TestAPIGrantDistributeProject(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	ownerSession := loginUser(t, owner.Name)
	ownerToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeAll)

	// We need a finalized round. Build one from scratch.
	// Create.
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/grant-rounds", map[string]any{
		"org_id":         3,
		"name":           "Distribute Single Project",
		"slug":           "distribute-single-project",
		"budget":         5000,
		"currency":       "USD",
		"budget_credits": 3000,
	}).AddTokenAuth(ownerToken)
	resp := MakeRequest(t, req, http.StatusCreated)
	var round hackforger_model.GrantRound
	DecodeJSON(t, resp, &round)
	roundID := round.ID

	// Open.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/open", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Submit project as user 5.
	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	user5Session := loginUser(t, user5.Name)
	user5Token := getTokenForLoggedInUser(t, user5Session, auth_model.AccessTokenScopeAll)

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/hackforger/grant-rounds/%d/projects", roundID), map[string]any{
		"title":       "Project for distribute test",
		"description": "Will receive credits",
	}).AddTokenAuth(user5Token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var project hackforger_model.GrantProject
	DecodeJSON(t, resp, &project)
	projectID := project.ID

	// Close.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/close", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Approve.
	req = NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/hackforger/grant-rounds/%d/projects/%d", roundID, projectID), map[string]any{
		"status": "approved",
	}).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Allocate.
	req = NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/hackforger/grant-rounds/%d/projects/%d/award", roundID, projectID), map[string]any{
		"amount":  1000,
		"credits": 750,
	}).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Finalize.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/finalize", roundID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Distribute single project.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/%d/projects/%d/distribute", roundID, projectID).AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify credits deposited to user 5.
	req = NewRequest(t, "GET", "/api/v1/hackforger/credits/balance").AddTokenAuth(user5Token)
	resp = MakeRequest(t, req, http.StatusOK)
	var balanceResp map[string]int64
	DecodeJSON(t, resp, &balanceResp)
	assert.Equal(t, int64(750), balanceResp["balance"])

	// Since this was the only approved project, round should auto-transition to Distributed.
	req = NewRequestf(t, "GET", "/api/v1/hackforger/grant-rounds/%d", roundID).AddTokenAuth(ownerToken)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &round)
	assert.Equal(t, hackforger_model.GrantRoundStatusDistributed, round.Status)
}

// TestAPIGrantRoundInvalidTransitions tests that invalid state transitions return 409.
func TestAPIGrantRoundInvalidTransitions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	ownerSession := loginUser(t, owner.Name)
	ownerToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeAll)

	// Fixture round 1 is Draft. Cannot close, finalize, or distribute directly.
	req := NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/1/close").AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusConflict)

	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/1/finalize").AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusConflict)

	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/1/distribute").AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusConflict)

	// Fixture round 2 is Open. Cannot open again or finalize.
	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/2/open").AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusConflict)

	req = NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/2/finalize").AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusConflict)
}

// TestAPIGrantDeleteNonDraft tests that deleting a non-draft round returns 409.
func TestAPIGrantDeleteNonDraft(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	ownerSession := loginUser(t, owner.Name)
	ownerToken := getTokenForLoggedInUser(t, ownerSession, auth_model.AccessTokenScopeAll)

	// Fixture round 2 is Open, cannot delete.
	req := NewRequestf(t, "DELETE", "/api/v1/hackforger/grant-rounds/2").AddTokenAuth(ownerToken)
	MakeRequest(t, req, http.StatusConflict)
}

// TestAPIGrantAccessDenied tests that a non-org-member cannot modify a round.
func TestAPIGrantAccessDenied(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 5 is NOT an owner/admin of org 3 (user 5 is in org 6 and 7, not 3).
	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	user5Session := loginUser(t, user5.Name)
	user5Token := getTokenForLoggedInUser(t, user5Session, auth_model.AccessTokenScopeAll)

	// Try to open fixture round 1 (org 3) as user 5 -> 403.
	req := NewRequestf(t, "POST", "/api/v1/hackforger/grant-rounds/1/open").AddTokenAuth(user5Token)
	MakeRequest(t, req, http.StatusForbidden)

	// Try to delete fixture round 1 as user 5 -> 403.
	req = NewRequestf(t, "DELETE", "/api/v1/hackforger/grant-rounds/1").AddTokenAuth(user5Token)
	MakeRequest(t, req, http.StatusForbidden)
}

// TestAPIGrantSubmitToNonOpenRound tests that submitting to a non-open round returns 409.
func TestAPIGrantSubmitToNonOpenRound(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	user5Session := loginUser(t, user5.Name)
	user5Token := getTokenForLoggedInUser(t, user5Session, auth_model.AccessTokenScopeAll)

	// Round 1 is Draft, cannot submit.
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/grant-rounds/1/projects", map[string]any{
		"title": "Should Fail",
	}).AddTokenAuth(user5Token)
	MakeRequest(t, req, http.StatusConflict)

	// Round 3 is Review, cannot submit.
	req = NewRequestWithJSON(t, "POST", "/api/v1/hackforger/grant-rounds/3/projects", map[string]any{
		"title": "Should Also Fail",
	}).AddTokenAuth(user5Token)
	MakeRequest(t, req, http.StatusConflict)
}
