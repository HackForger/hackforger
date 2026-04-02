// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/services/context"
)

const tplExploreSubmissions = "hackforger/submission/explore"

// SubmissionView is a presentation-layer struct for the explore template.
type SubmissionView struct {
	*hackforger_model.HackathonSubmission
	AuthorName   string
	AuthorAvatar string
	HackathonName   string
	HackathonSlug   string
	HackathonStatus hackforger_model.HackathonStatus
	TrackName       string
}

// ExploreSubmissions renders the Explore Submissions page.
func ExploreSubmissions(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.submissions")
	ctx.Data["PageIsExploreSubmissions"] = true

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	keyword := ctx.FormTrim("q")
	sortType := ctx.FormString("sort")
	if sortType == "" {
		sortType = "newest"
	}
	ctx.Data["Keyword"] = keyword
	ctx.Data["SortType"] = sortType
	ctx.Data["SearchPlaceholderKey"] = "hackforger.explore.submissions"

	opts := hackforger_model.ListSubmissionsOptions{
		Keyword: keyword,
		SortBy:  sortType,
	}
	opts.Page = page
	opts.PageSize = 20

	// Only show submitted (non-draft) submissions in explore
	submittedStatus := hackforger_model.SubmissionStatusSubmitted
	opts.Status = &submittedStatus

	submissions, total, err := hackforger_model.ListSubmissions(ctx, opts)
	if err != nil {
		ctx.ServerError("ListSubmissions", err)
		return
	}

	// Build view models with loaded relations
	views := make([]SubmissionView, 0, len(submissions))

	// Batch-collect unique IDs to minimize queries
	hackathonIDs := make(map[int64]bool)
	trackIDs := make(map[int64]bool)
	userIDs := make(map[int64]bool)
	for _, s := range submissions {
		hackathonIDs[s.HackathonID] = true
		if s.TrackID > 0 {
			trackIDs[s.TrackID] = true
		}
		userIDs[s.UserID] = true
	}

	// Load hackathons
	hackathonMap := make(map[int64]*hackforger_model.Hackathon)
	for id := range hackathonIDs {
		h, err := hackforger_model.GetHackathonByID(ctx, id)
		if err != nil {
			log.Error("GetHackathonByID(%d): %v", id, err)
			continue
		}
		hackathonMap[id] = h
	}

	// Load tracks
	trackMap := make(map[int64]*hackforger_model.HackathonTrack)
	for id := range trackIDs {
		t, err := hackforger_model.GetTrackByID(ctx, id)
		if err != nil {
			log.Error("GetTrackByID(%d): %v", id, err)
			continue
		}
		trackMap[id] = t
	}

	// Load users
	userMap := make(map[int64]*user_model.User)
	for id := range userIDs {
		u, err := user_model.GetUserByID(ctx, id)
		if err != nil {
			log.Error("GetUserByID(%d): %v", id, err)
			continue
		}
		userMap[id] = u
	}

	for _, s := range submissions {
		v := SubmissionView{
			HackathonSubmission: s,
		}
		if h, ok := hackathonMap[s.HackathonID]; ok {
			v.HackathonName = h.Name
			v.HackathonSlug = h.Slug
			v.HackathonStatus = h.Status
		}
		if t, ok := trackMap[s.TrackID]; ok {
			v.TrackName = t.Name
		}
		if u, ok := userMap[s.UserID]; ok {
			v.AuthorName = u.Name
			v.AuthorAvatar = u.AvatarLink(ctx)
		}
		views = append(views, v)
	}

	ctx.Data["Submissions"] = views
	ctx.Data["Total"] = total

	pager := context.NewPagination(int(total), 20, page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplExploreSubmissions)
}
