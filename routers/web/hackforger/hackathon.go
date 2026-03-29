// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"
	"strconv"
	"strings"

	hackforger_model "forgejo.org/models/hackforger"
	organization_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

const (
	tplExplore          = "hackforger/explore"
	tplExploreHackathon = "hackforger/hackathon/explore"
	tplNew              = "hackforger/hackathon/new"
	tplView        = "hackforger/hackathon/view"
	tplSubmit      = "hackforger/hackathon/submit"
	tplManage      = "hackforger/hackathon/manage"
	tplJudge       = "hackforger/hackathon/judge"
	tplLeaderboard = "hackforger/hackathon/leaderboard"
)

func ExploreHackathons(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.hackathons")
	ctx.Data["PageIsExploreHackathons"] = true

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	keyword := ctx.FormString("q")
	sortType := ctx.FormString("sort")
	if sortType == "" {
		sortType = "newest"
	}
	ctx.Data["Keyword"] = keyword
	ctx.Data["SortType"] = sortType

	opts := hackforger_model.ListHackathonsOptions{
		Keyword: keyword,
		SortBy:  sortType,
	}
	opts.Page = page
	opts.PageSize = 20

	statusFilter := ctx.FormString("status")
	ctx.Data["StatusFilter"] = statusFilter
	ctx.Data["SearchPlaceholderKey"] = "hackforger.hackathon.search.placeholder"

	// Status options for the shared search template (pass i18n keys, not strings)
	type statusOption struct {
		Value    string
		LabelKey string
	}
	ctx.Data["StatusOptions"] = []statusOption{
		{"open", "hackforger.hackathon.status.registration"},
		{"hacking", "hackforger.hackathon.status.hacking"},
		{"judging", "hackforger.hackathon.status.judging"},
		{"finished", "hackforger.hackathon.status.finished"},
	}
	statusLabelKeys := map[string]string{
		"open":     "hackforger.hackathon.status.registration",
		"hacking":  "hackforger.hackathon.status.hacking",
		"judging":  "hackforger.hackathon.status.judging",
		"finished": "hackforger.hackathon.status.finished",
	}
	ctx.Data["StatusLabelKey"] = statusLabelKeys[statusFilter]

	if statusFilter != "" {
		var st hackforger_model.HackathonStatus
		switch statusFilter {
		case "open":
			st = hackforger_model.HackathonStatusOpen
			opts.Status = &st
		case "hacking":
			st = hackforger_model.HackathonStatusHacking
			opts.Status = &st
		case "judging":
			st = hackforger_model.HackathonStatusJudging
			opts.Status = &st
		case "finished":
			st = hackforger_model.HackathonStatusFinished
			opts.Status = &st
		}
	}

	hackathons, total, err := hackforger_model.ListHackathons(ctx, opts)
	if err != nil {
		ctx.ServerError("ListHackathons", err)
		return
	}
	ctx.Data["Hackathons"] = hackathons
	ctx.Data["Total"] = total

	pager := context.NewPagination(int(total), 20, page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	if ctx.Doer != nil {
		ctx.Data["SignedUserID"] = ctx.Doer.ID
	}

	ctx.HTML(http.StatusOK, tplExploreHackathon)
}

// ExploreGrants renders the grants explore page.
func ExploreGrants(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.grants")
	ctx.Data["PageIsExploreGrants"] = true
	ctx.HTML(http.StatusOK, tplExplore)
}

func NewHackathon(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.create")
	ctx.HTML(http.StatusOK, tplNew)
}

func NewHackathonPost(ctx *context.Context) {
	maxTeamSize, _ := strconv.Atoi(ctx.FormString("max_team_size"))
	if maxTeamSize <= 0 {
		maxTeamSize = 5
	}
	orgID, _ := strconv.ParseInt(ctx.FormString("org_id"), 10, 64)
	h := &hackforger_model.Hackathon{
		OrgID:        orgID,
		OwnerID:      ctx.Doer.ID,
		Name:         ctx.FormString("name"),
		Slug:         ctx.FormString("slug"),
		Description:  ctx.FormString("description"),
		PrizeSummary: ctx.FormString("prize_summary"),
		MaxTeamSize:  maxTeamSize,
	}
	if err := hackforger_service.CreateHackathon(ctx, ctx.Doer, h); err != nil {
		ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.create")
		ctx.RenderWithErr(err.Error(), tplNew, nil)
		return
	}
	ctx.Redirect("/hackathon/" + h.Slug)
}

func loadHackathon(ctx *context.Context) *hackforger_model.Hackathon {
	h, err := hackforger_model.GetHackathonBySlug(ctx, ctx.Params(":slug"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound("hackathon not found", nil)
		} else {
			ctx.ServerError("GetHackathonBySlug", err)
		}
		return nil
	}
	return h
}

func ViewHackathon(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	ctx.Data["Title"] = h.Name
	ctx.Data["Hackathon"] = h
	ctx.Data["StatusLabel"] = hackforger_service.HackathonStatusLabel(h.Status)

	// Owner
	if h.OwnerID > 0 {
		if owner, err := user_model.GetUserByID(ctx, h.OwnerID); err == nil {
			ctx.Data["Owner"] = owner
		}
	}
	// Linked Org
	if h.LinkedOrgID > 0 {
		if org, err := user_model.GetUserByID(ctx, h.LinkedOrgID); err == nil {
			ctx.Data["LinkedOrg"] = org
		}
	}

	tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	ctx.Data["Tracks"] = tracks
	trackRepoNames := make(map[int64]string)
	for _, t := range tracks {
		if t.RepoID > 0 {
			if r, err := repo_model.GetRepositoryByID(ctx, t.RepoID); err == nil {
				trackRepoNames[t.ID] = r.Name
			}
		}
	}
	ctx.Data["TrackRepoNames"] = trackRepoNames

	regs, regCount, _ := hackforger_model.ListRegistrations(ctx, hackforger_model.ListRegistrationsOptions{HackathonID: h.ID})
	ctx.Data["Registrations"] = regs
	ctx.Data["RegistrationCount"] = regCount

	subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{HackathonID: h.ID})
	ctx.Data["Submissions"] = subs

	judges, _ := hackforger_model.ListJudges(ctx, h.ID)
	ctx.Data["JudgeCount"] = len(judges)

	if ctx.Doer != nil {
		ctx.Data["SignedUserID"] = ctx.Doer.ID

		_, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID)
		ctx.Data["IsRegistered"] = err == nil

		ctx.Data["IsOwner"] = ctx.Doer.ID == h.OwnerID

		isJudge, _ := hackforger_model.IsJudge(ctx, h.ID, ctx.Doer.ID)
		ctx.Data["IsJudge"] = isJudge

		orgs, _ := organization_model.GetUserOrgsList(ctx, ctx.Doer)
		ctx.Data["UserOrgs"] = orgs
	}
	ctx.HTML(http.StatusOK, tplView)
}

func RegisterPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	// Check duplicate registration first (more specific error)
	if _, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID); err == nil {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.already_registered"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}

	if h.Status != hackforger_model.HackathonStatusOpen {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}

	orgID, _ := strconv.ParseInt(ctx.FormString("org_id"), 10, 64)
	teamName := ctx.FormString("team_name")

	if orgID > 0 {
		isMember, err := organization_model.IsOrganizationMember(ctx, orgID, ctx.Doer.ID)
		if err != nil || !isMember {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.not_org_member"))
			ctx.Redirect("/hackathon/" + h.Slug)
			return
		}
		if org, err := organization_model.GetOrgByID(ctx, orgID); err == nil && teamName == "" {
			teamName = org.Name
		}
	} else if teamName == "" {
		teamName = ctx.Doer.Name
	}

	r := &hackforger_model.HackathonRegistration{
		HackathonID: h.ID,
		UserID:      ctx.Doer.ID,
		OrgID:       orgID,
		TeamName:    teamName,
	}
	if err := hackforger_model.CreateRegistration(ctx, r); err != nil {
		if hackforger_model.IsErrDuplicateRegistration(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.already_registered"))
		} else {
			ctx.Flash.Error(err.Error())
		}
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}

	// Add user to hackathon org
	if h.LinkedOrgID > 0 {
		_ = organization_model.AddOrgUser(ctx, h.LinkedOrgID, ctx.Doer.ID)
	}

	// Publish registered feed event
	_ = hackforger_service.PublishHackforgerAction(ctx, &hackforger_service.HackforgerActionOpts{
		ActUserID:    ctx.Doer.ID,
		OpType:       hackforger_model.ActionHackathonRegistered,
		AudienceType: hackforger_service.AudienceFollowers,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon",
			EntityID:   h.ID,
			EntityName: h.Name,
			EntitySlug: h.Slug,
		},
	})

	ctx.Flash.Success(ctx.Tr("hackforger.hackathon.register.success"))
	ctx.Redirect("/hackathon/" + h.Slug)
}

