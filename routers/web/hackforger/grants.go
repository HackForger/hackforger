// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"
	"net/http"
	"strconv"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	org_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/timeutil"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

const (
	tplGrantExplore       base.TplName = "hackforger/grants/explore"
	tplGrantNew           base.TplName = "hackforger/grants/new"
	tplGrantDetail        base.TplName = "hackforger/grants/detail"
	tplGrantProjects      base.TplName = "hackforger/grants/projects"
	tplGrantSubmit        base.TplName = "hackforger/grants/submit"
	tplGrantManage        base.TplName = "hackforger/grants/manage"
	tplGrantManageProject base.TplName = "hackforger/grants/manage_project"
)

// ExploreGrants renders the grants explore page with list of grant rounds.
func ExploreGrants(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.grants")
	ctx.Data["PageIsExploreGrants"] = true

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

	var orderBy string
	switch sortType {
	case "oldest":
		orderBy = "created_unix ASC"
	case "alphabetically":
		orderBy = "name ASC"
	default:
		sortType = "newest"
		orderBy = "created_unix DESC"
	}

	// Status filter
	statusFilterStr := ctx.FormString("status")
	ctx.Data["StatusFilter"] = statusFilterStr
	ctx.Data["SearchPlaceholderKey"] = "hackforger.grant.search_placeholder"

	type statusOption struct {
		Value    string
		LabelKey string
	}
	ctx.Data["StatusOptions"] = []statusOption{
		{"draft", "hackforger.grant.round.status.draft"},
		{"open", "hackforger.grant.round.status.open"},
		{"review", "hackforger.grant.round.status.review"},
		{"finalized", "hackforger.grant.round.status.finalized"},
		{"distributed", "hackforger.grant.round.status.distributed"},
		{"cancelled", "hackforger.grant.round.status.cancelled"},
	}
	statusLabelKeys := map[string]string{
		"draft":       "hackforger.grant.round.status.draft",
		"open":        "hackforger.grant.round.status.open",
		"review":      "hackforger.grant.round.status.review",
		"finalized":   "hackforger.grant.round.status.finalized",
		"distributed": "hackforger.grant.round.status.distributed",
		"cancelled":   "hackforger.grant.round.status.cancelled",
	}
	ctx.Data["StatusLabelKey"] = statusLabelKeys[statusFilterStr]

	var statusFilter *hackforger_model.GrantRoundStatus
	if statusFilterStr != "" {
		switch statusFilterStr {
		case "draft":
			st := hackforger_model.GrantRoundStatusDraft
			statusFilter = &st
		case "open":
			st := hackforger_model.GrantRoundStatusOpen
			statusFilter = &st
		case "review":
			st := hackforger_model.GrantRoundStatusReview
			statusFilter = &st
		case "finalized":
			st := hackforger_model.GrantRoundStatusFinalized
			statusFilter = &st
		case "distributed":
			st := hackforger_model.GrantRoundStatusDistributed
			statusFilter = &st
		case "cancelled":
			st := hackforger_model.GrantRoundStatusCancelled
			statusFilter = &st
		}
	}

	rounds, total, err := hackforger_model.ListGrantRounds(ctx, hackforger_model.ListGrantRoundsOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: 20},
		Status:      statusFilter,
		Keyword:     keyword,
		OrderBy:     orderBy,
	})
	if err != nil {
		ctx.ServerError("ListGrantRounds", err)
		return
	}

	ctx.Data["Grants"] = rounds
	ctx.Data["Total"] = total

	pager := context.NewPagination(int(total), 20, page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplGrantExplore)
}

// NewGrantRound renders the form to create a new grant round.
func NewGrantRound(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.grant.round.new")
	ctx.Data["PageIsExploreGrants"] = true
	ctx.HTML(http.StatusOK, tplGrantNew)
}

