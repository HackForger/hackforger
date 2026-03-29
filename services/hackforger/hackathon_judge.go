// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"
	"sort"

	hackforger_model "forgejo.org/models/hackforger"
)

// SubmitScore validates and records a judge's score for a submission.
func SubmitScore(ctx context.Context, judgeID, submissionID int64, score float64, comment string) error {
	sub, err := hackforger_model.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return err
	}
	h, err := hackforger_model.GetHackathonByID(ctx, sub.HackathonID)
	if err != nil {
		return err
	}
	if h.Status != hackforger_model.HackathonStatusJudging {
		return fmt.Errorf("hackathon must be in Judging status to score [id: %d, status: %d]", h.ID, h.Status)
	}
	isJudge, err := hackforger_model.IsJudge(ctx, h.ID, judgeID)
	if err != nil {
		return err
	}
	if !isJudge {
		return fmt.Errorf("user is not a judge for this hackathon [user_id: %d, hackathon_id: %d]", judgeID, h.ID)
	}
	if score < 0 || score > 10 {
		return fmt.Errorf("score must be between 0 and 10 [got: %f]", score)
	}

	existing, err := hackforger_model.GetScore(ctx, judgeID, submissionID)
	if err != nil {
		return err
	}
	if existing != nil {
		existing.Score = score
		existing.Comment = comment
		return hackforger_model.UpdateScore(ctx, existing)
	}

	s := &hackforger_model.HackathonJudgeScore{
		HackathonID:  h.ID,
		SubmissionID: submissionID,
		JudgeID:      judgeID,
		Score:        score,
		Comment:      comment,
	}
	if err := hackforger_model.CreateScore(ctx, s); err != nil {
		return err
	}

	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    judgeID,
		OpType:       hackforger_model.ActionHackathonScored,
		AudienceType: AudienceFollowers,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
			Extra: map[string]any{"submission_id": submissionID},
		},
	})
	return nil
}

// CalculateRanks computes average scores and assigns ranks for all submissions.
func CalculateRanks(ctx context.Context, hackathonID int64) error {
	subs, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
		HackathonID: hackathonID,
	})
	if err != nil {
		return err
	}

	type scored struct {
		id  int64
		avg float64
	}
	var scoredSubs []scored

	for _, sub := range subs {
		scores, err := hackforger_model.ListScoresBySubmission(ctx, sub.ID)
		if err != nil {
			return err
		}
		if len(scores) == 0 {
			scoredSubs = append(scoredSubs, scored{id: sub.ID, avg: 0})
			continue
		}
		var total float64
		for _, s := range scores {
			total += s.Score
		}
		scoredSubs = append(scoredSubs, scored{id: sub.ID, avg: total / float64(len(scores))})
	}

	sort.Slice(scoredSubs, func(i, j int) bool {
		return scoredSubs[i].avg > scoredSubs[j].avg
	})

	rankings := make([]hackforger_model.SubmissionRanking, len(scoredSubs))
	for i, s := range scoredSubs {
		rankings[i] = hackforger_model.SubmissionRanking{
			SubmissionID: s.id,
			TotalScore:   s.avg,
			Rank:         i + 1,
		}
	}

	return hackforger_model.UpdateSubmissionRanks(ctx, rankings)
}
