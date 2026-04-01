// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	hackforger_svc "forgejo.org/services/hackforger"
)

// ChatForm is the JSON body for the assistant chat endpoint.
type ChatForm struct {
	Query string `json:"query"`
}

// ChatAPI handles POST /api/v1/hackforger/assistant/chat
func ChatAPI(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/assistant/chat hackforger hackforgerAssistantChat
	// ---
	// summary: Chat with the HackForger AI assistant (placeholder)
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/ChatForm"
	// responses:
	//   "200":
	//     description: Assistant response
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	form := web.GetForm(ctx).(*ChatForm)
	result := hackforger_svc.Chat(form.Query)
	ctx.JSON(http.StatusOK, result)
}
