// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"
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
//
// swagger:operation GET /hackforger/hackathons/{id}/judges hackforger hackforgerListJudges
// ---
// summary: List judge assignments for a hackathon
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Judge list
func ListJudges(ctx *context.APIContext) {
	judges, err := hackforger_model.ListJudges(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, judges)
}

// AddJudge assigns a user as a judge for a hackathon track.
//
// swagger:operation POST /hackforger/hackathons/{id}/judges hackforger hackforgerAddJudge
// ---
// summary: Add a judge to a hackathon track
// consumes:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/AddJudgeForm"
// responses:
//   "201":
//     description: Judge added
//   "409":
//     description: Duplicate judge assignment
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
//
// swagger:operation DELETE /hackforger/hackathons/{id}/judges/{uid} hackforger hackforgerRemoveJudge
// ---
// summary: Remove a judge from a hackathon track
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: uid
//   in: path
//   description: User ID of the judge
//   type: integer
//   format: int64
//   required: true
// - name: track_id
//   in: query
//   description: Track ID to remove judge from
//   type: integer
//   format: int64
// responses:
//   "204":
//     description: Judge removed
func RemoveJudge(ctx *context.APIContext) {
	trackID := ctx.FormInt64("track_id")
	if err := hackforger_model.RemoveJudge(ctx, ctx.ParamsInt64(":id"), trackID, ctx.ParamsInt64(":uid")); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// SubmitScore records a judge's scores for all criteria on a submission.
//
// swagger:operation POST /hackforger/hackathons/{id}/submissions/{sid}/scores hackforger hackforgerSubmitScore
// ---
// summary: Submit scores for a submission
// consumes:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: sid
//   in: path
//   description: ID of the submission
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/SubmitScoresForm"
// responses:
//   "201":
//     description: Scores submitted
//   "400":
//     description: Invalid scores
func SubmitScore(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*SubmitScoresForm)
	scores := make([]hackforger_service.CriteriaScore, len(f.Scores))
	for i, s := range f.Scores {
		scores[i] = hackforger_service.CriteriaScore{CriteriaID: s.CriteriaID, Score: s.Score, Comment: s.Comment}
	}
	err := hackforger_service.SubmitScores(ctx, ctx.Doer.ID, ctx.ParamsInt64(":sid"), scores)
	if err == nil {
		ctx.Status(http.StatusCreated)
		return
	}
	switch {
	case hackforger_model.IsErrNotJudge(err):
		ctx.Error(http.StatusForbidden, "SubmitScores",
			ctx.Tr("hackforger.hackathon.error.not_judge"))
	case hackforger_service.IsErrNoRubricConfigured(err):
		ctx.Error(http.StatusBadRequest, "SubmitScores",
			ctx.Tr("hackforger.hackathon.error.no_rubric_configured"))
	case hackforger_service.IsErrIncompleteRubric(err):
		ctx.Error(http.StatusBadRequest, "SubmitScores",
			ctx.Tr("hackforger.hackathon.error.incomplete_rubric"))
	case hackforger_service.IsErrScoreOutOfRange(err):
		e := err.(hackforger_service.ErrScoreOutOfRange)
		ctx.Error(http.StatusBadRequest, "SubmitScores",
			ctx.Tr("hackforger.hackathon.error.score_out_of_range", fmt.Sprintf("%.0f", e.MaxScore)))
	case hackforger_model.IsErrInvalidHackathonPhase(err):
		ctx.Error(http.StatusBadRequest, "SubmitScores",
			ctx.Tr("hackforger.hackathon.error.invalid_phase"))
	default:
		ctx.Error(http.StatusBadRequest, "SubmitScores", err)
	}
}

// ListScores returns all judge scores for a submission.
//
// swagger:operation GET /hackforger/hackathons/{id}/submissions/{sid}/scores hackforger hackforgerListScores
// ---
// summary: List scores for a submission
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: sid
//   in: path
//   description: ID of the submission
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Score list
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
//
// swagger:operation GET /hackforger/hackathons/{id}/criteria hackforger hackforgerListCriteria
// ---
// summary: List judge criteria for a hackathon
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Criteria list
func ListCriteria(ctx *context.APIContext) {
	criteria, err := hackforger_model.ListCriteriaByHackathon(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, criteria)
}

