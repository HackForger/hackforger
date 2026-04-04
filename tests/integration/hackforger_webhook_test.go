// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"testing"

	hackforger_model "forgejo.org/models/hackforger"
	api "forgejo.org/modules/structs"
	webhook_module "forgejo.org/modules/webhook"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestHackForgerWebhookEventTypes(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Verify HackForger event type constants are registered with correct string values.
	assert.Equal(t, "hackathon_created", string(webhook_module.HookEventHackathonCreated))
	assert.Equal(t, "hackathon_status_changed", string(webhook_module.HookEventHackathonStatusChanged))
	assert.Equal(t, "hackathon_submission", string(webhook_module.HookEventHackathonSubmission))
	assert.Equal(t, "hackathon_scored", string(webhook_module.HookEventHackathonScored))
	assert.Equal(t, "hackathon_finalized", string(webhook_module.HookEventHackathonFinalized))
	assert.Equal(t, "bounty_created", string(webhook_module.HookEventBountyCreated))
	assert.Equal(t, "bounty_application", string(webhook_module.HookEventBountyApplication))
	assert.Equal(t, "bounty_claimed", string(webhook_module.HookEventBountyClaimed))
	assert.Equal(t, "bounty_completed", string(webhook_module.HookEventBountyCompleted))
	assert.Equal(t, "bounty_paid", string(webhook_module.HookEventBountyPaid))
	assert.Equal(t, "bounty_winners", string(webhook_module.HookEventBountyWinners))
	assert.Equal(t, "bounty_expired", string(webhook_module.HookEventBountyExpired))
	assert.Equal(t, "bounty_cancelled", string(webhook_module.HookEventBountyCancelled))
	assert.Equal(t, "grant_round_created", string(webhook_module.HookEventGrantRoundCreated))
	assert.Equal(t, "grant_round_opened", string(webhook_module.HookEventGrantRoundOpened))
	assert.Equal(t, "grant_project_submitted", string(webhook_module.HookEventGrantProjectSubmitted))
	assert.Equal(t, "grant_awarded", string(webhook_module.HookEventGrantAwarded))
	assert.Equal(t, "grant_round_finalized", string(webhook_module.HookEventGrantRoundFinalized))
	assert.Equal(t, "credits_deposited", string(webhook_module.HookEventCreditsDeposited))
	assert.Equal(t, "credits_redeemed", string(webhook_module.HookEventCreditsRedeemed))
}

func TestHackForgerWebhookEventMethod(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Verify the Event() method returns the correct string for each HackForger event.
	assert.Equal(t, "hackathon_created", webhook_module.HookEventHackathonCreated.Event())
	assert.Equal(t, "bounty_created", webhook_module.HookEventBountyCreated.Event())
	assert.Equal(t, "grant_round_created", webhook_module.HookEventGrantRoundCreated.Event())
	assert.Equal(t, "credits_deposited", webhook_module.HookEventCreditsDeposited.Event())
	assert.Equal(t, "credits_redeemed", webhook_module.HookEventCreditsRedeemed.Event())
}

func TestHackForgerActionTypeToHookEventMapping(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Verify ActionType -> HookEventType mapping covers expected action types.
	mapping := hackforger_model.ActionTypeToHookEvent

	assert.Equal(t, webhook_module.HookEventHackathonCreated, mapping[hackforger_model.ActionHackathonCreated])
	assert.Equal(t, webhook_module.HookEventHackathonStatusChanged, mapping[hackforger_model.ActionHackathonPhaseChanged])
	assert.Equal(t, webhook_module.HookEventHackathonSubmission, mapping[hackforger_model.ActionHackathonSubmitted])
	assert.Equal(t, webhook_module.HookEventHackathonScored, mapping[hackforger_model.ActionHackathonScored])
	assert.Equal(t, webhook_module.HookEventHackathonFinalized, mapping[hackforger_model.ActionHackathonFinalized])

	assert.Equal(t, webhook_module.HookEventBountyCreated, mapping[hackforger_model.ActionBountyCreated])
	assert.Equal(t, webhook_module.HookEventBountyClaimed, mapping[hackforger_model.ActionBountyClaimed])
	assert.Equal(t, webhook_module.HookEventBountyApplication, mapping[hackforger_model.ActionBountyDelivered])
	assert.Equal(t, webhook_module.HookEventBountyCompleted, mapping[hackforger_model.ActionBountyCompleted])
	assert.Equal(t, webhook_module.HookEventBountyWinners, mapping[hackforger_model.ActionBountyWinnersSelected])
	assert.Equal(t, webhook_module.HookEventBountyPaid, mapping[hackforger_model.ActionBountyPaid])
	assert.Equal(t, webhook_module.HookEventBountyExpired, mapping[hackforger_model.ActionBountyExpired])
	assert.Equal(t, webhook_module.HookEventBountyCancelled, mapping[hackforger_model.ActionBountyCancelled])

	assert.Equal(t, webhook_module.HookEventGrantRoundCreated, mapping[hackforger_model.ActionGrantRoundCreated])
	assert.Equal(t, webhook_module.HookEventGrantProjectSubmitted, mapping[hackforger_model.ActionGrantProjectSubmitted])
	assert.Equal(t, webhook_module.HookEventGrantAwarded, mapping[hackforger_model.ActionGrantAwarded])
	assert.Equal(t, webhook_module.HookEventGrantRoundOpened, mapping[hackforger_model.ActionGrantRoundOpened])
	assert.Equal(t, webhook_module.HookEventGrantRoundFinalized, mapping[hackforger_model.ActionGrantRoundFinalized])

	assert.Equal(t, webhook_module.HookEventCreditsRedeemed, mapping[hackforger_model.ActionCreditsRedeemed])

	// Verify unmapped action types return zero value.
	_, ok := mapping[hackforger_model.ActionHackathonRegistered]
	assert.False(t, ok, "ActionHackathonRegistered should not be in the webhook mapping")
}

func TestHackForgerWebhookPayloadInterface(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Verify HackforgerWebhookPayload implements the Payloader interface
	// and produces valid JSON.
	payload := &api.HackforgerWebhookPayload{
		Action:     "hackathon_created",
		EntityType: "hackathon",
		EntityID:   42,
		EntityName: "Test Hackathon",
	}
	data, err := payload.JSONPayload()
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"hackathon_created"`)
	assert.Contains(t, string(data), `"entity_id": 42`)
}

// TODO: Full webhook delivery integration test.
// This would require:
// 1. Creating a webhook via API with HackForger event subscriptions
// 2. Triggering a HackForger action (e.g., creating a hackathon)
// 3. Verifying a hook_task record was created in the database
// 4. Checking the payload content matches the expected structure
// This is complex because it requires the full webhook delivery pipeline
// to be running (queue workers, etc.) and is better tested as E2E.
