// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
)

// GetFeed returns the hackforger activity feed.
func GetFeed(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, map[string]any{
		"items":       []any{},
		"total_count": 0,
	})
}