func SubmitForm(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	if h.Status != hackforger_model.HackathonStatusHacking {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.submit")
	ctx.Data["Hackathon"] = h
	tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	ctx.Data["Tracks"] = tracks
	ctx.HTML(http.StatusOK, tplSubmit)
}

func SubmitPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	if h.Status != hackforger_model.HackathonStatusHacking {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}
	reg, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID)
	if err != nil {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.no_permission"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}
	trackID, _ := strconv.ParseInt(ctx.FormString("track_id"), 10, 64)
	s := &hackforger_model.HackathonSubmission{
		HackathonID:    h.ID,
		RegistrationID: reg.ID,
		UserID:         ctx.Doer.ID,
		Title:          ctx.FormString("title"),
		Description:    ctx.FormString("description"),
		DemoURL:        ctx.FormString("demo_url"),
		TrackID:        trackID,
		Status:         hackforger_model.SubmissionStatusSubmitted,
	}
	if err := hackforger_service.CreateSubmission(ctx, ctx.Doer, h, s); err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect("/hackathon/" + h.Slug + "/submit")
		return
	}
	ctx.Flash.Success(ctx.Tr("hackforger.hackathon.submit.success"))
	ctx.Redirect("/hackathon/" + h.Slug)
}

func ManageHackathon(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.manage")
	ctx.Data["Hackathon"] = h
	ctx.Data["StatusLabel"] = hackforger_service.HackathonStatusLabel(h.Status)
	tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	ctx.Data["Tracks"] = tracks
	regs, _, _ := hackforger_model.ListRegistrations(ctx, hackforger_model.ListRegistrationsOptions{HackathonID: h.ID})
	ctx.Data["Registrations"] = regs
	judges, _ := hackforger_model.ListJudges(ctx, h.ID)
	ctx.Data["Judges"] = judges
	judgeNames := make(map[int64]string)
	for _, j := range judges {
		if u, err := user_model.GetUserByID(ctx, j.UserID); err == nil {
			judgeNames[j.UserID] = u.Name
		}
	}
	ctx.Data["JudgeNames"] = judgeNames
	ctx.HTML(http.StatusOK, tplManage)
}

func ManagePhasePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	h, err := hackforger_model.GetHackathonByID(ctx, h.ID)
	if err != nil {
		ctx.ServerError("GetHackathonByID", err)
		return
	}
	path := ctx.Req.URL.Path
	segments := strings.Split(strings.TrimRight(path, "/"), "/")
	action := segments[len(segments)-1]
	switch action {
	case "publish":
		err = hackforger_service.PublishHackathon(ctx, ctx.Doer.ID, h)
	case "start":
		err = hackforger_service.StartHacking(ctx, ctx.Doer.ID, h)
	case "start-judging":
		err = hackforger_service.StartJudging(ctx, ctx.Doer.ID, h)
	case "finalize":
		err = hackforger_service.FinalizeHackathon(ctx, ctx.Doer.ID, h)
	case "cancel":
		err = hackforger_service.CancelHackathon(ctx, ctx.Doer.ID, h)
	default:
		ctx.Flash.Error("Unknown action")
		ctx.Redirect("/hackathon/" + h.Slug + "/manage")
		return
	}
	if err != nil {
		if hackforger_service.IsErrNoTracks(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.no_tracks"))
		} else if hackforger_service.IsErrNoSubmissions(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.no_submissions"))
		} else {
			ctx.Flash.Error(err.Error())
		}
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageTrackPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	t := &hackforger_model.HackathonTrack{HackathonID: h.ID, Name: ctx.FormString("name")}
	if err := hackforger_service.CreateTrackWithRepo(ctx, ctx.Doer, h, t); err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageRegistrationPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	rid := ctx.ParamsInt64(":rid")
	status, _ := strconv.Atoi(ctx.FormString("status"))
	if err := hackforger_model.UpdateRegistrationStatus(ctx, rid, hackforger_model.RegistrationStatus(status)); err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageJudgePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	userID, _ := strconv.ParseInt(ctx.FormString("user_id"), 10, 64)
	if err := hackforger_model.AddJudge(ctx, h.ID, userID); err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageJudgeRemovePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	userID := ctx.ParamsInt64(":uid")
	if err := hackforger_model.RemoveJudge(ctx, h.ID, userID); err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func JudgePage(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	if h.Status != hackforger_model.HackathonStatusJudging {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.judge")
	ctx.Data["Hackathon"] = h
	subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{HackathonID: h.ID})
	ctx.Data["Submissions"] = subs
	ctx.HTML(http.StatusOK, tplJudge)
}

func JudgeScorePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	sid := ctx.ParamsInt64(":sid")
	score, _ := strconv.ParseFloat(ctx.FormString("score"), 64)
	comment := ctx.FormString("comment")
	if err := hackforger_service.SubmitScore(ctx, ctx.Doer.ID, sid, score, comment); err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success("Score submitted")
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/judge")
}

func Leaderboard(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.leaderboard")
	ctx.Data["Hackathon"] = h
	subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{HackathonID: h.ID})
	ctx.Data["Submissions"] = subs
	ctx.HTML(http.StatusOK, tplLeaderboard)
}
