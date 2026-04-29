// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"sync"
	"time"

	hackforger_model "forgejo.org/models/hackforger"
	system_model "forgejo.org/models/system"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/validation"
)

const landingCacheTTL = 5 * time.Minute

// landingSlotOrder defines the 12 stage slots that appear on the landing page.
// MUST match the data-stage attributes in custom/public/assets/landing/index.html.
// If the landing HTML changes the slot list, update this array (and slot count).
var landingSlotOrder = []string{
	"s1-w1", "s1-w2", "s1-w3", "s1-w4",
	"s2-w1", "s2-w2", "s2-w3", "s2-w4",
	"s3-w1", "s3-w2", "s3-w3", "s3-w4",
}

var landingCache struct {
	sync.RWMutex
	payload   []byte
	expiresAt time.Time
}

// InvalidateLandingCache clears the in-memory cache so the next request
// will rebuild fresh data from the DB. Called on:
// - admin changes to hackforger.landing.league_prefix
// - hackathon publish/unpublish
// - hackathon phase transition
// - hackathon delete
func InvalidateLandingCache() {
	landingCache.Lock()
	landingCache.expiresAt = time.Time{}
	landingCache.payload = nil
	landingCache.Unlock()
}

// GetLandingPayload returns cached or freshly-built JSON bytes
// for GET /api/v1/hackforger/landing/stages.
func GetLandingPayload(ctx context.Context) ([]byte, error) {
	landingCache.RLock()
	if time.Now().Before(landingCache.expiresAt) && len(landingCache.payload) > 0 {
		b := landingCache.payload
		landingCache.RUnlock()
		return b, nil
	}
	landingCache.RUnlock()

	payload, err := buildLandingPayload(ctx)
	if err != nil {
		return nil, err
	}
	landingCache.Lock()
	landingCache.payload = payload
	landingCache.expiresAt = time.Now().Add(landingCacheTTL)
	landingCache.Unlock()
	return payload, nil
}

// buildLandingPayload assembles the JSON response from DB data.
// Slots without a matching published hackathon return {slug:null, enabled:false}.
func buildLandingPayload(ctx context.Context) ([]byte, error) {
	prefix, _ := system_model.GetSettingByKey(ctx, "hackforger.landing.league_prefix")

	response := map[string]interface{}{
		"league_prefix": prefix,
		"stages":        map[string]interface{}{},
		"fetched_at":    time.Now().UTC().Format(time.RFC3339),
	}
	stages := response["stages"].(map[string]interface{})
	// Initialize all 12 slots empty by default.
	for _, slot := range landingSlotOrder {
		stages[slot] = map[string]interface{}{"slug": nil, "enabled": false}
	}

	if prefix == "" {
		return json.Marshal(response)
	}
	// Defense-in-depth: validate prefix on read in case DB was tampered.
	if !validation.IsValidUsername(prefix + "-s1-w1") {
		log.Warn("Invalid landing prefix in DB: %q", prefix)
		return json.Marshal(response)
	}

	expectedSlugs := make([]string, 0, len(landingSlotOrder))
	slugToSlot := make(map[string]string, len(landingSlotOrder))
	for _, slot := range landingSlotOrder {
		s := prefix + "-" + slot
		expectedSlugs = append(expectedSlugs, s)
		slugToSlot[s] = slot
	}

	hackathons, err := hackforger_model.ListPublishedHackathonsBySlugs(ctx, expectedSlugs)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(hackathons))
	for _, h := range hackathons {
		ids = append(ids, h.ID)
	}
	tracksByHackathon, err := hackforger_model.ListTracksByHackathonIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, h := range hackathons {
		slot := slugToSlot[h.Slug]
		if slot == "" {
			continue
		}
		// GetCurrentPhase eager-loads phase.PhaseType — no separate fetch.
		phase, _ := hackforger_model.GetCurrentPhase(ctx, "hackathon", h.ID)

		slotData := map[string]interface{}{
			"slug":    h.Slug,
			"name":    h.Name,
			"enabled": determineEnabled(h, phase),
		}
		if phase != nil && phase.PhaseType != nil {
			slotData["current_phase"] = map[string]interface{}{
				"key":               phase.PhaseType.Key,
				"display_name_i18n": phase.PhaseType.DisplayNameI18n,
				"ends_at":           time.Unix(phase.EndTime, 0).UTC().Format(time.RFC3339),
			}
		}
		// registration_window: query the registration phase specifically.
		if regPhase := findPhaseByKey(ctx, h.ID, "registration"); regPhase != nil {
			slotData["registration_window"] = map[string]string{
				"start": time.Unix(regPhase.StartTime, 0).UTC().Format(time.RFC3339),
				"end":   time.Unix(regPhase.EndTime, 0).UTC().Format(time.RFC3339),
			}
		}

		names := []string{}
		if tracks, ok := tracksByHackathon[h.ID]; ok {
			for _, t := range tracks {
				names = append(names, t.Name)
			}
		}
		slotData["tracks"] = names

		stages[slot] = slotData
	}

	return json.Marshal(response)
}

// determineEnabled: registration phase active → true; otherwise fall back to status_cache.
func determineEnabled(h *hackforger_model.Hackathon, phase *hackforger_model.Phase) bool {
	if phase != nil && phase.PhaseType != nil && phase.PhaseType.Key == "registration" {
		return true
	}
	return h.StatusCache == hackforger_model.HackathonStatusOpen
}

func findPhaseByKey(ctx context.Context, hackathonID int64, key string) *hackforger_model.Phase {
	phases, err := hackforger_model.GetPhasesByActivity(ctx, "hackathon", hackathonID)
	if err != nil {
		return nil
	}
	for _, p := range phases {
		if p.PhaseType != nil && p.PhaseType.Key == key {
			return p
		}
	}
	return nil
}
