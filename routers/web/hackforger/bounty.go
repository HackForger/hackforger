// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"
	"net/http"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	"forgejo.org/services/context"
	hackforger_svc "forgejo.org/services/hackforger"
	notify_service "forgejo.org/services/notify"
)

// BountyView combines a Bounty with display info for templates.
type BountyView struct {
	*hackforger_model.Bounty
	RepoFullName string // "owner/repo"
	IssueIndex   int64  // issue index within repo (for URL)
	IssueLink    string // full link "/owner/repo/issues/N"
}

const tplBountyExplore = "hackforger/bounty/explore"

// ExploreBounties renders the bounty explore page with filters.
func ExploreBounties(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.bounties")
	ctx.Data["PageIsExploreBounties"] = true

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	var statusFilter *hackforger_model.BountyStatus
	if s := ctx.FormString("status"); s != "" {
		switch s {
		case "open":
			st := hackforger_model.BountyStatusOpen
			statusFilter = &st
		case "claimed":
			st := hackforger_model.BountyStatusClaimed
			statusFilter = &st
		case "in_review":
			st := hackforger_model.BountyStatusInReview
			statusFilter = &st
		case "completed":
			st := hackforger_model.BountyStatusCompleted
			statusFilter = &st
		}
	}

	bounties, total, err := hackforger_model.ListBounties(ctx, hackforger_model.ListBountiesOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: 20},
		Status:      statusFilter,
	})
	if err != nil {
		ctx.ServerError("ListBounties", err)
		return
	}

	// Build BountyViews with repo and issue info for display links
	views := make([]*BountyView, 0, len(bounties))
	for _, b := range bounties {
		v := &BountyView{Bounty: b}
		if repo, err := repo_model.GetRepositoryByID(ctx, b.RepoID); err == nil {
			v.RepoFullName = repo.FullName()
			if issue, err := issues_model.GetIssueByID(ctx, b.IssueID); err == nil {
				v.IssueIndex = issue.Index
				v.IssueLink = fmt.Sprintf("/%s/issues/%d", repo.FullName(), issue.Index)
			}
		}
		views = append(views, v)
	}

	ctx.Data["Bounties"] = views
	ctx.Data["Total"] = total

	statusFilterStr := ctx.FormString("status")
	ctx.Data["StatusFilter"] = statusFilterStr
	ctx.Data["Keyword"] = ctx.FormString("q")
	sortType := ctx.FormString("sort")
	if sortType == "" {
		sortType = "newest"
	}
	ctx.Data["SortType"] = sortType
	ctx.Data["SearchPlaceholderKey"] = "hackforger.bounty.search.placeholder"

	type statusOption struct {
		Value    string
		LabelKey string
	}
	ctx.Data["StatusOptions"] = []statusOption{
		{"open", "hackforger.bounty.status.open"},
		{"claimed", "hackforger.bounty.status.claimed"},
		{"in_review", "hackforger.bounty.status.in_review"},
		{"completed", "hackforger.bounty.status.completed"},
	}
	statusLabelKeys := map[string]string{
		"open":      "hackforger.bounty.status.open",
		"claimed":   "hackforger.bounty.status.claimed",
		"in_review": "hackforger.bounty.status.in_review",
		"completed": "hackforger.bounty.status.completed",
	}
	ctx.Data["StatusLabelKey"] = statusLabelKeys[statusFilterStr]

	pager := context.NewPagination(int(total), 20, page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplBountyExplore)
}

const tplBountyNew = "hackforger/bounty/new"

// NewBounty renders the create bounty form.
func NewBounty(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.bounty.new.title")
	ctx.Data["Query"] = map[string]string{
		"issue_id": ctx.FormString("issue_id"),
		"title":    ctx.FormString("title"),
	}
	ctx.HTML(http.StatusOK, tplBountyNew)
}

// NewBountyPost handles the create bounty form submission.
func NewBountyPost(ctx *context.Context) {
	issueIndex := ctx.FormInt64("issue_id")
	if issueIndex <= 0 {
		ctx.Flash.Error("Issue is required")
		ctx.Redirect(ctx.Repo.RepoLink + "/bounties/new")
		return
	}

	// Resolve issue index to database ID
	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, issueIndex)
	if err != nil {
		ctx.Flash.Error("Issue not found")
		ctx.Redirect(ctx.Repo.RepoLink + "/bounties/new")
		return
	}

	mode := hackforger_model.BountyMode(ctx.FormInt("mode"))
	deadline := timeutil.TimeStamp(ctx.FormInt64("deadline"))

	bounty := &hackforger_model.Bounty{
		RepoID:      ctx.Repo.Repository.ID,
		IssueID:     issue.ID, // DB ID, not index
		PublisherID: ctx.Doer.ID,
		Title:       ctx.FormString("title"),
		Mode:        mode,
		Deadline:    deadline,
	}

	if err := hackforger_model.CreateBounty(ctx, bounty); err != nil {
		if hackforger_model.IsErrBountyAlreadyExists(err) {
			ctx.Flash.Error("A bounty already exists for this issue")
			ctx.Redirect(ctx.Repo.RepoLink + "/bounties/new")
			return
		}
		ctx.ServerError("CreateBounty", err)
		return
	}

	// Publish feed event.
	notify_service.HackforgerEntityCreated(ctx, ctx.Doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionBountyCreated,
		EntityType:   "bounty",
		EntityID:     bounty.ID,
		EntityName:   bounty.Title,
		RepoID:       bounty.RepoID,
		AudienceType: notify_service.AudienceGlobal | notify_service.AudienceRepoWatchers,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: bounty.ID, EntityName: bounty.Title},
	})

	ctx.Flash.Success("Bounty created successfully")
	ctx.Redirect(fmt.Sprintf("%s/issues/%d", ctx.Repo.RepoLink, issue.Index))
}

