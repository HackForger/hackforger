// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
	hackforger_svc "forgejo.org/services/hackforger"
)

// ChatWeb handles POST /hackforger/assistant/chat (web route for the command-K modal).
// Returns JSON, used by the Vue frontend component.
func ChatWeb(ctx *context.Context) {
	query := ctx.FormString("query")
	result := hackforger_svc.Chat(query)
	ctx.JSON(http.StatusOK, result)
}
