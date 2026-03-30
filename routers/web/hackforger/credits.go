// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/base"
	hackforger_service "forgejo.org/services/hackforger"

	"forgejo.org/services/context"
)

const (
	tplCreditsOverview       base.TplName = "hackforger/credits/overview"
	tplCreditsRedeem         base.TplName = "hackforger/credits/redeem"
	tplCreditsOrders         base.TplName = "hackforger/credits/orders"
	tplAdminCredits          base.TplName = "hackforger/credits/admin/credits"
	tplAdminRedeemOptions    base.TplName = "hackforger/credits/admin/options"
	tplAdminCreditOrders     base.TplName = "hackforger/credits/admin/orders"
	tplAdminRedeemOptionKeys base.TplName = "hackforger/credits/admin/keys"
)

// CreditsOverview renders the user's credits overview page.
func CreditsOverview(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.credits.overview")

	acct, err := hackforger_service.GetOrCreateCreditAccount(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetOrCreateCreditAccount", err)
		return
	}

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	txns, total, err := hackforger_service.ListTransactions(ctx, ctx.Doer.ID, db.ListOptions{Page: page, PageSize: 20})
	if err != nil {
		ctx.ServerError("ListTransactions", err)
		return
	}

	options, err := hackforger_service.ListRedeemOptions(ctx)
	if err != nil {
		ctx.ServerError("ListRedeemOptions", err)
		return
	}

	ctx.Data["Account"] = acct
	ctx.Data["Transactions"] = txns
	ctx.Data["Total"] = total
	ctx.Data["Page"] = page
	ctx.Data["RedeemOptions"] = options
	ctx.HTML(http.StatusOK, tplCreditsOverview)
}

// RedeemConfirm renders the redemption confirmation page.
func RedeemConfirm(ctx *context.Context) {
	optionID := ctx.ParamsInt64(":id")
	option, err := hackforger_model.GetRedeemOptionByID(ctx, optionID)
	if err != nil {
		if hackforger_model.IsErrRedeemOptionNotExist(err) {
			ctx.NotFound("RedeemConfirm", err)
			return
		}
		ctx.ServerError("GetRedeemOptionByID", err)
		return
	}

	acct, err := hackforger_service.GetOrCreateCreditAccount(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetOrCreateCreditAccount", err)
		return
	}

	ctx.Data["Title"] = ctx.Tr("hackforger.credits.redeem.confirm")
	ctx.Data["Option"] = option
	ctx.Data["Account"] = acct
	ctx.HTML(http.StatusOK, tplCreditsRedeem)
}

// RedeemConfirmPost handles the POST to redeem credits.
func RedeemConfirmPost(ctx *context.Context) {
	optionID := ctx.ParamsInt64(":id")

	_, err := hackforger_service.Redeem(ctx, ctx.Doer.ID, optionID)
	if err != nil {
		if hackforger_model.IsErrInsufficientCredits(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.credits.insufficient"))
		} else if hackforger_model.IsErrOutOfStock(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.credits.out_of_stock"))
		} else {
			ctx.ServerError("Redeem", err)
			return
		}
		ctx.Redirect(fmt.Sprintf("/credits/redeem/%d", optionID))
		return
	}

	ctx.Flash.Success(ctx.Tr("hackforger.credits.redeemed"))
	ctx.Redirect("/credits/orders")
}

// CreditOrders renders the user's redeem order list.
func CreditOrders(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.credits.orders")

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	orders, total, err := hackforger_service.ListOrders(ctx, ctx.Doer.ID, db.ListOptions{Page: page, PageSize: 20})
	if err != nil {
		ctx.ServerError("ListOrders", err)
		return
	}

	ctx.Data["Orders"] = orders
	ctx.Data["Total"] = total
	ctx.Data["Page"] = page
	ctx.HTML(http.StatusOK, tplCreditsOrders)
}

// AdminCredits renders the admin credits management page.
func AdminCredits(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.credits.admin.title")
	ctx.Data["PageIsAdminCredits"] = true

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	accounts, total, err := hackforger_model.ListCreditAccounts(ctx, hackforger_model.ListCreditAccountsOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: 50},
	})
	if err != nil {
		ctx.ServerError("ListCreditAccounts", err)
		return
	}

	ctx.Data["Accounts"] = accounts
	ctx.Data["Total"] = total
	ctx.Data["Page"] = page
	ctx.HTML(http.StatusOK, tplAdminCredits)
}

