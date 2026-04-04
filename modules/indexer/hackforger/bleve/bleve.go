// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package bleve

import (
	"context"
	"fmt"

	indexer_internal "forgejo.org/modules/indexer/hackforger/internal"
	inner "forgejo.org/modules/indexer/internal"
	inner_bleve "forgejo.org/modules/indexer/internal/bleve"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/camelcase"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/unicodenorm"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
)

const (
	hackforgerIndexerAnalyzer      = "hackforgerIndexer"
	hackforgerIndexerDocType       = "hackforgerIndexerDocType"
	hackforgerIndexerLatestVersion = 1
	unicodeNormalizeName           = "unicodeNormalize"
	maxBatchSize                   = 16
)

// IndexerData wraps internal.IndexerData for bleve's Classifier interface.
type IndexerData indexer_internal.IndexerData

// Type returns the document type, for bleve's mapping.Classifier interface.
func (i *IndexerData) Type() string {
	return hackforgerIndexerDocType
}

// generateMapping creates the bleve index mapping for HackForger entities.
func generateMapping() (mapping.IndexMapping, error) {
	m := bleve.NewIndexMapping()
	docMapping := bleve.NewDocumentMapping()

	// Keyword fields (exact match, no analysis)
	keywordFieldMapping := bleve.NewKeywordFieldMapping()
	keywordFieldMapping.Store = false
	keywordFieldMapping.IncludeInAll = false
	docMapping.AddFieldMappingsAt("entity_type", keywordFieldMapping)

	// Numeric fields
	numericFieldMapping := bleve.NewNumericFieldMapping()
	numericFieldMapping.Store = false
	numericFieldMapping.IncludeInAll = false
	docMapping.AddFieldMappingsAt("status", numericFieldMapping)
	docMapping.AddFieldMappingsAt("owner_id", numericFieldMapping)
	docMapping.AddFieldMappingsAt("org_id", numericFieldMapping)
	docMapping.AddFieldMappingsAt("repo_id", numericFieldMapping)
	docMapping.AddFieldMappingsAt("hackathon_id", numericFieldMapping)
	docMapping.AddFieldMappingsAt("created_unix", numericFieldMapping)
	docMapping.AddFieldMappingsAt("updated_unix", numericFieldMapping)

	// Text fields for full-text search
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Store = false
	textFieldMapping.IncludeInAll = false
	docMapping.AddFieldMappingsAt("title", textFieldMapping)
	docMapping.AddFieldMappingsAt("content", textFieldMapping)

	// Add unicode normalization token filter
	if err := m.AddCustomTokenFilter(unicodeNormalizeName, map[string]any{
		"type": unicodenorm.Name,
		"form": unicodenorm.NFC,
	}); err != nil {
		return nil, err
	}

	// Add custom analyzer
	if err := m.AddCustomAnalyzer(hackforgerIndexerAnalyzer, map[string]any{
		"type":          custom.Name,
		"char_filters":  []string{},
		"tokenizer":     unicode.Name,
		"token_filters": []string{unicodeNormalizeName, camelcase.Name, lowercase.Name},
	}); err != nil {
		return nil, err
	}

	m.DefaultAnalyzer = hackforgerIndexerAnalyzer
	m.AddDocumentMapping(hackforgerIndexerDocType, docMapping)
	m.AddDocumentMapping("_all", bleve.NewDocumentDisabledMapping())
	m.DefaultMapping = bleve.NewDocumentDisabledMapping()

	return m, nil
}

var _ indexer_internal.Indexer = &Indexer{}

// Indexer implements the HackForger Indexer interface using bleve full-text search.
type Indexer struct {
	inner          *inner_bleve.Indexer
	inner.Indexer // do not composite inner_bleve.Indexer directly to avoid exposing too much
}

// NewIndexer creates a new bleve-backed HackForger indexer.
func NewIndexer(indexDir string) *Indexer {
	innerIndexer := inner_bleve.NewIndexer(indexDir, hackforgerIndexerLatestVersion, generateMapping)
	return &Indexer{
		Indexer: innerIndexer,
		inner:   innerIndexer,
	}
}

// Index adds or updates entities in the bleve index.
func (b *Indexer) Index(_ context.Context, data ...*indexer_internal.IndexerData) error {
	batch := inner_bleve.NewFlushingBatch(b.inner.Indexer, maxBatchSize)
	for _, d := range data {
		docID := docKey(d.EntityType, d.ID)
		if err := batch.Index(docID, (*IndexerData)(d)); err != nil {
			return err
		}
	}
	return batch.Flush()
}

// Delete removes entities from the bleve index.
func (b *Indexer) Delete(_ context.Context, entityType string, ids ...int64) error {
	batch := inner_bleve.NewFlushingBatch(b.inner.Indexer, maxBatchSize)
	for _, id := range ids {
		docID := docKey(entityType, id)
		if err := batch.Delete(docID); err != nil {
			return err
		}
	}
	return batch.Flush()
}

// Search queries the bleve index for HackForger entities.
func (b *Indexer) Search(ctx context.Context, opts *indexer_internal.SearchOptions) (*indexer_internal.SearchResult, error) {
	q := bleve.NewBooleanQuery()

	if opts.Keyword != "" {
		// Title boosted 2x, content normal weight
		titleQ := inner_bleve.MatchPhraseQuery(opts.Keyword, "title", hackforgerIndexerAnalyzer, false, 2.0)
		contentQ := inner_bleve.MatchPhraseQuery(opts.Keyword, "content", hackforgerIndexerAnalyzer, false, 1.0)
		q.AddMust(bleve.NewDisjunctionQuery(titleQ, contentQ))
	}

	// Entity type filter
	if opts.EntityType != "" {
		typeQ := bleve.NewTermQuery(opts.EntityType)
		typeQ.SetField("entity_type")
		q.AddMust(typeQ)
	}

	var indexerQuery query.Query = q
	if q.Must == nil && q.MustNot == nil && q.Should == nil {
		indexerQuery = bleve.NewMatchAllQuery()
	}

	skip, limit := inner.ParsePaginator(opts.Paginator)
	search := bleve.NewSearchRequestOptions(indexerQuery, limit, skip, false)
	search.SortBy([]string{"-_score", "-created_unix"})

	result, err := b.inner.Indexer.SearchInContext(ctx, search)
	if err != nil {
		return nil, err
	}

	ret := &indexer_internal.SearchResult{
		Total: int64(result.Total),
		Hits:  make([]indexer_internal.Match, 0, len(result.Hits)),
	}
	for _, hit := range result.Hits {
		entityType, id, err := parseDocKey(hit.ID)
		if err != nil {
			return nil, err
		}
		ret.Hits = append(ret.Hits, indexer_internal.Match{
			ID:         id,
			EntityType: entityType,
			Score:      hit.Score,
		})
	}
	return ret, nil
}

// docKey creates a composite document key: "entityType:base36(id)"
func docKey(entityType string, id int64) string {
	return entityType + ":" + inner.Base36(id)
}

// parseDocKey parses a composite document key back into entityType and id.
func parseDocKey(key string) (string, int64, error) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			id, err := inner.ParseBase36(key[i+1:])
			if err != nil {
				return "", 0, fmt.Errorf("parseDocKey(%q): %w", key, err)
			}
			return key[:i], id, nil
		}
	}
	return "", 0, fmt.Errorf("parseDocKey(%q): missing separator", key)
}
