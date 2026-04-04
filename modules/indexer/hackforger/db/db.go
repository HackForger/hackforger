// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package db

import (
	"context"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	indexer_internal "forgejo.org/modules/indexer/hackforger/internal"
	inner "forgejo.org/modules/indexer/internal"
	inner_db "forgejo.org/modules/indexer/internal/db"
)

var _ indexer_internal.Indexer = &Indexer{}

// Indexer implements the HackForger Indexer interface using database LIKE queries.
type Indexer struct {
	inner.Indexer
}

// NewIndexer creates a new DB-backed HackForger indexer.
func NewIndexer() *Indexer {
	return &Indexer{
		Indexer: &inner_db.Indexer{},
	}
}

// Index is a no-op for the DB backend (data is already in the database).
func (i *Indexer) Index(_ context.Context, _ ...*indexer_internal.IndexerData) error {
	return nil
}

// Delete is a no-op for the DB backend.
func (i *Indexer) Delete(_ context.Context, _ string, _ ...int64) error {
	return nil
}

// Search performs LIKE queries against the 4 HackForger tables.
func (i *Indexer) Search(ctx context.Context, opts *indexer_internal.SearchOptions) (*indexer_internal.SearchResult, error) {
	var hits []indexer_internal.Match
	keyword := "%" + opts.Keyword + "%"
	e := db.GetEngine(ctx)

	if opts.EntityType == "" || opts.EntityType == "hackathon" {
		var hackathons []*hackforger_model.Hackathon
		if err := e.Where("name LIKE ? OR description LIKE ?", keyword, keyword).
			OrderBy("created_unix DESC").Limit(5).Find(&hackathons); err != nil {
			return nil, err
		}
		for _, h := range hackathons {
			hits = append(hits, indexer_internal.Match{ID: h.ID, EntityType: "hackathon"})
		}
	}

	if opts.EntityType == "" || opts.EntityType == "bounty" {
		var bounties []*hackforger_model.Bounty
		if err := e.Where("title LIKE ?", keyword).
			OrderBy("created_unix DESC").Limit(5).Find(&bounties); err != nil {
			return nil, err
		}
		for _, b := range bounties {
			hits = append(hits, indexer_internal.Match{ID: b.ID, EntityType: "bounty"})
		}
	}

	if opts.EntityType == "" || opts.EntityType == "grant" {
		var rounds []*hackforger_model.GrantRound
		if err := e.Where("name LIKE ? OR description LIKE ?", keyword, keyword).
			OrderBy("created_unix DESC").Limit(5).Find(&rounds); err != nil {
			return nil, err
		}
		for _, r := range rounds {
			hits = append(hits, indexer_internal.Match{ID: r.ID, EntityType: "grant"})
		}
	}

	if opts.EntityType == "" || opts.EntityType == "submission" {
		var submissions []*hackforger_model.HackathonSubmission
		if err := e.Where("title LIKE ? OR description LIKE ?", keyword, keyword).
			OrderBy("created_unix DESC").Limit(5).Find(&submissions); err != nil {
			return nil, err
		}
		for _, s := range submissions {
			hits = append(hits, indexer_internal.Match{ID: s.ID, EntityType: "submission"})
		}
	}

	return &indexer_internal.SearchResult{Total: int64(len(hits)), Hits: hits}, nil
}
