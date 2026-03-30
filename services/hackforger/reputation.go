// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"math"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
)

const (
	defaultWeightsJSON = `{"stars":1,"bounties_completed":5,"hackathon_wins":10,"grants_received":3,"credits_earned":0.1}`
	defaultTiersJSON   = `[{"name":"Bronze","min":0},{"name":"Silver","min":50},{"name":"Gold","min":200},{"name":"Diamond","min":500}]`
)

// DefaultWeightsJSON returns the default JSON for reputation weights.
func DefaultWeightsJSON() string { return defaultWeightsJSON }

// DefaultTiersJSON returns the default JSON for reputation tiers.
func DefaultTiersJSON() string { return defaultTiersJSON }

// ReputationWeights holds the per-metric multipliers used to compute a
// user's reputation score.
type ReputationWeights struct {
	Stars             float64 `json:"stars"`
	BountiesCompleted float64 `json:"bounties_completed"`
	HackathonWins     float64 `json:"hackathon_wins"`
	GrantsReceived    float64 `json:"grants_received"`
	CreditsEarned     float64 `json:"credits_earned"`
}

// ReputationTier defines a named tier and the minimum score required to
// attain it.
type ReputationTier struct {
	Name string  `json:"name"`
	Min  float64 `json:"min"`
}

// GetReputationWeights loads the admin-configured weights from the settings
// table, falling back to the built-in defaults when the key is absent.
func GetReputationWeights(ctx context.Context) (*ReputationWeights, error) {
	raw := hackforger_model.GetSettingWithDefault(ctx, "reputation.weights", defaultWeightsJSON)
	var w ReputationWeights
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// GetReputationTiers loads the admin-configured tier thresholds from the
// settings table, falling back to the built-in defaults when the key is absent.
func GetReputationTiers(ctx context.Context) ([]ReputationTier, error) {
	raw := hackforger_model.GetSettingWithDefault(ctx, "reputation.tiers", defaultTiersJSON)
	var tiers []ReputationTier
	if err := json.Unmarshal([]byte(raw), &tiers); err != nil {
		return nil, err
	}
	return tiers, nil
}

// deriveTier returns the highest tier whose Min threshold is not exceeded by
// score. Tiers must be ordered ascending by Min for the logic to be correct.
func deriveTier(score int64, tiers []ReputationTier) string {
	tier := "Bronze"
	for _, t := range tiers {
		if float64(score) >= t.Min {
			tier = t.Name
		}
	}
	return tier
}

// RecalculateReputation recomputes the reputation score for a single user and
// persists the result. It queries raw DB metrics (stars, bounties, hackathon
// wins, grants, credits) and applies the admin-configured weights.
func RecalculateReputation(ctx context.Context, userID int64) error {
	rep, err := hackforger_model.GetOrCreateReputation(ctx, userID)
	if err != nil {
		return err
	}

	e := db.GetEngine(ctx)

	bountiesCompleted, _ := e.Table("bounty").
		Where("claimer_id = ? AND status IN (?, ?)", userID,
			hackforger_model.BountyStatusCompleted, hackforger_model.BountyStatusPaid).Count()

	hackathonWins, _ := e.Table("hackathon_submission").
		Where("user_id = ? AND rank = 1", userID).Count()

	grantsReceived, _ := e.Table("grant_project").
		Where("user_id = ? AND status IN (?, ?)", userID,
			hackforger_model.GrantProjectStatusApproved, hackforger_model.GrantProjectStatusFunded).Count()

	var totalStars int64
	_, _ = e.SQL("SELECT COALESCE(SUM(num_stars), 0) FROM repository WHERE owner_id = ?", userID).Get(&totalStars)

	var totalCreditsEarned int64
	_, _ = e.SQL(
		"SELECT COALESCE(SUM(amount), 0) FROM credit_transaction WHERE user_id = ? AND type IN (?, ?)",
		userID,
		string(hackforger_model.TransactionTypeDeposit),
		string(hackforger_model.TransactionTypeReward),
	).Get(&totalCreditsEarned)

	weights, err := GetReputationWeights(ctx)
	if err != nil {
		return err
	}

	scoreF := float64(totalStars)*weights.Stars +
		float64(bountiesCompleted)*weights.BountiesCompleted +
		float64(hackathonWins)*weights.HackathonWins +
		float64(grantsReceived)*weights.GrantsReceived +
		float64(totalCreditsEarned)*weights.CreditsEarned
	score := int64(math.Round(scoreF))

	tiers, err := GetReputationTiers(ctx)
	if err != nil {
		return err
	}

	rep.Score = score
	rep.BountiesCompleted = int(bountiesCompleted)
	rep.HackathonWins = int(hackathonWins)
	rep.GrantsReceived = int(grantsReceived)
	rep.TotalStars = totalStars
	rep.TotalCreditsEarned = totalCreditsEarned
	rep.Tier = deriveTier(score, tiers)

	return hackforger_model.UpdateReputation(ctx, rep)
}

// RecalculateAllReputations iterates every existing reputation record and
// recomputes it. Errors for individual users are logged but do not abort the
// loop, so a single bad record cannot block the rest of the batch.
func RecalculateAllReputations(ctx context.Context) error {
	var reps []*hackforger_model.Reputation
	if err := db.GetEngine(ctx).Find(&reps); err != nil {
		return err
	}
	for _, rep := range reps {
		if err := RecalculateReputation(ctx, rep.UserID); err != nil {
			log.Error("RecalculateReputation(user=%d): %v", rep.UserID, err)
		}
	}
	return nil
}
