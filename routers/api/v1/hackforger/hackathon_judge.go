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

// AddJudgeForm is the form for assigning a judge to a hackathon.
type AddJudgeForm struct {
	UserID int64 `json:"user_id" binding:"Required"`
}

// SubmitScoreForm is the form for submitting a judge's score.
type SubmitScoreForm struct {
	Score   float64 `json:"score"`
	Comment string  `json:"comment"`
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

// AddJudge assigns a user as a judge for a hackathon.
func AddJudge(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*AddJudgeForm)
	if err := hackforger_model.AddJudge(ctx, ctx.ParamsInt64(":id"), f.UserID); err != nil {
		if hackforger_model.IsErrDuplicateJudge(err) {
			ctx.Error(http.StatusConflict, "DuplicateJudge", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusCreated)
}

// RemoveJudge removes a judge assignment from a hackathon.
func RemoveJudge(ctx *context.APIContext) {
	if err := hackforger_model.RemoveJudge(ctx, ctx.ParamsInt64(":id"), ctx.ParamsInt64(":uid")); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// SubmitScore records a judge's score for a submission.
func SubmitScore(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*SubmitScoreForm)
	if err := hackforger_service.SubmitScore(ctx, ctx.Doer.ID, ctx.ParamsInt64(":sid"), f.Score, f.Comment); err != nil {
		ctx.Error(http.StatusBadRequest, "SubmitScore", err)
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
