// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// GetUserReputation returns the reputation record for a user.
//
// swagger:operation GET /hackforger/reputation/users/{username} hackforger hackforgerGetUserReputation
// ---
// summary: Get reputation for a user
// produces:
// - application/json
// parameters:
// - name: username
//   in: path
//   description: Username of the user
//   type: string
//   required: true
// responses:
//   "200":
//     description: Reputation record
//   "404":
//     description: User not found
func GetUserReputation(ctx *context.APIContext) {
	username := ctx.Params(":username")
	u, err := user_model.GetUserByName(ctx, username)
	if err != nil {
		ctx.NotFound("GetUserByName", err)
		return
	}
	rep, err := hackforger_model.GetOrCreateReputation(ctx, u.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetOrCreateReputation", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]any{
		"user_id":            u.ID,
		"username":           u.Name,
		"score":              rep.Score,
		"tier":               rep.Tier,
		"bounties_completed": rep.BountiesCompleted,
		"hackathon_wins":     rep.HackathonWins,
		"grants_received":    rep.GrantsReceived,
		"total_stars":        rep.TotalStars,
		"credits_earned":     rep.TotalCreditsEarned,
	})
}

// GetReputationLeaderboard returns the top users by reputation score.
//
// swagger:operation GET /hackforger/reputation/leaderboard hackforger hackforgerGetReputationLeaderboard
// ---
// summary: Get the reputation leaderboard
// produces:
// - application/json
// parameters:
// - name: limit
//   in: query
//   description: Number of results (1-100, default 50)
//   type: integer
//   default: 50
// responses:
//   "200":
//     description: List of top users by score
func GetReputationLeaderboard(ctx *context.APIContext) {
	limit := ctx.FormInt("limit")
	if limit < 1 || limit > 100 {
		limit = 50
	}
	records, err := hackforger_model.ReputationLeaderboard(ctx, limit)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ReputationLeaderboard", err)
		return
	}

	userIDs := make([]int64, 0, len(records))
	for _, r := range records {
		userIDs = append(userIDs, r.UserID)
	}
	userMap := make(map[int64]*user_model.User)
	if len(userIDs) > 0 {
		var users []*user_model.User
		if err := db.GetEngine(ctx).In("id", userIDs).Find(&users); err == nil {
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
	}

	items := make([]map[string]any, 0, len(records))
	for _, r := range records {
		username := ""
		if u, ok := userMap[r.UserID]; ok {
			username = u.Name
		}
		items = append(items, map[string]any{
			"user_id":  r.UserID,
			"username": username,
			"score":    r.Score,
			"tier":     r.Tier,
		})
	}
	ctx.JSON(http.StatusOK, items)
}

// AdminRecalculateReputation triggers a reputation recalculation for a user.
//
// swagger:operation POST /hackforger/reputation/recalculate/{username} hackforger hackforgerAdminRecalculateReputation
// ---
// summary: Recalculate reputation for a user (admin only)
// produces:
// - application/json
// parameters:
// - name: username
//   in: path
//   description: Username of the user
//   type: string
//   required: true
// responses:
//   "200":
//     description: Recalculation succeeded
//   "404":
//     description: User not found
func AdminRecalculateReputation(ctx *context.APIContext) {
	username := ctx.Params(":username")
	u, err := user_model.GetUserByName(ctx, username)
	if err != nil {
		ctx.NotFound("GetUserByName", err)
		return
	}
	if err := hackforger_service.RecalculateReputation(ctx, u.ID); err != nil {
		ctx.Error(http.StatusInternalServerError, "RecalculateReputation", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
