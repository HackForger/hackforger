// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
	notify_service "forgejo.org/services/notify"
)

// RegisterForm is the form for registering to a hackathon.
type RegisterForm struct {
	TeamName string `json:"team_name" binding:"Required"`
	TrackID  int64  `json:"track_id"`
}

// UpdateRegistrationForm is the form for updating a registration's status.
type UpdateRegistrationForm struct {
	Status int `json:"status" binding:"Required"`
}

// Register registers the authenticated user to a hackathon.
//
// swagger:operation POST /hackforger/hackathons/{id}/registrations hackforger hackforgerRegister
// ---
// summary: Register for a hackathon
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/RegisterForm"
// responses:
//   "201":
//     description: Registration created
//   "400":
//     description: Hackathon is not accepting registrations
//   "404":
//     "$ref": "#/responses/notFound"
//   "409":
//     description: Already registered
func Register(ctx *context.APIContext) {
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	canRegister, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register")
	if !canRegister {
		ctx.Error(http.StatusBadRequest, "InvalidPhase", "hackathon is not accepting registrations")
		return
	}
	f := web.GetForm(ctx).(*RegisterForm)
	r := &hackforger_model.HackathonRegistration{
		HackathonID: h.ID,
		UserID:      ctx.Doer.ID,
		TeamName:    f.TeamName,
		TrackID:     f.TrackID,
	}
	if err := hackforger_model.CreateRegistration(ctx, r); err != nil {
		if hackforger_model.IsErrDuplicateRegistration(err) {
			ctx.Error(http.StatusConflict, "AlreadyRegistered", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}

	// Publish feed event for registration (global + followers so it appears in entity timelines)
	notify_service.HackforgerEntityCreated(ctx, ctx.Doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionHackathonRegistered,
		EntityType:   "hackathon",
		EntityID:     h.ID,
		EntityName:   h.Name,
		EntitySlug:   h.Slug,
		AudienceType: notify_service.AudienceGlobal | notify_service.AudienceFollowers,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
		},
	})

	ctx.JSON(http.StatusCreated, r)
}

// ListRegistrations returns a paginated list of registrations for a hackathon.
//
// swagger:operation GET /hackforger/hackathons/{id}/registrations hackforger hackforgerListRegistrations
// ---
// summary: List registrations for a hackathon
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: page
//   in: query
//   description: page number of results to return (1-based)
//   type: integer
// - name: limit
//   in: query
//   description: page size of results
//   type: integer
// responses:
//   "200":
//     description: Registration list
func ListRegistrations(ctx *context.APIContext) {
	opts := hackforger_model.ListRegistrationsOptions{
		HackathonID: ctx.ParamsInt64(":id"),
	}
	opts.Page = ctx.FormInt("page")
	opts.PageSize = ctx.FormInt("limit")
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 50 {
		opts.PageSize = 20
	}
	regs, count, err := hackforger_model.ListRegistrations(ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, regs)
}

// UpdateRegistration updates a registration's status (approve/reject).
//
// swagger:operation PUT /hackforger/hackathons/{id}/registrations/{rid} hackforger hackforgerUpdateRegistration
// ---
// summary: Update registration status (approve/reject)
// consumes:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: rid
//   in: path
//   description: ID of the registration
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/UpdateRegistrationForm"
// responses:
//   "200":
//     description: Registration updated
func UpdateRegistration(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*UpdateRegistrationForm)
	rid := ctx.ParamsInt64(":rid")
	status := hackforger_model.RegistrationStatus(f.Status)
	if err := hackforger_model.UpdateRegistrationStatus(ctx, rid, status); err != nil {
		ctx.InternalServerError(err)
		return
	}
	// Publish feed event if approved
	if status == hackforger_model.RegistrationStatusApproved {
		r, err := hackforger_model.GetRegistrationByID(ctx, rid)
		if err == nil {
			h, err := hackforger_model.GetHackathonByID(ctx, r.HackathonID)
			if err == nil {
				if doer, err := user_model.GetUserByID(ctx, r.UserID); err == nil {
					notify_service.HackforgerEntityCreated(ctx, doer, &notify_service.HackforgerEventOpts{
						OpType:       hackforger_model.ActionHackathonRegistered,
						EntityType:   "hackathon",
						EntityID:     h.ID,
						EntityName:   h.Name,
						EntitySlug:   h.Slug,
						AudienceType: notify_service.AudienceFollowers,
						Content: hackforger_model.HackforgerActionContent{
							EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
						},
					})
				}
			}
		}
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "updated"})
}
