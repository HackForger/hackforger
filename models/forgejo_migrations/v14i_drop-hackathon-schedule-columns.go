// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Drop legacy hackathon schedule columns (migrated to phase table in v14b)",
		Upgrade: func(x *xorm.Engine) error {
			columns := []string{
				"registration_start",
				"registration_end",
				"hacking_start",
				"hacking_end",
				"judging_end",
			}
			for _, col := range columns {
				if _, err := x.Exec("ALTER TABLE hackathon DROP COLUMN " + col); err != nil {
					return err
				}
			}
			return nil
		},
	})
}
