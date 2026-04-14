// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Allow multiple judging phases (set is_unique=false for hackathon judging phase type)",
		Upgrade: func(x *xorm.Engine) error {
			_, err := x.Exec("UPDATE phase_type SET is_unique = 0 WHERE activity_kind = 'hackathon' AND key = 'judging'")
			return err
		},
	})
}
