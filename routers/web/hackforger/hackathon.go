// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	hackforger_model "forgejo.org/models/hackforger"
	organization_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
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

	// Build repo name map for submission repo links
	subRepos := make(map[int64]string)
	for _, s := range subs {
		if s.RepoID > 0 {
			if r, err := repo_model.GetRepositoryByID(ctx, s.RepoID); err == nil {
				subRepos[s.RepoID] = r.FullName()
			}
		}
	}
	ctx.Data["SubRepos"] = subRepos

	judges, _ := hackforger_model.ListJudges(ctx, h.ID)
	ctx.Data["JudgeCount"] = len(judges)

	if ctx.Doer != nil {
		ctx.Data["SignedUserID"] = ctx.Doer.ID

		_, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID)
		ctx.Data["IsRegistered"] = err == nil

		ctx.Data["IsOwner"] = ctx.Doer.ID == h.OwnerID

		isJudge, _ := hackforger_model.IsJudgeForAnyTrack(ctx, h.ID, ctx.Doer.ID)
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

	// Load user's repos for the project repo selector
	repos, _, _ := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		Actor:   ctx.Doer,
		OwnerID: ctx.Doer.ID,
		Private: true,
	})
	ctx.Data["UserRepos"] = repos

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
	repoID, _ := strconv.ParseInt(ctx.FormString("repo_id"), 10, 64)
	s := &hackforger_model.HackathonSubmission{
		HackathonID:    h.ID,
		RegistrationID: reg.ID,
		UserID:         ctx.Doer.ID,
		Title:          ctx.FormString("title"),
		Description:    ctx.FormString("description"),
		DemoURL:        ctx.FormString("demo_url"),
		TrackID:        trackID,
		RepoID:         repoID,
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
	criteria, _ := hackforger_model.ListCriteriaByHackathon(ctx, h.ID)
	ctx.Data["Criteria"] = criteria
	trackCriteria := make(map[int64][]*hackforger_model.HackathonTrackCriteria)
	for _, t := range tracks {
		tc, _ := hackforger_model.ListTrackCriteria(ctx, t.ID)
		trackCriteria[t.ID] = tc
	}
	ctx.Data["TrackCriteria"] = trackCriteria
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
		} else if hackforger_model.IsErrNoCriteria(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.no_criteria"))
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
	username := ctx.FormString("username")
	trackID, _ := strconv.ParseInt(ctx.FormString("track_id"), 10, 64)
	u, err := user_model.GetUserByName(ctx, username)
	if err != nil {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.user_not_found", username))
		ctx.Redirect("/hackathon/" + h.Slug + "/manage")
		return
	}
	if err := hackforger_model.AddJudge(ctx, h.ID, trackID, u.ID); err != nil {
		if hackforger_model.IsErrDuplicateJudge(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.duplicate_judge"))
		} else {
			log.Error("AddJudge: %v", err)
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		}
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageJudgeRemovePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	userID := ctx.ParamsInt64(":uid")
	trackID, _ := strconv.ParseInt(ctx.FormString("track_id"), 10, 64)
	if err := hackforger_model.RemoveJudge(ctx, h.ID, trackID, userID); err != nil {
		log.Error("RemoveJudge: %v", err)
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageCriteriaPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	name := ctx.FormString("name")
	description := ctx.FormString("description")
	maxScore, _ := strconv.ParseFloat(ctx.FormString("max_score"), 64)
	weight, _ := strconv.ParseFloat(ctx.FormString("weight"), 64)
	sortOrder, _ := strconv.Atoi(ctx.FormString("sort_order"))
	if maxScore <= 0 {
		maxScore = 10
	}
	if weight <= 0 {
		weight = 25
	}
	if err := hackforger_service.AddCriteria(ctx, h.ID, name, description, maxScore, weight, sortOrder); err != nil {
		if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.criteria_locked"))
		} else {
			log.Error("AddCriteria: %v", err)
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		}
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageCriteriaUpdatePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	cid := ctx.ParamsInt64(":cid")
	c, err := hackforger_model.GetCriteriaByID(ctx, cid)
	if err != nil {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		ctx.Redirect("/hackathon/" + h.Slug + "/manage")
		return
	}
	c.Name = ctx.FormString("name")
	c.Description = ctx.FormString("description")
	c.MaxScore, _ = strconv.ParseFloat(ctx.FormString("max_score"), 64)
	c.Weight, _ = strconv.ParseFloat(ctx.FormString("weight"), 64)
	c.SortOrder, _ = strconv.Atoi(ctx.FormString("sort_order"))
	if err := hackforger_service.UpdateCriteria(ctx, c); err != nil {
		if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.criteria_locked"))
		} else {
			log.Error("UpdateCriteria: %v", err)
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		}
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageCriteriaDeletePost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	cid := ctx.ParamsInt64(":cid")
	if err := hackforger_service.RemoveCriteria(ctx, cid); err != nil {
		if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.criteria_locked"))
		} else {
			log.Error("RemoveCriteria: %v", err)
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		}
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageTrackCriteriaPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	tid := ctx.ParamsInt64(":tid")
	cid, _ := strconv.ParseInt(ctx.FormString("criteria_id"), 10, 64)
	enabled := ctx.FormString("enabled") == "on" || ctx.FormString("enabled") == "true"
	weight, _ := strconv.ParseFloat(ctx.FormString("weight"), 64)
	if err := hackforger_service.SetTrackCriteriaOverride(ctx, tid, cid, enabled, weight); err != nil {
		ctx.Flash.Error(err.Error())
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func FinalizePreview(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	rankings, err := hackforger_service.PreviewFinalize(ctx, h.ID)
	if err != nil {
		if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
		} else {
			log.Error("PreviewFinalize: %v", err)
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		}
		ctx.Redirect("/hackathon/" + h.Slug + "/manage")
		return
	}
	tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	rubrics := make(map[int64][]*hackforger_service.EffectiveCriteria)
	for _, t := range tracks {
		r, _ := hackforger_service.GetEffectiveRubric(ctx, t.ID)
		rubrics[t.ID] = r
	}
	// Load criteria + trackCriteria for the manage page template
	criteria, _ := hackforger_model.ListCriteriaByHackathon(ctx, h.ID)
	judges, _ := hackforger_model.ListJudges(ctx, h.ID)
	judgeNames := make(map[int64]string)
	for _, j := range judges {
		if u, err := user_model.GetUserByID(ctx, j.UserID); err == nil {
			judgeNames[j.UserID] = u.Name
		}
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.manage.finalize_preview")
	ctx.Data["Hackathon"] = h
	ctx.Data["Rankings"] = rankings
	ctx.Data["Tracks"] = tracks
	ctx.Data["Rubrics"] = rubrics
	ctx.Data["Criteria"] = criteria
	ctx.Data["Judges"] = judges
	ctx.Data["JudgeNames"] = judgeNames
	ctx.Data["ShowPreview"] = true
	ctx.HTML(http.StatusOK, tplManage)
}

func FinalizeConfirm(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	h, _ = hackforger_model.GetHackathonByID(ctx, h.ID)
	if err := hackforger_service.ConfirmFinalize(ctx, ctx.Doer.ID, h); err != nil {
		if hackforger_model.IsErrInvalidHackathonPhase(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
		} else {
			log.Error("ConfirmFinalize: %v", err)
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		}
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.hackathon.manage.finalize_success"))
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
	isJudge, _ := hackforger_model.IsJudgeForAnyTrack(ctx, h.ID, ctx.Doer.ID)
	if !isJudge {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.not_judge"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}

	allJudges, _ := hackforger_model.ListJudges(ctx, h.ID)
	allTracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	trackMap := make(map[int64]*hackforger_model.HackathonTrack)
	for _, t := range allTracks {
		trackMap[t.ID] = t
	}

	type trackInfo struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	type subInfo struct {
		ID             int64  `json:"id"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		DemoURL        string `json:"demo_url"`
		ExistingScores []struct {
			CriteriaID int64   `json:"criteria_id"`
			Score      float64 `json:"score"`
			Comment    string  `json:"comment"`
		} `json:"existing_scores"`
	}
	type criteriaInfo struct {
		CriteriaID  int64   `json:"criteria_id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		MaxScore    float64 `json:"max_score"`
		Weight      float64 `json:"weight"`
	}

	var tracks []trackInfo
	submissions := make(map[string][]subInfo)  // trackID as string key for JSON
	rubrics := make(map[string][]criteriaInfo)

	seen := make(map[int64]bool)
	for _, j := range allJudges {
		if j.UserID != ctx.Doer.ID || seen[j.TrackID] {
			continue
		}
		seen[j.TrackID] = true
		t := trackMap[j.TrackID]
		if t == nil {
			continue
		}
		tracks = append(tracks, trackInfo{ID: t.ID, Name: t.Name})

		// Load submissions for this track
		subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
			HackathonID: h.ID, TrackID: t.ID,
		})
		tidStr := strconv.FormatInt(t.ID, 10)
		var subList []subInfo
		for _, s := range subs {
			si := subInfo{ID: s.ID, Title: s.Title, Description: s.Description, DemoURL: s.DemoURL}
			// Load judge's existing scores for this submission
			existingScores, _ := hackforger_model.ListScoresByJudgeAndSubmission(ctx, ctx.Doer.ID, s.ID)
			for _, es := range existingScores {
				si.ExistingScores = append(si.ExistingScores, struct {
					CriteriaID int64   `json:"criteria_id"`
					Score      float64 `json:"score"`
					Comment    string  `json:"comment"`
				}{CriteriaID: es.CriteriaID, Score: es.Score, Comment: es.Comment})
			}
			subList = append(subList, si)
		}
		submissions[tidStr] = subList

		// Load effective rubric
		rubric, _ := hackforger_service.GetEffectiveRubric(ctx, t.ID)
		var critList []criteriaInfo
		for _, c := range rubric {
			critList = append(critList, criteriaInfo{
				CriteriaID: c.CriteriaID, Name: c.Name, Description: c.Description,
				MaxScore: c.MaxScore, Weight: c.Weight,
			})
		}
		rubrics[tidStr] = critList
	}

	// JSON encode for Vue component data attributes
	tracksJSON, _ := json.Marshal(tracks)
	subsJSON, _ := json.Marshal(submissions)
	rubricsJSON, _ := json.Marshal(rubrics)

	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.judge")
	ctx.Data["Hackathon"] = h
	ctx.Data["TracksJSON"] = string(tracksJSON)
	ctx.Data["SubmissionsJSON"] = string(subsJSON)
	ctx.Data["RubricsJSON"] = string(rubricsJSON)
	ctx.HTML(http.StatusOK, tplJudge)
}

func JudgeScoresPost(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		ctx.JSON(http.StatusNotFound, map[string]string{"message": "hackathon not found"})
		return
	}
	sid := ctx.ParamsInt64(":sid")
	type scoreReq struct {
		Scores []struct {
			CriteriaID int64   `json:"criteria_id"`
			Score      float64 `json:"score"`
			Comment    string  `json:"comment"`
		} `json:"scores"`
	}
	var req scoreReq
	if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
		return
	}
	scores := make([]hackforger_service.CriteriaScore, len(req.Scores))
	for i, s := range req.Scores {
		scores[i] = hackforger_service.CriteriaScore{CriteriaID: s.CriteriaID, Score: s.Score, Comment: s.Comment}
	}
	if err := hackforger_service.SubmitScores(ctx, ctx.Doer.ID, sid, scores); err != nil {
		msg := string(ctx.Tr("hackforger.hackathon.error.internal"))
		status := http.StatusInternalServerError
		switch {
		case hackforger_model.IsErrNotJudge(err):
			msg = string(ctx.Tr("hackforger.hackathon.error.not_judge"))
			status = http.StatusForbidden
		case hackforger_service.IsErrIncompleteRubric(err):
			msg = string(ctx.Tr("hackforger.hackathon.error.incomplete_rubric"))
			status = http.StatusBadRequest
		case hackforger_service.IsErrScoreOutOfRange(err):
			e := err.(hackforger_service.ErrScoreOutOfRange)
			msg = string(ctx.Tr("hackforger.hackathon.error.score_out_of_range", fmt.Sprintf("%.0f", e.MaxScore)))
			status = http.StatusBadRequest
		case hackforger_model.IsErrInvalidHackathonPhase(err):
			msg = string(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
			status = http.StatusBadRequest
		}
		ctx.JSON(status, map[string]string{"message": msg})
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func Leaderboard(ctx *context.Context) {
	h := loadHackathon(ctx)
	if h == nil {
		return
	}
	tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	subsByTrack := make(map[int64][]*hackforger_model.HackathonSubmission)
	for _, t := range tracks {
		subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
			HackathonID: h.ID, TrackID: t.ID,
		})
		subsByTrack[t.ID] = subs
	}
	rubrics := make(map[int64][]*hackforger_service.EffectiveCriteria)
	for _, t := range tracks {
		r, _ := hackforger_service.GetEffectiveRubric(ctx, t.ID)
		rubrics[t.ID] = r
	}
	ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.leaderboard")
	ctx.Data["Hackathon"] = h
	ctx.Data["Tracks"] = tracks
	ctx.Data["SubsByTrack"] = subsByTrack
	ctx.Data["Rubrics"] = rubrics
	ctx.Data["ShowBreakdown"] = h.Status == hackforger_model.HackathonStatusFinished
	ctx.HTML(http.StatusOK, tplLeaderboard)
}
