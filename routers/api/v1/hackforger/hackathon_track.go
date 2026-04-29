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

// CreateTrackForm is the form for creating a hackathon track.
type CreateTrackForm struct {
	Name            string  `json:"name" binding:"Required"`
	Description     string  `json:"description"`
	PrizeAmount     float64 `json:"prize_amount"`
	PrizeCurrency   string  `json:"prize_currency"`
	PrizeCredits    int64   `json:"prize_credits"`
	PrizeDistMode   string  `json:"prize_dist_mode"`
	PrizeDistRatios string  `json:"prize_dist_ratios"`
}

// UpdateTrackForm is the form for updating a hackathon track.
type UpdateTrackForm struct {
	Name            *string  `json:"name"`
	Description     *string  `json:"description"`
	PrizeAmount     *float64 `json:"prize_amount"`
	PrizeCurrency   *string  `json:"prize_currency"`
	PrizeCredits    *int64   `json:"prize_credits"`
	PrizeDistMode   *string  `json:"prize_dist_mode"`
	PrizeDistRatios *string  `json:"prize_dist_ratios"`
}

// ListTracks returns all tracks for a hackathon.
//
// swagger:operation GET /hackforger/hackathons/{id}/tracks hackforger hackforgerListTracks
// ---
// summary: List tracks for a hackathon
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
//     description: Track list
func ListTracks(ctx *context.APIContext) {
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, tracks)
}

// CreateTrack adds a new track to a hackathon, creating a repo in the linked org.
//
// swagger:operation POST /hackforger/hackathons/{id}/tracks hackforger hackforgerCreateTrack
// ---
// summary: Create a track for a hackathon
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
//     "$ref": "#/definitions/CreateTrackForm"
// responses:
//   "201":
//     description: Track created
//   "404":
//     "$ref": "#/responses/notFound"
func CreateTrack(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*CreateTrackForm)
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.InternalServerError(err)
		}
		return
	}

	// API parity with web: empty mode defaults to winner_takes_all.
	mode := f.PrizeDistMode
	if mode == "" {
		mode = "winner_takes_all"
	}
	if err := hackforger_model.ValidatePrizeDistConfig(mode, f.PrizeDistRatios); err != nil {
		if hackforger_model.IsErrInvalidPrizeDistMode(err) || hackforger_model.IsErrInvalidDistRatios(err) {
			ctx.Error(http.StatusBadRequest, "ValidatePrizeDistConfig", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}

	t := &hackforger_model.HackathonTrack{
		HackathonID:     h.ID,
		Name:            f.Name,
		Description:     f.Description,
		PrizeAmount:     f.PrizeAmount,
		PrizeCurrency:   f.PrizeCurrency,
		PrizeCredits:    f.PrizeCredits,
		PrizeDistMode:   mode,
		PrizeDistRatios: f.PrizeDistRatios,
	}
	if err := hackforger_service.CreateTrackWithRepo(ctx, ctx.Doer, h, t); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusCreated, t)
}

// UpdateTrack updates an existing hackathon track.
//
// swagger:operation PUT /hackforger/hackathons/{id}/tracks/{tid} hackforger hackforgerUpdateTrack
// ---
// summary: Update a hackathon track
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
// - name: tid
//   in: path
//   description: ID of the track
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/UpdateTrackForm"
// responses:
//   "200":
//     description: Updated track
//   "404":
//     "$ref": "#/responses/notFound"
func UpdateTrack(ctx *context.APIContext) {
	t, err := hackforger_model.GetTrackByID(ctx, ctx.ParamsInt64(":tid"))
	if err != nil {
		if hackforger_model.IsErrTrackNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	f := web.GetForm(ctx).(*UpdateTrackForm)
	if f.Name != nil {
		t.Name = *f.Name
	}
	if f.Description != nil {
		t.Description = *f.Description
	}
	if f.PrizeAmount != nil {
		t.PrizeAmount = *f.PrizeAmount
	}
	if f.PrizeCurrency != nil {
		t.PrizeCurrency = *f.PrizeCurrency
	}
	if f.PrizeCredits != nil {
		t.PrizeCredits = *f.PrizeCredits
	}
	if f.PrizeDistMode != nil {
		mode := *f.PrizeDistMode
		if mode == "" {
			mode = "winner_takes_all"
		}
		t.PrizeDistMode = mode
	}
	if f.PrizeDistRatios != nil {
		t.PrizeDistRatios = *f.PrizeDistRatios
	}

	// Validate the resulting state (after merging request into stored).
	if err := hackforger_model.ValidatePrizeDistConfig(t.PrizeDistMode, t.PrizeDistRatios); err != nil {
		if hackforger_model.IsErrInvalidPrizeDistMode(err) || hackforger_model.IsErrInvalidDistRatios(err) {
			ctx.Error(http.StatusBadRequest, "ValidatePrizeDistConfig", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}

	if err := hackforger_model.UpdateTrack(ctx, t); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, t)
}

// DeleteTrack removes a track from a hackathon.
//
// swagger:operation DELETE /hackforger/hackathons/{id}/tracks/{tid} hackforger hackforgerDeleteTrack
// ---
// summary: Delete a hackathon track
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: tid
//   in: path
//   description: ID of the track
//   type: integer
//   format: int64
//   required: true
// responses:
//   "204":
//     description: Track deleted
func DeleteTrack(ctx *context.APIContext) {
	if err := hackforger_model.DeleteTrack(ctx, ctx.ParamsInt64(":tid")); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
