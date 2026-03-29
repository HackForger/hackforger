// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "add org/repo integration fields to hackathon tables",
		Upgrade:     addHackathonOrgRepoFields,
	})
}

type v14fHackathon struct {
	LinkedOrgID int64 `xorm:"INDEX DEFAULT 0"`
}

func (v14fHackathon) TableName() string { return "hackathon" }

type v14fHackathonRegistration struct {
	OrgID int64 `xorm:"INDEX DEFAULT 0"`
}

func (v14fHackathonRegistration) TableName() string { return "hackathon_registration" }

type v14fHackathonSubmission struct {
	ForkRepoID int64 `xorm:"INDEX DEFAULT 0"`
	PRID       int64 `xorm:"INDEX DEFAULT 0"`
	PullIndex  int64 `xorm:"DEFAULT 0"`
}

func (v14fHackathonSubmission) TableName() string { return "hackathon_submission" }

func addHackathonOrgRepoFields(x *xorm.Engine) error {
	if err := x.Sync(new(v14fHackathon)); err != nil {
		return err
	}
	if err := x.Sync(new(v14fHackathonRegistration)); err != nil {
		return err
	}
	return x.Sync(new(v14fHackathonSubmission))
}
