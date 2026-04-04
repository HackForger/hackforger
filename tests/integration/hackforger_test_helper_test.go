// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http/httptest"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hackforgerLoginAs logs in the user by ID and returns session + token.
func hackforgerLoginAs(t *testing.T, userID int64) (*TestSession, string) {
	t.Helper()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)
	return session, token
}

// hackforgerAssertFeedEvent checks that a hackforger_action record exists
// for the given ActionType and entity.
func hackforgerAssertFeedEvent(t *testing.T, opType int, entityID int64) {
	t.Helper()
	actions := make([]*hackforger_model.HackforgerAction, 0)
	err := db.GetEngine(db.DefaultContext).Where("op_type = ? AND entity_id = ?", opType, entityID).Find(&actions)
	require.NoError(t, err)
	assert.NotEmpty(t, actions, "expected feed event op_type=%d entity_id=%d", opType, entityID)
}

// hackforgerGet is a shorthand for GET /api/v1/hackforger/* requests.
func hackforgerGet(t *testing.T, token, path string, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/hackforger/%s", path)).AddTokenAuth(token)
	return MakeRequest(t, req, expectedStatus)
}

// hackforgerPost is a shorthand for POST /api/v1/hackforger/* requests with JSON body.
func hackforgerPost(t *testing.T, token, path string, body any, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/hackforger/%s", path), body).AddTokenAuth(token)
	return MakeRequest(t, req, expectedStatus)
}
