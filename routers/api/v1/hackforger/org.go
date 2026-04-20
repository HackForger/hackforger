// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_service "forgejo.org/services/hackforger"
	"forgejo.org/services/context"
)

// OrgJoinRequestResponse is the API response for the join-request endpoint.
// swagger:response HackforgerOrgJoinRequest
type OrgJoinRequestResponse struct {
	Status string `json:"status"`
}

// CreateOrgJoinRequestAPI publishes a "user wants to join this org" notification.
// Fire-and-forget: returns 202 because no resource is persisted.
func CreateOrgJoinRequestAPI(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/orgs/{org}/join-request hackforgerOrg hackforgerCreateOrgJoinRequest
	// ---
	// summary: Request to join a HackForger organization
	// parameters:
	//   - name: org
	//     in: path
	//     type: string
	//     required: true
	// responses:
	//   "202":
	//     "$ref": "#/responses/HackforgerOrgJoinRequest"
	//   "422":
	//     "$ref": "#/responses/validationError"
	//   "404":
	//     "$ref": "#/responses/notFound"
	orgLink := ctx.Org.Organization.HomeLink()
	if err := hackforger_service.RequestOrgJoin(ctx, ctx.Org.Organization, ctx.Doer, orgLink); err != nil {
		if hackforger_service.IsErrAlreadyOrgMember(err) {
			ctx.Error(http.StatusUnprocessableEntity, "AlreadyMember", "user is already a member of this organization")
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusAccepted, OrgJoinRequestResponse{Status: "notified"})
}
