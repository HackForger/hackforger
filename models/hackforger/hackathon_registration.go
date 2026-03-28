// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
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

// TableName returns the XORM table name for HackathonRegistration.
func (r *HackathonRegistration) TableName() string {
	return "hackforger_hackathon_registration"
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
