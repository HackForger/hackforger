// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
)

// ListBounties returns a list of bounties.
func ListBounties(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, []any{})
}
