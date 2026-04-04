// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"
	"strconv"

	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	hackforger_svc "forgejo.org/services/hackforger"
	notify_service "forgejo.org/services/notify"
)

// --------------------------------------------------------------------------
// Form structs (bound via bind() in route registration)
// --------------------------------------------------------------------------

// CreateBountyForm is the JSON body for creating a bounty.
type CreateBountyForm struct {
	IssueID  int64  `json:"issue_id" binding:"Required"`
	Title    string `json:"title" binding:"Required"`
	Mode     int    `json:"mode"`
	Deadline int64  `json:"deadline"`
}

// UpdateBountyForm is the JSON body for updating bounty metadata.
type UpdateBountyForm struct {
	Title    string `json:"title" binding:"Required"`
	Deadline int64  `json:"deadline"`
}

// AddRewardForm is the JSON body for adding a reward to a bounty.
type AddRewardForm struct {
	Type     string  `json:"type" binding:"Required"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Credits  int64   `json:"credits"`
	Rank     int     `json:"rank"`
	Note     string  `json:"note"`
}

// ApplyForm is the JSON body for applying to a bounty.
type ApplyForm struct {
	Message string `json:"message"`
}

// ReviewApplicationForm is the JSON body for reviewing an application.
type ReviewApplicationForm struct {
	Action string `json:"action" binding:"Required"`
}

// SelectWinnersForm is the JSON body for selecting bounty winners.
type SelectWinnersForm struct {
	Winners []WinnerEntry `json:"winners" binding:"Required"`
}

// WinnerEntry represents a single winner in the SelectWinnersForm.
type WinnerEntry struct {
	UserID int64 `json:"user_id"`
	Rank   int   `json:"rank"`
}

// --------------------------------------------------------------------------
// Repo-level handlers
// --------------------------------------------------------------------------

// CreateBounty creates a new bounty for a repository.
func CreateBounty(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties bounty bountyCreateBounty
	// ---
	// summary: Create a bounty
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/CreateBountyForm"
	// responses:
	//   "201":
	//     description: "Bounty"
	//     schema:
	//       "$ref": "#/definitions/Bounty"
	//   "409":
	//     description: bounty already exists for this issue
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*CreateBountyForm)

	// Resolve issue index to database ID
	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, form.IssueID)
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.Error(http.StatusNotFound, "GetIssueByIndex", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetIssueByIndex", err)
		return
	}

	bounty := &hackforger_model.Bounty{
		RepoID:      ctx.Repo.Repository.ID,
		IssueID:     issue.ID, // Use DB ID, not index
		PublisherID: ctx.Doer.ID,
		Title:       form.Title,
		Mode:        hackforger_model.BountyMode(form.Mode),
		Deadline:    timeutil.TimeStamp(form.Deadline),
	}

	if err := hackforger_model.CreateBounty(ctx, bounty); err != nil {
		if hackforger_model.IsErrBountyAlreadyExists(err) {
			ctx.Error(http.StatusConflict, "CreateBounty", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "CreateBounty", err)
		return
	}

	// Publish feed event.
	notify_service.HackforgerEntityCreated(ctx, ctx.Doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionBountyCreated,
		EntityType:   "bounty",
		EntityID:     bounty.ID,
		EntityName:   bounty.Title,
		RepoID:       ctx.Repo.Repository.ID,
		AudienceType: notify_service.AudienceGlobal | notify_service.AudienceRepoWatchers,
		Content: &hackforger_model.HackforgerActionContent{
			EntityType: "bounty",
			EntityID:   bounty.ID,
			EntityName: bounty.Title,
		},
	})

	ctx.JSON(http.StatusCreated, bounty)
}

// ListRepoBounties lists bounties of a repository.
func ListRepoBounties(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/bounties bounty bountyListRepoBounties
	// ---
	// summary: List a repository's bounties
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: status
	//   in: query
	//   description: filter by status
	//   type: integer
	// - name: page
	//   in: query
	//   description: page number of results
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     description: "BountyList"
	//     schema:
	//       type: array
	//       items:
	//         "$ref": "#/definitions/Bounty"

	opts := hackforger_model.ListBountiesOptions{
		ListOptions: utils.GetListOptions(ctx),
		RepoID:      ctx.Repo.Repository.ID,
	}

	if statusStr := ctx.FormString("status"); statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err != nil {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		status := hackforger_model.BountyStatus(s)
		opts.Status = &status
	}

	bounties, total, err := hackforger_model.ListBounties(ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListBounties", err)
		return
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, bounties)
}

// GetBounty returns a single bounty.
func GetBounty(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/bounties/{bounty_id} bounty bountyGetBounty
	// ---
	// summary: Get a bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: "Bounty"
	//     schema:
	//       "$ref": "#/definitions/Bounty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	bountyID := ctx.ParamsInt64(":bounty_id")
	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetBountyByID", err)
		return
	}

	ctx.JSON(http.StatusOK, bounty)
}

// UpdateBounty updates bounty metadata.
func UpdateBounty(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/bounties/{bounty_id} bounty bountyUpdateBounty
	// ---
	// summary: Update a bounty
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/UpdateBountyForm"
	// responses:
	//   "200":
	//     description: "Bounty"
	//     schema:
	//       "$ref": "#/definitions/Bounty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*UpdateBountyForm)
	bountyID := ctx.ParamsInt64(":bounty_id")

	if err := hackforger_svc.UpdateBountyMeta(ctx, bountyID, ctx.Doer.ID, form.Title, timeutil.TimeStamp(form.Deadline)); err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_svc.IsErrNotPublisher(err) {
			ctx.Error(http.StatusForbidden, "RequirePublisher", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "UpdateBountyMeta", err)
		return
	}

	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetBountyByID", err)
		return
	}

	ctx.JSON(http.StatusOK, bounty)
}

// DeleteBountyAPI deletes a bounty.
func DeleteBountyAPI(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/bounties/{bounty_id} bounty bountyDeleteBounty
	// ---
	// summary: Delete a bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	bountyID := ctx.ParamsInt64(":bounty_id")

	if err := hackforger_svc.DeleteBounty(ctx, bountyID, ctx.Doer.ID); err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_svc.IsErrNotPublisher(err) {
			ctx.Error(http.StatusForbidden, "RequirePublisher", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		if hackforger_svc.IsErrBountyHasApplications(err) {
			ctx.Error(http.StatusConflict, "BountyHasApplications", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "DeleteBounty", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AddReward adds a reward to a bounty.
func AddReward(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties/{bounty_id}/rewards bounty bountyAddReward
	// ---
	// summary: Add a reward to a bounty
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/AddRewardForm"
	// responses:
	//   "201":
	//     description: "BountyReward"
	//     schema:
	//       "$ref": "#/definitions/BountyReward"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*AddRewardForm)
	bountyID := ctx.ParamsInt64(":bounty_id")

	// Verify bounty exists.
	if _, err := hackforger_model.GetBountyByID(ctx, bountyID); err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetBountyByID", err)
		return
	}

	reward := &hackforger_model.BountyReward{
		BountyID: bountyID,
		Type:     hackforger_model.RewardType(form.Type),
		Amount:   form.Amount,
		Currency: form.Currency,
		Credits:  form.Credits,
		Rank:     form.Rank,
		Note:     form.Note,
	}

	if err := hackforger_model.CreateBountyReward(ctx, reward); err != nil {
		ctx.Error(http.StatusInternalServerError, "CreateBountyReward", err)
		return
	}

	ctx.JSON(http.StatusCreated, reward)
}

// ListRewards lists rewards for a bounty.
func ListRewards(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/bounties/{bounty_id}/rewards bounty bountyListRewards
	// ---
	// summary: List rewards for a bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: "BountyRewardList"
	//     schema:
	//       type: array
	//       items:
	//         "$ref": "#/definitions/BountyReward"
	//   "404":
	//     "$ref": "#/responses/notFound"

	bountyID := ctx.ParamsInt64(":bounty_id")

	rewards, err := hackforger_model.ListBountyRewards(ctx, bountyID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListBountyRewards", err)
		return
	}

	ctx.JSON(http.StatusOK, rewards)
}

// DeleteReward deletes a reward from a bounty.
func DeleteReward(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/bounties/{bounty_id}/rewards/{reward_id} bounty bountyDeleteReward
	// ---
	// summary: Delete a reward from a bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// - name: reward_id
	//   in: path
	//   description: id of the reward
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	rewardID := ctx.ParamsInt64(":reward_id")

	if err := hackforger_model.DeleteBountyReward(ctx, rewardID); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteBountyReward", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ApplyForBounty creates a new application for a bounty.
func ApplyForBounty(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties/{bounty_id}/applications bounty bountyApplyForBounty
	// ---
	// summary: Apply for a bounty
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/ApplyForm"
	// responses:
	//   "201":
	//     description: "BountyApplication"
	//     schema:
	//       "$ref": "#/definitions/BountyApplication"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     description: already applied
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*ApplyForm)
	bountyID := ctx.ParamsInt64(":bounty_id")

	app, err := hackforger_svc.ApplyForBounty(ctx, bountyID, ctx.Doer.ID, form.Message)
	if err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_model.IsErrAlreadyApplied(err) {
			ctx.Error(http.StatusConflict, "AlreadyApplied", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "ApplyForBounty", err)
		return
	}

	ctx.JSON(http.StatusCreated, app)
}

// ListApplications lists applications for a bounty.
func ListApplications(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/bounties/{bounty_id}/applications bounty bountyListApplications
	// ---
	// summary: List applications for a bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     description: "BountyApplicationList"
	//     schema:
	//       type: array
	//       items:
	//         "$ref": "#/definitions/BountyApplication"

	bountyID := ctx.ParamsInt64(":bounty_id")

	apps, total, err := hackforger_model.ListBountyApplications(ctx, hackforger_model.ListBountyApplicationsOptions{
		ListOptions: utils.GetListOptions(ctx),
		BountyID:    bountyID,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListBountyApplications", err)
		return
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, apps)
}

// ReviewApplication accepts or rejects an application.
func ReviewApplication(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/bounties/{bounty_id}/applications/{application_id} bounty bountyReviewApplication
	// ---
	// summary: Accept or reject an application
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// - name: application_id
	//   in: path
	//   description: id of the application
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/ReviewApplicationForm"
	// responses:
	//   "200":
	//     description: "success"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*ReviewApplicationForm)
	applicationID := ctx.ParamsInt64(":application_id")

	var err error
	switch form.Action {
	case "accept":
		err = hackforger_svc.AcceptApplication(ctx, applicationID, ctx.Doer.ID)
	case "reject":
		err = hackforger_svc.RejectApplication(ctx, applicationID, ctx.Doer.ID)
	default:
		ctx.Error(http.StatusUnprocessableEntity, "InvalidAction", nil)
		return
	}

	if err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_svc.IsErrNotPublisher(err) {
			ctx.Error(http.StatusForbidden, "RequirePublisher", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "ReviewApplication", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// StartReviewAPI transitions a competitive bounty from Open to InReview.
func StartReviewAPI(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties/{bounty_id}/review bounty bountyStartReview
	// ---
	// summary: Start review of a competitive bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: "success"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	bountyID := ctx.ParamsInt64(":bounty_id")

	if err := hackforger_svc.StartReview(ctx, bountyID, ctx.Doer.ID); err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_svc.IsErrNotPublisher(err) {
			ctx.Error(http.StatusForbidden, "RequirePublisher", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "StartReview", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// CompleteBountyAPI transitions a bounty from InReview to Completed.
func CompleteBountyAPI(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties/{bounty_id}/complete bounty bountyComplete
	// ---
	// summary: Complete a bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: "success"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	bountyID := ctx.ParamsInt64(":bounty_id")

	if err := hackforger_svc.CompleteBounty(ctx, bountyID, ctx.Doer.ID); err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_svc.IsErrNotPublisher(err) {
			ctx.Error(http.StatusForbidden, "RequirePublisher", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "CompleteBounty", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// RejectDeliveryAPI transitions an exclusive bounty from InReview back to Claimed.
func RejectDeliveryAPI(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties/{bounty_id}/reject bounty bountyRejectDelivery
	// ---
	// summary: Reject a bounty delivery
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: "success"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	bountyID := ctx.ParamsInt64(":bounty_id")

	if err := hackforger_svc.RejectDelivery(ctx, bountyID, ctx.Doer.ID); err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_svc.IsErrNotPublisher(err) {
			ctx.Error(http.StatusForbidden, "RequirePublisher", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "RejectDelivery", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// MarkPaidAPI transitions a bounty from Completed to Paid.
func MarkPaidAPI(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties/{bounty_id}/pay bounty bountyMarkPaid
	// ---
	// summary: Mark a bounty as paid
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: "success"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	bountyID := ctx.ParamsInt64(":bounty_id")

	if err := hackforger_svc.MarkPaid(ctx, bountyID, ctx.Doer.ID); err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_svc.IsErrNotPublisher(err) {
			ctx.Error(http.StatusForbidden, "RequirePublisher", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "MarkPaid", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// CancelBountyAPI cancels a bounty.
func CancelBountyAPI(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties/{bounty_id}/cancel bounty bountyCancelBounty
	// ---
	// summary: Cancel a bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: "success"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	bountyID := ctx.ParamsInt64(":bounty_id")

	if err := hackforger_svc.CancelBounty(ctx, bountyID, ctx.Doer.ID); err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_svc.IsErrNotPublisher(err) {
			ctx.Error(http.StatusForbidden, "RequirePublisher", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "CancelBounty", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ExpireBountyAPI runs the expired bounties check.
func ExpireBountyAPI(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties/{bounty_id}/expire bounty bountyExpire
	// ---
	// summary: Expire a bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: "success"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	bountyID := ctx.ParamsInt64(":bounty_id")

	bounty, err := hackforger_model.GetBountyByID(ctx, bountyID)
	if err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetBountyByID", err)
		return
	}

	if err := hackforger_svc.ExpireBounty(ctx, bounty); err != nil {
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "ExpireBounty", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// SelectWinnersAPI selects winners for a competitive bounty.
func SelectWinnersAPI(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/bounties/{bounty_id}/winners bounty bountySelectWinners
	// ---
	// summary: Select winners for a competitive bounty
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/SelectWinnersForm"
	// responses:
	//   "200":
	//     description: "success"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*SelectWinnersForm)
	bountyID := ctx.ParamsInt64(":bounty_id")

	winners := make([]hackforger_svc.WinnerInput, len(form.Winners))
	for i, w := range form.Winners {
		winners[i] = hackforger_svc.WinnerInput{
			UserID: w.UserID,
			Rank:   w.Rank,
		}
	}

	if err := hackforger_svc.SelectWinners(ctx, bountyID, ctx.Doer.ID, winners); err != nil {
		if hackforger_model.IsErrBountyNotExist(err) {
			ctx.NotFound()
			return
		}
		if hackforger_svc.IsErrNotPublisher(err) {
			ctx.Error(http.StatusForbidden, "RequirePublisher", err)
			return
		}
		if hackforger_svc.IsErrInvalidBountyStatus(err) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "SelectWinners", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ListWinnersAPI lists winners for a bounty.
func ListWinnersAPI(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/bounties/{bounty_id}/winners bounty bountyListWinners
	// ---
	// summary: List winners for a bounty
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: bounty_id
	//   in: path
	//   description: id of the bounty
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     description: "BountyWinnerList"
	//     schema:
	//       type: array
	//       items:
	//         "$ref": "#/definitions/BountyWinner"

	bountyID := ctx.ParamsInt64(":bounty_id")

	winners, err := hackforger_model.ListBountyWinners(ctx, bountyID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListBountyWinners", err)
		return
	}

	ctx.JSON(http.StatusOK, winners)
}

// --------------------------------------------------------------------------
// Global handlers
// --------------------------------------------------------------------------

// ListAllBounties lists all bounties across all repositories.
func ListAllBounties(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/bounties bounty bountyListAllBounties
	// ---
	// summary: List all bounties
	// produces:
	// - application/json
	// parameters:
	// - name: status
	//   in: query
	//   description: filter by status
	//   type: integer
	// - name: page
	//   in: query
	//   description: page number of results
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     description: "BountyList"
	//     schema:
	//       type: array
	//       items:
	//         "$ref": "#/definitions/Bounty"

	opts := hackforger_model.ListBountiesOptions{
		ListOptions: utils.GetListOptions(ctx),
	}

	if statusStr := ctx.FormString("status"); statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err != nil {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidStatus", err)
			return
		}
		status := hackforger_model.BountyStatus(s)
		opts.Status = &status
	}

	bounties, total, err := hackforger_model.ListBounties(ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListBounties", err)
		return
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, bounties)
}

// BountyStats returns aggregate bounty statistics.
func BountyStats(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/bounties/stats bounty bountyStats
	// ---
	// summary: Get bounty statistics
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     description: "BountyStats"
	//     schema:
	//       "$ref": "#/definitions/BountyStatsResponse"

	stats, err := hackforger_svc.GetBountyStats(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetBountyStats", err)
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

// HunterLeaderboard returns the bounty hunter leaderboard.
func HunterLeaderboard(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/bounties/leaderboard bounty bountyHunterLeaderboard
	// ---
	// summary: Get bounty hunter leaderboard
	// produces:
	// - application/json
	// parameters:
	// - name: limit
	//   in: query
	//   description: number of entries
	//   type: integer
	// responses:
	//   "200":
	//     description: "LeaderboardEntryList"
	//     schema:
	//       type: array
	//       items:
	//         "$ref": "#/definitions/LeaderboardEntry"

	limit := ctx.FormInt("limit")
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	entries, err := hackforger_svc.GetBountyLeaderboard(ctx, limit)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetBountyLeaderboard", err)
		return
	}

	ctx.JSON(http.StatusOK, entries)
}
