// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"sort"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
)

// CriteriaScore holds a judge's score for a single criterion.
type CriteriaScore struct {
	CriteriaID int64
	Score      float64
	Comment    string
}

// SubmitScores validates and records a judge's scores for all criteria on a submission.
func SubmitScores(ctx context.Context, judgeID, submissionID int64, scores []CriteriaScore) error {
	// 1. Load submission
	sub, err := hackforger_model.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return err
	}

	// 2. Load hackathon, verify status == Judging
	h, err := hackforger_model.GetHackathonByID(ctx, sub.HackathonID)
	if err != nil {
		return err
	}
	if h.Status != hackforger_model.HackathonStatusJudging {
		return hackforger_model.ErrInvalidHackathonPhase{
			HackathonID: h.ID,
			Current:     h.Status,
			Expected:    hackforger_model.HackathonStatusJudging,
		}
	}

	// 3. Verify judge is assigned to submission's track
	isJudge, err := hackforger_model.IsJudge(ctx, h.ID, sub.TrackID, judgeID)
	if err != nil {
		return err
	}
	if !isJudge {
		return hackforger_model.ErrNotJudge{UserID: judgeID, HackathonID: h.ID, TrackID: sub.TrackID}
	}

	// 4. Load effective rubric for track
	rubric, err := GetEffectiveRubric(ctx, sub.TrackID)
	if err != nil {
		return err
	}

	// 5. Build criteria map for validation
	rubricMap := make(map[int64]*EffectiveCriteria, len(rubric))
	for _, c := range rubric {
		rubricMap[c.CriteriaID] = c
	}

	// Validate all enabled criteria are present
	scoreMap := make(map[int64]CriteriaScore, len(scores))
	for _, s := range scores {
		scoreMap[s.CriteriaID] = s
	}
	var missing []int64
	for _, c := range rubric {
		if _, ok := scoreMap[c.CriteriaID]; !ok {
			missing = append(missing, c.CriteriaID)
		}
	}
	if len(missing) > 0 {
		return ErrIncompleteRubric{Missing: missing}
	}

	// 6. Validate score ranges
	for _, s := range scores {
		c, ok := rubricMap[s.CriteriaID]
		if !ok {
			continue // extra criteria ignored
		}
		if s.Score < 0 || s.Score > c.MaxScore {
			return ErrScoreOutOfRange{CriteriaID: s.CriteriaID, Score: s.Score, MaxScore: c.MaxScore}
		}
	}

	// 7. Upsert scores within transaction
	if err := db.WithTx(ctx, func(txCtx context.Context) error {
		for _, s := range scores {
			if _, ok := rubricMap[s.CriteriaID]; !ok {
				continue
			}
			existing, err := hackforger_model.GetScore(txCtx, judgeID, submissionID, s.CriteriaID)
			if err != nil {
				return err
			}
			if existing != nil {
				existing.Score = s.Score
				existing.Comment = s.Comment
				if err := hackforger_model.UpdateScore(txCtx, existing); err != nil {
					return err
				}
			} else {
				newScore := &hackforger_model.HackathonJudgeScore{
					HackathonID:  h.ID,
					SubmissionID: submissionID,
					JudgeID:      judgeID,
					CriteriaID:   s.CriteriaID,
					Score:        s.Score,
					Comment:      s.Comment,
				}
				if err := hackforger_model.CreateScore(txCtx, newScore); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// 8. Publish feed event
	_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
		ActUserID:    judgeID,
		OpType:       hackforger_model.ActionHackathonScored,
		AudienceType: AudienceFollowers,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
			Extra: map[string]any{"submission_id": submissionID, "track_id": sub.TrackID},
		},
	})
	return nil
}

// RankedSubmission holds a submission's computed score and rank within a track.
type RankedSubmission struct {
	SubmissionID   int64
	Title          string
	UserID         int64
	TrackID        int64
	WeightedTotal  float64
	CriteriaScores map[int64]float64 // criteriaID -> average score across judges
	Rank           int
}

// CalculateRanks computes weighted average scores and assigns ranks per track.
// Pure computation — does NOT persist to database.
func CalculateRanks(ctx context.Context, hackathonID int64) (map[int64][]RankedSubmission, error) {
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, hackathonID)
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]RankedSubmission)
	for _, track := range tracks {
		rubric, err := GetEffectiveRubric(ctx, track.ID)
		if err != nil {
			return nil, err
		}
		if len(rubric) == 0 {
			continue
		}

		subs, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
			HackathonID: hackathonID,
			TrackID:     track.ID,
		})
		if err != nil {
			return nil, err
		}

		// Compute total weight for normalization
		var totalWeight float64
		for _, c := range rubric {
			totalWeight += c.Weight
		}
		if totalWeight == 0 {
			continue
		}

		var ranked []RankedSubmission
		for _, sub := range subs {
			allScores, err := hackforger_model.ListScoresBySubmission(ctx, sub.ID)
			if err != nil {
				return nil, err
			}

			// Group scores by criteria, compute averages
			criteriaScores := make(map[int64]float64)
			criteriaCounts := make(map[int64]int)
			for _, s := range allScores {
				criteriaScores[s.CriteriaID] += s.Score
				criteriaCounts[s.CriteriaID]++
			}
			for cid, total := range criteriaScores {
				if criteriaCounts[cid] > 0 {
					criteriaScores[cid] = total / float64(criteriaCounts[cid])
				}
			}

			// Compute weighted total
			var weightedSum float64
			for _, c := range rubric {
				avg := criteriaScores[c.CriteriaID] // 0 if not scored
				weightedSum += avg * c.Weight
			}
			weightedTotal := weightedSum / totalWeight

			ranked = append(ranked, RankedSubmission{
				SubmissionID:   sub.ID,
				Title:          sub.Title,
				UserID:         sub.UserID,
				TrackID:        track.ID,
				WeightedTotal:  weightedTotal,
				CriteriaScores: criteriaScores,
			})
		}

		// Sort by weighted total descending
		sort.Slice(ranked, func(i, j int) bool {
			return ranked[i].WeightedTotal > ranked[j].WeightedTotal
		})
		// Assign ranks 1-based
		for i := range ranked {
			ranked[i].Rank = i + 1
		}
		result[track.ID] = ranked
	}
	return result, nil
}

// PersistRanks writes computed rankings to the hackathon_submission table.
func PersistRanks(ctx context.Context, rankings map[int64][]RankedSubmission) error {
	var allRankings []hackforger_model.SubmissionRanking
	for _, subs := range rankings {
		for _, s := range subs {
			allRankings = append(allRankings, hackforger_model.SubmissionRanking{
				SubmissionID: s.SubmissionID,
				TotalScore:   s.WeightedTotal,
				Rank:         s.Rank,
			})
		}
	}
	return hackforger_model.UpdateSubmissionRanks(ctx, allRankings)
}
