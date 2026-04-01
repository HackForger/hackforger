// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// SearchAPI performs a cross-entity keyword search.
//
// swagger:operation GET /hackforger/search hackforger hackforgerSearch
// ---
// summary: Search hackathons, bounties, and grant rounds
// produces:
// - application/json
// parameters:
// - name: q
//   in: query
//   description: Search keyword
//   type: string
//   required: true
// - name: scope
//   in: query
//   description: Search scope (all, hackathons, bounties, grants)
//   type: string
//   default: all
// - name: page
//   in: query
//   description: Page number
//   type: integer
//   default: 1
// - name: limit
//   in: query
//   description: Page size
//   type: integer
//   default: 20
// responses:
//   "200":
//     description: Search results
func SearchAPI(ctx *context.APIContext) {
	keyword := ctx.FormTrim("q")
	scope := ctx.FormTrim("scope")
	if scope == "" {
		scope = "all"
	}

	results, total, err := hackforger_service.Search(ctx, &hackforger_service.SearchOptions{
		Keyword: keyword,
		Scope:   scope,
		Page:    ctx.FormInt("page"),
		Limit:   ctx.FormInt("limit"),
	})
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"results": results,
		"total":   total,
	})
}
