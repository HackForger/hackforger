// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHackForgerReputationGet tests fetching a user's reputation by username.
func TestHackForgerReputationGet(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 4 (user4) has reputation fixture: score=1250, tier=Silver.
	req := NewRequest(t, "GET", "/api/v1/hackforger/reputation/users/user4")
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)

	assert.Equal(t, float64(4), result["user_id"])
	assert.Equal(t, "user4", result["username"])
	assert.Equal(t, float64(1250), result["score"])
	assert.Equal(t, "Silver", result["tier"])
	assert.Equal(t, float64(3), result["bounties_completed"])
	assert.Equal(t, float64(1), result["hackathon_wins"])
	assert.Equal(t, float64(2), result["grants_received"])
	assert.Equal(t, float64(15), result["total_stars"])
	assert.Equal(t, float64(5000), result["credits_earned"])
}

// TestHackForgerReputationGetUser2 tests fetching user2's reputation (Gold tier).
func TestHackForgerReputationGetUser2(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/reputation/users/user2")
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)

	assert.Equal(t, float64(2), result["user_id"])
	assert.Equal(t, "user2", result["username"])
	assert.Equal(t, float64(3500), result["score"])
	assert.Equal(t, "Gold", result["tier"])
}

// TestHackForgerReputationGetNewUser tests that a user with no reputation record gets defaults.
func TestHackForgerReputationGetNewUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 1 has no reputation fixture -- GetOrCreateReputation should create one with defaults.
	req := NewRequest(t, "GET", "/api/v1/hackforger/reputation/users/user1")
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)

	assert.Equal(t, float64(1), result["user_id"])
	assert.Equal(t, "user1", result["username"])
	assert.Equal(t, float64(0), result["score"])
	assert.Equal(t, "Bronze", result["tier"])
}

// TestHackForgerReputationGetNonExistent tests that a non-existent user returns 404.
func TestHackForgerReputationGetNonExistent(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/reputation/users/nonexistentuser99")
	MakeRequest(t, req, http.StatusNotFound)
}

// TestHackForgerReputationLeaderboard tests the reputation leaderboard endpoint.
func TestHackForgerReputationLeaderboard(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/reputation/leaderboard")
	resp := MakeRequest(t, req, http.StatusOK)

	var items []map[string]any
	DecodeJSON(t, resp, &items)

	// Should have at least 3 entries from fixtures (user2=3500, user4=1250, user5=800).
	require.GreaterOrEqual(t, len(items), 3)

	// Verify descending order by score.
	for i := 1; i < len(items); i++ {
		prevScore := items[i-1]["score"].(float64)
		currScore := items[i]["score"].(float64)
		assert.GreaterOrEqual(t, prevScore, currScore, "leaderboard should be sorted descending by score")
	}

	// First entry should be user2 (highest score = 3500).
	assert.Equal(t, "user2", items[0]["username"])
	assert.Equal(t, float64(3500), items[0]["score"])
	assert.Equal(t, "Gold", items[0]["tier"])
}

// TestHackForgerReputationLeaderboardWithLimit tests the leaderboard with a custom limit.
func TestHackForgerReputationLeaderboardWithLimit(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/reputation/leaderboard?limit=2")
	resp := MakeRequest(t, req, http.StatusOK)

	var items []map[string]any
	DecodeJSON(t, resp, &items)

	assert.Equal(t, 2, len(items), "limit=2 should return exactly 2 entries")
}

// TestHackForgerReputationRecalculate tests admin recalculating reputation for a user.
func TestHackForgerReputationRecalculate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	// Recalculate for user4.
	req := NewRequestf(t, "POST", "/api/v1/hackforger/reputation/recalculate/user4").AddTokenAuth(adminToken)
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]string
	DecodeJSON(t, resp, &result)
	assert.Equal(t, "ok", result["status"])
}

// TestHackForgerReputationRecalcNonAdmin tests that a non-admin cannot recalculate reputation.
func TestHackForgerReputationRecalcNonAdmin(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 2 is not a site admin.
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequestf(t, "POST", "/api/v1/hackforger/reputation/recalculate/user4").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)
}

// TestHackForgerReputationRecalcNonExistentUser tests recalculating for a non-existent user.
func TestHackForgerReputationRecalcNonExistentUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	req := NewRequestf(t, "POST", "/api/v1/hackforger/reputation/recalculate/nonexistentuser99").AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusNotFound)
}

// TestHackForgerReputationRecalcUnauthenticated tests that unauthenticated recalculate returns 401.
func TestHackForgerReputationRecalcUnauthenticated(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "POST", "/api/v1/hackforger/reputation/recalculate/user4")
	MakeRequest(t, req, http.StatusUnauthorized)
}

// TestAPIHackforgerReputationSettings exercises GET/PUT /hackforger/admin/reputation/settings.
func TestAPIHackforgerReputationSettings(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Admin PUT then GET roundtrip", func(t *testing.T) {
		_, adminToken := hackforgerLoginAs(t, 1) // user1 = admin per fixtures
		putBody := map[string]string{
			"weights": `{"hackathon":0.5,"bounty":0.3,"grant":0.2}`,
			"tiers":   `[{"name":"bronze","min":0},{"name":"silver","min":100}]`,
		}
		req := NewRequestWithJSON(t, "PUT", "/api/v1/hackforger/admin/reputation/settings", putBody).
			AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusOK)

		req = NewRequest(t, "GET", "/api/v1/hackforger/admin/reputation/settings").
			AddTokenAuth(adminToken)
		resp := MakeRequest(t, req, http.StatusOK)
		var got struct {
			Weights string `json:"weights"`
			Tiers   string `json:"tiers"`
		}
		DecodeJSON(t, resp, &got)
		assert.Contains(t, got.Weights, "hackathon")
		assert.Contains(t, got.Tiers, "bronze")
	})

	t.Run("403 for non-admin", func(t *testing.T) {
		_, userToken := hackforgerLoginAs(t, 2) // user2 = regular user
		req := NewRequestWithJSON(t, "PUT", "/api/v1/hackforger/admin/reputation/settings",
			map[string]string{"weights": `{"x":1}`}).
			AddTokenAuth(userToken)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("422 on invalid JSON", func(t *testing.T) {
		_, adminToken := hackforgerLoginAs(t, 1)
		req := NewRequestWithJSON(t, "PUT", "/api/v1/hackforger/admin/reputation/settings",
			map[string]string{"weights": "not-json"}).
			AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
	})
}

// silence unused import warnings in test (auth_model + unittest + user_model imported elsewhere in this file)
var _ = auth_model.AccessTokenScopeAll
var _ = unittest.AssertExistsAndLoadBean[user_model.User]
