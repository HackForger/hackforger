// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
)

// GetFeed returns the hackforger activity feed.
func GetFeed(ctx *context.APIContext) {
	feedType := ctx.FormString("type")
	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}
	limit := ctx.FormInt("limit")
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// Build audience filter condition.
	audienceCond := "op_type >= 30"
	var audienceArgs []any

	switch feedType {
	case "global":
		audienceCond += " AND user_id = 0"
	case "following":
		if ctx.Doer != nil {
			audienceCond += " AND (user_id = ? OR user_id = 0)"
			audienceArgs = append(audienceArgs, ctx.Doer.ID)
		} else {
			audienceCond += " AND user_id = 0"
		}
	default:
		if ctx.Doer != nil {
			audienceCond += " AND (user_id = ? OR user_id = 0)"
			audienceArgs = append(audienceArgs, ctx.Doer.ID)
		} else {
			audienceCond += " AND user_id = 0"
		}
	}

	// Count total matching actions.
	total, _ := db.GetEngine(ctx).Where(audienceCond, audienceArgs...).Count(new(activities_model.Action))

	// Fetch actions.
	var actions []*activities_model.Action
	err := db.GetEngine(ctx).Where(audienceCond, audienceArgs...).
		OrderBy("created_unix DESC").
		Limit(limit, (page-1)*limit).
		Find(&actions)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetFeed", err)
		return
	}

	// Transform to response format.
	type feedItem struct {
		ID        int64  `json:"id"`
		OpType    int    `json:"op_type"`
		OpName    string `json:"op_name"`
		ActUserID int64  `json:"act_user_id"`
		RepoID    int64  `json:"repo_id"`
		Content   string `json:"content"`
		CreatedAt int64  `json:"created_at"`
	}

	items := make([]feedItem, 0, len(actions))
	for _, a := range actions {
		name := hackforger_model.HackforgerActionTypeName[a.OpType]
		items = append(items, feedItem{
			ID:        a.ID,
			OpType:    int(a.OpType),
			OpName:    name,
			ActUserID: a.ActUserID,
			RepoID:    a.RepoID,
			Content:   a.Content,
			CreatedAt: int64(a.CreatedUnix),
		})
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"items":       items,
		"total_count": total,
	})
}