// NewGrantRoundPost handles the POST to create a new grant round.
func NewGrantRoundPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.grant.round.new")
	ctx.Data["PageIsExploreGrants"] = true

	budget, _ := strconv.ParseFloat(ctx.Req.FormValue("budget"), 64)
	budgetCredits, _ := strconv.ParseInt(ctx.Req.FormValue("budget_credits"), 10, 64)
	orgID, _ := strconv.ParseInt(ctx.Req.FormValue("org_id"), 10, 64)
	deadlineUnix, _ := strconv.ParseInt(ctx.Req.FormValue("deadline"), 10, 64)

	name := ctx.Req.FormValue("name")
	slug := ctx.Req.FormValue("slug")
	description := ctx.Req.FormValue("description")
	currency := ctx.Req.FormValue("currency")
	if currency == "" {
		currency = "USD"
	}

	if name == "" || slug == "" {
		ctx.Flash.Error(ctx.Tr("hackforger.grant.round.name_required"))
		ctx.HTML(http.StatusOK, tplGrantNew)
		return
	}

	if orgID <= 0 {
		// Use the doer's ID as the org (personal grant round).
		orgID = ctx.Doer.ID
	}

	round, err := hackforger_service.CreateGrantRound(ctx, ctx.Doer.ID, orgID, hackforger_service.CreateGrantRoundOpts{
		Name:          name,
		Slug:          slug,
		Description:   description,
		Budget:        budget,
		Currency:      currency,
		BudgetCredits: budgetCredits,
		Deadline:      timeutil.TimeStamp(deadlineUnix),
	})
	if err != nil {
		if hackforger_model.IsErrGrantRoundSlugExists(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.grant.round.slug_exists"))
			ctx.HTML(http.StatusOK, tplGrantNew)
			return
		}
		ctx.ServerError("CreateGrantRound", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("hackforger.grant.round.created"))
	ctx.Redirect(fmt.Sprintf("/grants/%s", round.Slug))
}

// GrantRoundDetail renders the detail page for a grant round.
func GrantRoundDetail(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		if hackforger_model.IsErrGrantRoundNotExist(err) {
			ctx.NotFound("GrantRoundDetail", err)
			return
		}
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}

	projectCount, err := hackforger_model.CountGrantProjectsByRound(ctx, round.ID, nil)
	if err != nil {
		ctx.ServerError("CountGrantProjectsByRound", err)
		return
	}

	usedAmount, usedCredits, err := hackforger_model.GetGrantRoundBudgetUsage(ctx, round.ID)
	if err != nil {
		ctx.ServerError("GetGrantRoundBudgetUsage", err)
		return
	}

	isOwner := false
	if ctx.Doer != nil {
		if ctx.Doer.ID == round.OwnerID {
			isOwner = true
		} else {
			isOrgOwner, _ := org_model.IsOrganizationOwner(ctx, round.OrgID, ctx.Doer.ID)
			isOrgAdmin, _ := org_model.IsOrganizationAdmin(ctx, round.OrgID, ctx.Doer.ID)
			isOwner = isOrgOwner || isOrgAdmin
		}
	}

	ctx.Data["Title"] = round.Name
	ctx.Data["PageIsExploreGrants"] = true
	ctx.Data["Round"] = round
	ctx.Data["ProjectCount"] = projectCount
	ctx.Data["UsedAmount"] = usedAmount
	ctx.Data["UsedCredits"] = usedCredits
	ctx.Data["StatusName"] = hackforger_model.GrantRoundStatusNames[round.Status]
	ctx.Data["IsOwner"] = isOwner
	ctx.HTML(http.StatusOK, tplGrantDetail)
}

