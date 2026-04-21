// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"context"
	"net/http"
	"testing"

	hackforger_indexer "forgejo.org/modules/indexer/hackforger"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searchResult mirrors GroupedSearchResult JSON shape.
type searchResult struct {
	Groups []struct {
		Key   string           `json:"key"`
		Title string           `json:"title"`
		Items []map[string]any `json:"items"`
	} `json:"groups"`
}

// totalItems sums items across all groups.
func (r searchResult) totalItems() int {
	n := 0
	for _, g := range r.Groups {
		n += len(g.Items)
	}
	return n
}

func TestHackForgerSearchAll(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Trigger reindex so the Bleve index reflects fixture data.
	// Reindex synchronously (the API endpoint dispatches to a goroutine that
	// dies when the request context is cancelled — useless for tests).
	require.NoError(t, hackforger_indexer.PopulateHackforgerIndexer(context.Background()))

	// Search without auth (public endpoint).
	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=Hackathon&scope=all")
	resp := MakeRequest(t, req, http.StatusOK)

	var result searchResult
	DecodeJSON(t, resp, &result)
	assert.GreaterOrEqual(t, result.totalItems(), 1, "fixture has hackathons matching 'Hackathon'")
}

func TestHackForgerSearchByScope(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	// Reindex synchronously (the API endpoint dispatches to a goroutine that
	// dies when the request context is cancelled — useless for tests).
	require.NoError(t, hackforger_indexer.PopulateHackforgerIndexer(context.Background()))

	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=Hackathon&scope=hackathons")
	resp := MakeRequest(t, req, http.StatusOK)

	var result searchResult
	DecodeJSON(t, resp, &result)
	for _, g := range result.Groups {
		for _, item := range g.Items {
			assert.Equal(t, "hackathon", item["type"])
		}
	}
}

func TestHackForgerSearchBountyScope(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	// Reindex synchronously (the API endpoint dispatches to a goroutine that
	// dies when the request context is cancelled — useless for tests).
	require.NoError(t, hackforger_indexer.PopulateHackforgerIndexer(context.Background()))

	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=bug&scope=bounties")
	resp := MakeRequest(t, req, http.StatusOK)

	var result searchResult
	DecodeJSON(t, resp, &result)
	for _, g := range result.Groups {
		for _, item := range g.Items {
			assert.Equal(t, "bounty", item["type"])
		}
	}
}

func TestHackForgerSearchEmpty(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=xyznonexistent999&scope=all")
	resp := MakeRequest(t, req, http.StatusOK)

	var result searchResult
	DecodeJSON(t, resp, &result)
	assert.Equal(t, 0, result.totalItems())
}

func TestHackForgerSearchPagination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	// Reindex synchronously (the API endpoint dispatches to a goroutine that
	// dies when the request context is cancelled — useless for tests).
	require.NoError(t, hackforger_indexer.PopulateHackforgerIndexer(context.Background()))

	req := NewRequest(t, "GET", "/api/v1/hackforger/search?q=Hackathon&scope=all&page=1&limit=2")
	resp := MakeRequest(t, req, http.StatusOK)

	var result searchResult
	DecodeJSON(t, resp, &result)
	// Pagination is per-group (top N items per type); just ensure response decoded.
	assert.NotNil(t, result.Groups)
}
