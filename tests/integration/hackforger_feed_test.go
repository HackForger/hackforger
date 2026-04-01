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

// TestHackForgerFeedGlobal tests the global feed endpoint (no auth required).
// Global feed returns actions where user_id=0 (broadcast events).
func TestHackForgerFeedGlobal(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/feed?type=global")
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)

	items, ok := result["items"].([]any)
	require.True(t, ok, "items should be an array")
	assert.Greater(t, len(items), 0, "global feed should have entries")

	totalCount, ok := result["total_count"].(float64)
	require.True(t, ok, "total_count should be a number")
	assert.Greater(t, totalCount, float64(0))

	// Verify structure of first item.
	first := items[0].(map[string]any)
	assert.Contains(t, first, "id")
	assert.Contains(t, first, "op_type")
	assert.Contains(t, first, "op_name")
	assert.Contains(t, first, "actor")
	assert.Contains(t, first, "created_at")
}

// TestHackForgerFeedDefaultIsGlobal tests that omitting type defaults to global.
func TestHackForgerFeedDefaultIsGlobal(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/feed")
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)

	items, ok := result["items"].([]any)
	require.True(t, ok)
	assert.Greater(t, len(items), 0, "default feed should return global items")
}

// TestHackForgerFeedFollowing tests the following feed for an authenticated user.
// User 4 follows user 2 and user 5 (from follow.yml fixtures).
// The following feed returns global events (user_id=0) + events targeted at user 4 (user_id=4).
func TestHackForgerFeedFollowing(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// User 4 follows user 2 (organizer) and user 5 (hacker2).
	user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session := loginUser(t, user4.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "GET", "/api/v1/hackforger/feed?type=following").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)

	items, ok := result["items"].([]any)
	require.True(t, ok)
	assert.Greater(t, len(items), 0, "following feed should have entries for user 4")

	// All items should either be global (user_id=0) or targeted at user 4 (user_id=4).
	// The feed handler uses GetHackforgerFeeds with UserID=4 and IncludeGlobal=true.
}

// TestHackForgerFeedFollowingUnauthenticated tests that following feed requires auth.
func TestHackForgerFeedFollowingUnauthenticated(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/feed?type=following")
	MakeRequest(t, req, http.StatusUnauthorized)
}

// TestHackForgerFeedPagination tests feed pagination with page and limit.
func TestHackForgerFeedPagination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Request page 1 with limit 2.
	req := NewRequest(t, "GET", "/api/v1/hackforger/feed?type=global&page=1&limit=2")
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)

	items, ok := result["items"].([]any)
	require.True(t, ok)
	assert.Equal(t, 2, len(items), "page 1 with limit 2 should return exactly 2 items")

	totalCount := result["total_count"].(float64)
	assert.Greater(t, totalCount, float64(2), "total count should be more than the page limit")

	// Request page 2 with limit 2.
	req = NewRequest(t, "GET", "/api/v1/hackforger/feed?type=global&page=2&limit=2")
	resp = MakeRequest(t, req, http.StatusOK)

	var result2 map[string]any
	DecodeJSON(t, resp, &result2)

	items2, ok := result2["items"].([]any)
	require.True(t, ok)
	assert.Greater(t, len(items2), 0, "page 2 should have items")

	// Items on page 1 and page 2 should be different.
	firstID := items[0].(map[string]any)["id"]
	secondPageFirstID := items2[0].(map[string]any)["id"]
	assert.NotEqual(t, firstID, secondPageFirstID, "different pages should have different items")
}

// TestHackForgerFeedEntityFilter tests filtering feed by entity type and ID.
func TestHackForgerFeedEntityFilter(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Filter by hackathon entity_id=1 -- fixtures have actions 1, 2, 3 for hackathon 1.
	req := NewRequest(t, "GET", "/api/v1/hackforger/feed?type=entity&entity_type=hackathon&entity_id=1")
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)

	items, ok := result["items"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(items), 3, "should have at least 3 hackathon-1 events")

	// Every item should reference hackathon entity.
	for _, item := range items {
		m := item.(map[string]any)
		entity := m["entity"].(map[string]any)
		assert.Equal(t, "hackathon", entity["type"])
		assert.Equal(t, float64(1), entity["id"])
	}
}

// TestHackForgerFeedUserFilter tests filtering feed by acting user.
func TestHackForgerFeedUserFilter(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Filter by user 5's actions.
	req := NewRequest(t, "GET", "/api/v1/hackforger/feed?type=user&user_id=5")
	resp := MakeRequest(t, req, http.StatusOK)

	var result map[string]any
	DecodeJSON(t, resp, &result)

	items, ok := result["items"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(items), 2, "user 5 should have at least 2 actions")

	// All items should have actor.id == 5.
	for _, item := range items {
		m := item.(map[string]any)
		actor := m["actor"].(map[string]any)
		assert.Equal(t, float64(5), actor["id"])
	}
}

// TestHackForgerFeedInvalidType tests that an invalid feed type returns 400.
func TestHackForgerFeedInvalidType(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/feed?type=invalid")
	MakeRequest(t, req, http.StatusBadRequest)
}
