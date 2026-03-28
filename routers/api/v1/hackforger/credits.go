// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
	hackforger_svc "forgejo.org/services/hackforger"
)

// GetBalance returns the user's credit balance.
func GetBalance(ctx *context.APIContext) {
	if ctx.Doer == nil {
		ctx.JSON(http.StatusOK, map[string]int64{"balance": 0})
		return
	}

	userID := ctx.Doer.ID
	// Admin can query other users' balance
	if qUID := ctx.FormInt64("user_id"); qUID > 0 && ctx.Doer.IsAdmin {
		userID = qUID
	}

	acct, err := hackforger_svc.GetOrCreateCreditAccount(ctx, userID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetOrCreateCreditAccount", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]int64{"balance": acct.Balance, "user_id": userID})
}