// BountyAction handles POST actions on a bounty (apply, accept, reject, complete, etc.).
func BountyAction(ctx *context.Context) {
	bountyID := ctx.ParamsInt64("bounty_id")
	action := ctx.Params("action")

	var err error
	switch action {
	case "apply":
		message := ctx.FormString("message")
		_, err = hackforger_svc.ApplyForBounty(ctx, bountyID, ctx.Doer.ID, message)
	case "complete":
		err = hackforger_svc.CompleteBounty(ctx, bountyID, ctx.Doer.ID)
	case "reject":
		err = hackforger_svc.RejectDelivery(ctx, bountyID, ctx.Doer.ID)
	case "pay":
		err = hackforger_svc.MarkPaid(ctx, bountyID, ctx.Doer.ID)
	case "deliver":
		err = hackforger_svc.SubmitDelivery(ctx, bountyID, ctx.Doer.ID)
	case "cancel":
		err = hackforger_svc.CancelBounty(ctx, bountyID, ctx.Doer.ID)
	case "review":
		err = hackforger_svc.StartReview(ctx, bountyID, ctx.Doer.ID)
	default:
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "unknown action"})
		return
	}

	if err != nil {
		log.Error("BountyAction: %v", err)
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "internal error"})
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// BountyApplicationAction handles accept/reject of an application.
func BountyApplicationAction(ctx *context.Context) {
	applicationID := ctx.ParamsInt64("application_id")
	action := ctx.FormString("action")

	var err error
	switch action {
	case "accept":
		err = hackforger_svc.AcceptApplication(ctx, applicationID, ctx.Doer.ID)
	case "reject":
		err = hackforger_svc.RejectApplication(ctx, applicationID, ctx.Doer.ID)
	default:
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "unknown action"})
		return
	}

	if err != nil {
		log.Error("BountyAction: %v", err)
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "internal error"})
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// BountySelectWinners handles winner selection for competitive bounties.
func BountySelectWinners(ctx *context.Context) {
	bountyID := ctx.ParamsInt64("bounty_id")

	type winnersForm struct {
		Winners []hackforger_svc.WinnerInput `json:"winners"`
	}
	var form winnersForm
	if err := json.NewDecoder(ctx.Req.Body).Decode(&form); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := hackforger_svc.SelectWinners(ctx, bountyID, ctx.Doer.ID, form.Winners); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "internal error"})
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// BountyListApplications returns applications as JSON for the Vue component, with resolved usernames.
func BountyListApplications(ctx *context.Context) {
	bountyID := ctx.ParamsInt64("bounty_id")
	apps, _, err := hackforger_model.ListBountyApplications(ctx, hackforger_model.ListBountyApplicationsOptions{
		BountyID: bountyID,
	})
	if err != nil {
		log.Error("BountyQuery: %v", err)
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	type appJSON struct {
		ID       int64                              `json:"id"`
		BountyID int64                              `json:"bounty_id"`
		UserID   int64                              `json:"user_id"`
		Username string                             `json:"username"`
		Status   hackforger_model.ApplicationStatus `json:"status"`
		Message  string                             `json:"message"`
	}
	result := make([]appJSON, 0, len(apps))
	for _, a := range apps {
		aj := appJSON{ID: a.ID, BountyID: a.BountyID, UserID: a.UserID, Status: a.Status, Message: a.Message}
		if u, err := user_model.GetUserByID(ctx, a.UserID); err == nil {
			aj.Username = u.Name
		}
		result = append(result, aj)
	}
	ctx.JSON(http.StatusOK, result)
}

// BountyListWinners returns winners as JSON for the Vue component, with resolved usernames.
func BountyListWinners(ctx *context.Context) {
	bountyID := ctx.ParamsInt64("bounty_id")
	winners, err := hackforger_model.ListBountyWinners(ctx, bountyID)
	if err != nil {
		log.Error("BountyQuery: %v", err)
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	type winnerJSON struct {
		ID       int64  `json:"ID"`
		BountyID int64  `json:"BountyID"`
		UserID   int64  `json:"UserID"`
		Username string `json:"Username"`
		Rank     int    `json:"Rank"`
	}
	result := make([]winnerJSON, 0, len(winners))
	for _, w := range winners {
		wj := winnerJSON{ID: w.ID, BountyID: w.BountyID, UserID: w.UserID, Rank: w.Rank}
		if u, err := user_model.GetUserByID(ctx, w.UserID); err == nil {
			wj.Username = u.Name
		}
		result = append(result, wj)
	}
	ctx.JSON(http.StatusOK, result)
}