// GrantRoundProjects renders the projects list for a grant round.
func GrantRoundProjects(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		if hackforger_model.IsErrGrantRoundNotExist(err) {
			ctx.NotFound("GrantRoundProjects", err)
			return
		}
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	projects, total, err := hackforger_model.ListGrantProjectsByRound(ctx, hackforger_model.ListGrantProjectsByRoundOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: 20},
		RoundID:     round.ID,
	})
	if err != nil {
		ctx.ServerError("ListGrantProjectsByRound", err)
		return
	}

	// Load user info for each project
	userMap := make(map[int64]*user_model.User)
	for _, p := range projects {
		if _, ok := userMap[p.UserID]; !ok {
			u, err := user_model.GetUserByID(ctx, p.UserID)
			if err == nil {
				userMap[p.UserID] = u
			}
		}
	}

	// Load repo info for each project that has a linked repo
	repoMap := make(map[int64]*repo_model.Repository)
	for _, p := range projects {
		if p.RepoID > 0 {
			if _, ok := repoMap[p.RepoID]; !ok {
				r, err := repo_model.GetRepositoryByID(ctx, p.RepoID)
				if err == nil {
					repoMap[p.RepoID] = r
				}
			}
		}
	}

	pager := context.NewPagination(int(total), 20, page, 5)
	pager.SetDefaultParams(ctx)

	ctx.Data["Title"] = fmt.Sprintf("%s - %s", round.Name, ctx.Tr("hackforger.grant.projects"))
	ctx.Data["PageIsExploreGrants"] = true
	ctx.Data["Round"] = round
	ctx.Data["Projects"] = projects
	ctx.Data["UserMap"] = userMap
	ctx.Data["RepoMap"] = repoMap
	ctx.Data["Total"] = total
	ctx.Data["Page"] = pager
	ctx.HTML(http.StatusOK, tplGrantProjects)
}

// SubmitGrantProject renders the form to submit a project to a grant round.
func SubmitGrantProject(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		if hackforger_model.IsErrGrantRoundNotExist(err) {
			ctx.NotFound("SubmitGrantProject", err)
			return
		}
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}

	// Load the current user's repos for the dropdown
	repos, _, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		Actor:       ctx.Doer,
		OwnerID:     ctx.Doer.ID,
		Private:     true,
		ListOptions: db.ListOptions{PageSize: 100},
	})
	if err != nil {
		ctx.ServerError("SearchRepository", err)
		return
	}

	ctx.Data["Title"] = ctx.Tr("hackforger.grant.project.submit")
	ctx.Data["PageIsExploreGrants"] = true
	ctx.Data["Round"] = round
	ctx.Data["Repos"] = repos
	ctx.HTML(http.StatusOK, tplGrantSubmit)
}

// loadUserRepos is a helper to load the current user's repos for the submit form dropdown.
func loadUserRepos(ctx *context.Context) (repo_model.RepositoryList, error) {
	repos, _, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		Actor:       ctx.Doer,
		OwnerID:     ctx.Doer.ID,
		Private:     true,
		ListOptions: db.ListOptions{PageSize: 100},
	})
	return repos, err
}

// SubmitGrantProjectPost handles the POST to submit a project.
func SubmitGrantProjectPost(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		if hackforger_model.IsErrGrantRoundNotExist(err) {
			ctx.NotFound("SubmitGrantProjectPost", err)
			return
		}
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}

	ctx.Data["Title"] = ctx.Tr("hackforger.grant.project.submit")
	ctx.Data["PageIsExploreGrants"] = true
	ctx.Data["Round"] = round

	title := ctx.Req.FormValue("title")
	description := ctx.Req.FormValue("description")
	repoID, _ := strconv.ParseInt(ctx.Req.FormValue("repo_id"), 10, 64)

	// Helper to reload repos on validation errors so the dropdown is repopulated
	reloadRepos := func() {
		repos, repoErr := loadUserRepos(ctx)
		if repoErr == nil {
			ctx.Data["Repos"] = repos
		}
	}

	if title == "" {
		ctx.Flash.Error(ctx.Tr("hackforger.grant.project.title_required"))
		reloadRepos()
		ctx.HTML(http.StatusOK, tplGrantSubmit)
		return
	}

	if repoID <= 0 {
		ctx.Flash.Error(ctx.Tr("hackforger.grant.project.repo_required"))
		reloadRepos()
		ctx.HTML(http.StatusOK, tplGrantSubmit)
		return
	}

	// Verify the repo exists and belongs to the current user
	repo, err := repo_model.GetRepositoryByID(ctx, repoID)
	if err != nil || repo.OwnerID != ctx.Doer.ID {
		ctx.Flash.Error(ctx.Tr("hackforger.grant.project.invalid_repo"))
		reloadRepos()
		ctx.HTML(http.StatusOK, tplGrantSubmit)
		return
	}

	_, err = hackforger_service.SubmitProject(ctx, ctx.Doer.ID, round.ID, hackforger_service.SubmitProjectOpts{
		Title:       title,
		Description: description,
		RepoID:      repoID,
	})
	if err != nil {
		if hackforger_model.IsErrGrantProjectAlreadyExists(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.grant.project.already_submitted"))
			ctx.Redirect(fmt.Sprintf("/grants/%s", slug))
			return
		}
		if hackforger_model.IsErrGrantRoundNotOpen(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.grant.round.not_open"))
			ctx.Redirect(fmt.Sprintf("/grants/%s", slug))
			return
		}
		ctx.ServerError("SubmitProject", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("hackforger.grant.project.submitted"))
	ctx.Redirect(fmt.Sprintf("/grants/%s/projects", round.Slug))
}

