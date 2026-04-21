// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"
)

// TestAPIHackforgerOrgJoinRequest covers POST /hackforger/orgs/{org}/join-request.
// Uses standard Forgejo fixtures: org3 has members {user2, user4, user28};
// user5 is not a member.
func TestAPIHackforgerOrgJoinRequest(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Non-member request happy path", func(t *testing.T) {
		_, token := hackforgerLoginAs(t, 5)
		req := NewRequest(t, "POST", "/api/v1/hackforger/orgs/org3/join-request").
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusAccepted)
	})

	t.Run("Existing member returns 422", func(t *testing.T) {
		_, token := hackforgerLoginAs(t, 2)
		req := NewRequest(t, "POST", "/api/v1/hackforger/orgs/org3/join-request").
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
	})

	t.Run("401 without token", func(t *testing.T) {
		req := NewRequest(t, "POST", "/api/v1/hackforger/orgs/org3/join-request")
		MakeRequest(t, req, http.StatusUnauthorized)
	})
}
