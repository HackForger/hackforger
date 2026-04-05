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

// HackathonStatus represents the status of a hackathon.
type HackathonStatus int

const (
	HackathonStatusDraft      HackathonStatus = iota // 0
	HackathonStatusOpen                               // 1
	HackathonStatusHacking                            // 2
	HackathonStatusJudging                            // 3
	HackathonStatusFinished                           // 4
	HackathonStatusCancelled                          // 5
)

// HackathonStatusNames maps status values to human-readable names.
var HackathonStatusNames = map[HackathonStatus]string{
	HackathonStatusDraft:     "draft",
	HackathonStatusOpen:      "open",
	HackathonStatusHacking:   "hacking",
	HackathonStatusJudging:   "judging",
	HackathonStatusFinished:  "finished",
	HackathonStatusCancelled: "cancelled",
}

// Hackathon represents a hackathon event.
type Hackathon struct {
	ID                int64              `xorm:"pk autoincr"`
	OrgID             int64              `xorm:"INDEX NOT NULL"`
	OwnerID           int64              `xorm:"INDEX NOT NULL"`
	Name              string             `xorm:"NOT NULL"`
	Slug              string             `xorm:"UNIQUE NOT NULL"`
	Description       string             `xorm:"TEXT"`
	// StatusCache is computed from the Phase system. Do NOT set directly.
	// Use SyncStatusCache() to update. Source of truth is the phase table.
	StatusCache       HackathonStatus    `xorm:"'status_cache' NOT NULL DEFAULT 0"`
	IsPublished       bool               `xorm:"NOT NULL DEFAULT false"`
	MaxTeamSize       int                `xorm:"NOT NULL DEFAULT 5"`
	RegistrationStart timeutil.TimeStamp `xorm:""`
	RegistrationEnd   timeutil.TimeStamp `xorm:""`
	HackingStart      timeutil.TimeStamp `xorm:""`
	HackingEnd        timeutil.TimeStamp `xorm:""`
	JudgingEnd        timeutil.TimeStamp `xorm:""`
	PrizeSummary      string             `xorm:"TEXT"`
	TemplateRepoID    int64              `xorm:"INDEX"`
	LinkedOrgID       int64              `xorm:"INDEX"` // auto-created Forgejo Organization for this hackathon
	CreatedUnix       timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix       timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(Hackathon))
}

// StatusCacheName returns the human-readable name for the hackathon's cached status.
func (h *Hackathon) StatusCacheName() string {
	if name, ok := HackathonStatusNames[h.StatusCache]; ok {
		return name
	}
	return "unknown"
}

// ErrHackathonNotExist represents a "HackathonNotExist" kind of error.
type ErrHackathonNotExist struct {
	ID   int64
	Slug string
}

// IsErrHackathonNotExist checks if an error is a ErrHackathonNotExist.
func IsErrHackathonNotExist(err error) bool {
	_, ok := err.(ErrHackathonNotExist)
	return ok
}

func (err ErrHackathonNotExist) Error() string {
	return fmt.Sprintf("hackathon does not exist [id: %d, slug: %s]", err.ID, err.Slug)
}

func (err ErrHackathonNotExist) Unwrap() error {
	return util.ErrNotExist
}

// ErrHackathonSlugAlreadyExist represents a "HackathonSlugAlreadyExist" kind of error.
type ErrHackathonSlugAlreadyExist struct {
	Slug string
}

// IsErrHackathonSlugAlreadyExist checks if an error is a ErrHackathonSlugAlreadyExist.
func IsErrHackathonSlugAlreadyExist(err error) bool {
	_, ok := err.(ErrHackathonSlugAlreadyExist)
	return ok
}

func (err ErrHackathonSlugAlreadyExist) Error() string {
	return fmt.Sprintf("hackathon slug already exists [slug: %s]", err.Slug)
}

func (err ErrHackathonSlugAlreadyExist) Unwrap() error {
	return util.ErrAlreadyExist
}

// ErrInvalidHackathonPhase represents a phase transition error (e.g. judging requires Judging status).
type ErrInvalidHackathonPhase struct {
	HackathonID int64
	Current     HackathonStatus
	Expected    HackathonStatus
}

// IsErrInvalidHackathonPhase checks if an error is a ErrInvalidHackathonPhase.
func IsErrInvalidHackathonPhase(err error) bool {
	_, ok := err.(ErrInvalidHackathonPhase)
	return ok
}

func (err ErrInvalidHackathonPhase) Error() string {
	return fmt.Sprintf("invalid hackathon phase [id: %d, current: %d, expected: %d]", err.HackathonID, err.Current, err.Expected)
}

func (err ErrInvalidHackathonPhase) Unwrap() error {
	return util.ErrInvalidArgument
}

