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
)

func TestHackForgerHackathonJudgeEndpoint(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Hackathon 4 is Hacking status — transition to Judging via new /judge endpoint
	req := NewRequest(t, "POST", "/api/v1/hackforger/hackathons/4/judge").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)
}

func TestHackForgerHackathonStartJudgingRemoved(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	req := NewRequest(t, "POST", "/api/v1/hackforger/hackathons/4/start-judging").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}
