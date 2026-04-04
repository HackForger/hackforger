// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	hackforger_model "forgejo.org/models/hackforger"
	organization_model "forgejo.org/models/organization"
	"forgejo.org/modules/log"
	"forgejo.org/services/context"
	notify_service "forgejo.org/services/notify"
)

// JoinOrgRequest handles a signed-in user requesting to join an organization.
// It sends a feed notification to the org owner with a link to the add-member page.
func JoinOrgRequest(ctx *context.Context) {
	org := ctx.Org.Organization

	// Guard: must be signed in and not already a member or owner.
	if ctx.Doer == nil {
		ctx.Redirect(org.HomeLink())
		return
	}
	isMember, err := organization_model.IsOrganizationMember(ctx, org.ID, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("IsOrganizationMember", err)
		return
	}
	if isMember {
		ctx.Flash.Info(ctx.Tr("hackforger.org.already_member"))
		ctx.Redirect(org.HomeLink())
		return
	}

	// Find the owner team to get the org owner(s).
	ownerTeam, err := organization_model.GetOwnerTeam(ctx, org.ID)
	if err != nil {
		ctx.ServerError("GetOwnerTeam", err)
		return
	}
	if err := ownerTeam.LoadMembers(ctx); err != nil {
		ctx.ServerError("LoadMembers", err)
		return
	}

	// Build the add-member link for convenience.
	addMemberLink := fmt.Sprintf("%s/teams/%s?username=%s",
		ctx.Org.OrgLink, ownerTeam.LowerName, ctx.Doer.Name)

	// Publish a feed event directed at each owner.
	for _, owner := range ownerTeam.Members {
		notify_service.HackforgerEntityCreated(ctx, ctx.Doer, &notify_service.HackforgerEventOpts{
			OpType:       hackforger_model.ActionOrgJoinRequest,
			EntityType:   "org",
			EntityID:     org.ID,
			EntityName:   org.Name,
			OrgID:        org.ID,
			AudienceType: notify_service.AudienceDirectUser,
			TargetUserID: owner.ID,
			Content: map[string]string{
				"username":        ctx.Doer.Name,
				"add_member_link": addMemberLink,
			},
		})
	}

	log.Info("HackForger: user %s requested to join org %s", ctx.Doer.Name, org.Name)

	ctx.Flash.Success(ctx.Tr("hackforger.org.join_request_sent"))
	ctx.Redirect(org.HomeLink())
}
