// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_indexer "forgejo.org/modules/indexer/hackforger"
	"forgejo.org/modules/log"
	"forgejo.org/services/context"
)

// AdminReindex triggers a full reindex of HackForger entities.
func AdminReindex(ctx *context.APIContext) {
	// swagger:operation POST /admin/hackforger/reindex admin adminHackforgerReindex
	// ---
	// summary: Trigger full reindex of HackForger entities
	// produces:
	// - application/json
	// responses:
	//   "204":
	//     description: reindex started
	//   "403":
	//     "$ref": "#/responses/forbidden"

	go func() {
		if err := hackforger_indexer.PopulateHackforgerIndexer(ctx); err != nil {
			log.Error("Admin HackForger reindex failed: %v", err)
		}
	}()
	ctx.Status(http.StatusNoContent)
}
