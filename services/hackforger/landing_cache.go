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

// landingCardSlots defines the 12 card slot identifiers shown on the landing page.
// MUST match the data-card attributes in custom/public/assets/landing/index.html.
// Slot identifiers are deliberately neutral numeric strings — HackForger has no
// "stage" or "wave" concept; the visual grouping (S1/S2/S3 etc) is purely
// design-level CSS, not a data concept.
// If the landing HTML changes the card count, update this array.
var landingCardSlots = []string{
	"1", "2", "3", "4",
	"5", "6", "7", "8",
	"9", "10", "11", "12",
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
// Cards without a matching published hackathon return {slug:null, enabled:false}.
func buildLandingPayload(ctx context.Context) ([]byte, error) {
	prefix, _ := system_model.GetSettingByKey(ctx, "hackforger.landing.hackathon_prefix")

	response := map[string]interface{}{
		"hackathon_prefix": prefix,
		"cards":            map[string]interface{}{},
		"fetched_at":       time.Now().UTC().Format(time.RFC3339),
	}
	cards := response["cards"].(map[string]interface{})
	// Initialize all 12 card slots empty by default.
	for _, slot := range landingCardSlots {
		cards[slot] = map[string]interface{}{"slug": nil, "enabled": false}
	}

	if prefix == "" {
		return json.Marshal(response)
	}
	// Defense-in-depth: validate prefix on read in case DB was tampered.
	if !validation.IsValidUsername(prefix + "-h1") {
		log.Warn("Invalid landing prefix in DB: %q", prefix)
		return json.Marshal(response)
	}

	// Slug pattern: <prefix>-h{N} for N=1..12. The "h" prefix maps to "hackathon".
	expectedSlugs := make([]string, 0, len(landingCardSlots))
	slugToSlot := make(map[string]string, len(landingCardSlots))
	for _, slot := range landingCardSlots {
		s := prefix + "-h" + slot
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
		// Pre-fetch all phases for this hackathon once (PhaseType eager-loaded).
		// Avoids repeated round-trips for each phase-key window query.
		allPhases, _ := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)

		// Find the currently active phase from the pre-fetched list.
		var currentPhase *hackforger_model.Phase
		nowUnix := time.Now().Unix()
		for _, p := range allPhases {
			if p.StartTime <= nowUnix && p.EndTime > nowUnix {
				currentPhase = p
				break
			}
		}

		cardData := map[string]interface{}{
			"slug":    h.Slug,
			"name":    h.Name,
			"enabled": determineEnabled(h, currentPhase),
		}
		if currentPhase != nil && currentPhase.PhaseType != nil {
			cardData["current_phase"] = map[string]interface{}{
				"key":               currentPhase.PhaseType.Key,
				"display_name_i18n": currentPhase.PhaseType.DisplayNameI18n,
				"ends_at":           time.Unix(currentPhase.EndTime, 0).UTC().Format(time.RFC3339),
			}
		}

		// All 4 phase windows hydrated for badge updates (D1).
		// Phase keys match HackForger's phase_type catalog (activity_kind='hackathon'):
		//   registration, development, judging, results
		// JS maps these keys to badge prefixes via PHASE_LABELS dictionary.
		windows := map[string]map[string]string{}
		for _, key := range []string{"registration", "development", "judging", "results"} {
			for _, p := range allPhases {
				if p.PhaseType != nil && p.PhaseType.Key == key {
					windows[key+"_window"] = map[string]string{
						"start": time.Unix(p.StartTime, 0).UTC().Format(time.RFC3339),
						"end":   time.Unix(p.EndTime, 0).UTC().Format(time.RFC3339),
					}
					break
				}
			}
		}
		for k, v := range windows {
			cardData[k] = v
		}

		names := []string{}
		if tracks, ok := tracksByHackathon[h.ID]; ok {
			for _, t := range tracks {
				names = append(names, t.Name)
			}
		}
		cardData["tracks"] = names

		cards[slot] = cardData
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
