// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIHackforgerTrackPrizeDistValidation exercises the five cases in the
// prize_dist_mode API contract introduced by the #77 fix.
//
// Setup: creates a fresh draft hackathon per-test run so sub-tests are
// independent. Uses user 2 (organizer, member of org 3) — the same fixture
// user used by all other hackathon integration tests.
func TestAPIHackforgerTrackPrizeDistValidation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user 2 is the organizer; org_id=3 is the org they belong to.
	token := hackathonToken(t, 2)

	hackathonBody := map[string]any{
		"name":   "Track Validation Test",
		"slug":   "track-validation-test",
		"org_id": 3,
	}
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/hackathons", hackathonBody).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var h struct {
		ID int64 `json:"ID"`
	}
	DecodeJSON(t, resp, &h)
	require.NotZero(t, h.ID, "hackathon ID must be assigned before running sub-tests")
	tracksURL := fmt.Sprintf("/api/v1/hackforger/hackathons/%d/tracks", h.ID)

	t.Run("OmittingMode_DefaultsToWinnerTakesAll", func(t *testing.T) {
		body := map[string]any{
			"name":          "Default Mode Track",
			"prize_credits": 100,
			// prize_dist_mode intentionally omitted
		}
		req := NewRequestWithJSON(t, "POST", tracksURL, body).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)
		var got struct {
			ID            int64  `json:"ID"`
			PrizeDistMode string `json:"PrizeDistMode"`
		}
		DecodeJSON(t, resp, &got)
		assert.Equal(t, "winner_takes_all", got.PrizeDistMode,
			"omitted prize_dist_mode must default to winner_takes_all")
	})

	t.Run("ExplicitInvalidMode_Returns400", func(t *testing.T) {
		body := map[string]any{
			"name":            "Bogus Mode Track",
			"prize_credits":   100,
			"prize_dist_mode": "definitely_not_a_mode",
		}
		req := NewRequestWithJSON(t, "POST", tracksURL, body).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusBadRequest)
	})

	t.Run("Tiered_ValidRatios_Returns201", func(t *testing.T) {
		body := map[string]any{
			"name":              "Tiered OK Track",
			"prize_credits":     100,
			"prize_dist_mode":   "tiered",
			"prize_dist_ratios": `[{"rank":1,"pct":60},{"rank":2,"pct":40}]`,
		}
		req := NewRequestWithJSON(t, "POST", tracksURL, body).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)
	})

	t.Run("Tiered_BadRatios_SumNot100_Returns400", func(t *testing.T) {
		body := map[string]any{
			"name":              "Tiered Bad Track",
			"prize_credits":     100,
			"prize_dist_mode":   "tiered",
			"prize_dist_ratios": `[{"rank":1,"pct":50},{"rank":2,"pct":40}]`,
		}
		req := NewRequestWithJSON(t, "POST", tracksURL, body).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusBadRequest)
	})

	t.Run("Tiered_MalformedJSON_Returns400", func(t *testing.T) {
		body := map[string]any{
			"name":              "Tiered Malformed Track",
			"prize_credits":     100,
			"prize_dist_mode":   "tiered",
			"prize_dist_ratios": "not even close to json",
		}
		req := NewRequestWithJSON(t, "POST", tracksURL, body).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusBadRequest)
	})
}
