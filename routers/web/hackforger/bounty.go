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
	"forgejo.org/modules/timeutil"
	"forgejo.org/services/context"
	hackforger_svc "forgejo.org/services/hackforger"
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
	ctx.Data["StatusFilter"] = ctx.FormString("status")

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
	_ = hackforger_svc.PublishHackforgerAction(ctx, &hackforger_svc.HackforgerActionOpts{
		ActUserID:    ctx.Doer.ID,
		OpType:       hackforger_model.ActionBountyCreated,
		RepoID:       bounty.RepoID,
		Content:      &hackforger_model.HackforgerActionContent{EntityType: "bounty", EntityID: bounty.ID, EntityName: bounty.Title},
		AudienceType: hackforger_svc.AudienceGlobal | hackforger_svc.AudienceRepoWatchers,
	})

	ctx.Flash.Success("Bounty created successfully")
	ctx.Redirect(fmt.Sprintf("%s/issues/%d", ctx.Repo.RepoLink, issue.Index))
}
