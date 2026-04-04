// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"errors"
	"net/http"
	"strconv"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// --- Request / Response forms ---

// CreateGrantRoundForm represents the JSON body for creating a grant round.
type CreateGrantRoundForm struct {
	OrgID         int64  `json:"org_id" binding:"Required"`
	Name          string `json:"name" binding:"Required"`
	Slug          string `json:"slug" binding:"Required"`
	Description   string `json:"description"`
	Budget        float64 `json:"budget"`
	Currency      string `json:"currency"`
	BudgetCredits int64  `json:"budget_credits"`
	Deadline      int64  `json:"deadline"`
}

// UpdateGrantRoundForm represents the JSON body for updating a grant round.
type UpdateGrantRoundForm struct {
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	Description   string  `json:"description"`
	Budget        float64 `json:"budget"`
	Currency      string  `json:"currency"`
	BudgetCredits int64   `json:"budget_credits"`
	Deadline      int64   `json:"deadline"`
}

// SubmitProjectForm represents the JSON body for submitting a project.
type SubmitProjectForm struct {
	Title       string `json:"title" binding:"Required"`
	Description string `json:"description"`
	RepoID      int64  `json:"repo_id"`
}

// ApproveOrRejectForm represents the JSON body for approving or rejecting a project.
type ApproveOrRejectForm struct {
	Status string `json:"status" binding:"Required"`
}

// AllocateAwardForm represents the JSON body for allocating an award.
type AllocateAwardForm struct {
	Amount  float64 `json:"amount"`
	Credits int64   `json:"credits"`
}

// --- Helpers ---

// handleGrantError maps service/model errors to HTTP responses.
func handleGrantError(ctx *context.APIContext, err error) {
	if hackforger_model.IsErrGrantRoundNotExist(err) {
		ctx.NotFound()
		return
	}
	if hackforger_model.IsErrGrantProjectNotExist(err) {
		ctx.NotFound()
		return
	}
	if hackforger_service.IsErrAccessDenied(err) {
		ctx.Error(http.StatusForbidden, "AccessDenied", err)
		return
	}
	if hackforger_service.IsErrInvalidTransition(err) {
		ctx.Error(http.StatusConflict, "InvalidTransition", err)
		return
	}
	if hackforger_service.IsErrRoundNotFinalized(err) {
		ctx.Error(http.StatusConflict, "RoundNotFinalized", err)
		return
	}
	if hackforger_model.IsErrGrantRoundNotDraft(err) {
		ctx.Error(http.StatusConflict, "RoundNotDraft", err)
		return
	}
	if hackforger_model.IsErrGrantRoundNotOpen(err) {
		ctx.Error(http.StatusConflict, "RoundNotOpen", err)
		return
	}
	if hackforger_model.IsErrGrantProjectAlreadyExists(err) {
		ctx.Error(http.StatusConflict, "ProjectAlreadyExists", err)
		return
	}
	if hackforger_model.IsErrExceedsBudget(err) {
		ctx.Error(http.StatusUnprocessableEntity, "ExceedsBudget", err)
		return
	}
	if hackforger_model.IsErrUnallocatedProjects(err) {
		ctx.Error(http.StatusUnprocessableEntity, "UnallocatedProjects", err)
		return
	}
	if errors.Is(err, util.ErrInvalidArgument) {
		ctx.Error(http.StatusBadRequest, "InvalidArgument", err)
		return
	}
	ctx.Error(http.StatusInternalServerError, "InternalError", err)
}

// --- Grant Round Handlers ---

// CreateGrantRound creates a new grant round.
func CreateGrantRound(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/grant-rounds hackforger hackforgerCreateGrantRound
	// ---
	// summary: Create a grant round
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/CreateGrantRoundForm"
	// responses:
	//   "201":
	//     description: Grant round created
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*CreateGrantRoundForm)

	round, err := hackforger_service.CreateGrantRound(ctx, ctx.Doer.ID, form.OrgID, hackforger_service.CreateGrantRoundOpts{
		Name:          form.Name,
		Slug:          form.Slug,
		Description:   form.Description,
		Budget:        form.Budget,
		Currency:      form.Currency,
		BudgetCredits: form.BudgetCredits,
		Deadline:      timeutil.TimeStamp(form.Deadline),
	})
	if err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, round)
}

// ListGrantRounds returns a paginated list of grant rounds.
func ListGrantRounds(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/grant-rounds hackforger hackforgerListGrantRounds
	// ---
	// summary: List grant rounds
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// - name: org_id
	//   in: query
	//   description: filter by organization ID
	//   type: integer
	// - name: status
	//   in: query
	//   description: filter by status (0=draft, 1=open, 2=review, 3=finalized, 4=distributed, 5=cancelled)
	//   type: integer
	// responses:
	//   "200":
	//     description: Grant round list

	listOpts := utils.GetListOptions(ctx)
	opts := hackforger_model.ListGrantRoundsOptions{
		ListOptions: listOpts,
	}

	if orgID := ctx.FormInt64("org_id"); orgID > 0 {
		opts.OrgID = orgID
	}

	if statusStr := ctx.FormString("status"); statusStr != "" {
		statusInt, err := strconv.Atoi(statusStr)
		if err == nil {
			status := hackforger_model.GrantRoundStatus(statusInt)
			opts.Status = &status
		}
	}

	rounds, total, err := hackforger_model.ListGrantRounds(ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListGrantRounds", err)
		return
	}

	ctx.SetLinkHeader(int(total), listOpts.PageSize)
	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, rounds)
}

