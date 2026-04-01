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

// ---------------------------------------------------------------------------
// Repo-level CRUD
// ---------------------------------------------------------------------------

// TestHackForgerBountyCreate tests POST /repos/{owner}/{repo}/bounties.
func TestHackForgerBountyCreate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 owns repo1; issue index 4 (DB id 5) has no bounty yet.
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Happy path: create bounty on issue index 4.
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/bounties", map[string]any{
		"issue_id": 4,
		"title":    "New Integration Bounty",
		"mode":     0,
		"deadline": 1893456000,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var bounty hackforger_model.Bounty
	DecodeJSON(t, resp, &bounty)
	assert.Equal(t, "New Integration Bounty", bounty.Title)
	assert.Equal(t, hackforger_model.BountyStatusOpen, bounty.Status)
	assert.Equal(t, hackforger_model.BountyModeExclusive, bounty.Mode)
	assert.Equal(t, int64(2), bounty.PublisherID)
	require.Greater(t, bounty.ID, int64(0))

	// Duplicate: creating another bounty on the same issue should 409.
	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/bounties", map[string]any{
		"issue_id": 4,
		"title":    "Duplicate",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusConflict)

	// Non-existent issue index should 404.
	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/bounties", map[string]any{
		"issue_id": 9999,
		"title":    "Ghost",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

