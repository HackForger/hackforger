// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestHackForgerSearchAll(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Search without auth (public endpoint)
	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=Hackathon&scope=all")
	resp := MakeRequest(t, req, http.StatusOK)

	var result struct {
		Results []map[string]any `json:"results"`
		Total   int64            `json:"total"`
	}
	DecodeJSON(t, resp, &result)
	assert.Greater(t, result.Total, int64(0))
	// Fixture has 6 hackathons with "Hackathon" in the name
	assert.GreaterOrEqual(t, len(result.Results), 1)
}

func TestHackForgerSearchByScope(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=Hackathon&scope=hackathons")
	resp := MakeRequest(t, req, http.StatusOK)

	var result struct {
		Results []map[string]any `json:"results"`
		Total   int64            `json:"total"`
	}
	DecodeJSON(t, resp, &result)
	for _, r := range result.Results {
		assert.Equal(t, "hackathon", r["type"])
	}
}

func TestHackForgerSearchBountyScope(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=bug&scope=bounties")
	resp := MakeRequest(t, req, http.StatusOK)

	var result struct {
		Results []map[string]any `json:"results"`
		Total   int64            `json:"total"`
	}
	DecodeJSON(t, resp, &result)
	for _, r := range result.Results {
		assert.Equal(t, "bounty", r["type"])
	}
}

func TestHackForgerSearchEmpty(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=xyznonexistent999&scope=all")
	resp := MakeRequest(t, req, http.StatusOK)

	var result struct {
		Results []map[string]any `json:"results"`
		Total   int64            `json:"total"`
	}
	DecodeJSON(t, resp, &result)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.Results)
}

func TestHackForgerSearchPagination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=Hackathon&scope=all&page=1&limit=2")
	resp := MakeRequest(t, req, http.StatusOK)

	var result struct {
		Results []map[string]any `json:"results"`
		Total   int64            `json:"total"`
	}
	DecodeJSON(t, resp, &result)
	assert.LessOrEqual(t, len(result.Results), 2)
}
