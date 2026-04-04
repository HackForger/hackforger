// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package internal

import (
	"context"
	"errors"

	inner "forgejo.org/modules/indexer/internal"
)

// Indexer defines the HackForger search indexer interface.
type Indexer interface {
	inner.Indexer
	Index(ctx context.Context, data ...*IndexerData) error
	Delete(ctx context.Context, entityType string, ids ...int64) error
	Search(ctx context.Context, opts *SearchOptions) (*SearchResult, error)
}

// NewDummyIndexer returns a dummy indexer that returns errors for all operations.
func NewDummyIndexer() Indexer {
	return &dummyIndexer{
		Indexer: inner.NewDummyIndexer(),
	}
}

type dummyIndexer struct {
	inner.Indexer
}

func (d *dummyIndexer) Index(_ context.Context, _ ...*IndexerData) error {
	return errors.New("indexer is not ready")
}

func (d *dummyIndexer) Delete(_ context.Context, _ string, _ ...int64) error {
	return errors.New("indexer is not ready")
}

func (d *dummyIndexer) Search(_ context.Context, _ *SearchOptions) (*SearchResult, error) {
	return nil, errors.New("indexer is not ready")
}
