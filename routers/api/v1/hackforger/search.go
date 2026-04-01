// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// SearchAPI performs a unified cross-entity keyword search.
//
// swagger:operation GET /hackforger/search hackforger hackforgerSearch
// ---
// summary: Search across hackathons, bounties, grants, submissions, repos, users, issues
// produces:
// - application/json
// parameters:
// - name: q
//   in: query
//   description: Search keyword
//   type: string
//   required: true
// responses:
//   "200":
//     description: Grouped search results
func SearchAPI(ctx *context.APIContext) {
	keyword := ctx.FormTrim("q")

	result, err := hackforger_service.UnifiedSearch(ctx, &hackforger_service.UnifiedSearchOptions{
		Keyword: keyword,
		Doer:    ctx.Doer,
	})
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}
