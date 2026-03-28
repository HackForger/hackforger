// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/models/db"
	"forgejo.org/services/context"
	hackforger_svc "forgejo.org/services/hackforger"
)

const tplCredits = "hackforger/credits/dashboard"

// CreditsDashboard renders the user's credits page.
func CreditsDashboard(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.credits.title")

	acct, err := hackforger_svc.GetOrCreateCreditAccount(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetOrCreateCreditAccount", err)
		return
	}
	ctx.Data["Balance"] = acct.Balance

	// Load transactions
	txns, _, err := hackforger_svc.ListTransactions(ctx, ctx.Doer.ID, db.ListOptions{Page: 1, PageSize: 50})
	if err != nil {
		ctx.ServerError("ListTransactions", err)
		return
	}
	ctx.Data["Transactions"] = txns

	ctx.HTML(http.StatusOK, tplCredits)
}