// ManageGrantRound renders the management panel for a grant round.
func ManageGrantRound(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		if hackforger_model.IsErrGrantRoundNotExist(err) {
			ctx.NotFound("ManageGrantRound", err)
			return
		}
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}

	projects, _, err := hackforger_model.ListGrantProjectsByRound(ctx, hackforger_model.ListGrantProjectsByRoundOptions{
		ListOptions: db.ListOptions{Page: 1, PageSize: 100},
		RoundID:     round.ID,
	})
	if err != nil {
		ctx.ServerError("ListGrantProjectsByRound", err)
		return
	}

	usedAmount, usedCredits, err := hackforger_model.GetGrantRoundBudgetUsage(ctx, round.ID)
	if err != nil {
		ctx.ServerError("GetGrantRoundBudgetUsage", err)
		return
	}

	ctx.Data["Title"] = fmt.Sprintf("%s - %s", ctx.Tr("hackforger.grant.manage"), round.Name)
	ctx.Data["PageIsExploreGrants"] = true
	ctx.Data["Round"] = round
	ctx.Data["Projects"] = projects
	ctx.Data["UsedAmount"] = usedAmount
	ctx.Data["UsedCredits"] = usedCredits
	ctx.Data["StatusName"] = hackforger_model.GrantRoundStatusNames[round.Status]
	ctx.HTML(http.StatusOK, tplGrantManage)
}

// ManageGrantRoundOpen transitions a round to Open status.
func ManageGrantRoundOpen(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}
	if err := hackforger_service.OpenRound(ctx, ctx.Doer.ID, round.ID); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to open round: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.grant.round.opened"))
	}
	ctx.Redirect(fmt.Sprintf("/grants/%s/manage", slug))
}

// ManageGrantRoundClose transitions a round to Review status.
func ManageGrantRoundClose(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}
	if err := hackforger_service.CloseRound(ctx, ctx.Doer.ID, round.ID); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to close round: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.grant.round.closed"))
	}
	ctx.Redirect(fmt.Sprintf("/grants/%s/manage", slug))
}

// ManageGrantRoundFinalize transitions a round to Finalized status.
func ManageGrantRoundFinalize(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}
	if err := hackforger_service.FinalizeRound(ctx, ctx.Doer.ID, round.ID); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to finalize round: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.grant.round.finalized"))
	}
	ctx.Redirect(fmt.Sprintf("/grants/%s/manage", slug))
}

// ManageGrantRoundDistribute distributes all approved projects in the round.
func ManageGrantRoundDistribute(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}
	if err := hackforger_service.DistributeRound(ctx, ctx.Doer.ID, round.ID); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to distribute: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.grant.round.distributed"))
	}
	ctx.Redirect(fmt.Sprintf("/grants/%s/manage", slug))
}

// ManageGrantRoundCancel transitions a round to Cancelled status.
func ManageGrantRoundCancel(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}
	if err := hackforger_service.CancelRound(ctx, ctx.Doer.ID, round.ID); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to cancel round: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.grant.round.cancelled"))
	}
	ctx.Redirect(fmt.Sprintf("/grants/%s/manage", slug))
}