// AdminCreditsDeposit handles admin deposit of credits to a user.
func AdminCreditsDeposit(ctx *context.Context) {
	userID, _ := strconv.ParseInt(ctx.Req.FormValue("user_id"), 10, 64)
	amount, _ := strconv.ParseInt(ctx.Req.FormValue("amount"), 10, 64)
	ref := ctx.Req.FormValue("reference")
	note := ctx.Req.FormValue("note")

	if userID <= 0 || amount <= 0 {
		ctx.Flash.Error(ctx.Tr("hackforger.credits.admin.invalid_input"))
		ctx.Redirect("/-/admin/credits")
		return
	}

	if err := hackforger_service.AdminDeposit(ctx, ctx.Doer, userID, amount, ref, note); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to deposit: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.deposited"))
	}
	ctx.Redirect("/-/admin/credits")
}

// AdminCreditsDeduct handles admin deduction of credits from a user.
func AdminCreditsDeduct(ctx *context.Context) {
	userID, _ := strconv.ParseInt(ctx.Req.FormValue("user_id"), 10, 64)
	amount, _ := strconv.ParseInt(ctx.Req.FormValue("amount"), 10, 64)
	ref := ctx.Req.FormValue("reference")
	note := ctx.Req.FormValue("note")

	if userID <= 0 || amount <= 0 {
		ctx.Flash.Error(ctx.Tr("hackforger.credits.admin.invalid_input"))
		ctx.Redirect("/-/admin/credits")
		return
	}

	if err := hackforger_service.AdminDeduct(ctx, ctx.Doer, userID, amount, ref, note); err != nil {
		if hackforger_model.IsErrInsufficientCredits(err) {
			ctx.Flash.Error(ctx.Tr("hackforger.credits.insufficient"))
		} else {
			ctx.Flash.Error(fmt.Sprintf("Failed to deduct: %v", err))
		}
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.deducted"))
	}
	ctx.Redirect("/-/admin/credits")
}

// AdminRedeemOptions renders the admin redeem options management page.
func AdminRedeemOptions(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.credits.admin.options")
	ctx.Data["PageIsAdminCredits"] = true

	options, err := hackforger_model.ListAllRedeemOptions(ctx)
	if err != nil {
		ctx.ServerError("ListAllRedeemOptions", err)
		return
	}

	ctx.Data["Options"] = options
	ctx.HTML(http.StatusOK, tplAdminRedeemOptions)
}

// AdminRedeemOptionsCreate handles creating a new redeem option.
func AdminRedeemOptionsCreate(ctx *context.Context) {
	name := ctx.Req.FormValue("name")
	description := ctx.Req.FormValue("description")
	cost, _ := strconv.ParseInt(ctx.Req.FormValue("cost"), 10, 64)
	stock, _ := strconv.Atoi(ctx.Req.FormValue("stock"))

	if name == "" || cost <= 0 {
		ctx.Flash.Error(ctx.Tr("hackforger.credits.admin.option_invalid"))
		ctx.Redirect("/-/admin/credits/options")
		return
	}

	opt := &hackforger_model.RedeemOption{
		Name:        name,
		Description: description,
		Cost:        cost,
		Stock:       stock,
		IsActive:    true,
	}

	if err := hackforger_service.CreateRedeemOptionAsAdmin(ctx, ctx.Doer, opt); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to create option: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.option_created"))
	}
	ctx.Redirect("/-/admin/credits/options")
}

// AdminRedeemOptionsUpdate handles updating an existing redeem option.
func AdminRedeemOptionsUpdate(ctx *context.Context) {
	optionID := ctx.ParamsInt64(":id")
	option, err := hackforger_model.GetRedeemOptionByID(ctx, optionID)
	if err != nil {
		ctx.Flash.Error(fmt.Sprintf("Option not found: %v", err))
		ctx.Redirect("/-/admin/credits/options")
		return
	}

	option.Name = ctx.Req.FormValue("name")
	option.Description = ctx.Req.FormValue("description")
	option.Cost, _ = strconv.ParseInt(ctx.Req.FormValue("cost"), 10, 64)
	option.Stock, _ = strconv.Atoi(ctx.Req.FormValue("stock"))
	option.IsActive = ctx.Req.FormValue("is_active") == "on"

	if err := hackforger_service.UpdateRedeemOptionAsAdmin(ctx, ctx.Doer, option); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to update option: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.option_updated"))
	}
	ctx.Redirect("/-/admin/credits/options")
}

// AdminCreditOrders renders the admin orders management page.
func AdminCreditOrders(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.credits.admin.orders")
	ctx.Data["PageIsAdminCredits"] = true

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	orders, total, err := hackforger_model.ListAllRedeemOrders(ctx, db.ListOptions{Page: page, PageSize: 50})
	if err != nil {
		ctx.ServerError("ListAllRedeemOrders", err)
		return
	}

	ctx.Data["Orders"] = orders
	ctx.Data["Total"] = total
	ctx.Data["Page"] = page
	ctx.HTML(http.StatusOK, tplAdminCreditOrders)
}

