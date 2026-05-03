// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	organization_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
	notify_service "forgejo.org/services/notify"
)

// RegisterForm is the form for registering to a hackathon.
type RegisterForm struct {
	TeamName    string `json:"team_name" binding:"Required"`
	TrackID     int64  `json:"track_id"`
	RepoID      int64  `json:"repo_id,omitempty"`     // NEW: optional; if 0, solo registration (no repo binding)
	Title       string `json:"title,omitempty"`       // NEW: submission title (defaults to repo name)
	Description string `json:"description,omitempty"` // NEW: submission description
	DemoURL     string `json:"demo_url,omitempty"`    // NEW: submission demo URL
}

// UpdateRegistrationForm is the form for updating a registration's status.
type UpdateRegistrationForm struct {
	Status int `json:"status" binding:"Required"`
}

// Register registers the authenticated user to a hackathon.
//
// swagger:operation POST /hackforger/hackathons/{id}/registrations hackforger hackforgerRegister
// ---
// summary: Register for a hackathon
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/RegisterForm"
// responses:
//   "201":
//     description: Registration created
//   "400":
//     description: Hackathon is not accepting registrations
//   "404":
//     "$ref": "#/responses/notFound"
//   "409":
//     description: Already registered
func Register(ctx *context.APIContext) {
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	if !h.IsPublished {
		ctx.Error(http.StatusBadRequest, "InvalidPhase", "hackathon is not accepting registrations")
		return
	}
	// Block organizer self-registration (parity with web RegisterPost)
	if h.OwnerID == ctx.Doer.ID {
		ctx.Error(http.StatusForbidden, "CannotRegisterOwn", "organizer cannot register for own hackathon")
		return
	}
	if h.LinkedOrgID > 0 {
		if org, err := organization_model.GetOrgByID(ctx, h.LinkedOrgID); err == nil {
			if isOwner, _ := org.IsOwnedBy(ctx, ctx.Doer.ID); isOwner {
				ctx.Error(http.StatusForbidden, "CannotRegisterOwn", "organizer cannot register for own hackathon")
				return
			}
		}
	}

	canRegister, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register")
	if !canRegister {
		ctx.Error(http.StatusBadRequest, "InvalidPhase", "hackathon is not accepting registrations")
		return
	}
	// Judges cannot register as participants in their own hackathon (parity with web handler).
	isJudge, err := hackforger_model.IsJudgeForAnyTrack(ctx, h.ID, ctx.Doer.ID)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	if isJudge {
		ctx.Error(http.StatusForbidden, "JudgeCannotRegister", "judges cannot register as participants in their own hackathon")
		return
	}
	f := web.GetForm(ctx).(*RegisterForm)

	// Optional repo binding — hoisted so submission block below can use `repo`.
	var repo *repo_model.Repository
	if f.RepoID > 0 {
		var rerr error
		repo, rerr = hackforger_service.UserCanRegisterRepo(ctx, ctx.Doer, f.RepoID)
		if rerr != nil {
			if hackforger_service.IsErrRepoAccessDenied(rerr) {
				ctx.Error(http.StatusForbidden, "RepoAccessDenied", rerr)
				return
			}
			if repo_model.IsErrRepoNotExist(rerr) {
				ctx.Error(http.StatusNotFound, "RepoNotExist", rerr)
				return
			}
			ctx.InternalServerError(rerr)
			return
		}
	}

	r := &hackforger_model.HackathonRegistration{
		HackathonID: h.ID,
		UserID:      ctx.Doer.ID,
		TeamName:    f.TeamName,
		TrackID:     f.TrackID,
		Status:      hackforger_model.RegistrationStatusApproved, // ALWAYS — match web; was previously zero-value
	}
	if repo != nil {
		r.RepoID = repo.ID
		if repo.Owner.IsOrganization() {
			r.OrgID = repo.Owner.ID
			r.TeamName = repo.Owner.Name // override caller's TeamName for team/org registration
		}
	}
	if err := hackforger_model.CreateRegistration(ctx, r); err != nil {
		if hackforger_model.IsErrDuplicateRegistration(err) {
			ctx.Error(http.StatusConflict, "AlreadyRegistered", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}

	// Repo-bound side-effects (parity with web): submission + AddOrgUser.
	if repo != nil {
		title := f.Title
		if title == "" {
			title = repo.Name
		}
		s := &hackforger_model.HackathonSubmission{
			HackathonID:    h.ID,
			RegistrationID: r.ID,
			UserID:         ctx.Doer.ID,
			Title:          title,
			Description:    f.Description,
			DemoURL:        f.DemoURL,
			TrackID:        f.TrackID,
			RepoID:         repo.ID,
			Status:         hackforger_model.SubmissionStatusSubmitted,
		}
		if err := hackforger_service.CreateSubmission(ctx, ctx.Doer, h, s); err != nil {
			log.Error("CreateSubmission on register: %v", err)
			// Don't fail registration — log and continue (parity with web)
		}
	}

	// AddOrgUser side-effect runs whether or not a repo was bound (matches web).
	if h.LinkedOrgID > 0 {
		_ = organization_model.AddOrgUser(ctx, h.LinkedOrgID, ctx.Doer.ID)
	}

	// Publish feed event for registration (global + followers so it appears in entity timelines)
	notify_service.HackforgerEntityCreated(ctx, ctx.Doer, &notify_service.HackforgerEventOpts{
		OpType:       hackforger_model.ActionHackathonRegistered,
		EntityType:   "hackathon",
		EntityID:     h.ID,
		EntityName:   h.Name,
		EntitySlug:   h.Slug,
		AudienceType: notify_service.AudienceGlobal | notify_service.AudienceFollowers,
		Content: hackforger_model.HackforgerActionContent{
			EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
		},
	})

	ctx.JSON(http.StatusCreated, r)
}

// ListRegistrations returns a paginated list of registrations for a hackathon.
//
// swagger:operation GET /hackforger/hackathons/{id}/registrations hackforger hackforgerListRegistrations
// ---
// summary: List registrations for a hackathon
// produces:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: page
//   in: query
//   description: page number of results to return (1-based)
//   type: integer
// - name: limit
//   in: query
//   description: page size of results
//   type: integer
// responses:
//   "200":
//     description: Registration list
func ListRegistrations(ctx *context.APIContext) {
	opts := hackforger_model.ListRegistrationsOptions{
		HackathonID: ctx.ParamsInt64(":id"),
	}
	opts.Page = ctx.FormInt("page")
	opts.PageSize = ctx.FormInt("limit")
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 50 {
		opts.PageSize = 20
	}
	regs, count, err := hackforger_model.ListRegistrations(ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, regs)
}

// UpdateRegistration updates a registration's status (approve/reject).
//
// swagger:operation PUT /hackforger/hackathons/{id}/registrations/{rid} hackforger hackforgerUpdateRegistration
// ---
// summary: Update registration status (approve/reject)
// consumes:
// - application/json
// parameters:
// - name: id
//   in: path
//   description: ID of the hackathon
//   type: integer
//   format: int64
//   required: true
// - name: rid
//   in: path
//   description: ID of the registration
//   type: integer
//   format: int64
//   required: true
// - name: body
//   in: body
//   required: true
//   schema:
//     "$ref": "#/definitions/UpdateRegistrationForm"
// responses:
//   "200":
//     description: Registration updated
func UpdateRegistration(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*UpdateRegistrationForm)
	rid := ctx.ParamsInt64(":rid")
	status := hackforger_model.RegistrationStatus(f.Status)
	if err := hackforger_model.UpdateRegistrationStatus(ctx, rid, status); err != nil {
		ctx.InternalServerError(err)
		return
	}
	// Publish feed event if approved
	if status == hackforger_model.RegistrationStatusApproved {
		r, err := hackforger_model.GetRegistrationByID(ctx, rid)
		if err == nil {
			h, err := hackforger_model.GetHackathonByID(ctx, r.HackathonID)
			if err == nil {
				if doer, err := user_model.GetUserByID(ctx, r.UserID); err == nil {
					notify_service.HackforgerEntityCreated(ctx, doer, &notify_service.HackforgerEventOpts{
						OpType:       hackforger_model.ActionHackathonRegistered,
						EntityType:   "hackathon",
						EntityID:     h.ID,
						EntityName:   h.Name,
						EntitySlug:   h.Slug,
						AudienceType: notify_service.AudienceFollowers,
						Content: hackforger_model.HackforgerActionContent{
							EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
						},
					})
				}
			}
		}
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "updated"})
}