// AddCriteriaAPI creates a new judge criterion for a hackathon.
//
// swagger:operation POST /hackforger/hackathons/{id}/criteria hackforger hackforgerAddCriteria
// ---
// summary: Add a judge criterion to a hackathon
// consumes:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/AddCriteriaForm"
// responses:
//   "201":
//     description: Criterion created
//   "409":
//     description: Criteria locked (hackathon not in draft/open)
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
//
// swagger:operation PUT /hackforger/hackathons/{id}/criteria/{cid} hackforger hackforgerUpdateCriteria
// ---
// summary: Update a judge criterion
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: cid
//   in: path
//   description: ID of the criterion
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/UpdateCriteriaForm"
// responses:
//   "200":
//     description: Updated criterion
//   "404":
//     "$ref": "#/responses/notFound"
//   "409":
//     description: Criteria locked
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
//
// swagger:operation DELETE /hackforger/hackathons/{id}/criteria/{cid} hackforger hackforgerDeleteCriteria
// ---
// summary: Delete a judge criterion
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: cid
//   in: path
//   description: ID of the criterion
//   type: integer
//   format: int64
//   required: true
// responses:
//   "204":
//     description: Criterion deleted
//   "404":
//     "$ref": "#/responses/notFound"
//   "409":
//     description: Criteria locked
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
//
// swagger:operation GET /hackforger/hackathons/{id}/tracks/{tid}/rubric hackforger hackforgerGetTrackRubric
// ---
// summary: Get effective scoring rubric for a track
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: tid
//   in: path
//   description: ID of the track
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Effective rubric with criteria and weights
func GetTrackEffectiveRubric(ctx *context.APIContext) {
	rubric, err := hackforger_service.GetEffectiveRubric(ctx, ctx.ParamsInt64(":tid"))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, rubric)
}

// SetTrackCriteriaOverrideAPI sets a track-level override for a criterion.
//
// swagger:operation PUT /hackforger/hackathons/{id}/tracks/{tid}/criteria/{cid} hackforger hackforgerSetTrackCriteriaOverride
// ---
// summary: Set track-level criterion override
// consumes:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: tid
//   in: path
//   description: ID of the track
//   type: integer
//   format: int64
//   required: true
// - name: cid
//   in: path
//   description: ID of the criterion
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/TrackCriteriaOverrideForm"
// responses:
//   "200":
//     description: Override set
//   "400":
//     description: Invalid override
func SetTrackCriteriaOverrideAPI(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*TrackCriteriaOverrideForm)
	if err := hackforger_service.SetTrackCriteriaOverride(ctx, ctx.ParamsInt64(":tid"), ctx.ParamsInt64(":cid"), f.Enabled, f.Weight); err != nil {
		ctx.Error(http.StatusBadRequest, "InvalidOverride", err)
		return
	}
	ctx.Status(http.StatusOK)
}

// FinalizePreviewAPI returns computed rankings without changing hackathon status.
//
// swagger:operation GET /hackforger/hackathons/{id}/finalize/preview hackforger hackforgerFinalizePreview
// ---
// summary: Preview finalization rankings
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Computed rankings preview
//   "409":
//     description: Invalid hackathon phase
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
//
// swagger:operation POST /hackforger/hackathons/{id}/finalize/confirm hackforger hackforgerFinalizeConfirm
// ---
// summary: Confirm finalization and lock rankings
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Hackathon finalized
//   "404":
//     "$ref": "#/responses/notFound"
//   "409":
//     description: Invalid hackathon phase
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
		switch {
		case hackforger_model.IsErrInvalidHackathonPhase(err):
			ctx.Error(http.StatusConflict, "InvalidPhase", err)
		case hackforger_model.IsErrInvalidPrizeDistMode(err):
			ctx.Error(http.StatusUnprocessableEntity, "InvalidPrizeDistMode", err)
		default:
			ctx.InternalServerError(err)
		}
		return
	}
	ctx.Status(http.StatusOK)
}

// GetLeaderboard returns ranked submissions for a hackathon, grouped by track.
//
// swagger:operation GET /hackforger/hackathons/{id}/leaderboard hackforger hackforgerGetLeaderboard
// ---
// summary: Get hackathon leaderboard grouped by track
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// responses:
//   "200":
//     description: Leaderboard with ranked submissions per track
func GetLeaderboard(ctx *context.APIContext) {
	hackathonID := ctx.ParamsInt64(":id")
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, hackathonID)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	type entry struct {
		Rank       int     `json:"rank"`
		Title      string  `json:"title"`
		UserID     int64   `json:"user_id"`
		TrackID    int64   `json:"track_id"`
		TotalScore float64 `json:"total_score"`
		DemoURL    string  `json:"demo_url"`
	}
	type trackLeaderboard struct {
		TrackID   int64   `json:"track_id"`
		TrackName string  `json:"track_name"`
		Entries   []entry `json:"entries"`
	}

	result := make([]trackLeaderboard, 0, len(tracks))
	for _, t := range tracks {
		subs, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
			HackathonID: hackathonID, TrackID: t.ID,
		})
		if err != nil {
			ctx.InternalServerError(err)
			return
		}
		entries := make([]entry, 0, len(subs))
		for _, s := range subs {
			entries = append(entries, entry{
				Rank:       s.Rank,
				Title:      s.Title,
				UserID:     s.UserID,
				TrackID:    s.TrackID,
				TotalScore: s.TotalScore,
				DemoURL:    s.DemoURL,
			})
		}
		result = append(result, trackLeaderboard{
			TrackID:   t.ID,
			TrackName: t.Name,
			Entries:   entries,
		})
	}
	ctx.JSON(http.StatusOK, result)
}
