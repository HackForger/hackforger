// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
)

// CreatePhaseTypeForm is the JSON body for POST.
type CreatePhaseTypeForm struct {
	ActivityKind    string `json:"activity_kind" binding:"Required;In(hackathon,bounty,grant)"`
	Key             string `json:"key" binding:"Required"`
	DisplayNameI18n string `json:"display_name_i18n" binding:"Required"`
	IsUnique        bool   `json:"is_unique"`
	AllowedActions  string `json:"allowed_actions"`
	DefaultOrder    int    `json:"default_order"`
}

// UpdatePhaseTypeForm is the JSON body for PUT — same shape, full overwrite.
type UpdatePhaseTypeForm = CreatePhaseTypeForm

// ListPhaseTypesAPI returns the global phase type catalog (flat array).
func ListPhaseTypesAPI(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/admin/phase-types hackforgerAdmin hackforgerListPhaseTypes
	// ---
	// summary: List all phase types (admin)
	// responses:
	//   "200":
	//     description: array of phase types
	pts, err := hackforger_model.GetAllPhaseTypes(ctx)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, pts)
}

// CreatePhaseTypeAPI inserts a new phase type.
func CreatePhaseTypeAPI(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/admin/phase-types hackforgerAdmin hackforgerCreatePhaseType
	// ---
	// summary: Create a phase type (admin)
	// parameters:
	//   - name: body
	//     in: body
	//     schema:
	//       "$ref": "#/definitions/CreatePhaseTypeForm"
	// responses:
	//   "201":
	//     description: created phase type
	form := web.GetForm(ctx).(*CreatePhaseTypeForm)
	allowed := form.AllowedActions
	if allowed == "" {
		allowed = "[]"
	}
	pt := &hackforger_model.PhaseType{
		ActivityKind:    form.ActivityKind,
		Key:             form.Key,
		DisplayNameI18n: form.DisplayNameI18n,
		IsUnique:        form.IsUnique,
		AllowedActions:  allowed,
		DefaultOrder:    form.DefaultOrder,
	}
	if err := hackforger_model.CreatePhaseType(ctx, pt); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusCreated, pt)
}

// UpdatePhaseTypeAPI replaces all fields of an existing phase type.
func UpdatePhaseTypeAPI(ctx *context.APIContext) {
	// swagger:operation PUT /hackforger/admin/phase-types/{id} hackforgerAdmin hackforgerUpdatePhaseType
	// ---
	// summary: Update a phase type (admin)
	// parameters:
	//   - name: id
	//     in: path
	//     type: integer
	//     required: true
	//   - name: body
	//     in: body
	//     schema:
	//       "$ref": "#/definitions/CreatePhaseTypeForm"
	// responses:
	//   "200":
	//     description: updated phase type
	//   "404":
	//     "$ref": "#/responses/notFound"
	id := ctx.ParamsInt64(":id")
	pt, err := hackforger_model.GetPhaseTypeByID(ctx, id)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	if pt == nil {
		ctx.NotFound()
		return
	}
	form := web.GetForm(ctx).(*UpdatePhaseTypeForm)
	pt.ActivityKind = form.ActivityKind
	pt.Key = form.Key
	pt.DisplayNameI18n = form.DisplayNameI18n
	pt.IsUnique = form.IsUnique
	pt.AllowedActions = form.AllowedActions
	if pt.AllowedActions == "" {
		pt.AllowedActions = "[]"
	}
	pt.DefaultOrder = form.DefaultOrder
	if err := hackforger_model.UpdatePhaseType(ctx, pt); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, pt)
}

// DeletePhaseTypeAPI removes a phase type.
func DeletePhaseTypeAPI(ctx *context.APIContext) {
	// swagger:operation DELETE /hackforger/admin/phase-types/{id} hackforgerAdmin hackforgerDeletePhaseType
	// ---
	// summary: Delete a phase type (admin)
	// parameters:
	//   - name: id
	//     in: path
	//     type: integer
	//     required: true
	// responses:
	//   "204":
	//     description: deleted
	//   "404":
	//     "$ref": "#/responses/notFound"
	id := ctx.ParamsInt64(":id")
	pt, err := hackforger_model.GetPhaseTypeByID(ctx, id)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	if pt == nil {
		ctx.NotFound()
		return
	}
	if err := hackforger_model.DeletePhaseType(ctx, id); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
