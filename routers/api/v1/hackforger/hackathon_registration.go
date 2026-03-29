// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
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
	if h.Status != hackforger_model.HackathonStatusOpen {
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
	ctx.JSON(http.StatusCreated, r)
}

// ListRegistrations returns a paginated list of registrations for a hackathon.
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
				_ = hackforger_service.PublishHackforgerAction(ctx, &hackforger_service.HackforgerActionOpts{
					ActUserID:    r.UserID,
					OpType:       hackforger_model.ActionHackathonRegistered,
					AudienceType: hackforger_service.AudienceFollowers,
					Content: hackforger_model.HackforgerActionContent{
						EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
					},
				})
			}
		}
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "updated"})
}
