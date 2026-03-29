// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"
	"strconv"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// --- Form structs for bind() middleware ---

// CreateHackathonForm is the form for creating a hackathon.
type CreateHackathonForm struct {
	Name        string `json:"name" binding:"Required"`
	Slug        string `json:"slug" binding:"Required"`
	Description string `json:"description"`
	OrgID       int64  `json:"org_id" binding:"Required"`
	MaxTeamSize int    `json:"max_team_size"`
}

// UpdateHackathonForm is the form for updating a hackathon.
type UpdateHackathonForm struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	MaxTeamSize  *int    `json:"max_team_size"`
	PrizeSummary *string `json:"prize_summary"`
}

// --- Handlers ---

// ListHackathons returns a paginated list of hackathons.
func ListHackathons(ctx *context.APIContext) {
	opts := hackforger_model.ListHackathonsOptions{
		Keyword: ctx.FormString("q"),
	}
	opts.Page = ctx.FormInt("page")
	opts.PageSize = ctx.FormInt("limit")
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 50 {
		opts.PageSize = 20
	}
	if orgID := ctx.FormInt64("org_id"); orgID > 0 {
		opts.OrgID = orgID
	}
	if statusStr := ctx.FormString("status"); statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			status := hackforger_model.HackathonStatus(s)
			opts.Status = &status
		}
	}
	hackathons, count, err := hackforger_model.ListHackathons(ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, hackathons)
}

// GetHackathon returns a single hackathon by ID.
func GetHackathon(ctx *context.APIContext) {
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, h)
}

// CreateHackathon creates a new hackathon.
func CreateHackathon(ctx *context.APIContext) {
	form := web.GetForm(ctx).(*CreateHackathonForm)
	h := &hackforger_model.Hackathon{
		OrgID:       form.OrgID,
		OwnerID:     ctx.Doer.ID,
		Name:        form.Name,
		Slug:        form.Slug,
		Description: form.Description,
		MaxTeamSize: form.MaxTeamSize,
	}
	if h.MaxTeamSize <= 0 {
		h.MaxTeamSize = 5
	}
	if err := hackforger_service.CreateHackathon(ctx, ctx.Doer, h); err != nil {
		if hackforger_model.IsErrHackathonSlugAlreadyExist(err) {
			ctx.Error(http.StatusConflict, "SlugExists", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusCreated, h)
}

// UpdateHackathon updates fields of an existing hackathon.
func UpdateHackathon(ctx *context.APIContext) {
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	form := web.GetForm(ctx).(*UpdateHackathonForm)
	if form.Name != nil {
		h.Name = *form.Name
	}
	if form.Description != nil {
		h.Description = *form.Description
	}
	if form.MaxTeamSize != nil {
		h.MaxTeamSize = *form.MaxTeamSize
	}
	if form.PrizeSummary != nil {
		h.PrizeSummary = *form.PrizeSummary
	}
	if err := hackforger_model.UpdateHackathon(ctx, h); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, h)
}

// DeleteHackathon deletes a hackathon (only allowed in Draft status).
func DeleteHackathon(ctx *context.APIContext) {
	if err := hackforger_model.DeleteHackathon(ctx, ctx.ParamsInt64(":id")); err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.Error(http.StatusForbidden, "DeleteHackathon", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// getHackathonFromPath is a helper that loads a hackathon from the :id path param.
// It writes the error response itself and returns nil on failure.
func getHackathonFromPath(ctx *context.APIContext) *hackforger_model.Hackathon {
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.InternalServerError(err)
		}
		return nil
	}
	return h
}

// PublishHackathon transitions a hackathon from Draft to Open.
func PublishHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.PublishHackathon(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "PublishHackathon", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "open"})
}

// StartHackathon transitions a hackathon from Open to Hacking.
func StartHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.StartHacking(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "StartHacking", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "hacking"})
}

// StartJudgingHackathon transitions a hackathon from Hacking to Judging.
func StartJudgingHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.StartJudging(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "StartJudging", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "judging"})
}

// FinalizeHackathon transitions a hackathon from Judging to Finished and computes ranks.
func FinalizeHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.FinalizeHackathon(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "FinalizeHackathon", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "finished"})
}

// CancelHackathon cancels a hackathon (allowed in any non-Finished status).
func CancelHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.CancelHackathon(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "CancelHackathon", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "cancelled"})
}
