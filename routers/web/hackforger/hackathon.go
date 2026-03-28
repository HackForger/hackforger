// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/services/context"
)

const tplExplore = "hackforger/explore"

// ExploreHackathons renders the hackathon explore page.
func ExploreHackathons(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.hackathons")
	ctx.Data["PageIsExploreHackathons"] = true
	ctx.HTML(200, tplExplore)
}

// ExploreGrants renders the grants explore page.
func ExploreGrants(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.grants")
	ctx.Data["PageIsExploreGrants"] = true
	ctx.HTML(200, tplExplore)
}