// GetGrantRound returns a single grant round by ID.
func GetGrantRound(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/grant-rounds/{id} hackforger hackforgerGetGrantRound
	// ---
	// summary: Get a grant round
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: Grant round
	//   "404":
	//     "$ref": "#/responses/notFound"

	round, err := hackforger_model.GetGrantRoundByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, round)
}

// UpdateGrantRound updates a grant round.
func UpdateGrantRound(ctx *context.APIContext) {
	// swagger:operation PUT /hackforger/grant-rounds/{id} hackforger hackforgerUpdateGrantRound
	// ---
	// summary: Update a grant round
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/UpdateGrantRoundForm"
	// responses:
	//   "200":
	//     description: Updated grant round
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*UpdateGrantRoundForm)
	id := ctx.ParamsInt64(":id")

	round, err := hackforger_model.GetGrantRoundByID(ctx, id)
	if err != nil {
		handleGrantError(ctx, err)
		return
	}

	// Apply updates
	if form.Name != "" {
		round.Name = form.Name
	}
	if form.Slug != "" {
		round.Slug = form.Slug
	}
	if form.Description != "" {
		round.Description = form.Description
	}
	if form.Budget != 0 {
		round.Budget = form.Budget
	}
	if form.Currency != "" {
		round.Currency = form.Currency
	}
	if form.BudgetCredits != 0 {
		round.BudgetCredits = form.BudgetCredits
	}
	if form.Deadline != 0 {
		round.Deadline = timeutil.TimeStamp(form.Deadline)
	}

	if err := hackforger_service.UpdateGrantRoundService(ctx, ctx.Doer.ID, round); err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, round)
}