// TestHackForgerBountyGet tests GET /repos/{owner}/{repo}/bounties/{bounty_id}.
func TestHackForgerBountyGet(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Bounty 1 exists in repo1.
	req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties/1").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var bounty hackforger_model.Bounty
	DecodeJSON(t, resp, &bounty)
	assert.Equal(t, int64(1), bounty.ID)
	assert.Equal(t, "Fix authentication bug", bounty.Title)
	assert.Equal(t, hackforger_model.BountyStatusOpen, bounty.Status)

	// Non-existent bounty should 404.
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties/9999").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

// TestHackForgerBountyList tests GET /repos/{owner}/{repo}/bounties.
func TestHackForgerBountyList(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// repo1 has bounties 1 and 2.
	req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var bounties []hackforger_model.Bounty
	DecodeJSON(t, resp, &bounties)
	assert.Len(t, bounties, 2)

	// Filter by status=0 (Open) -- should still return 2.
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties?status=0").AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &bounties)
	assert.Len(t, bounties, 2)

	// Filter by status=1 (Claimed) -- should return 0 in repo1.
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties?status=1").AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &bounties)
	assert.Len(t, bounties, 0)
}

// TestHackForgerBountyUpdate tests PUT /repos/{owner}/{repo}/bounties/{bounty_id}.
func TestHackForgerBountyUpdate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 is the publisher of bounty 1.
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "PUT", "/api/v1/repos/user2/repo1/bounties/1", map[string]any{
		"title":    "Updated Title",
		"deadline": 1893456000,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var bounty hackforger_model.Bounty
	DecodeJSON(t, resp, &bounty)
	assert.Equal(t, "Updated Title", bounty.Title)

	// Non-publisher (user4) should be forbidden.
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session4 := loginUser(t, user4.Name)
	token4 := getTokenForLoggedInUser(t, session4, auth_model.AccessTokenScopeAll)

	req = NewRequestWithJSON(t, "PUT", "/api/v1/repos/user2/repo1/bounties/1", map[string]any{
		"title": "Unauthorized Update",
	}).AddTokenAuth(token4)
	MakeRequest(t, req, http.StatusForbidden)
}

// TestHackForgerBountyDelete tests DELETE /repos/{owner}/{repo}/bounties/{bounty_id}.
func TestHackForgerBountyDelete(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// First create a fresh bounty to delete (bounty 1 has applications).
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/bounties", map[string]any{
		"issue_id": 4,
		"title":    "To Be Deleted",
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var created hackforger_model.Bounty
	DecodeJSON(t, resp, &created)

	// Delete it.
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/user2/repo1/bounties/%d", created.ID)).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify it's gone.
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/user2/repo1/bounties/%d", created.ID)).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

// ---------------------------------------------------------------------------
// Rewards
// ---------------------------------------------------------------------------

// TestHackForgerBountyRewardCRUD tests POST/GET/DELETE for bounty rewards.
func TestHackForgerBountyRewardCRUD(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// List existing rewards for bounty 1 (fixture has 2 rewards: ids 1,2).
	req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties/1/rewards").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var rewards []hackforger_model.BountyReward
	DecodeJSON(t, resp, &rewards)
	assert.Len(t, rewards, 2)

	// Add a new reward.
	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/bounties/1/rewards", map[string]any{
		"type":     "credits",
		"credits":  200,
		"rank":     2,
		"note":     "runner-up prize",
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)

	var reward hackforger_model.BountyReward
	DecodeJSON(t, resp, &reward)
	assert.Equal(t, int64(1), reward.BountyID)
	assert.Equal(t, hackforger_model.RewardTypeCredits, reward.Type)
	assert.Equal(t, int64(200), reward.Credits)
	require.Greater(t, reward.ID, int64(0))

	// Delete the new reward.
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/user2/repo1/bounties/1/rewards/%d", reward.ID)).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify count is back to 2.
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties/1/rewards").AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &rewards)
	assert.Len(t, rewards, 2)
}

// ---------------------------------------------------------------------------
// Applications
// ---------------------------------------------------------------------------

// TestHackForgerBountyApply tests POST /repos/{owner}/{repo}/bounties/{bounty_id}/applications.
func TestHackForgerBountyApply(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user5 (hacker2) applies to bounty 2 (Open, Competitive, repo1).
	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	session := loginUser(t, user5.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/bounties/2/applications", map[string]any{
		"message": "I want to participate",
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var app hackforger_model.BountyApplication
	DecodeJSON(t, resp, &app)
	assert.Equal(t, int64(2), app.BountyID)
	assert.Equal(t, int64(5), app.UserID)
	assert.Equal(t, hackforger_model.ApplicationStatusPending, app.Status)

	// Duplicate application should 409.
	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/bounties/2/applications", map[string]any{
		"message": "Again",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusConflict)
}

// TestHackForgerBountyListApplications tests GET /repos/{owner}/{repo}/bounties/{bounty_id}/applications.
func TestHackForgerBountyListApplications(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Publisher (user2) can list applications for bounty 1.
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Bounty 1 has 1 application (id=1, user_id=4).
	req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties/1/applications").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var apps []hackforger_model.BountyApplication
	DecodeJSON(t, resp, &apps)
	assert.Len(t, apps, 1)
	assert.Equal(t, int64(4), apps[0].UserID)
}

// TestHackForgerBountyReviewApplication tests PUT /repos/{owner}/{repo}/bounties/{bounty_id}/applications/{application_id}.
func TestHackForgerBountyReviewApplication(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Publisher user2 reviews the pending application on bounty 1.
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Application 1 is pending on bounty 1.
	// Accept it.
	req := NewRequestWithJSON(t, "PUT", "/api/v1/repos/user2/repo1/bounties/1/applications/1", map[string]any{
		"action": "accept",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// Verify bounty 1 is now Claimed (exclusive mode: accepting moves to Claimed).
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties/1").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var bounty hackforger_model.Bounty
	DecodeJSON(t, resp, &bounty)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, bounty.Status)
	assert.Equal(t, int64(4), bounty.ClaimerID)

	// Invalid action should 422.
	req = NewRequestWithJSON(t, "PUT", "/api/v1/repos/user2/repo1/bounties/1/applications/1", map[string]any{
		"action": "invalid",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

// ---------------------------------------------------------------------------
// State transitions
// ---------------------------------------------------------------------------

// TestHackForgerBountyComplete tests POST /repos/{owner}/{repo}/bounties/{bounty_id}/complete.
// Bounty 3 (Claimed, repo2) should transition to Completed.
func TestHackForgerBountyComplete(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 is publisher of bounty 3 in repo2.
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "POST", "/api/v1/repos/user2/repo2/bounties/3/complete").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// Verify status is now Completed.
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo2/bounties/3").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var bounty hackforger_model.Bounty
	DecodeJSON(t, resp, &bounty)
	assert.Equal(t, hackforger_model.BountyStatusCompleted, bounty.Status)

	// Cannot complete an Open bounty (bounty 1 in repo1).
	req = NewRequest(t, "POST", "/api/v1/repos/user2/repo1/bounties/1/complete").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

// TestHackForgerBountyReject tests POST /repos/{owner}/{repo}/bounties/{bounty_id}/reject.
// Bounty 4 (InReview, repo2) should transition back to Claimed.
func TestHackForgerBountyReject(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "POST", "/api/v1/repos/user2/repo2/bounties/4/reject").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// Verify status is now Claimed (delivery rejected, back to work).
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo2/bounties/4").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var bounty hackforger_model.Bounty
	DecodeJSON(t, resp, &bounty)
	assert.Equal(t, hackforger_model.BountyStatusClaimed, bounty.Status)
}

// TestHackForgerBountyCancel tests POST /repos/{owner}/{repo}/bounties/{bounty_id}/cancel.
// Bounty 1 (Open, repo1) can be cancelled.
func TestHackForgerBountyCancel(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "POST", "/api/v1/repos/user2/repo1/bounties/1/cancel").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// Verify status is now Cancelled.
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties/1").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var bounty hackforger_model.Bounty
	DecodeJSON(t, resp, &bounty)
	assert.Equal(t, hackforger_model.BountyStatusCancelled, bounty.Status)

	// Non-publisher should be forbidden.
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session4 := loginUser(t, user4.Name)
	token4 := getTokenForLoggedInUser(t, session4, auth_model.AccessTokenScopeAll)

	req = NewRequest(t, "POST", "/api/v1/repos/user2/repo1/bounties/2/cancel").AddTokenAuth(token4)
	MakeRequest(t, req, http.StatusForbidden)
}

// TestHackForgerBountyPay tests POST /repos/{owner}/{repo}/bounties/{bounty_id}/pay.
// Bounty 5 (Completed, repo3/org3) should transition to Paid.
func TestHackForgerBountyPay(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 is publisher of bounty 5 in repo3 (owned by org3).
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "POST", "/api/v1/repos/org3/repo3/bounties/5/pay").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// Verify status is now Paid.
	req = NewRequest(t, "GET", "/api/v1/repos/org3/repo3/bounties/5").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var bounty hackforger_model.Bounty
	DecodeJSON(t, resp, &bounty)
	assert.Equal(t, hackforger_model.BountyStatusPaid, bounty.Status)

	// Cannot pay an Open bounty.
	req = NewRequest(t, "POST", "/api/v1/repos/user2/repo1/bounties/1/pay").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

// ---------------------------------------------------------------------------
// Competitive bounty: select winners
// ---------------------------------------------------------------------------

// TestHackForgerBountySelectWinners tests POST /repos/{owner}/{repo}/bounties/{bounty_id}/winners.
// Bounty 2 is Competitive and Open in repo1. We need to transition it to InReview first.
func TestHackForgerBountySelectWinners(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Bounty 2 is Open+Competitive. Transition to InReview via /review endpoint.
	req := NewRequest(t, "POST", "/api/v1/repos/user2/repo1/bounties/2/review").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// Now select winners.
	req = NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/bounties/2/winners", map[string]any{
		"winners": []map[string]any{
			{"user_id": 4, "rank": 1},
			{"user_id": 5, "rank": 2},
		},
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// List winners.
	req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/bounties/2/winners").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var winners []hackforger_model.BountyWinner
	DecodeJSON(t, resp, &winners)
	assert.Len(t, winners, 2)
}

// ---------------------------------------------------------------------------
// Global endpoints
// ---------------------------------------------------------------------------

// TestHackForgerBountyGlobalList tests GET /hackforger/bounties.
func TestHackForgerBountyGlobalList(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// List all bounties across repos.
	req := NewRequest(t, "GET", "/api/v1/hackforger/bounties").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var bounties []hackforger_model.Bounty
	DecodeJSON(t, resp, &bounties)
	// Fixtures have 8 bounties total.
	assert.GreaterOrEqual(t, len(bounties), 8)

	// X-Total-Count header should be set.
	assert.NotEmpty(t, resp.Header().Get("X-Total-Count"))

	// Filter by status=0 (Open).
	req = NewRequest(t, "GET", "/api/v1/hackforger/bounties?status=0").AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &bounties)
	for _, b := range bounties {
		assert.Equal(t, hackforger_model.BountyStatusOpen, b.Status)
	}
}

// TestHackForgerBountyStats tests GET /hackforger/bounties/stats.
func TestHackForgerBountyStats(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "GET", "/api/v1/hackforger/bounties/stats").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var stats map[string]any
	DecodeJSON(t, resp, &stats)
	// Stats should contain some keys (exact structure depends on implementation).
	assert.NotEmpty(t, stats)
}

// TestHackForgerBountyLeaderboard tests GET /hackforger/bounties/leaderboard.
func TestHackForgerBountyLeaderboard(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "GET", "/api/v1/hackforger/bounties/leaderboard").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var entries []map[string]any
	DecodeJSON(t, resp, &entries)
	// With fixtures, leaderboard may or may not have entries, but endpoint must succeed.
	assert.NotNil(t, entries)

	// With limit param.
	req = NewRequest(t, "GET", "/api/v1/hackforger/bounties/leaderboard?limit=5").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)
}
