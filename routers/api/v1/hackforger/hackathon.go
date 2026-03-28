// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
)

// ListHackathons returns a list of hackathons.
func ListHackathons(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, []any{})
}
