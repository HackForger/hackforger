// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

// GrantProjectStatus represents the status of a grant project.
type GrantProjectStatus int

const (
	GrantProjectStatusPending  GrantProjectStatus = iota // 0
	GrantProjectStatusApproved                           // 1
	GrantProjectStatusRejected                           // 2
	GrantProjectStatusFunded                             // 3
)

// GrantProject represents a project submitted to a grant round.
type GrantProject struct {
	ID           int64              `xorm:"pk autoincr"`
	RoundID      int64              `xorm:"INDEX NOT NULL"`
	UserID       int64              `xorm:"INDEX NOT NULL"`
	RepoID       int64              `xorm:"INDEX"`
	Title        string             `xorm:"NOT NULL"`
	Description  string             `xorm:"TEXT"`
	Status       GrantProjectStatus `xorm:"NOT NULL DEFAULT 0"`
	AwardAmount  float64            `xorm:"NOT NULL DEFAULT 0"`
	AwardCredits int64              `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix  timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(GrantProject))
}

// ErrGrantProjectNotExist represents a "GrantProjectNotExist" kind of error.
type ErrGrantProjectNotExist struct {
	ID int64
}

// IsErrGrantProjectNotExist checks if an error is a ErrGrantProjectNotExist.
func IsErrGrantProjectNotExist(err error) bool {
	_, ok := err.(ErrGrantProjectNotExist)
	return ok
}

func (err ErrGrantProjectNotExist) Error() string {
	return fmt.Sprintf("grant project does not exist [id: %d]", err.ID)
}

func (err ErrGrantProjectNotExist) Unwrap() error {
	return util.ErrNotExist
}

// CreateGrantProject creates a new grant project.
func CreateGrantProject(ctx context.Context, p *GrantProject) error {
	_, err := db.GetEngine(ctx).Insert(p)
	return err
}

// GetGrantProjectByID returns a grant project by its ID.
func GetGrantProjectByID(ctx context.Context, id int64) (*GrantProject, error) {
	p := new(GrantProject)
	has, err := db.GetEngine(ctx).ID(id).Get(p)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrGrantProjectNotExist{ID: id}
	}
	return p, nil
}

// ListGrantProjectsByRoundOptions holds options for listing grant projects in a round.
type ListGrantProjectsByRoundOptions struct {
	db.ListOptions
	RoundID int64
	Status  *GrantProjectStatus
}

func (opts ListGrantProjectsByRoundOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.RoundID > 0 {
		cond = cond.And(builder.Eq{"round_id": opts.RoundID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"status": *opts.Status})
	}
	return cond
}

// ListGrantProjectsByRound returns grant projects matching the given options.
func ListGrantProjectsByRound(ctx context.Context, opts ListGrantProjectsByRoundOptions) ([]*GrantProject, int64, error) {
	return db.FindAndCount[GrantProject](ctx, opts)
}

// UpdateGrantProject updates an existing grant project.
func UpdateGrantProject(ctx context.Context, p *GrantProject) error {
	_, err := db.GetEngine(ctx).ID(p.ID).AllCols().Update(p)
	return err
}

// ErrGrantProjectAlreadyExists represents a duplicate project submission.
type ErrGrantProjectAlreadyExists struct {
	UserID  int64
	RoundID int64
}

// IsErrGrantProjectAlreadyExists checks if an error is a ErrGrantProjectAlreadyExists.
func IsErrGrantProjectAlreadyExists(err error) bool {
	_, ok := err.(ErrGrantProjectAlreadyExists)
	return ok
}

func (err ErrGrantProjectAlreadyExists) Error() string {
	return fmt.Sprintf("grant project already exists [user_id: %d, round_id: %d]", err.UserID, err.RoundID)
}

func (err ErrGrantProjectAlreadyExists) Unwrap() error {
	return util.ErrAlreadyExist
}

// ErrGrantRoundNotOpen is returned when a round is not in Open status.
type ErrGrantRoundNotOpen struct {
	RoundID int64
	Status  GrantRoundStatus
}

// IsErrGrantRoundNotOpen checks if an error is a ErrGrantRoundNotOpen.
func IsErrGrantRoundNotOpen(err error) bool {
	_, ok := err.(ErrGrantRoundNotOpen)
	return ok
}

func (err ErrGrantRoundNotOpen) Error() string {
	return fmt.Sprintf("grant round is not open [round_id: %d, status: %d]", err.RoundID, err.Status)
}

func (err ErrGrantRoundNotOpen) Unwrap() error {
	return util.ErrInvalidArgument
}

// ErrExceedsBudget is returned when an award would exceed the round's budget.
type ErrExceedsBudget struct {
	RoundID     int64
	BudgetField string
	Budget      float64
	Used        float64
	Requested   float64
}

// IsErrExceedsBudget checks if an error is a ErrExceedsBudget.
func IsErrExceedsBudget(err error) bool {
	_, ok := err.(ErrExceedsBudget)
	return ok
}

func (err ErrExceedsBudget) Error() string {
	return fmt.Sprintf("exceeds budget [round_id: %d, field: %s, budget: %.2f, used: %.2f, requested: %.2f]",
		err.RoundID, err.BudgetField, err.Budget, err.Used, err.Requested)
}

func (err ErrExceedsBudget) Unwrap() error {
	return util.ErrInvalidArgument
}

// ErrUnallocatedProjects is returned when there are unallocated projects preventing finalization.
type ErrUnallocatedProjects struct {
	RoundID int64
	Count   int64
}

// IsErrUnallocatedProjects checks if an error is a ErrUnallocatedProjects.
func IsErrUnallocatedProjects(err error) bool {
	_, ok := err.(ErrUnallocatedProjects)
	return ok
}

func (err ErrUnallocatedProjects) Error() string {
	return fmt.Sprintf("unallocated projects remain [round_id: %d, count: %d]", err.RoundID, err.Count)
}

func (err ErrUnallocatedProjects) Unwrap() error {
	return util.ErrInvalidArgument
}

// GetGrantProjectByUserAndRound returns a grant project for a specific user in a round.
func GetGrantProjectByUserAndRound(ctx context.Context, userID, roundID int64) (*GrantProject, error) {
	p := new(GrantProject)
	has, err := db.GetEngine(ctx).Where("user_id = ? AND round_id = ?", userID, roundID).Get(p)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrGrantProjectNotExist{ID: 0}
	}
	return p, nil
}

// DeleteGrantProject deletes a grant project. Only Pending projects can be deleted.
func DeleteGrantProject(ctx context.Context, id int64) error {
	p, err := GetGrantProjectByID(ctx, id)
	if err != nil {
		return err
	}
	if p.Status != GrantProjectStatusPending {
		return fmt.Errorf("grant project cannot be deleted [id: %d, status: %d]: %w", id, p.Status, util.ErrInvalidArgument)
	}
	_, err = db.GetEngine(ctx).ID(id).Delete(new(GrantProject))
	return err
}

// CountGrantProjectsByRoundOptions holds options for counting grant projects.
type CountGrantProjectsByRoundOptions struct {
	db.ListOptions
	RoundID int64
	Status  *GrantProjectStatus
}

func (opts CountGrantProjectsByRoundOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.RoundID > 0 {
		cond = cond.And(builder.Eq{"round_id": opts.RoundID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"status": *opts.Status})
	}
	return cond
}

// CountGrantProjectsByRound counts grant projects in a round with optional status filter.
func CountGrantProjectsByRound(ctx context.Context, roundID int64, status *GrantProjectStatus) (int64, error) {
	return db.Count[GrantProject](ctx, CountGrantProjectsByRoundOptions{
		RoundID: roundID,
		Status:  status,
	})
}

// GetUserGrantRounds returns grant rounds the user has submitted projects to, most recent first.
func GetUserGrantRounds(ctx context.Context, userID int64, limit int) ([]*GrantRound, error) {
	rounds := make([]*GrantRound, 0, limit)
	return rounds, db.GetEngine(ctx).
		Join("INNER", "grant_project", "grant_project.round_id = grant_round.id").
		Where("grant_project.user_id = ?", userID).
		OrderBy("grant_project.created_unix DESC").
		Limit(limit).
		Find(&rounds)
}
