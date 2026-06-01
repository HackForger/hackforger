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

// --- Form structs for bind() middleware ---

// AddPhaseForm is the form for adding a phase to a hackathon.
type AddPhaseForm struct {
	PhaseTypeID int64 `json:"phase_type_id" binding:"Required"`
	StartTime   int64 `json:"start_time" binding:"Required"`
	EndTime     int64 `json:"end_time" binding:"Required"`
	SortOrder   int   `json:"sort_order"`
}

// UpdatePhaseForm is the form for updating an existing phase.
// CustomName is a pointer so callers can omit it (leave the name untouched on a
// time-only edit) vs. send "" to clear it.
type UpdatePhaseForm struct {
	StartTime  int64   `json:"start_time" binding:"Required"`
	EndTime    int64   `json:"end_time" binding:"Required"`
	CustomName *string `json:"custom_name"`
}

// ReorderPhasesForm is the form for batch-updating phase sort orders.
type ReorderPhasesForm struct {
	Orders []hackforger_model.PhaseSortOrder `json:"orders" binding:"Required"`
}

// --- Handlers ---

// ListPhases returns all phases for a hackathon.
//
// swagger:operation GET /hackforger/hackathons/{id}/phases hackforger hackforgerListPhases
// ---
// summary: List phases for a hackathon
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
//     description: Phase list
//   "404":
//     "$ref": "#/responses/notFound"
func ListPhases(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	phases, err := hackforger_service.GetPhases(ctx, "hackathon", h.ID)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, phases)
}

// AddPhaseAPI creates a new phase for a hackathon.
//
// swagger:operation POST /hackforger/hackathons/{id}/phases hackforger hackforgerAddPhase
// ---
// summary: Add a phase to a hackathon
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
//     "$ref": "#/definitions/AddPhaseForm"
// responses:
//   "201":
//     description: Phase created
//   "400":
//     description: Validation error (overlap, unique violation, etc.)
//   "404":
//     "$ref": "#/responses/notFound"
func AddPhaseAPI(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	form := web.GetForm(ctx).(*AddPhaseForm)
	phase, err := hackforger_service.AddPhase(ctx, "hackathon", h.ID, form.PhaseTypeID, form.SortOrder, form.StartTime, form.EndTime)
	if err != nil {
		if hackforger_service.IsErrPhaseOverlap(err) || hackforger_service.IsErrPhaseUniqueViolation(err) {
			ctx.Error(http.StatusBadRequest, "AddPhase", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}
	if syncErr := hackforger_service.SyncStatusCache(ctx, h.ID, ctx.Doer.ID); syncErr != nil {
		ctx.InternalServerError(syncErr)
		return
	}
	ctx.JSON(http.StatusCreated, phase)
}

// UpdatePhaseAPI updates the time window and custom name of a phase.
//
// swagger:operation PUT /hackforger/hackathons/{id}/phases/{phase_id} hackforger hackforgerUpdatePhase
// ---
// summary: Update a phase
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
// - name: phase_id
//   in: path
//   description: ID of the phase
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/UpdatePhaseForm"
// responses:
//   "200":
//     description: Updated phase
//   "400":
//     description: Validation error (locked, overlap, etc.)
//   "404":
//     "$ref": "#/responses/notFound"
func UpdatePhaseAPI(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	phaseID := ctx.ParamsInt64(":phase_id")
	form := web.GetForm(ctx).(*UpdatePhaseForm)

	phase, err := hackforger_model.GetPhaseByID(ctx, phaseID)
	if err != nil || phase == nil {
		ctx.NotFound()
		return
	}

	// The time window is guarded (locked/active phases reject changes). Only run
	// the guarded path when the times actually change, so a pure rename (incl.
	// an ended phase) isn't rejected and a failed time update never leaves a
	// half-applied custom_name behind.
	if form.StartTime != phase.StartTime || form.EndTime != phase.EndTime {
		if err := hackforger_service.UpdatePhaseTime(ctx, phaseID, form.StartTime, form.EndTime); err != nil {
			if hackforger_service.IsErrPhaseLocked(err) || hackforger_service.IsErrActivePhaseStartLocked(err) || hackforger_service.IsErrPhaseOverlap(err) {
				ctx.Error(http.StatusBadRequest, "UpdatePhase", err)
				return
			}
			ctx.InternalServerError(err)
			return
		}
	}

	// The custom name is independent of the time window and can be changed on
	// any phase. nil = not provided (leave untouched); non-nil sets it, "" clears.
	if form.CustomName != nil && *form.CustomName != phase.CustomName {
		if err := hackforger_model.UpdatePhaseCustomName(ctx, phaseID, *form.CustomName); err != nil {
			ctx.InternalServerError(err)
			return
		}
	}

	if syncErr := hackforger_service.SyncStatusCache(ctx, h.ID, ctx.Doer.ID); syncErr != nil {
		ctx.InternalServerError(syncErr)
		return
	}

	// Return the updated phase
	phase, err = hackforger_model.GetPhaseByID(ctx, phaseID)
	if err != nil || phase == nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, phase)
}

// DeletePhaseAPI removes a future phase from a hackathon.
//
// swagger:operation DELETE /hackforger/hackathons/{id}/phases/{phase_id} hackforger hackforgerDeletePhase
// ---
// summary: Delete a phase (future only)
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: phase_id
//   in: path
//   description: ID of the phase
//   type: integer
//   format: int64
//   required: true
// responses:
//   "204":
//     description: Phase deleted
//   "400":
//     description: Phase is not in the future
//   "404":
//     "$ref": "#/responses/notFound"
func DeletePhaseAPI(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	phaseID := ctx.ParamsInt64(":phase_id")
	if err := hackforger_service.RemovePhase(ctx, phaseID); err != nil {
		if hackforger_service.IsErrPhaseNotFuture(err) {
			ctx.Error(http.StatusBadRequest, "DeletePhase", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}
	if syncErr := hackforger_service.SyncStatusCache(ctx, h.ID, ctx.Doer.ID); syncErr != nil {
		ctx.InternalServerError(syncErr)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// GetCurrentPhaseAPI returns the currently active phase for a hackathon, or null.
//
// swagger:operation GET /hackforger/hackathons/{id}/current-phase hackforger hackforgerGetCurrentPhase
// ---
// summary: Get the current active phase
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
//     description: Current phase (or null if none active)
//   "404":
//     "$ref": "#/responses/notFound"
func GetCurrentPhaseAPI(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	phase, err := hackforger_service.CurrentPhase(ctx, "hackathon", h.ID)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, phase)
}

// ReorderPhasesAPI batch-updates the sort_order of phases for a hackathon.
//
// swagger:operation POST /hackforger/hackathons/{id}/phases/reorder hackforger hackforgerReorderPhases
// ---
// summary: Reorder phases
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
//     "$ref": "#/definitions/ReorderPhasesForm"
// responses:
//   "200":
//     description: Phases reordered
//   "404":
//     "$ref": "#/responses/notFound"
func ReorderPhasesAPI(ctx *context.APIContext) {
	h := getHackathonFromPath(ctx)
	if h == nil {
		return
	}
	form := web.GetForm(ctx).(*ReorderPhasesForm)
	if err := hackforger_model.BatchUpdatePhaseSortOrder(ctx, form.Orders); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
