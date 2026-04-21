// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"encoding/json"
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
)

// UpdateReputationSettingsForm is the body for PUT /hackforger/admin/reputation/settings.
// Empty fields are skipped (partial update); non-empty fields must be valid JSON.
type UpdateReputationSettingsForm struct {
	Weights string `json:"weights"`
	Tiers   string `json:"tiers"`
}

// ReputationSettingsResponse is the GET/PUT response.
// swagger:response HackforgerReputationSettings
type ReputationSettingsResponse struct {
	Weights string `json:"weights"`
	Tiers   string `json:"tiers"`
}

// GetReputationSettingsAPI returns the current platform reputation algorithm config.
func GetReputationSettingsAPI(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/admin/reputation/settings hackforgerAdmin hackforgerGetReputationSettings
	// ---
	// summary: Get platform reputation algorithm settings
	// produces:
	//   - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/HackforgerReputationSettings"
	weights, _ := hackforger_model.GetSetting(ctx, "reputation.weights")
	tiers, _ := hackforger_model.GetSetting(ctx, "reputation.tiers")
	ctx.JSON(http.StatusOK, ReputationSettingsResponse{Weights: weights, Tiers: tiers})
}

// UpdateReputationSettingsAPI partially updates weights and/or tiers.
func UpdateReputationSettingsAPI(ctx *context.APIContext) {
	// swagger:operation PUT /hackforger/admin/reputation/settings hackforgerAdmin hackforgerUpdateReputationSettings
	// ---
	// summary: Update platform reputation algorithm settings
	// consumes:
	//   - application/json
	// parameters:
	//   - name: body
	//     in: body
	//     schema:
	//       "$ref": "#/definitions/UpdateReputationSettingsForm"
	// responses:
	//   "200":
	//     "$ref": "#/responses/HackforgerReputationSettings"
	//   "422":
	//     "$ref": "#/responses/validationError"
	form := web.GetForm(ctx).(*UpdateReputationSettingsForm)

	if form.Weights != "" {
		if !json.Valid([]byte(form.Weights)) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidJSON", "weights is not valid JSON")
			return
		}
		if err := hackforger_model.SetSetting(ctx, "reputation.weights", form.Weights); err != nil {
			ctx.InternalServerError(err)
			return
		}
	}
	if form.Tiers != "" {
		if !json.Valid([]byte(form.Tiers)) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidJSON", "tiers is not valid JSON")
			return
		}
		if err := hackforger_model.SetSetting(ctx, "reputation.tiers", form.Tiers); err != nil {
			ctx.InternalServerError(err)
			return
		}
	}

	weights, _ := hackforger_model.GetSetting(ctx, "reputation.weights")
	tiers, _ := hackforger_model.GetSetting(ctx, "reputation.tiers")
	ctx.JSON(http.StatusOK, ReputationSettingsResponse{Weights: weights, Tiers: tiers})
}
