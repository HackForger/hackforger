// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/graceful"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// AdminReputation renders the reputation settings admin page.
func AdminReputation(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.admin.reputation")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminHackforgerReputation"] = true

	weights := hackforger_model.GetSettingWithDefault(ctx, "reputation.weights", hackforger_service.DefaultWeightsJSON())
	tiers := hackforger_model.GetSettingWithDefault(ctx, "reputation.tiers", hackforger_service.DefaultTiersJSON())
	ctx.Data["Weights"] = weights
	ctx.Data["Tiers"] = tiers

	ctx.HTML(http.StatusOK, "admin/hackforger/reputation")
}

// AdminReputationPost handles form submission to save reputation weights and tiers.
func AdminReputationPost(ctx *context.Context) {
	weights := ctx.FormString("weights")
	tiers := ctx.FormString("tiers")

	if weights != "" {
		if err := hackforger_model.SetSetting(ctx, "reputation.weights", weights); err != nil {
			ctx.ServerError("SetSetting weights", err)
			return
		}
	}
	if tiers != "" {
		if err := hackforger_model.SetSetting(ctx, "reputation.tiers", tiers); err != nil {
			ctx.ServerError("SetSetting tiers", err)
			return
		}
	}

	ctx.Flash.Success(ctx.Tr("hackforger.admin.reputation.save_success"))
	ctx.Redirect(ctx.Req.URL.Path)
}

// AdminReputationRecalc triggers a background recalculation of all user reputations.
func AdminReputationRecalc(ctx *context.Context) {
	// Use graceful context — HTTP request context is canceled after response
	go hackforger_service.RecalculateAllReputations(graceful.GetManager().HammerContext())
	ctx.Flash.Success(ctx.Tr("hackforger.admin.reputation.recalc_success"))
	ctx.Redirect("/admin/hackforger/reputation")
}
