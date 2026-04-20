// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// JoinOrgRequest handles a signed-in user requesting to join an organization.
// Delegates to hackforger_service.RequestOrgJoin so the API and web call the same code.
func JoinOrgRequest(ctx *context.Context) {
	org := ctx.Org.Organization

	if ctx.Doer == nil {
		ctx.Redirect(org.HomeLink())
		return
	}

	err := hackforger_service.RequestOrgJoin(ctx, org, ctx.Doer, ctx.Org.OrgLink)
	switch {
	case err == nil:
		ctx.Flash.Success(ctx.Tr("hackforger.org.join_request_sent"))
	case hackforger_service.IsErrAlreadyOrgMember(err):
		ctx.Flash.Info(ctx.Tr("hackforger.org.already_member"))
	default:
		ctx.ServerError("RequestOrgJoin", err)
		return
	}
	ctx.Redirect(org.HomeLink())
}
