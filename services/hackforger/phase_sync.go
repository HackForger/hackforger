// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/log"
)

// phaseKeyToStatus maps PhaseType.Key to HackathonStatus.
var phaseKeyToStatus = map[string]hackforger_model.HackathonStatus{
	"registration": hackforger_model.HackathonStatusOpen,
	"development":  hackforger_model.HackathonStatusHacking,
	"judging":      hackforger_model.HackathonStatusJudging,
	"results":      hackforger_model.HackathonStatusFinished,
}

// SyncStatusCache recomputes a hackathon's StatusCache from its Phase records.
// If the status changes, it fires lifecycle hooks and publishes a feed event.
// doerID is the user who triggered the change (0 for system/lazy sync).
func SyncStatusCache(ctx context.Context, hackathonID int64, doerID int64) error {
	h, err := hackforger_model.GetHackathonByID(ctx, hackathonID)
	if err != nil {
		return err
	}

	newStatus := computeStatusFromPhases(ctx, h)
	if newStatus == h.StatusCache {
		return nil // no change
	}

	oldStatus := h.StatusCache
	if err := hackforger_model.UpdateHackathonStatusCache(ctx, h.ID, newStatus); err != nil {
		return err
	}

	// Fire lifecycle hooks
	runLifecycleHooks(ctx, h, oldStatus, newStatus, doerID)

	// Publish feed event
	eventDoerID := doerID
	if eventDoerID == 0 {
		eventDoerID = h.OwnerID
	}
	publishPhaseChange(ctx, eventDoerID, h, oldStatus, newStatus)

	return nil
}

// computeStatusFromPhases derives the HackathonStatus from Phase records.
func computeStatusFromPhases(ctx context.Context, h *hackforger_model.Hackathon) hackforger_model.HackathonStatus {
	// Cancelled is a manual override — never auto-change away from it
	if h.StatusCache == hackforger_model.HackathonStatusCancelled {
		return hackforger_model.HackathonStatusCancelled
	}

	// Not published → always Draft
	if !h.IsPublished {
		return hackforger_model.HackathonStatusDraft
	}

	phases, err := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
	if err != nil || len(phases) == 0 {
		return hackforger_model.HackathonStatusDraft
	}

	// Check for active phase
	current, _ := hackforger_model.GetCurrentPhase(ctx, "hackathon", h.ID)
	if current != nil && current.PhaseType != nil {
		if status, ok := phaseKeyToStatus[current.PhaseType.Key]; ok {
			return status
		}
	}

	// No active phase — find next future phase (gap handling)
	for _, p := range phases {
		if p.IsFuture() && p.PhaseType != nil {
			if status, ok := phaseKeyToStatus[p.PhaseType.Key]; ok {
				return status
			}
		}
	}

	// All phases past — use last phase's mapped status
	last := phases[len(phases)-1]
	if last.PhaseType != nil {
		if status, ok := phaseKeyToStatus[last.PhaseType.Key]; ok {
			return status
		}
	}

	return hackforger_model.HackathonStatusDraft
}

// runLifecycleHooks fires side effects when status transitions occur.
func runLifecycleHooks(ctx context.Context, h *hackforger_model.Hackathon, oldStatus, newStatus hackforger_model.HackathonStatus, doerID int64) {
	switch {
	case oldStatus == hackforger_model.HackathonStatusOpen && newStatus == hackforger_model.HackathonStatusHacking:
		// Open → Hacking: tag track repos with v0-kickoff baseline
		if doerID > 0 {
			tagTrackRepos(ctx, doerID, h.ID, "v0-kickoff", "Hackathon kickoff baseline")
		}

	case oldStatus == hackforger_model.HackathonStatusHacking && newStatus == hackforger_model.HackathonStatusJudging:
		// Hacking → Judging: warn if no criteria or submissions
		criteriaCount, _ := hackforger_model.CountCriteriaByHackathon(ctx, h.ID)
		subCount, _ := hackforger_model.CountSubmissions(ctx, h.ID)
		if criteriaCount == 0 {
			log.Warn("Hackathon %d entered judging phase with no scoring criteria", h.ID)
		}
		if subCount == 0 {
			log.Warn("Hackathon %d entered judging phase with no submissions", h.ID)
		}
	}
}

// EnsureStatusCacheFresh lazily syncs StatusCache when loading a hackathon.
// Call this in route handlers after loading a hackathon for display.
func EnsureStatusCacheFresh(ctx context.Context, h *hackforger_model.Hackathon) {
	if h.StatusCache == hackforger_model.HackathonStatusCancelled {
		return
	}
	if !h.IsPublished {
		return
	}
	computed := computeStatusFromPhases(ctx, h)
	if computed != h.StatusCache {
		if err := SyncStatusCache(ctx, h.ID, 0); err != nil {
			log.Error("EnsureStatusCacheFresh: %v", err)
		}
		h.StatusCache = computed
	}
}
