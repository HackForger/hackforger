// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"
	"strings"

	system_model "forgejo.org/models/system"
	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/validation"
	hackforger_service "forgejo.org/services/hackforger"
	"forgejo.org/services/context"
)

const (
	tplAdminLandingConfig base.TplName = "hackforger/admin/landing_config"

	settingKeyLandingPrefix = "hackforger.landing.league_prefix"
)

// AdminLandingConfig renders the landing-page configuration admin form.
// GET /-/admin/hackforger/landing-config
func AdminLandingConfig(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.landing.config.title")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminHackforgerLanding"] = true

	prefix, _ := system_model.GetSettingByKey(ctx, settingKeyLandingPrefix)
	ctx.Data["LandingLeaguePrefix"] = prefix

	ctx.HTML(http.StatusOK, tplAdminLandingConfig)
}

// AdminLandingConfigPost saves the league_prefix setting.
// POST /-/admin/hackforger/landing-config
func AdminLandingConfigPost(ctx *context.Context) {
	prefix := strings.TrimSpace(ctx.FormString("league_prefix"))

	// Empty prefix is allowed — it disables the landing page.
	if prefix != "" {
		// Validate by attempting Forgejo's username/org-name check on a synthesized slug.
		// If "<prefix>-s1-w1" wouldn't be a valid org name, the prefix is unusable.
		if !validation.IsValidUsername(prefix + "-s1-w1") {
			ctx.Flash.Error(ctx.Tr("hackforger.landing.invalid_prefix"))
			ctx.Redirect(setting.AppSubURL + "/admin/hackforger/landing-config")
			return
		}
	}

	if err := system_model.SetSettings(ctx, map[string]string{
		settingKeyLandingPrefix: prefix,
	}); err != nil {
		ctx.ServerError("SetSettings", err)
		return
	}

	hackforger_service.InvalidateLandingCache()
	ctx.Flash.Success(ctx.Tr("hackforger.landing.saved"))
	ctx.Redirect(setting.AppSubURL + "/admin/hackforger/landing-config")
}
