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

// CreateSubmissionForm is the form for creating a hackathon submission.
type CreateSubmissionForm struct {
	Title       string `json:"title" binding:"Required"`
	Description string `json:"description"`
	DemoURL     string `json:"demo_url"`
	TrackID     int64  `json:"track_id"`
	RepoID      int64  `json:"repo_id"`
}

// UpdateSubmissionForm is the form for updating a hackathon submission.
type UpdateSubmissionForm struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	DemoURL     *string `json:"demo_url"`
}

// CreateSubmission creates a new project submission for a hackathon.
func CreateSubmission(ctx *context.APIContext) {
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	if h.Status != hackforger_model.HackathonStatusHacking {
		ctx.Error(http.StatusBadRequest, "InvalidPhase", "hackathon is not accepting submissions")
		return
	}
	reg, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusForbidden, "NotRegistered", "user is not registered for this hackathon")
		return
	}
	f := web.GetForm(ctx).(*CreateSubmissionForm)
	s := &hackforger_model.HackathonSubmission{
		HackathonID:    h.ID,
		RegistrationID: reg.ID,
		UserID:         ctx.Doer.ID,
		Title:          f.Title,
		Description:    f.Description,
		DemoURL:        f.DemoURL,
		TrackID:        f.TrackID,
		RepoID:         f.RepoID,
		Status:         hackforger_model.SubmissionStatusSubmitted,
	}
	if err := hackforger_service.CreateSubmission(ctx, ctx.Doer, h, s); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusCreated, s)
}

// ListSubmissions returns a paginated list of submissions for a hackathon.
func ListSubmissions(ctx *context.APIContext) {
	opts := hackforger_model.ListSubmissionsOptions{HackathonID: ctx.ParamsInt64(":id")}
	opts.Page = ctx.FormInt("page")
	opts.PageSize = ctx.FormInt("limit")
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 50 {
		opts.PageSize = 20
	}
	subs, count, err := hackforger_model.ListSubmissions(ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, subs)
}

// GetSubmission returns a single submission by ID.
func GetSubmission(ctx *context.APIContext) {
	s, err := hackforger_model.GetSubmissionByID(ctx, ctx.ParamsInt64(":sid"))
	if err != nil {
		if hackforger_model.IsErrSubmissionNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, s)
}

// DeleteSubmission removes a submission from a hackathon.
func DeleteSubmission(ctx *context.APIContext) {
	sid := ctx.ParamsInt64(":sid")
	sub, err := hackforger_model.GetSubmissionByID(ctx, sid)
	if err != nil {
		if hackforger_model.IsErrSubmissionNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.InternalServerError(err)
		}
		return
	}
	// Only the submitter or hackathon owner can delete
	h, err := hackforger_model.GetHackathonByID(ctx, sub.HackathonID)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	if ctx.Doer.ID != sub.UserID && ctx.Doer.ID != h.OwnerID {
		ctx.Error(http.StatusForbidden, "Forbidden", "only the submitter or hackathon owner can delete a submission")
		return
	}
	if err := hackforger_service.DeleteSubmission(ctx, ctx.Doer, sid); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// UpdateSubmission updates title, description, or demo URL of a submission.
func UpdateSubmission(ctx *context.APIContext) {
	s, err := hackforger_model.GetSubmissionByID(ctx, ctx.ParamsInt64(":sid"))
	if err != nil {
		if hackforger_model.IsErrSubmissionNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	f := web.GetForm(ctx).(*UpdateSubmissionForm)
	if f.Title != nil {
		s.Title = *f.Title
	}
	if f.Description != nil {
		s.Description = *f.Description
	}
	if f.DemoURL != nil {
		s.DemoURL = *f.DemoURL
	}
	if err := hackforger_model.UpdateSubmission(ctx, s); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, s)
}
