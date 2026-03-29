// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
)

// CreateTrackForm is the form for creating a hackathon track.
type CreateTrackForm struct {
	Name          string  `json:"name" binding:"Required"`
	Description   string  `json:"description"`
	PrizeAmount   float64 `json:"prize_amount"`
	PrizeCurrency string  `json:"prize_currency"`
	PrizeCredits  int64   `json:"prize_credits"`
}

// UpdateTrackForm is the form for updating a hackathon track.
type UpdateTrackForm struct {
	Name          *string  `json:"name"`
	Description   *string  `json:"description"`
	PrizeAmount   *float64 `json:"prize_amount"`
	PrizeCurrency *string  `json:"prize_currency"`
	PrizeCredits  *int64   `json:"prize_credits"`
}

// ListTracks returns all tracks for a hackathon.
func ListTracks(ctx *context.APIContext) {
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, tracks)
}

// CreateTrack adds a new track to a hackathon.
func CreateTrack(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*CreateTrackForm)
	t := &hackforger_model.HackathonTrack{
		HackathonID:   ctx.ParamsInt64(":id"),
		Name:          f.Name,
		Description:   f.Description,
		PrizeAmount:   f.PrizeAmount,
		PrizeCurrency: f.PrizeCurrency,
		PrizeCredits:  f.PrizeCredits,
	}
	if err := hackforger_model.CreateTrack(ctx, t); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusCreated, t)
}

// UpdateTrack updates an existing hackathon track.
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
	if err := hackforger_model.UpdateTrack(ctx, t); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, t)
}

// DeleteTrack removes a track from a hackathon.
func DeleteTrack(ctx *context.APIContext) {
	if err := hackforger_model.DeleteTrack(ctx, ctx.ParamsInt64(":tid")); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
