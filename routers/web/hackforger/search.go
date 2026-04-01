// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// SearchWeb returns JSON grouped search results for the web frontend modal.
func SearchWeb(ctx *context.Context) {
	keyword := ctx.FormTrim("q")

	result, err := hackforger_service.UnifiedSearch(ctx, &hackforger_service.UnifiedSearchOptions{
		Keyword: keyword,
		Doer:    ctx.Doer,
	})
	if err != nil {
		ctx.ServerError("UnifiedSearch", err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}
