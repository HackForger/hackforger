// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// White-box test (same package) for internal helpers.

package hackforger

import (
	"fmt"
	"strings"
	"testing"

	hackforger_model "forgejo.org/models/hackforger"

	"github.com/stretchr/testify/assert"
)

func TestTrackRepoContributingTemplateIsBusinessNeutral(t *testing.T) {
	rendered := fmt.Sprintf(trackRepoContributingTemplate, "Example Track", "Example Event", "Example Event")

	assert.NotContains(t, strings.ToLower(rendered), "synno"+"vator")
	assert.NotContains(t, rendered, "%!")
	assert.Contains(t, rendered, "Example Track")
	assert.Equal(t, 2, strings.Count(rendered, "Example Event"))
}

func TestPhasesToMilestones_SkipsPhasesWithoutEndTime(t *testing.T) {
	regType := &hackforger_model.PhaseType{Key: "registration"}
	devType := &hackforger_model.PhaseType{Key: "development"}

	phases := []*hackforger_model.Phase{
		{PhaseType: regType, StartTime: 1_700_000_000, EndTime: 1_700_100_000}, // scheduled
		{PhaseType: devType, StartTime: 1_700_100_000, EndTime: 0},             // unscheduled — must be skipped
	}

	got := phasesToMilestones(phases, 42)

	assert.Len(t, got, 1, "phases with EndTime==0 must not produce milestones")
	assert.Equal(t, int64(42), got[0].RepoID)
	assert.Equal(t, "registration", got[0].Name)
	assert.Equal(t, int64(1_700_100_000), int64(got[0].DeadlineUnix))
}

func TestPhasesToMilestones_UsesCustomNameWhenSet(t *testing.T) {
	phases := []*hackforger_model.Phase{
		{CustomName: "Final Submission", StartTime: 1, EndTime: 2},
	}

	got := phasesToMilestones(phases, 1)

	assert.Len(t, got, 1)
	assert.Equal(t, "Final Submission", got[0].Name,
		"CustomName should take precedence over PhaseType.Key")
}

func TestPhasesToMilestones_FallsBackToPhaseTypeKey(t *testing.T) {
	phases := []*hackforger_model.Phase{
		{PhaseType: &hackforger_model.PhaseType{Key: "judging"}, StartTime: 1, EndTime: 2},
	}

	got := phasesToMilestones(phases, 1)

	assert.Len(t, got, 1)
	assert.Equal(t, "judging", got[0].Name)
}

func TestPhasesToMilestones_EmptyInput(t *testing.T) {
	got := phasesToMilestones(nil, 1)
	assert.Len(t, got, 0)
}
