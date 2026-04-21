// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIHackforgerPhaseType(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	_, adminToken := hackforgerLoginAs(t, 1)

	var createdID int64

	t.Run("Create", func(t *testing.T) {
		body := map[string]any{
			"activity_kind":     "hackathon",
			"key":               "test_phase_apicover",
			"display_name_i18n": "test.phase.display",
			"is_unique":         false,
			"allowed_actions":   `["test"]`,
			"default_order":     99,
		}
		req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/admin/phase-types", body).
			AddTokenAuth(adminToken)
		resp := MakeRequest(t, req, http.StatusCreated)
		var got struct {
			ID int64 `json:"ID"`
		}
		DecodeJSON(t, resp, &got)
		assert.NotZero(t, got.ID)
		createdID = got.ID
	})

	t.Run("List includes the new entry", func(t *testing.T) {
		req := NewRequest(t, "GET", "/api/v1/hackforger/admin/phase-types").
			AddTokenAuth(adminToken)
		resp := MakeRequest(t, req, http.StatusOK)
		var pts []struct {
			ID int64 `json:"ID"`
		}
		DecodeJSON(t, resp, &pts)
		found := false
		for _, p := range pts {
			if p.ID == createdID {
				found = true
				break
			}
		}
		assert.True(t, found, "newly created phase type should appear in list")
	})

	t.Run("Update", func(t *testing.T) {
		body := map[string]any{
			"activity_kind":     "hackathon",
			"key":               "test_phase_updated",
			"display_name_i18n": "test.phase.updated",
			"is_unique":         true,
			"allowed_actions":   `["updated"]`,
			"default_order":     100,
		}
		req := NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/hackforger/admin/phase-types/%d", createdID), body).
			AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusOK)
	})

	t.Run("Delete", func(t *testing.T) {
		req := NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/hackforger/admin/phase-types/%d", createdID)).
			AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusNoContent)
	})

	t.Run("403 for non-admin POST", func(t *testing.T) {
		_, userToken := hackforgerLoginAs(t, 2)
		req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/admin/phase-types",
			map[string]any{"activity_kind": "hackathon", "key": "x_nonadmin", "display_name_i18n": "x"}).
			AddTokenAuth(userToken)
		MakeRequest(t, req, http.StatusForbidden)
	})
}
