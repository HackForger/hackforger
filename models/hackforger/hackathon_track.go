// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

// HackathonTrack represents a prize track within a hackathon.
type HackathonTrack struct {
	ID            int64              `xorm:"pk autoincr"`
	HackathonID   int64              `xorm:"INDEX NOT NULL"`
	Name          string             `xorm:"NOT NULL"`
	Description   string             `xorm:"TEXT"`
	PrizeAmount   float64            `xorm:""`
	PrizeCurrency string             `xorm:"VARCHAR(16)"`
	PrizeCredits  int64              `xorm:""`
	CreatedUnix   timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(HackathonTrack))
}

// TableName returns the XORM table name for HackathonTrack.
func (t *HackathonTrack) TableName() string {
	return "hackforger_hackathon_track"
}