// ErrNotJudge represents a permission error when a user is not assigned as judge.
type ErrNotJudge struct {
	UserID      int64
	HackathonID int64
	TrackID     int64
}

// IsErrNotJudge checks if an error is a ErrNotJudge.
func IsErrNotJudge(err error) bool {
	_, ok := err.(ErrNotJudge)
	return ok
}

func (err ErrNotJudge) Error() string {
	return fmt.Sprintf("user is not a judge [user_id: %d, hackathon_id: %d, track_id: %d]", err.UserID, err.HackathonID, err.TrackID)
}

func (err ErrNotJudge) Unwrap() error {
	return util.ErrPermissionDenied
}

// ErrNoCriteria represents a missing scoring criteria error.
type ErrNoCriteria struct {
	HackathonID int64
}

// IsErrNoCriteria checks if an error is a ErrNoCriteria.
func IsErrNoCriteria(err error) bool {
	_, ok := err.(ErrNoCriteria)
	return ok
}

func (err ErrNoCriteria) Error() string {
	return fmt.Sprintf("no scoring criteria defined [hackathon_id: %d]", err.HackathonID)
}

func (err ErrNoCriteria) Unwrap() error {
	return util.ErrInvalidArgument
}

// CreateHackathon creates a new hackathon in the database.
func CreateHackathon(ctx context.Context, h *Hackathon) error {
	return db.Insert(ctx, h)
}

// GetHackathonByID returns a hackathon by its ID.
func GetHackathonByID(ctx context.Context, id int64) (*Hackathon, error) {
	h, exists, err := db.GetByID[Hackathon](ctx, id)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, ErrHackathonNotExist{ID: id}
	}
	return h, nil
}

// GetHackathonBySlug returns a hackathon by its slug.
func GetHackathonBySlug(ctx context.Context, slug string) (*Hackathon, error) {
	h := &Hackathon{Slug: slug}
	has, err := db.GetEngine(ctx).Get(h)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrHackathonNotExist{Slug: slug}
	}
	return h, nil
}

// ListHackathonsOptions holds options for listing hackathons.
type ListHackathonsOptions struct {
	db.ListOptions
	OrgID   int64
	Status  *HackathonStatus
	Keyword string
	SortBy  string
}

func (opts ListHackathonsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.OrgID != 0 {
		cond = cond.And(builder.Eq{"hackathon.org_id": opts.OrgID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"hackathon.status_cache": *opts.Status})
	}
	if opts.Keyword != "" {
		cond = cond.And(builder.Or(
			builder.Like{"hackathon.name", opts.Keyword},
			builder.Like{"hackathon.description", opts.Keyword},
		))
	}
	return cond
}

func (opts ListHackathonsOptions) ToOrders() string {
	switch opts.SortBy {
	case "oldest":
		return "hackathon.created_unix ASC"
	case "alphabetically":
		return "hackathon.name ASC"
	default: // "newest"
		return "hackathon.created_unix DESC"
	}
}

// ListHackathons returns a list of hackathons matching the given options.
// ListPublishedHackathons returns all hackathons where is_published = true.
func ListPublishedHackathons(ctx context.Context) ([]*Hackathon, error) {
	hackathons := make([]*Hackathon, 0)
	return hackathons, db.GetEngine(ctx).Where("is_published = ?", true).Find(&hackathons)
}

func ListHackathons(ctx context.Context, opts ListHackathonsOptions) ([]*Hackathon, int64, error) {
	return db.FindAndCount[Hackathon](ctx, opts)
}

// UpdateHackathonStatusCache updates the status_cache field. Internal use only.
// External callers should use services/hackforger.SyncStatusCache() instead.
func UpdateHackathonStatusCache(ctx context.Context, id int64, status HackathonStatus) error {
	_, err := db.GetEngine(ctx).ID(id).Cols("status_cache").Update(&Hackathon{StatusCache: status})
	return err
}

// SetIsPublished updates the is_published flag.
func SetIsPublished(ctx context.Context, id int64, published bool) error {
	_, err := db.GetEngine(ctx).ID(id).Cols("is_published").Update(&Hackathon{IsPublished: published})
	return err
}

// UpdateHackathon updates an existing hackathon.
func UpdateHackathon(ctx context.Context, h *Hackathon) error {
	_, err := db.GetEngine(ctx).ID(h.ID).AllCols().Update(h)
	return err
}

// DeleteHackathon deletes a hackathon. Only hackathons in Draft status can be deleted.
func DeleteHackathon(ctx context.Context, id int64) error {
	h, err := GetHackathonByID(ctx, id)
	if err != nil {
		return err
	}
	if h.StatusCache != HackathonStatusDraft {
		return fmt.Errorf("hackathon can only be deleted in draft status [id: %d, status: %d]", id, h.StatusCache)
	}
	_, err = db.GetEngine(ctx).ID(id).Delete(new(Hackathon))
	return err
}