// ManageGrantProject renders the management page for a single project.
func ManageGrantProject(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		if hackforger_model.IsErrGrantRoundNotExist(err) {
			ctx.NotFound("ManageGrantProject", err)
			return
		}
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}

	pid := ctx.ParamsInt64(":pid")
	project, err := hackforger_model.GetGrantProjectByID(ctx, pid)
	if err != nil {
		if hackforger_model.IsErrGrantProjectNotExist(err) {
			ctx.NotFound("ManageGrantProject", err)
			return
		}
		ctx.ServerError("GetGrantProjectByID", err)
		return
	}

	ctx.Data["Title"] = fmt.Sprintf("%s - %s", ctx.Tr("hackforger.grant.manage"), project.Title)
	ctx.Data["PageIsExploreGrants"] = true
	ctx.Data["Round"] = round
	ctx.Data["Project"] = project
	ctx.Data["StatusName"] = hackforger_model.GrantRoundStatusNames[round.Status]
	ctx.HTML(http.StatusOK, tplGrantManageProject)
}

// ManageGrantProjectApprove approves a pending project.
func ManageGrantProjectApprove(ctx *context.Context) {
	slug := ctx.Params(":slug")
	pid := ctx.ParamsInt64(":pid")

	if err := hackforger_service.ApproveProject(ctx, ctx.Doer.ID, pid); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to approve project: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.grant.project.approved"))
	}
	ctx.Redirect(fmt.Sprintf("/grants/%s/manage/projects/%d", slug, pid))
}

// ManageGrantProjectReject rejects a pending project.
func ManageGrantProjectReject(ctx *context.Context) {
	slug := ctx.Params(":slug")
	pid := ctx.ParamsInt64(":pid")

	if err := hackforger_service.RejectProject(ctx, ctx.Doer.ID, pid); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to reject project: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.grant.project.rejected"))
	}
	ctx.Redirect(fmt.Sprintf("/grants/%s/manage/projects/%d", slug, pid))
}

// ManageGrantProjectAward sets the award allocation for a project.
func ManageGrantProjectAward(ctx *context.Context) {
	slug := ctx.Params(":slug")
	pid := ctx.ParamsInt64(":pid")

	amount, _ := strconv.ParseFloat(ctx.Req.FormValue("award_amount"), 64)
	credits, _ := strconv.ParseInt(ctx.Req.FormValue("award_credits"), 10, 64)

	if err := hackforger_service.AllocateAward(ctx, ctx.Doer.ID, pid, amount, credits); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to allocate award: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.grant.project.award_set"))
	}
	ctx.Redirect(fmt.Sprintf("/grants/%s/manage/projects/%d", slug, pid))
}

// ManageGrantProjectDistribute distributes the award for a single project.
func ManageGrantProjectDistribute(ctx *context.Context) {
	slug := ctx.Params(":slug")
	pid := ctx.ParamsInt64(":pid")

	if err := hackforger_service.DistributeProject(ctx, ctx.Doer.ID, pid); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to distribute: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.grant.project.distributed"))
	}
	ctx.Redirect(fmt.Sprintf("/grants/%s/manage/projects/%d", slug, pid))
}

// ExportGrantRoundCSV exports all projects in a round as CSV.
func ExportGrantRoundCSV(ctx *context.Context) {
	slug := ctx.Params(":slug")
	round, err := hackforger_model.GetGrantRoundBySlug(ctx, slug)
	if err != nil {
		if hackforger_model.IsErrGrantRoundNotExist(err) {
			ctx.NotFound("ExportGrantRoundCSV", err)
			return
		}
		ctx.ServerError("GetGrantRoundBySlug", err)
		return
	}

	data, err := hackforger_service.ExportRoundCSV(ctx, round.ID)
	if err != nil {
		ctx.ServerError("ExportRoundCSV", err)
		return
	}

	ctx.Resp.Header().Set("Content-Type", "text/csv")
	ctx.Resp.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-projects.csv", round.Slug))
	ctx.Resp.WriteHeader(http.StatusOK)
	ctx.Resp.Write(data) //nolint:errcheck
}
