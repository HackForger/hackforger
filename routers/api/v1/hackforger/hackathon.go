// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"
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
//
// swagger:operation GET /hackforger/hackathons hackforger hackforgerListHackathons
// ---
// summary: List hackathons
// produces:
// - application/json
// parameters:
// - name: q
//   in: query
//   description: search keyword
//   type: string
// - name: org_id
//   in: query
//   description: filter by organization ID
//   type: integer
//   format: int64
// - name: status
//   in: query
//   description: filter by status (0=draft, 1=open, 2=hacking, 3=judging, 4=finished, 5=cancelled)
//   type: integer
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
//     description: Hackathon list
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
//
// swagger:operation GET /hackforger/hackathons/{id} hackforger hackforgerGetHackathon
// ---
// summary: Get a hackathon
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Hackathon details
//   "404":
//     "$ref": "#/responses/notFound"
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
//
// swagger:operation POST /hackforger/hackathons hackforger hackforgerCreateHackathon
// ---
// summary: Create a hackathon
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/CreateHackathonForm"
// responses:
//   "201":
//     description: Hackathon created
//   "409":
//     description: Slug already exists
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
//
// swagger:operation PUT /hackforger/hackathons/{id} hackforger hackforgerUpdateHackathon
// ---
// summary: Update a hackathon
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
//     "$ref": "#/definitions/UpdateHackathonForm"
// responses:
//   "200":
//     description: Updated hackathon
//   "404":
//     "$ref": "#/responses/notFound"
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
//
// swagger:operation DELETE /hackforger/hackathons/{id} hackforger hackforgerDeleteHackathon
// ---
// summary: Delete a hackathon (draft only)
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "204":
//     description: Hackathon deleted
//   "403":
//     description: Not allowed (not in draft status)
//   "404":
//     "$ref": "#/responses/notFound"
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
//
// swagger:operation POST /hackforger/hackathons/{id}/publish hackforger hackforgerPublishHackathon
// ---
// summary: Publish a hackathon (draft to open)
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Hackathon published
//   "400":
//     description: Invalid state transition
//   "404":
//     "$ref": "#/responses/notFound"
func PublishHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	err := hackforger_service.PublishHackathon(ctx, ctx.Doer.ID, h)
	if err == nil {
		ctx.JSON(http.StatusOK, map[string]string{"status": "open"})
		return
	}
	switch {
	case hackforger_service.IsErrNoTracks(err):
		ctx.Error(http.StatusBadRequest, "PublishHackathon",
			ctx.Tr("hackforger.hackathon.error.no_tracks"))
	case hackforger_service.IsErrNoCriteria(err):
		typed := err.(hackforger_service.ErrNoCriteria)
		if typed.TrackID == 0 {
			ctx.Error(http.StatusBadRequest, "PublishHackathon",
				ctx.Tr("hackforger.hackathon.error.publish_requires_criteria"))
		} else {
			trackName := fmt.Sprintf("#%d", typed.TrackID)
			if t, terr := hackforger_model.GetTrackByID(ctx, typed.TrackID); terr == nil && t != nil {
				trackName = t.Name
			}
			ctx.Error(http.StatusBadRequest, "PublishHackathon",
				ctx.Tr("hackforger.hackathon.error.track_requires_criteria", trackName))
		}
	case hackforger_service.IsErrNoRegistrationPhase(err):
		ctx.Error(http.StatusBadRequest, "PublishHackathon",
			ctx.Tr("hackforger.hackathon.error.no_registration_phase"))
	case hackforger_service.IsErrNoDevelopmentPhase(err):
		ctx.Error(http.StatusBadRequest, "PublishHackathon",
			ctx.Tr("hackforger.hackathon.error.no_development_phase"))
	default:
		ctx.Error(http.StatusBadRequest, "PublishHackathon", err)
	}
}

// StartHackathon transitions a hackathon from Open to Hacking.
//
// FinalizeHackathon transitions a hackathon from Judging to Finished and computes ranks.
//
// swagger:operation POST /hackforger/hackathons/{id}/finalize hackforger hackforgerFinalizeHackathon
// ---
// summary: Finalize hackathon (judging to finished)
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Hackathon finalized
//   "400":
//     description: Invalid state transition
//   "404":
//     "$ref": "#/responses/notFound"
func FinalizeHackathon(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	if err := hackforger_service.ConfirmFinalize(ctx, ctx.Doer.ID, h); err != nil {
		ctx.Error(http.StatusBadRequest, "FinalizeHackathon", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "finished"})
}

// CancelHackathon cancels a hackathon (allowed in any non-Finished status).
//
// swagger:operation POST /hackforger/hackathons/{id}/cancel hackforger hackforgerCancelHackathon
// ---
// summary: Cancel a hackathon
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Hackathon cancelled
//   "400":
//     description: Invalid state transition
//   "404":
//     "$ref": "#/responses/notFound"
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
