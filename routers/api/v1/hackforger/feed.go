// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
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
//   description: Feed type (global, following, entity, user)
//   type: string
//   default: global
// - name: entity_type
//   in: query
//   description: Entity type filter (used with type=entity)
//   type: string
// - name: entity_id
//   in: query
//   description: Entity ID filter (used with type=entity)
//   type: integer
// - name: user_id
//   in: query
//   description: User ID filter (used with type=user)
//   type: integer
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

	var actions []*hackforger_model.HackforgerAction
	var count int64
	var err error

	switch feedType {
	case "global", "following", "entity", "user":
		// valid types, handled below
	default:
		ctx.Error(http.StatusBadRequest, "InvalidFeedType", nil)
		return
	}

	switch feedType {
	case "following":
		if ctx.Doer == nil {
			ctx.Error(http.StatusUnauthorized, "following feed requires authentication", nil)
			return
		}
		actions, count, err = hackforger_model.GetHackforgerFeeds(ctx, hackforger_model.GetHackforgerFeedsOptions{
			ListOptions:   listOpts,
			UserID:        ctx.Doer.ID,
			IncludeGlobal: true,
		})

	case "entity":
		entityType := ctx.FormString("entity_type")
		entityID := ctx.FormInt64("entity_id")
		actions, count, err = hackforger_model.GetEntityTimeline(ctx, entityType, entityID, listOpts)

	case "user":
		userID := ctx.FormInt64("user_id")
		actions, count, err = hackforger_model.GetHackforgerFeeds(ctx, hackforger_model.GetHackforgerFeedsOptions{
			ListOptions: listOpts,
			ActUserID:   userID,
		})

	default: // "global"
		actions, count, err = hackforger_model.GetHackforgerFeeds(ctx, hackforger_model.GetHackforgerFeedsOptions{
			ListOptions:   listOpts,
			IncludeGlobal: true,
		})
	}

	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetHackforgerFeeds", err)
		return
	}

	if err := hackforger_model.LoadActUsers(ctx, actions); err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadActUsers", err)
		return
	}

	items := make([]*feedItem, 0, len(actions))
	for _, a := range actions {
		opName := hackforger_model.HackforgerActionTypeName[a.OpType]
		item := &feedItem{
			ID:        a.ID,
			OpType:    int(a.OpType),
			OpName:    opName,
			CreatedAt: int64(a.CreatedUnix),
		}

		if a.ActUser != nil {
			item.Actor = &feedActor{
				ID:       a.ActUser.ID,
				Username: a.ActUser.Name,
			}
		}

		if a.EntityType != "" {
			item.Entity = &feedEntity{
				Type: a.EntityType,
				ID:   a.EntityID,
				Name: a.EntityName,
				Slug: a.EntitySlug,
			}
		}

		items = append(items, item)
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"items":       items,
		"total_count": count,
	})
}
