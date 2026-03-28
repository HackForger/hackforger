// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"errors"
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/util"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// --- Request / Response forms ---

// RedeemForm represents the JSON body for redeeming credits.
type RedeemForm struct {
	OptionID int64 `json:"option_id" binding:"Required"`
}

// AdminDepositForm represents the JSON body for admin deposit.
type AdminDepositForm struct {
	UserID    int64  `json:"user_id" binding:"Required"`
	Amount    int64  `json:"amount" binding:"Required"`
	Reference string `json:"reference"`
	Note      string `json:"note"`
}

// AdminDeductForm represents the JSON body for admin deduct.
type AdminDeductForm struct {
	UserID    int64  `json:"user_id" binding:"Required"`
	Amount    int64  `json:"amount" binding:"Required"`
	Reference string `json:"reference"`
	Note      string `json:"note"`
}

// CreateRedeemOptionForm represents the JSON body for creating a redeem option.
type CreateRedeemOptionForm struct {
	Name        string `json:"name" binding:"Required"`
	Description string `json:"description"`
	Cost        int64  `json:"cost" binding:"Required"`
	Stock       int    `json:"stock"`
	IsActive    bool   `json:"is_active"`
}

// UpdateRedeemOptionForm represents the JSON body for updating a redeem option.
type UpdateRedeemOptionForm struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Cost        int64  `json:"cost"`
	Stock       int    `json:"stock"`
	IsActive    *bool  `json:"is_active"`
}

// FulfillOrderForm represents the JSON body for fulfilling an order.
type FulfillOrderForm struct {
	Note string `json:"note"`
}

// --- Helpers ---

// handleCreditsError maps service/model errors to HTTP responses.
func handleCreditsError(ctx *context.APIContext, err error) {
	if hackforger_model.IsErrInsufficientCredits(err) {
		ctx.Error(http.StatusUnprocessableEntity, "InsufficientCredits", err)
		return
	}
	if hackforger_model.IsErrOutOfStock(err) {
		ctx.Error(http.StatusUnprocessableEntity, "OutOfStock", err)
		return
	}
	if hackforger_model.IsErrRedeemOptionNotExist(err) {
		ctx.NotFound()
		return
	}
	if hackforger_model.IsErrRedeemOrderNotExist(err) {
		ctx.NotFound()
		return
	}
	if hackforger_service.IsErrNotAdmin(err) {
		ctx.Error(http.StatusForbidden, "NotAdmin", err)
		return
	}
	if hackforger_service.IsErrOrderNotPending(err) {
		ctx.Error(http.StatusConflict, "OrderNotPending", err)
		return
	}
	if errors.Is(err, util.ErrInvalidArgument) {
		ctx.Error(http.StatusBadRequest, "InvalidArgument", err)
		return
	}
	ctx.Error(http.StatusInternalServerError, "InternalError", err)
}

// --- Credit Handlers ---

// GetBalance returns the authenticated user's credit balance.
func GetBalance(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/credits/balance hackforger hackforgerGetBalance
	// ---
	// summary: Get current user's credit balance
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     description: Credit balance
	//   "401":
	//     "$ref": "#/responses/unauthorized"

	acct, err := hackforger_service.GetOrCreateCreditAccount(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetOrCreateCreditAccount", err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]int64{"balance": acct.Balance})
}

// ListTransactions returns the authenticated user's credit transactions.
func ListTransactions(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/credits/transactions hackforger hackforgerListTransactions
	// ---
	// summary: List current user's credit transactions
	// produces:
	// - application/json
	// parameters:
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
	//     description: Transaction list
	//   "401":
	//     "$ref": "#/responses/unauthorized"

	listOpts := utils.GetListOptions(ctx)

	txns, total, err := hackforger_service.ListTransactions(ctx, ctx.Doer.ID, listOpts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListTransactions", err)
		return
	}

	ctx.SetLinkHeader(int(total), listOpts.PageSize)
	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, txns)
}

// ListRedeemOptions returns all active redeem options.
func ListRedeemOptions(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/credits/redeem/options hackforger hackforgerListRedeemOptions
	// ---
	// summary: List available redeem options
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     description: Redeem option list

	options, err := hackforger_service.ListRedeemOptions(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListRedeemOptions", err)
		return
	}

	ctx.JSON(http.StatusOK, options)
}

// CreateRedeemOption creates a new redeem option (admin only).
func CreateRedeemOption(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/credits/redeem/options hackforger hackforgerCreateRedeemOption
	// ---
	// summary: Create a redeem option (admin only)
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/CreateRedeemOptionForm"
	// responses:
	//   "201":
	//     description: Redeem option created
	//   "403":
	//     "$ref": "#/responses/forbidden"

	form := web.GetForm(ctx).(*CreateRedeemOptionForm)

	opt := &hackforger_model.RedeemOption{
		Name:        form.Name,
		Description: form.Description,
		Cost:        form.Cost,
		Stock:       form.Stock,
		IsActive:    form.IsActive,
	}

	if err := hackforger_service.CreateRedeemOptionAsAdmin(ctx, ctx.Doer, opt); err != nil {
		handleCreditsError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, opt)
}

