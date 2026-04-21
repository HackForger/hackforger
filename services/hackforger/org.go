// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"errors"
	"fmt"

	hackforger_model "forgejo.org/models/hackforger"
	organization_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	notify_service "forgejo.org/services/notify"
)

// ErrAlreadyOrgMember is returned when the doer is already a member or owner.
var ErrAlreadyOrgMember = errors.New("user is already a member of the org")

// IsErrAlreadyOrgMember reports whether err == ErrAlreadyOrgMember.
func IsErrAlreadyOrgMember(err error) bool { return errors.Is(err, ErrAlreadyOrgMember) }

// RequestOrgJoin notifies the org's owners (per-owner direct notification)
// that the doer wants to join. Returns ErrAlreadyOrgMember if the doer already
// belongs to the org. Fire-and-forget: no row is persisted; this only
// publishes one feed event per owner.
//
// orgLink is the org's web link (e.g. "/web3-innovation"); the API caller
// supplies it because ctx.Org.OrgLink is web-context only.
func RequestOrgJoin(ctx context.Context, org *organization_model.Organization, doer *user_model.User, orgLink string) error {
	isMember, err := organization_model.IsOrganizationMember(ctx, org.ID, doer.ID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyOrgMember
	}

	ownerTeam, err := organization_model.GetOwnerTeam(ctx, org.ID)
	if err != nil {
		return err
	}
	if err := ownerTeam.LoadMembers(ctx); err != nil {
		return err
	}

	addMemberLink := fmt.Sprintf("%s/teams/%s?username=%s",
		orgLink, ownerTeam.LowerName, doer.Name)

	for _, owner := range ownerTeam.Members {
		notify_service.HackforgerEntityCreated(ctx, doer, &notify_service.HackforgerEventOpts{
			OpType:       hackforger_model.ActionOrgJoinRequest,
			EntityType:   "org",
			EntityID:     org.ID,
			EntityName:   org.Name,
			OrgID:        org.ID,
			AudienceType: notify_service.AudienceDirectUser,
			TargetUserID: owner.ID,
			Content: map[string]string{
				"username":        doer.Name,
				"add_member_link": addMemberLink,
			},
		})
	}

	log.Info("HackForger: user %s requested to join org %s", doer.Name, org.Name)
	return nil
}
