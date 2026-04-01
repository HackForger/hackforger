// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package internal

import "forgejo.org/models/db"

// IndexerData represents a HackForger entity in the search index.
type IndexerData struct {
	ID          int64  `json:"id"`
	EntityType  string `json:"entity_type"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Status      int    `json:"status"`
	OwnerID     int64  `json:"owner_id"`
	OrgID       int64  `json:"org_id"`
	RepoID      int64  `json:"repo_id"`
	HackathonID int64  `json:"hackathon_id"`
	CreatedUnix int64  `json:"created_unix"`
	UpdatedUnix int64  `json:"updated_unix"`
}

// SearchOptions holds search parameters.
type SearchOptions struct {
	Keyword    string
	EntityType string // filter by type, empty = all
	Paginator  *db.ListOptions
}

// Match represents a single search hit.
type Match struct {
	ID         int64   `json:"id"`
	EntityType string  `json:"entity_type"`
	Score      float64 `json:"score"`
}

// SearchResult holds search results.
type SearchResult struct {
	Total int64
	Hits  []Match
}
