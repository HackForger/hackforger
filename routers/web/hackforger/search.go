// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// SearchWeb returns JSON search results for the web frontend.
func SearchWeb(ctx *context.Context) {
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
		ctx.ServerError("Search", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"results": results,
		"total":   total,
	})
}