// AdminCreditOrdersFulfill handles fulfilling a pending order.
func AdminCreditOrdersFulfill(ctx *context.Context) {
	orderID := ctx.ParamsInt64(":oid")
	note := ctx.Req.FormValue("note")
	deliveryType := ctx.Req.FormValue("delivery_type")
	deliveryValue := ctx.Req.FormValue("delivery_value")

	if err := hackforger_service.FulfillOrder(ctx, ctx.Doer, orderID, note, deliveryType, deliveryValue); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to fulfill order: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.order_fulfilled"))
	}
	ctx.Redirect("/-/admin/credits/orders")
}

// AdminCreditOrdersCancel handles cancelling a pending order.
func AdminCreditOrdersCancel(ctx *context.Context) {
	orderID := ctx.ParamsInt64(":oid")

	if err := hackforger_service.CancelOrder(ctx, ctx.Doer, orderID); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to cancel order: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.order_cancelled"))
	}
	ctx.Redirect("/-/admin/credits/orders")
}

// AdminRedeemOptionKeys renders the key pool management page for an option.
func AdminRedeemOptionKeys(ctx *context.Context) {
	optionID := ctx.ParamsInt64(":id")
	option, err := hackforger_model.GetRedeemOptionByID(ctx, optionID)
	if err != nil {
		if hackforger_model.IsErrRedeemOptionNotExist(err) {
			ctx.NotFound("AdminRedeemOptionKeys", err)
			return
		}
		ctx.ServerError("GetRedeemOptionByID", err)
		return
	}

	keys, err := hackforger_model.ListKeysByOption(ctx, optionID)
	if err != nil {
		ctx.ServerError("ListKeysByOption", err)
		return
	}

	total, available, err := hackforger_service.GetKeyPoolStatus(ctx, optionID)
	if err != nil {
		ctx.ServerError("GetKeyPoolStatus", err)
		return
	}

	ctx.Data["Title"] = ctx.Tr("hackforger.credits.admin.keys.title")
	ctx.Data["PageIsAdminCredits"] = true
	ctx.Data["Option"] = option
	ctx.Data["Keys"] = keys
	ctx.Data["KeysTotal"] = total
	ctx.Data["KeysAvailable"] = available
	ctx.HTML(http.StatusOK, tplAdminRedeemOptionKeys)
}

// AdminRedeemOptionKeysAdd handles bulk-adding keys to a redeem option's pool.
func AdminRedeemOptionKeysAdd(ctx *context.Context) {
	optionID := ctx.ParamsInt64(":id")
	rawKeys := ctx.Req.FormValue("keys")

	var keys []string
	for _, line := range strings.Split(rawKeys, "\n") {
		k := strings.TrimSpace(line)
		if k != "" {
			keys = append(keys, k)
		}
	}

	if len(keys) == 0 {
		ctx.Flash.Error(ctx.Tr("hackforger.credits.admin.keys.empty"))
		ctx.Redirect(fmt.Sprintf("/-/admin/credits/options/%d/keys", optionID))
		return
	}

	if err := hackforger_service.AddKeysToOption(ctx, ctx.Doer, optionID, keys); err != nil {
		ctx.Flash.Error(fmt.Sprintf("Failed to add keys: %v", err))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.keys.added", len(keys)))
	}
	ctx.Redirect(fmt.Sprintf("/-/admin/credits/options/%d/keys", optionID))
}

// AdminCreditOrdersBatchFulfill handles batch fulfillment of multiple pending orders.
func AdminCreditOrdersBatchFulfill(ctx *context.Context) {
	if err := ctx.Req.ParseForm(); err != nil {
		ctx.ServerError("ParseForm", err)
		return
	}

	orderIDStrs := ctx.Req.Form["order_ids"]
	if len(orderIDStrs) == 0 {
		ctx.Flash.Error(ctx.Tr("hackforger.credits.admin.batch.no_selection"))
		ctx.Redirect("/-/admin/credits/orders")
		return
	}

	var orderIDs []int64
	for _, s := range orderIDStrs {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			continue
		}
		orderIDs = append(orderIDs, id)
	}

	note := ctx.Req.FormValue("note")
	deliveryType := ctx.Req.FormValue("delivery_type")
	deliveryValue := ctx.Req.FormValue("delivery_value")

	success, failed, err := hackforger_service.BatchFulfillOrders(ctx, ctx.Doer, orderIDs, note, deliveryType, deliveryValue)
	if err != nil {
		ctx.Flash.Error(fmt.Sprintf("Batch fulfill error: %v", err))
	} else if len(failed) > 0 {
		ctx.Flash.Warning(ctx.Tr("hackforger.credits.admin.batch.partial", success, len(failed)))
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.credits.admin.batch.success", success))
	}
	ctx.Redirect("/-/admin/credits/orders")
}
