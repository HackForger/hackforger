// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "add hackathon judge assignment table",
		Upgrade:     addHackathonJudgeTable,
	})
}

type v14eHackathonJudge struct {
	ID          int64 `xorm:"pk autoincr"`
	HackathonID int64 `xorm:"UNIQUE(s) INDEX NOT NULL"`
	UserID      int64 `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CreatedUnix int64 `xorm:"created"`
}

func (v14eHackathonJudge) TableName() string { return "hackathon_judge" }

func addHackathonJudgeTable(x *xorm.Engine) error {
	return x.Sync(new(v14eHackathonJudge))
}
