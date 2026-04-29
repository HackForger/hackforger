// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_service "forgejo.org/services/hackforger"
	"forgejo.org/services/context"
)

// GetLandingCardsAPI serves the landing-page card data.
// Public endpoint, no authentication required.
//
// GET /api/v1/hackforger/landing/cards
//
// Cached 5 minutes server-side via services/hackforger/landing_cache.go.
// Returns 12 numbered card slots; each slot has either a hydrated hackathon
// (slug + name + phases + tracks) or empty defaults (slug=null, enabled=false).
func GetLandingCardsAPI(ctx *context.APIContext) {
	payload, err := hackforger_service.GetLandingPayload(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetLandingPayload", err)
		return
	}
	ctx.Resp.Header().Set("Content-Type", "application/json")
	// Cache hint to browsers (5 min server cache; allow short browser cache).
	ctx.Resp.Header().Set("Cache-Control", "public, max-age=60")
	ctx.Resp.Write(payload)
}
