// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// AddJudgeForm is the form for assigning a judge to a hackathon track.
type AddJudgeForm struct {
	UserID  int64 `json:"user_id" binding:"Required"`
	TrackID int64 `json:"track_id" binding:"Required"`
}

// SubmitScoresForm is the form for submitting a judge's scores for all criteria.
type SubmitScoresForm struct {
	Scores []struct {
		CriteriaID int64   `json:"criteria_id"`
		Score      float64 `json:"score"`
		Comment    string  `json:"comment"`
	} `json:"scores"`
}

// ListJudges returns all judge assignments for a hackathon.
func ListJudges(ctx *context.APIContext) {
	judges, err := hackforger_model.ListJudges(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, judges)
}

// AddJudge assigns a user as a judge for a hackathon track.
func AddJudge(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*AddJudgeForm)
	if err := hackforger_model.AddJudge(ctx, ctx.ParamsInt64(":id"), f.TrackID, f.UserID); err != nil {
		if hackforger_model.IsErrDuplicateJudge(err) {
			ctx.Error(http.StatusConflict, "DuplicateJudge", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusCreated)
}

// RemoveJudge removes a judge assignment from a hackathon track.
func RemoveJudge(ctx *context.APIContext) {
	trackID := ctx.FormInt64("track_id")
	if err := hackforger_model.RemoveJudge(ctx, ctx.ParamsInt64(":id"), trackID, ctx.ParamsInt64(":uid")); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// SubmitScore records a judge's scores for all criteria on a submission.
func SubmitScore(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*SubmitScoresForm)
	scores := make([]hackforger_service.CriteriaScore, len(f.Scores))
	for i, s := range f.Scores {
		scores[i] = hackforger_service.CriteriaScore{CriteriaID: s.CriteriaID, Score: s.Score, Comment: s.Comment}
	}
	if err := hackforger_service.SubmitScores(ctx, ctx.Doer.ID, ctx.ParamsInt64(":sid"), scores); err != nil {
		ctx.Error(http.StatusBadRequest, "SubmitScores", err)
		return
	}
	ctx.Status(http.StatusCreated)
}

// ListScores returns all judge scores for a submission.
func ListScores(ctx *context.APIContext) {
	scores, err := hackforger_model.ListScoresBySubmission(ctx, ctx.ParamsInt64(":sid"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, scores)
}

// AddCriteriaForm is the form for creating a new judge criterion.
type AddCriteriaForm struct {
	Name        string  `json:"name" binding:"Required"`
	Description string  `json:"description"`
	MaxScore    float64 `json:"max_score"`
	Weight      float64 `json:"weight"`
	SortOrder   int     `json:"sort_order"`
}

// UpdateCriteriaForm is the form for updating an existing judge criterion.
type UpdateCriteriaForm struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	MaxScore    float64 `json:"max_score"`
	Weight      float64 `json:"weight"`
	SortOrder   int     `json:"sort_order"`
}

// TrackCriteriaOverrideForm is the form for setting a track-level criterion override.
type TrackCriteriaOverrideForm struct {
	Enabled bool    `json:"enabled"`
	Weight  float64 `json:"weight"`
}

// ListCriteria returns all judge criteria for a hackathon.
func ListCriteria(ctx *context.APIContext) {
	criteria, err := hackforger_model.ListCriteriaByHackathon(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, criteria)
}

// AddCriteriaAPI creates a new judge criterion for a hackathon.
func AddCriteriaAPI(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*AddCriteriaForm)
	maxScore := f.MaxScore
	if maxScore <= 0 {
		maxScore = 10
	}
	weight := f.Weight
	if weight <= 0 {
		weight = 25
	}
	if err := hackforger_service.AddCriteria(ctx, ctx.ParamsInt64(":id"), f.Name, f.Description, maxScore, weight, f.SortOrder); err != nil {
		if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Error(http.StatusConflict, "CriteriaLocked", err)
		} else {
			ctx.InternalServerError(err)
		}
		return
	}
	ctx.Status(http.StatusCreated)
}