// DeleteGrantRound deletes a draft grant round.
func DeleteGrantRound(ctx *context.APIContext) {
	// swagger:operation DELETE /hackforger/grant-rounds/{id} hackforger hackforgerDeleteGrantRound
	// ---
	// summary: Delete a grant round (draft only)
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     description: Grant round deleted
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     description: Round is not in draft status

	if err := hackforger_service.DeleteGrantRoundService(ctx, ctx.Doer.ID, ctx.ParamsInt64(":id")); err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// OpenRound transitions a grant round from Draft to Open.
func OpenRound(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/grant-rounds/{id}/open hackforger hackforgerOpenRound
	// ---
	// summary: Open a grant round
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     description: Round opened
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "409":
	//     description: Invalid state transition

	if err := hackforger_service.OpenRound(ctx, ctx.Doer.ID, ctx.ParamsInt64(":id")); err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// CloseRound transitions a grant round from Open to Review.
func CloseRound(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/grant-rounds/{id}/close hackforger hackforgerCloseRound
	// ---
	// summary: Close a grant round (move to review)
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     description: Round closed
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "409":
	//     description: Invalid state transition

	if err := hackforger_service.CloseRound(ctx, ctx.Doer.ID, ctx.ParamsInt64(":id")); err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// FinalizeRound transitions a grant round from Review to Finalized.
func FinalizeRound(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/grant-rounds/{id}/finalize hackforger hackforgerFinalizeRound
	// ---
	// summary: Finalize a grant round
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     description: Round finalized
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "409":
	//     description: Invalid state transition
	//   "422":
	//     description: Unallocated projects remain

	if err := hackforger_service.FinalizeRound(ctx, ctx.Doer.ID, ctx.ParamsInt64(":id")); err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// DistributeRound distributes awards for all approved projects in a finalized round.
func DistributeRound(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/grant-rounds/{id}/distribute hackforger hackforgerDistributeRound
	// ---
	// summary: Distribute all awards in a finalized round
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     description: Awards distributed
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "409":
	//     description: Round is not finalized

	if err := hackforger_service.DistributeRound(ctx, ctx.Doer.ID, ctx.ParamsInt64(":id")); err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// CancelRound transitions a grant round to Cancelled.
func CancelRound(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/grant-rounds/{id}/cancel hackforger hackforgerCancelRound
	// ---
	// summary: Cancel a grant round
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     description: Round cancelled
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "409":
	//     description: Invalid state transition

	if err := hackforger_service.CancelRound(ctx, ctx.Doer.ID, ctx.ParamsInt64(":id")); err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ExportRoundCSV exports all projects in a round as CSV.
func ExportRoundCSV(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/grant-rounds/{id}/export hackforger hackforgerExportRoundCSV
	// ---
	// summary: Export grant round projects as CSV
	// produces:
	// - text/csv
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: CSV file
	//   "404":
	//     "$ref": "#/responses/notFound"

	data, err := hackforger_service.ExportRoundCSV(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.Resp.Header().Set("Content-Type", "text/csv")
	ctx.Resp.Header().Set("Content-Disposition", "attachment; filename=grant-round-export.csv")
	ctx.Resp.WriteHeader(http.StatusOK)
	ctx.Resp.Write(data) //nolint:errcheck
}

// --- Grant Project Handlers ---

// SubmitProject submits a project to a grant round.
func SubmitProject(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/grant-rounds/{id}/projects hackforger hackforgerSubmitProject
	// ---
	// summary: Submit a project to a grant round
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/SubmitProjectForm"
	// responses:
	//   "201":
	//     description: Project submitted
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "409":
	//     description: Round not open or project already exists

	form := web.GetForm(ctx).(*SubmitProjectForm)

	project, err := hackforger_service.SubmitProject(ctx, ctx.Doer.ID, ctx.ParamsInt64(":id"), hackforger_service.SubmitProjectOpts{
		Title:       form.Title,
		Description: form.Description,
		RepoID:      form.RepoID,
	})
	if err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, project)
}

// ListGrantProjects returns a paginated list of projects in a grant round.
func ListGrantProjects(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/grant-rounds/{id}/projects hackforger hackforgerListGrantProjects
	// ---
	// summary: List projects in a grant round
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
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
	// - name: status
	//   in: query
	//   description: filter by status (0=pending, 1=approved, 2=rejected, 3=funded)
	//   type: integer
	// responses:
	//   "200":
	//     description: Grant project list

	listOpts := utils.GetListOptions(ctx)
	opts := hackforger_model.ListGrantProjectsByRoundOptions{
		ListOptions: listOpts,
		RoundID:     ctx.ParamsInt64(":id"),
	}

	if statusStr := ctx.FormString("status"); statusStr != "" {
		statusInt, err := strconv.Atoi(statusStr)
		if err == nil {
			status := hackforger_model.GrantProjectStatus(statusInt)
			opts.Status = &status
		}
	}

	projects, total, err := hackforger_model.ListGrantProjectsByRound(ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListGrantProjects", err)
		return
	}

	ctx.SetLinkHeader(int(total), listOpts.PageSize)
	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, projects)
}

// GetGrantProject returns a single grant project by ID.
func GetGrantProject(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/grant-rounds/{id}/projects/{pid} hackforger hackforgerGetGrantProject
	// ---
	// summary: Get a grant project
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// - name: pid
	//   in: path
	//   description: ID of the grant project
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: Grant project
	//   "404":
	//     "$ref": "#/responses/notFound"

	project, err := hackforger_model.GetGrantProjectByID(ctx, ctx.ParamsInt64(":pid"))
	if err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, project)
}

// ApproveOrRejectProject approves or rejects a grant project.
func ApproveOrRejectProject(ctx *context.APIContext) {
	// swagger:operation PUT /hackforger/grant-rounds/{id}/projects/{pid} hackforger hackforgerApproveOrRejectProject
	// ---
	// summary: Approve or reject a grant project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// - name: pid
	//   in: path
	//   description: ID of the grant project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/ApproveOrRejectForm"
	// responses:
	//   "204":
	//     description: Project status updated
	//   "400":
	//     description: Invalid status value
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*ApproveOrRejectForm)
	pid := ctx.ParamsInt64(":pid")

	switch form.Status {
	case "approved":
		if err := hackforger_service.ApproveProject(ctx, ctx.Doer.ID, pid); err != nil {
			handleGrantError(ctx, err)
			return
		}
	case "rejected":
		if err := hackforger_service.RejectProject(ctx, ctx.Doer.ID, pid); err != nil {
			handleGrantError(ctx, err)
			return
		}
	default:
		ctx.Error(http.StatusBadRequest, "InvalidStatus", errors.New("status must be 'approved' or 'rejected'"))
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AllocateAward sets the award amounts for a grant project.
func AllocateAward(ctx *context.APIContext) {
	// swagger:operation PUT /hackforger/grant-rounds/{id}/projects/{pid}/award hackforger hackforgerAllocateAward
	// ---
	// summary: Allocate an award to a grant project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// - name: pid
	//   in: path
	//   description: ID of the grant project
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/AllocateAwardForm"
	// responses:
	//   "204":
	//     description: Award allocated
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     description: Exceeds budget

	form := web.GetForm(ctx).(*AllocateAwardForm)

	if err := hackforger_service.AllocateAward(ctx, ctx.Doer.ID, ctx.ParamsInt64(":pid"), form.Amount, form.Credits); err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// DistributeProject distributes the award for a single grant project.
func DistributeProject(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/grant-rounds/{id}/projects/{pid}/distribute hackforger hackforgerDistributeProject
	// ---
	// summary: Distribute award for a single project
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the grant round
	//   type: integer
	//   format: int64
	//   required: true
	// - name: pid
	//   in: path
	//   description: ID of the grant project
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     description: Project award distributed
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "409":
	//     description: Round not finalized or project not approved

	if err := hackforger_service.DistributeProject(ctx, ctx.Doer.ID, ctx.ParamsInt64(":pid")); err != nil {
		handleGrantError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
