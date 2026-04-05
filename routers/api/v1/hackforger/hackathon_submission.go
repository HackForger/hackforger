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
//
// swagger:operation POST /hackforger/hackathons/{id}/submissions hackforger hackforgerCreateSubmission
// ---
// summary: Submit a project to a hackathon
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
//     "$ref": "#/definitions/CreateSubmissionForm"
// responses:
//   "201":
//     description: Submission created
//   "400":
//     description: Hackathon is not accepting submissions
//   "403":
//     description: User is not registered
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
	canSubmit, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "submit_work")
	if !canSubmit {
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
//
// swagger:operation GET /hackforger/hackathons/{id}/submissions hackforger hackforgerListSubmissions
// ---
// summary: List submissions for a hackathon
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
//     description: Submission list
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
//
// swagger:operation GET /hackforger/hackathons/{id}/submissions/{sid} hackforger hackforgerGetSubmission
// ---
// summary: Get a submission
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: sid
//   in: path
//   description: ID of the submission
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Submission details
//   "404":
//     "$ref": "#/responses/notFound"
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
//
// swagger:operation DELETE /hackforger/hackathons/{id}/submissions/{sid} hackforger hackforgerDeleteSubmission
// ---
// summary: Delete a submission
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: sid
//   in: path
//   description: ID of the submission
//   type: integer
//   format: int64
//   required: true
// responses:
//   "204":
//     description: Submission deleted
//   "403":
//     description: Only submitter or hackathon owner can delete
//   "404":
//     "$ref": "#/responses/notFound"
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
//
// swagger:operation PUT /hackforger/hackathons/{id}/submissions/{sid} hackforger hackforgerUpdateSubmission
// ---
// summary: Update a submission
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
// - name: sid
//   in: path
//   description: ID of the submission
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/UpdateSubmissionForm"
// responses:
//   "200":
//     description: Updated submission
//   "404":
//     "$ref": "#/responses/notFound"
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
