// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
)

const tplBountyExplore = "hackforger/bounty/explore"

// ExploreBounties renders the bounty explore page with filters.
func ExploreBounties(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.bounties")
	ctx.Data["PageIsExploreBounties"] = true

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	var statusFilter *hackforger_model.BountyStatus
	if s := ctx.FormString("status"); s != "" {
		switch s {
		case "open":
			st := hackforger_model.BountyStatusOpen
			statusFilter = &st
		case "claimed":
			st := hackforger_model.BountyStatusClaimed
			statusFilter = &st
		case "in_review":
			st := hackforger_model.BountyStatusInReview
			statusFilter = &st
		case "completed":
			st := hackforger_model.BountyStatusCompleted
			statusFilter = &st
		}
	}

	bounties, total, err := hackforger_model.ListBounties(ctx, hackforger_model.ListBountiesOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: 20},
		Status:      statusFilter,
	})
	if err != nil {
		ctx.ServerError("ListBounties", err)
		return
	}

	ctx.Data["Bounties"] = bounties
	ctx.Data["Total"] = total
	ctx.Data["StatusFilter"] = ctx.FormString("status")

	pager := context.NewPagination(int(total), 20, page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplBountyExplore)
}
