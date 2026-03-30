// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/services/context"

	"xorm.io/builder"
)

// feedItem is the JSON representation of a single feed entry.
type feedItem struct {
	ID        int64       `json:"id"`
	OpType    int         `json:"op_type"`
	OpName    string      `json:"op_name"`
	Actor     *feedActor  `json:"actor"`
	Entity    *feedEntity `json:"entity,omitempty"`
	Message   string      `json:"message,omitempty"`
	CreatedAt int64       `json:"created_at"`
}

type feedActor struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type feedEntity struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

// hackforgerOpTypes returns all HackForger action type values for use in queries.
func hackforgerOpTypes() []int {
	types := make([]int, 0, len(hackforger_model.HackforgerActionTypeName))
	for at := range hackforger_model.HackforgerActionTypeName {
		types = append(types, int(at))
	}
	return types
}

// GetFeed returns the HackForger activity feed.
//
// swagger:operation GET /hackforger/feed hackforger hackforgerGetFeed
// ---
// summary: Get the HackForger activity feed
// produces:
// - application/json
// parameters:
// - name: type
//   in: query
//   description: Feed type (global or following)
//   type: string
//   default: global
// - name: page
//   in: query
//   description: Page number
//   type: integer
//   default: 1
// - name: limit
//   in: query
//   description: Page size
//   type: integer
//   default: 20
// responses:
//   "200":
//     description: Feed items
func GetFeed(ctx *context.APIContext) {
	feedType := ctx.FormString("type")
	if feedType == "" {
		feedType = "global"
	}

	listOpts := db.ListOptions{
		Page:     ctx.FormInt("page"),
		PageSize: ctx.FormInt("limit"),
	}
	if listOpts.Page < 1 {
		listOpts.Page = 1
	}
	if listOpts.PageSize < 1 || listOpts.PageSize > 50 {
		listOpts.PageSize = 20
	}

	opTypes := hackforgerOpTypes()
	cond := builder.In("op_type", opTypes)

	if feedType == "following" && ctx.Doer != nil {
		// Restrict to actions performed by users the doer follows.
		followingCond := builder.In("act_user_id",
			builder.Select("follow_id").From("follow").Where(builder.Eq{"user_id": ctx.Doer.ID}),
		)
		cond = cond.And(followingCond)
	}

	sess := db.GetEngine(ctx).Where(cond)
	sess = db.SetSessionPagination(sess, &listOpts)

	var actions []*activities_model.Action
	count, err := sess.Desc("created_unix").FindAndCount(&actions)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FindAndCount", err)
		return
	}

	// Load act users in bulk.
	actUserIDs := make([]int64, 0, len(actions))
	seen := make(map[int64]bool)
	for _, a := range actions {
		if !seen[a.ActUserID] {
			actUserIDs = append(actUserIDs, a.ActUserID)
			seen[a.ActUserID] = true
		}
	}
	userMap := make(map[int64]*user_model.User)
	if len(actUserIDs) > 0 {
		var users []*user_model.User
		if err := db.GetEngine(ctx).In("id", actUserIDs).Find(&users); err != nil {
			ctx.Error(http.StatusInternalServerError, "FindUsers", err)
			return
		}
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	items := make([]*feedItem, 0, len(actions))
	for _, a := range actions {
		opName := hackforger_model.HackforgerActionTypeName[activities_model.ActionType(a.OpType)]
		item := &feedItem{
			ID:        a.ID,
			OpType:    int(a.OpType),
			OpName:    opName,
			CreatedAt: int64(a.CreatedUnix),
		}

		if u, ok := userMap[a.ActUserID]; ok {
			item.Actor = &feedActor{
				ID:       u.ID,
				Username: u.Name,
			}
		}

		// Parse HackForger content JSON.
		if a.Content != "" {
			var content hackforger_model.HackforgerActionContent
			if err := json.Unmarshal([]byte(a.Content), &content); err == nil {
				item.Entity = &feedEntity{
					Type: content.EntityType,
					ID:   content.EntityID,
					Name: content.EntityName,
					Slug: content.EntitySlug,
				}
			}
		}

		items = append(items, item)
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"items":       items,
		"total_count": count,
	})
}