// UpdateCriteriaAPI updates an existing judge criterion.
func UpdateCriteriaAPI(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*UpdateCriteriaForm)
	c, err := hackforger_model.GetCriteriaByID(ctx, ctx.ParamsInt64(":cid"))
	if err != nil {
		if hackforger_model.IsErrCriteriaNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.InternalServerError(err)
		}
		return
	}
	if f.Name != "" {
		c.Name = f.Name
	}
	if f.Description != "" {
		c.Description = f.Description
	}
	if f.MaxScore > 0 {
		c.MaxScore = f.MaxScore
	}
	if f.Weight > 0 {
		c.Weight = f.Weight
	}
	if f.SortOrder > 0 {
		c.SortOrder = f.SortOrder
	}
	if err := hackforger_service.UpdateCriteria(ctx, c); err != nil {
		if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Error(http.StatusConflict, "CriteriaLocked", err)
		} else {
			ctx.InternalServerError(err)
		}
		return
	}
	ctx.JSON(http.StatusOK, c)
}

// DeleteCriteriaAPI deletes a judge criterion.
func DeleteCriteriaAPI(ctx *context.APIContext) {
	if err := hackforger_service.RemoveCriteria(ctx, ctx.ParamsInt64(":cid")); err != nil {
		if hackforger_model.IsErrCriteriaNotExist(err) {
			ctx.NotFound()
		} else if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Error(http.StatusConflict, "CriteriaLocked", err)
		} else {
			ctx.InternalServerError(err)
		}
		return
	}
	ctx.Status(http.StatusNoContent)
}

// GetTrackEffectiveRubric returns the resolved scoring rubric for a track.
func GetTrackEffectiveRubric(ctx *context.APIContext) {
	rubric, err := hackforger_service.GetEffectiveRubric(ctx, ctx.ParamsInt64(":tid"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, rubric)
}

// SetTrackCriteriaOverrideAPI sets a track-level override for a criterion.
func SetTrackCriteriaOverrideAPI(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*TrackCriteriaOverrideForm)
	if err := hackforger_service.SetTrackCriteriaOverride(ctx, ctx.ParamsInt64(":tid"), ctx.ParamsInt64(":cid"), f.Enabled, f.Weight); err != nil {
		ctx.Error(http.StatusBadRequest, "InvalidOverride", err)
		return
	}
	ctx.Status(http.StatusOK)
}

// FinalizePreviewAPI returns computed rankings without changing hackathon status.
func FinalizePreviewAPI(ctx *context.APIContext) {
	rankings, err := hackforger_service.PreviewFinalize(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Error(http.StatusConflict, "InvalidPhase", err)
		} else {
			ctx.InternalServerError(err)
		}
		return
	}
	ctx.JSON(http.StatusOK, rankings)
}

// FinalizeConfirmAPI locks rankings and transitions hackathon to Finished.
func FinalizeConfirmAPI(ctx *context.APIContext) {
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.InternalServerError(err)
		}
		return
	}
	if err := hackforger_service.ConfirmFinalize(ctx, ctx.Doer.ID, h); err != nil {
		if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Error(http.StatusConflict, "InvalidPhase", err)
		} else {
			ctx.InternalServerError(err)
		}
		return
	}
	ctx.Status(http.StatusOK)
}

// GetLeaderboard returns ranked submissions for a hackathon.
func GetLeaderboard(ctx *context.APIContext) {
	subs, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
		HackathonID: ctx.ParamsInt64(":id"),
	})
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	type entry struct {
		Rank       int     `json:"rank"`
		Title      string  `json:"title"`
		UserID     int64   `json:"user_id"`
		TotalScore float64 `json:"total_score"`
		DemoURL    string  `json:"demo_url"`
	}
	result := make([]entry, 0, len(subs))
	for _, s := range subs {
		result = append(result, entry{
			Rank:       s.Rank,
			Title:      s.Title,
			UserID:     s.UserID,
			TotalScore: s.TotalScore,
			DemoURL:    s.DemoURL,
		})
	}
	ctx.JSON(http.StatusOK, result)
}