// UpdateRedeemOption updates an existing redeem option (admin only).
func UpdateRedeemOption(ctx *context.APIContext) {
	// swagger:operation PUT /hackforger/credits/redeem/options/{id} hackforger hackforgerUpdateRedeemOption
	// ---
	// summary: Update a redeem option (admin only)
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: ID of the redeem option
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/UpdateRedeemOptionForm"
	// responses:
	//   "200":
	//     description: Updated redeem option
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*UpdateRedeemOptionForm)
	id := ctx.ParamsInt64(":id")

	opt, err := hackforger_model.GetRedeemOptionByID(ctx, id)
	if err != nil {
		handleCreditsError(ctx, err)
		return
	}

	if form.Name != "" {
		opt.Name = form.Name
	}
	if form.Description != "" {
		opt.Description = form.Description
	}
	if form.Cost != 0 {
		opt.Cost = form.Cost
	}
	if form.Stock != 0 {
		opt.Stock = form.Stock
	}
	if form.IsActive != nil {
		opt.IsActive = *form.IsActive
	}

	if err := hackforger_service.UpdateRedeemOptionAsAdmin(ctx, ctx.Doer, opt); err != nil {
		handleCreditsError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, opt)
}

// Redeem redeems credits for a redeem option.
func Redeem(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/credits/redeem hackforger hackforgerRedeem
	// ---
	// summary: Redeem credits for an option
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/RedeemForm"
	// responses:
	//   "201":
	//     description: Order created
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "422":
	//     description: Insufficient credits or out of stock

	form := web.GetForm(ctx).(*RedeemForm)

	order, err := hackforger_service.Redeem(ctx, ctx.Doer.ID, form.OptionID)
	if err != nil {
		handleCreditsError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, order)
}

// ListRedeemOrders returns the authenticated user's redeem orders.
func ListRedeemOrders(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/credits/redeem/orders hackforger hackforgerListRedeemOrders
	// ---
	// summary: List current user's redeem orders
	// produces:
	// - application/json
	// parameters:
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
	//     description: Redeem order list
	//   "401":
	//     "$ref": "#/responses/unauthorized"

	listOpts := utils.GetListOptions(ctx)

	orders, total, err := hackforger_service.ListOrders(ctx, ctx.Doer.ID, listOpts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListRedeemOrders", err)
		return
	}

	ctx.SetLinkHeader(int(total), listOpts.PageSize)
	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, orders)
}

// FulfillOrder marks a redeem order as fulfilled (admin only).
func FulfillOrder(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/credits/redeem/orders/{oid}/fulfill hackforger hackforgerFulfillOrder
	// ---
	// summary: Fulfill a redeem order (admin only)
	// consumes:
	// - application/json
	// parameters:
	// - name: oid
	//   in: path
	//   description: ID of the redeem order
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/FulfillOrderForm"
	// responses:
	//   "204":
	//     description: Order fulfilled
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     description: Order is not pending

	form := web.GetForm(ctx).(*FulfillOrderForm)

	if err := hackforger_service.FulfillOrder(ctx, ctx.Doer, ctx.ParamsInt64(":oid"), form.Note); err != nil {
		handleCreditsError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// CancelOrder cancels a redeem order and refunds credits (admin only).
func CancelOrder(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/credits/redeem/orders/{oid}/cancel hackforger hackforgerCancelOrder
	// ---
	// summary: Cancel a redeem order (admin only)
	// parameters:
	// - name: oid
	//   in: path
	//   description: ID of the redeem order
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     description: Order cancelled and credits refunded
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     description: Order is not pending

	if err := hackforger_service.CancelOrder(ctx, ctx.Doer, ctx.ParamsInt64(":oid")); err != nil {
		handleCreditsError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AdminDeposit deposits credits to a user's account (admin only).
func AdminDeposit(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/credits/admin/deposit hackforger hackforgerAdminDeposit
	// ---
	// summary: Deposit credits to a user (admin only)
	// consumes:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/AdminDepositForm"
	// responses:
	//   "204":
	//     description: Credits deposited
	//   "403":
	//     "$ref": "#/responses/forbidden"

	form := web.GetForm(ctx).(*AdminDepositForm)

	if err := hackforger_service.AdminDeposit(ctx, ctx.Doer, form.UserID, form.Amount, form.Reference, form.Note); err != nil {
		handleCreditsError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AdminDeduct deducts credits from a user's account (admin only).
func AdminDeduct(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/credits/admin/deduct hackforger hackforgerAdminDeduct
	// ---
	// summary: Deduct credits from a user (admin only)
	// consumes:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/AdminDeductForm"
	// responses:
	//   "204":
	//     description: Credits deducted
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     description: Insufficient credits

	form := web.GetForm(ctx).(*AdminDeductForm)

	if err := hackforger_service.AdminDeduct(ctx, ctx.Doer, form.UserID, form.Amount, form.Reference, form.Note); err != nil {
		handleCreditsError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
