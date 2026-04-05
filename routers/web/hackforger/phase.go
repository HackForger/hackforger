// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

type phaseTypeJSON struct {
	ID              int64  `json:"id"`
	Key             string `json:"key"`
	DisplayName     string `json:"display_name"`
	IsUnique        bool   `json:"is_unique"`
	DefaultOrder    int    `json:"default_order"`
	AlreadyAddedTip string `json:"already_added_tip"`
}

// ManagePhases returns all phases + phase type catalog as JSON.
func ManagePhases(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}

	phases, err := hackforger_service.GetPhases(ctx, "hackathon", h.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	phaseTypes, err := hackforger_model.GetPhaseTypesByActivityKind(ctx, "hackathon")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Resolve i18n keys to translated strings for Vue component
	alreadyAddedTip := ctx.Locale.TrString("hackforger.phase.already_added")
	ptList := make([]phaseTypeJSON, len(phaseTypes))
	for i, pt := range phaseTypes {
		ptList[i] = phaseTypeJSON{
			ID:              pt.ID,
			Key:             pt.Key,
			DisplayName:     ctx.Locale.TrString(pt.DisplayNameI18n),
			IsUnique:        pt.IsUnique,
			DefaultOrder:    pt.DefaultOrder,
			AlreadyAddedTip: alreadyAddedTip,
		}
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"phases":       phases,
		"phase_types":  ptList,
		"status_cache": h.StatusCache,
	})
}

// ManagePhasesAdd adds a new phase.
func ManagePhasesAdd(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}

	var req struct {
		PhaseTypeID int64 `json:"phase_type_id"`
		StartTime   int64 `json:"start_time"`
		EndTime     int64 `json:"end_time"`
		SortOrder   int   `json:"sort_order"`
	}
	if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	_, err := hackforger_service.AddPhase(ctx, "hackathon", h.ID, req.PhaseTypeID, req.SortOrder, req.StartTime, req.EndTime)
	if err != nil {
		handlePhaseError(ctx, err)
		return
	}

	if syncErr := hackforger_service.SyncStatusCache(ctx, h.ID, ctx.Doer.ID); syncErr != nil {
		log.Error("SyncStatusCache: %v", syncErr)
	}

	respondWithPhases(ctx, h.ID)
}

// ManagePhasesUpdate updates a phase's time window.
func ManagePhasesUpdate(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	phaseID := ctx.ParamsInt64("phase_id")

	var req struct {
		StartTime  int64  `json:"start_time"`
		EndTime    int64  `json:"end_time"`
		CustomName string `json:"custom_name"`
	}
	if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	// Update custom_name if provided (can be empty string to clear)
	if phase, _ := hackforger_model.GetPhaseByID(ctx, phaseID); phase != nil {
		phase.CustomName = req.CustomName
		_ = hackforger_model.UpdatePhase(ctx, phase)
	}

	if err := hackforger_service.UpdatePhaseTime(ctx, phaseID, req.StartTime, req.EndTime); err != nil {
		handlePhaseError(ctx, err)
		return
	}

	if syncErr := hackforger_service.SyncStatusCache(ctx, h.ID, ctx.Doer.ID); syncErr != nil {
		log.Error("SyncStatusCache: %v", syncErr)
	}

	respondWithPhases(ctx, h.ID)
}

// ManagePhasesDelete deletes a future phase.
func ManagePhasesDelete(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	phaseID := ctx.ParamsInt64("phase_id")

	if err := hackforger_service.RemovePhase(ctx, phaseID); err != nil {
		handlePhaseError(ctx, err)
		return
	}

	if syncErr := hackforger_service.SyncStatusCache(ctx, h.ID, ctx.Doer.ID); syncErr != nil {
		log.Error("SyncStatusCache: %v", syncErr)
	}

	respondWithPhases(ctx, h.ID)
}

// ManagePhasesReorder reorders phases by updating sort_order.
func ManagePhasesReorder(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	var req struct {
		Orders []hackforger_model.PhaseSortOrder `json:"orders"`
	}
	if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := hackforger_model.BatchUpdatePhaseSortOrder(ctx, req.Orders); err != nil {
		log.Error("BatchUpdatePhaseSortOrder: %v", err)
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	respondWithPhases(ctx, h.ID)
}

// respondWithPhases returns the updated phases list + status_cache.
func respondWithPhases(ctx *context.Context, hackathonID int64) {
	phases, _ := hackforger_service.GetPhases(ctx, "hackathon", hackathonID)
	h, _ := hackforger_model.GetHackathonByID(ctx, hackathonID)
	ctx.JSON(http.StatusOK, map[string]any{
		"phases":       phases,
		"status_cache": h.StatusCache,
	})
}

// handlePhaseError maps typed phase errors to JSON responses.
func handlePhaseError(ctx *context.Context, err error) {
	status := http.StatusInternalServerError
	msg := "internal error"

	switch {
	case hackforger_service.IsErrPhaseOverlap(err):
		status = http.StatusConflict
		msg = ctx.Locale.TrString("hackforger.phase.error.overlap")
	case hackforger_service.IsErrPhaseLocked(err):
		status = http.StatusForbidden
		msg = ctx.Locale.TrString("hackforger.phase.error.locked")
	case hackforger_service.IsErrPhaseNotFuture(err):
		status = http.StatusForbidden
		msg = ctx.Locale.TrString("hackforger.phase.error.not_future")
	case hackforger_service.IsErrPhaseUniqueViolation(err):
		status = http.StatusConflict
		msg = ctx.Locale.TrString("hackforger.phase.error.unique_violation")
	case hackforger_service.IsErrActivePhaseStartLocked(err):
		status = http.StatusForbidden
		msg = ctx.Locale.TrString("hackforger.phase.error.active_start_locked")
	default:
		log.Error("Phase operation error: %v", err)
	}

	ctx.JSON(status, map[string]string{"error": msg})
}
