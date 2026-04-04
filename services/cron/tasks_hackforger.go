// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package cron

import (
	"context"

	user_model "forgejo.org/models/user"
	hackforger_service "forgejo.org/services/hackforger"
)

func registerHackforgerHackathonStatus() {
	RegisterTaskFatal("hackforger_hackathon_status", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@every 5m",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		// Phase 1: call hackforger_service.CheckHackathonTransitions(ctx)
		return nil
	})
}

func registerHackforgerBountyExpiry() {
	RegisterTaskFatal("hackforger_bounty_expiry", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@every 5m",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		// Phase 1: call hackforger_service.ExpireOldBounties(ctx)
		return nil
	})
}

func registerHackforgerReputationRecalc() {
	RegisterTaskFatal("hackforger_reputation_recalc", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@every 1h",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		return hackforger_service.RecalculateAllReputations(ctx)
	})
}

func registerHackforgerGrantDeadline() {
	RegisterTaskFatal("hackforger_grant_deadline", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@every 1h",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		// Phase 1: call hackforger_service.CheckGrantDeadlines(ctx)
		return nil
	})
}

func initHackforgerTasks() {
	registerHackforgerHackathonStatus()
	registerHackforgerBountyExpiry()
	registerHackforgerReputationRecalc()
	registerHackforgerGrantDeadline()
}
