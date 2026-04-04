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
)

func TestHackForgerAssistantChat(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/assistant/chat",
		map[string]string{"query": "What bounties are available?"}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var result struct {
		Message    string `json:"message"`
		Status     string `json:"status"`
		Disclaimer string `json:"disclaimer"`
	}
	DecodeJSON(t, resp, &result)
	assert.Equal(t, "placeholder", result.Status)
	assert.NotEmpty(t, result.Message)
	assert.NotEmpty(t, result.Disclaimer)
}

func TestHackForgerAssistantChatNoAuth(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/assistant/chat",
		map[string]string{"query": "test"})
	MakeRequest(t, req, http.StatusUnauthorized)
}

func TestHackForgerAssistantChatEmptyQuery(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/assistant/chat",
		map[string]string{"query": ""}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var result struct {
		Status string `json:"status"`
	}
	DecodeJSON(t, resp, &result)
	assert.Equal(t, "placeholder", result.Status)
}
