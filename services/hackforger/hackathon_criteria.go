// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/util"
)

// EffectiveCriteria represents a criterion with track-level weight resolution.
type EffectiveCriteria struct {
	CriteriaID  int64
	Name        string
	Description string
	MaxScore    float64
	Weight      float64 // track-level weight if >0, otherwise criteria default
	SortOrder   int
}

// ErrScoreOutOfRange represents a score that exceeds the criteria's max score.
type ErrScoreOutOfRange struct {
	CriteriaID int64
	Score      float64
	MaxScore   float64
}

// IsErrScoreOutOfRange checks if an error is a ErrScoreOutOfRange.
func IsErrScoreOutOfRange(err error) bool {
	_, ok := err.(ErrScoreOutOfRange)
	return ok
}

func (err ErrScoreOutOfRange) Error() string {
	return fmt.Sprintf("score out of range [criteria_id: %d, score: %.1f, max: %.1f]", err.CriteriaID, err.Score, err.MaxScore)
}

func (err ErrScoreOutOfRange) Unwrap() error {
	return util.ErrInvalidArgument
}

// ErrIncompleteRubric represents a submission that has not been scored on all criteria.
type ErrIncompleteRubric struct {
	Missing []int64 // criteria IDs not scored
}

// IsErrIncompleteRubric checks if an error is a ErrIncompleteRubric.
func IsErrIncompleteRubric(err error) bool {
	_, ok := err.(ErrIncompleteRubric)
	return ok
}

func (err ErrIncompleteRubric) Error() string {
	return fmt.Sprintf("incomplete rubric [missing criteria: %v]", err.Missing)
}

func (err ErrIncompleteRubric) Unwrap() error {
	return util.ErrInvalidArgument
}

// checkCriteriaModifiable verifies the hackathon is in a phase that allows
// criteria modification (Draft or Open). Returns ErrInvalidHackathonPhase
// if the hackathon has progressed past Open.
func checkCriteriaModifiable(ctx context.Context, hackathonID int64) error {
	h, err := hackforger_model.GetHackathonByID(ctx, hackathonID)
	if err != nil {
		return err
	}
	// Allow criteria changes during Draft, Registration, and Hacking phases.
	// Lock once judging starts (StatusCache >= Judging).
	if h.StatusCache >= hackforger_model.HackathonStatusJudging {
		return hackforger_model.ErrInvalidHackathonPhase{
			HackathonID: h.ID,
			Current:     h.StatusCache,
			Expected:    hackforger_model.HackathonStatusHacking,
		}
	}
	return nil
}

// AddCriteria creates a new judge criterion for a hackathon and seeds it
// to all existing tracks. The hackathon must be in Draft or Open status.
func AddCriteria(ctx context.Context, hackathonID int64, name, description string, maxScore, weight float64, sortOrder int) error {
	if err := checkCriteriaModifiable(ctx, hackathonID); err != nil {
		return err
	}

	c := &hackforger_model.HackathonJudgeCriteria{
		HackathonID: hackathonID,
		Name:        name,
		Description: description,
		MaxScore:    maxScore,
		Weight:      weight,
		SortOrder:   sortOrder,
	}
	if err := hackforger_model.CreateCriteria(ctx, c); err != nil {
		return err
	}

	// Auto-seed all existing tracks with the new criterion.
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, hackathonID)
	if err != nil {
		return err
	}
	for _, track := range tracks {
		if err := hackforger_model.SeedTrackCriteria(ctx, track.ID, []*hackforger_model.HackathonJudgeCriteria{c}); err != nil {
			return err
		}
	}

	return nil
}

// UpdateCriteria updates an existing judge criterion. The hackathon must be
// in Draft or Open status.
func UpdateCriteria(ctx context.Context, c *hackforger_model.HackathonJudgeCriteria) error {
	existing, err := hackforger_model.GetCriteriaByID(ctx, c.ID)
	if err != nil {
		return err
	}
	if err := checkCriteriaModifiable(ctx, existing.HackathonID); err != nil {
		return err
	}
	return hackforger_model.UpdateCriteria(ctx, c)
}

// RemoveCriteria deletes a judge criterion and cascades to track criteria
// links and judge scores. The hackathon must be in Draft or Open status.
func RemoveCriteria(ctx context.Context, criteriaID int64) error {
	existing, err := hackforger_model.GetCriteriaByID(ctx, criteriaID)
	if err != nil {
		return err
	}
	if err := checkCriteriaModifiable(ctx, existing.HackathonID); err != nil {
		return err
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		// Cascade: delete scores referencing this criterion.
		if _, err := db.GetEngine(ctx).Where("criteria_id = ?", criteriaID).
			Delete(new(hackforger_model.HackathonJudgeScore)); err != nil {
			return err
		}
		// Cascade: delete track-criteria links referencing this criterion.
		if _, err := db.GetEngine(ctx).Where("criteria_id = ?", criteriaID).
			Delete(new(hackforger_model.HackathonTrackCriteria)); err != nil {
			return err
		}
		// Delete the criterion itself.
		return hackforger_model.DeleteCriteria(ctx, criteriaID)
	})
}

// SetTrackCriteriaOverride creates or updates a track-level override for a
// criterion's enabled state and weight. Enabled criteria must have a positive weight.
func SetTrackCriteriaOverride(ctx context.Context, trackID, criteriaID int64, enabled bool, weight float64) error {
	if enabled && weight <= 0 {
		return fmt.Errorf("enabled criteria must have positive weight [track_id: %d, criteria_id: %d, weight: %.2f]", trackID, criteriaID, weight)
	}

	existing, err := hackforger_model.GetTrackCriteria(ctx, trackID, criteriaID)
	if err != nil {
		return err
	}
	if existing != nil {
		existing.Enabled = enabled
		existing.Weight = weight
		return hackforger_model.UpdateTrackCriteria(ctx, existing)
	}

	return hackforger_model.CreateTrackCriteria(ctx, &hackforger_model.HackathonTrackCriteria{
		TrackID:    trackID,
		CriteriaID: criteriaID,
		Enabled:    enabled,
		Weight:     weight,
	})
}

// GetEffectiveRubric returns the resolved list of scoring criteria for a track.
// Track-level overrides can disable criteria or override their weight. Criteria
// without an override use the hackathon-level defaults.
func GetEffectiveRubric(ctx context.Context, trackID int64) ([]*EffectiveCriteria, error) {
	track, err := hackforger_model.GetTrackByID(ctx, trackID)
	if err != nil {
		return nil, err
	}

	criteria, err := hackforger_model.ListCriteriaByHackathon(ctx, track.HackathonID)
	if err != nil {
		return nil, err
	}

	overrides, err := hackforger_model.ListTrackCriteria(ctx, trackID)
	if err != nil {
		return nil, err
	}

	// Build a map of track-level overrides keyed by criteria ID.
	overrideMap := make(map[int64]*hackforger_model.HackathonTrackCriteria, len(overrides))
	for _, o := range overrides {
		overrideMap[o.CriteriaID] = o
	}

	var result []*EffectiveCriteria
	for _, c := range criteria {
		override, hasOverride := overrideMap[c.ID]

		// If override exists and disabled, skip this criterion.
		if hasOverride && !override.Enabled {
			continue
		}

		weight := c.Weight
		if hasOverride && override.Weight > 0 {
			weight = override.Weight
		}

		result = append(result, &EffectiveCriteria{
			CriteriaID:  c.ID,
			Name:        c.Name,
			Description: c.Description,
			MaxScore:    c.MaxScore,
			Weight:      weight,
			SortOrder:   c.SortOrder,
		})
	}

	return result, nil
}
