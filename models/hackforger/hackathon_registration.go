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

// RegistrationStatus represents the status of a hackathon registration.
type RegistrationStatus int

const (
	RegistrationStatusPending  RegistrationStatus = iota // 0
	RegistrationStatusApproved                           // 1
	RegistrationStatusRejected                           // 2
)

// HackathonRegistration represents a user's registration to a hackathon.
type HackathonRegistration struct {
	ID          int64              `xorm:"pk autoincr"`
	HackathonID int64              `xorm:"INDEX NOT NULL"`
	UserID      int64              `xorm:"INDEX NOT NULL"`
	OrgID       int64              `xorm:"INDEX"` // 0=solo, >0=team via this org
	TeamName    string             `xorm:"NOT NULL"`
	TrackID     int64              `xorm:"INDEX"`
	Status      RegistrationStatus `xorm:"NOT NULL DEFAULT 0"`
	TeamID      int64              `xorm:"INDEX"`
	RepoID      int64              `xorm:"INDEX"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(HackathonRegistration))
}

// ErrRegistrationNotExist represents a "RegistrationNotExist" kind of error.
type ErrRegistrationNotExist struct {
	ID int64
}

// IsErrRegistrationNotExist checks if an error is a ErrRegistrationNotExist.
func IsErrRegistrationNotExist(err error) bool {
	_, ok := err.(ErrRegistrationNotExist)
	return ok
}

func (err ErrRegistrationNotExist) Error() string {
	return fmt.Sprintf("hackathon registration does not exist [id: %d]", err.ID)
}

func (err ErrRegistrationNotExist) Unwrap() error {
	return util.ErrNotExist
}

// ErrDuplicateRegistration represents a "DuplicateRegistration" kind of error.
type ErrDuplicateRegistration struct {
	HackathonID int64
	UserID      int64
}

// IsErrDuplicateRegistration checks if an error is a ErrDuplicateRegistration.
func IsErrDuplicateRegistration(err error) bool {
	_, ok := err.(ErrDuplicateRegistration)
	return ok
}

func (err ErrDuplicateRegistration) Error() string {
	return fmt.Sprintf("user already registered [hackathon_id: %d, user_id: %d]", err.HackathonID, err.UserID)
}

func (err ErrDuplicateRegistration) Unwrap() error {
	return util.ErrAlreadyExist
}

// GetRegistration returns a registration by hackathon and user ID.
func GetRegistration(ctx context.Context, hackathonID, userID int64) (*HackathonRegistration, error) {
	r := &HackathonRegistration{}
	has, err := db.GetEngine(ctx).Where("hackathon_id = ? AND user_id = ?", hackathonID, userID).Get(r)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrRegistrationNotExist{ID: 0}
	}
	return r, nil
}

// GetRegistrationByID returns a registration by its ID.
func GetRegistrationByID(ctx context.Context, id int64) (*HackathonRegistration, error) {
	r, exists, err := db.GetByID[HackathonRegistration](ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrRegistrationNotExist{ID: id}
	}
	return r, nil
}

// ListRegistrationsOptions holds options for listing registrations.
type ListRegistrationsOptions struct {
	db.ListOptions
	HackathonID int64
	Status      *RegistrationStatus
}

func (opts ListRegistrationsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.HackathonID != 0 {
		cond = cond.And(builder.Eq{"hackathon_registration.hackathon_id": opts.HackathonID})
	}
	if opts.Status != nil {
		cond = cond.And(builder.Eq{"hackathon_registration.status": *opts.Status})
	}
	return cond
}

// ListRegistrations returns a paginated list of registrations matching the given options.
func ListRegistrations(ctx context.Context, opts ListRegistrationsOptions) ([]*HackathonRegistration, int64, error) {
	return db.FindAndCount[HackathonRegistration](ctx, opts)
}

// CreateRegistration inserts a new registration, returning ErrDuplicateRegistration if already exists.
// If the user already has a solo registration (OrgID == 0) and the new registration carries an OrgID,
// the existing row is upgraded to a team registration (Solo→Team upgrade) instead of returning an error.
func CreateRegistration(ctx context.Context, r *HackathonRegistration) error {
	existing := &HackathonRegistration{}
	has, err := db.GetEngine(ctx).Where("hackathon_id = ? AND user_id = ?", r.HackathonID, r.UserID).Get(existing)
	if err != nil {
		return err
	}
	if has {
		if r.OrgID > 0 && existing.OrgID == 0 {
			// Solo → Team upgrade
			existing.OrgID = r.OrgID
			existing.TeamName = r.TeamName
			cols := []string{"org_id", "team_name"}
			// Carry over RepoID if the upgrade also brought a repo binding (e.g. user
			// initially registered solo, then re-registers with an org repo to bind it).
			if r.RepoID > 0 {
				existing.RepoID = r.RepoID
				cols = append(cols, "repo_id")
			}
			if _, err := db.GetEngine(ctx).ID(existing.ID).Cols(cols...).Update(existing); err != nil {
				return err
			}
			// Mutate caller's `r` to reflect persisted state. Critical for callers
			// that build downstream rows referencing r.ID — without this they
			// silently set RegistrationID = 0 (orphaned submission).
			r.ID = existing.ID
			r.CreatedUnix = existing.CreatedUnix
			r.UpdatedUnix = existing.UpdatedUnix
			r.Status = existing.Status
			return nil
		}
		return ErrDuplicateRegistration{HackathonID: r.HackathonID, UserID: r.UserID}
	}
	return db.Insert(ctx, r)
}

// UpdateRegistrationStatus updates only the status field of a registration.
func UpdateRegistrationStatus(ctx context.Context, id int64, status RegistrationStatus) error {
	_, err := db.GetEngine(ctx).ID(id).Cols("status").Update(&HackathonRegistration{Status: status})
	return err
}

// CountRegistrations returns the number of registrations for a given hackathon.
func CountRegistrations(ctx context.Context, hackathonID int64) (int64, error) {
	return db.GetEngine(ctx).Where("hackathon_id = ?", hackathonID).Count(new(HackathonRegistration))
}

// GetUserHackathons returns hackathons the user is registered for, most recent first.
func GetUserHackathons(ctx context.Context, userID int64, limit int) ([]*Hackathon, error) {
	hackathons := make([]*Hackathon, 0, limit)
	return hackathons, db.GetEngine(ctx).
		Join("INNER", "hackathon_registration", "hackathon_registration.hackathon_id = hackathon.id").
		Where("hackathon_registration.user_id = ?", userID).
		OrderBy("hackathon_registration.created_unix DESC").
		Limit(limit).
		Find(&hackathons)
}
