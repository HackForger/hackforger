// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/base"
	"forgejo.org/services/context"
)

const (
	tplAdminPhaseTypes base.TplName = "hackforger/admin/phase_types"
)

// AdminPhaseTypes renders the phase type management admin page.
func AdminPhaseTypes(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.admin.phase_types.title")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminHackforgerPhaseTypes"] = true

	phaseTypes, err := hackforger_model.GetAllPhaseTypes(ctx)
	if err != nil {
		ctx.ServerError("GetAllPhaseTypes", err)
		return
	}

	grouped := make(map[string][]*hackforger_model.PhaseType)
	for _, pt := range phaseTypes {
		grouped[pt.ActivityKind] = append(grouped[pt.ActivityKind], pt)
	}

	ctx.Data["GroupedPhaseTypes"] = grouped
	ctx.Data["ActivityKinds"] = []string{"hackathon", "bounty", "grant"}
	ctx.HTML(http.StatusOK, tplAdminPhaseTypes)
}

// AdminPhaseTypeCreate handles creating a new phase type.
func AdminPhaseTypeCreate(ctx *context.Context) {
	activityKind := ctx.FormString("activity_kind")
	key := ctx.FormString("key")
	displayNameI18n := ctx.FormString("display_name_i18n")
	isUnique := ctx.FormBool("is_unique")
	allowedActions := ctx.FormString("allowed_actions")
	defaultOrder := ctx.FormInt("default_order")

	if activityKind == "" || key == "" || displayNameI18n == "" {
		ctx.Flash.Error(ctx.Tr("hackforger.admin.phase_types.missing_fields"))
		ctx.Redirect("/admin/hackforger/phase-types")
		return
	}

	if allowedActions == "" {
		allowedActions = "[]"
	}

	pt := &hackforger_model.PhaseType{
		ActivityKind:    activityKind,
		Key:             key,
		DisplayNameI18n: displayNameI18n,
		IsUnique:        isUnique,
		AllowedActions:  allowedActions,
		DefaultOrder:    defaultOrder,
	}

	if err := hackforger_model.CreatePhaseType(ctx, pt); err != nil {
		ctx.ServerError("CreatePhaseType", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("hackforger.admin.phase_types.created"))
	ctx.Redirect("/admin/hackforger/phase-types")
}

// AdminPhaseTypeUpdate handles updating an existing phase type.
func AdminPhaseTypeUpdate(ctx *context.Context) {
	id := ctx.ParamsInt64(":id")

	pt, err := hackforger_model.GetPhaseTypeByID(ctx, id)
	if err != nil {
		ctx.ServerError("GetPhaseTypeByID", err)
		return
	}
	if pt == nil {
		ctx.Flash.Error(ctx.Tr("hackforger.admin.phase_types.not_found"))
		ctx.Redirect("/admin/hackforger/phase-types")
		return
	}

	pt.ActivityKind = ctx.FormString("activity_kind")
	pt.Key = ctx.FormString("key")
	pt.DisplayNameI18n = ctx.FormString("display_name_i18n")
	pt.IsUnique = ctx.FormBool("is_unique")
	pt.AllowedActions = ctx.FormString("allowed_actions")
	pt.DefaultOrder = ctx.FormInt("default_order")

	if pt.ActivityKind == "" || pt.Key == "" || pt.DisplayNameI18n == "" {
		ctx.Flash.Error(ctx.Tr("hackforger.admin.phase_types.missing_fields"))
		ctx.Redirect("/admin/hackforger/phase-types")
		return
	}

	if pt.AllowedActions == "" {
		pt.AllowedActions = "[]"
	}

	if err := hackforger_model.UpdatePhaseType(ctx, pt); err != nil {
		ctx.ServerError("UpdatePhaseType", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("hackforger.admin.phase_types.updated"))
	ctx.Redirect("/admin/hackforger/phase-types")
}

// AdminPhaseTypeDelete handles deleting a phase type.
func AdminPhaseTypeDelete(ctx *context.Context) {
	id := ctx.ParamsInt64(":id")

	pt, err := hackforger_model.GetPhaseTypeByID(ctx, id)
	if err != nil {
		ctx.ServerError("GetPhaseTypeByID", err)
		return
	}
	if pt == nil {
		ctx.Flash.Error(ctx.Tr("hackforger.admin.phase_types.not_found"))
		ctx.Redirect("/admin/hackforger/phase-types")
		return
	}

	if err := hackforger_model.DeletePhaseType(ctx, id); err != nil {
		ctx.ServerError("DeletePhaseType", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("hackforger.admin.phase_types.deleted"))
	ctx.Redirect("/admin/hackforger/phase-types")
}
