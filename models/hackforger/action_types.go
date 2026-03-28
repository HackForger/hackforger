// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	activities_model "forgejo.org/models/activities"
)

// HackForger action types, starting from 30 to avoid collision with
// Forgejo's built-in ActionType (1-27).
const (
	// User behavior events (30-42)
	ActionHackathonCreated      activities_model.ActionType = 30
	ActionHackathonRegistered   activities_model.ActionType = 31
	ActionHackathonSubmitted    activities_model.ActionType = 32
	ActionHackathonScored       activities_model.ActionType = 33
	ActionBountyCreated         activities_model.ActionType = 34
	ActionBountyClaimed         activities_model.ActionType = 35
	ActionBountyDelivered       activities_model.ActionType = 36
	ActionBountyCompleted       activities_model.ActionType = 37
	ActionBountyWinnersSelected activities_model.ActionType = 38
	ActionGrantRoundCreated     activities_model.ActionType = 39
	ActionGrantProjectSubmitted activities_model.ActionType = 40
	ActionGrantAwarded          activities_model.ActionType = 41
	ActionCreditsRedeemed       activities_model.ActionType = 42
	ActionBountyPaid            activities_model.ActionType = 43

	// Entity lifecycle events (50-56)
	ActionHackathonPhaseChanged activities_model.ActionType = 50
	ActionHackathonFinalized    activities_model.ActionType = 51
	ActionBountyExpired         activities_model.ActionType = 52
	ActionBountyCancelled       activities_model.ActionType = 53
	ActionGrantRoundOpened      activities_model.ActionType = 54
	ActionGrantRoundClosed      activities_model.ActionType = 55
	ActionGrantRoundFinalized   activities_model.ActionType = 56
	ActionGrantRoundCancelled   activities_model.ActionType = 57

	// Milestone events (60, future)
	ActionMilestone activities_model.ActionType = 60
)

// HackforgerActionTypeName maps HackForger action types to string names.
var HackforgerActionTypeName = map[activities_model.ActionType]string{
	ActionHackathonCreated:      "hackathon_created",
	ActionHackathonRegistered:   "hackathon_registered",
	ActionHackathonSubmitted:    "hackathon_submitted",
	ActionHackathonScored:       "hackathon_scored",
	ActionBountyCreated:         "bounty_created",
	ActionBountyClaimed:         "bounty_claimed",
	ActionBountyDelivered:       "bounty_delivered",
	ActionBountyCompleted:       "bounty_completed",
	ActionBountyWinnersSelected: "bounty_winners_selected",
	ActionGrantRoundCreated:     "grant_round_created",
	ActionGrantProjectSubmitted: "grant_project_submitted",
	ActionGrantAwarded:          "grant_awarded",
	ActionCreditsRedeemed:       "credits_redeemed",
	ActionBountyPaid:            "bounty_paid",
	ActionHackathonPhaseChanged: "hackathon_phase_changed",
	ActionHackathonFinalized:    "hackathon_finalized",
	ActionBountyExpired:         "bounty_expired",
	ActionBountyCancelled:       "bounty_cancelled",
	ActionGrantRoundOpened:      "grant_round_opened",
	ActionGrantRoundClosed:      "grant_round_closed",
	ActionGrantRoundFinalized:   "grant_round_finalized",
	ActionGrantRoundCancelled:   "grant_round_cancelled",
	ActionMilestone:             "milestone",
}

// IsHackforgerAction returns true if the ActionType is a HackForger event.
func IsHackforgerAction(at activities_model.ActionType) bool {
	_, ok := HackforgerActionTypeName[at]
	return ok
}

// HackforgerActionContent is the JSON content stored in action.content
// for most HackForger events.
type HackforgerActionContent struct {
	EntityType string         `json:"entity_type"`
	EntityID   int64          `json:"entity_id"`
	EntityName string         `json:"entity_name"`
	EntitySlug string         `json:"entity_slug,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// HackforgerPhaseContent extends HackforgerActionContent with status change info.
type HackforgerPhaseContent struct {
	HackforgerActionContent
	OldStatus   string `json:"old_status"`
	NewStatus   string `json:"new_status"`
	StatusLabel string `json:"status_label,omitempty"`
	WinnerName  string `json:"winner_name,omitempty"`
}
