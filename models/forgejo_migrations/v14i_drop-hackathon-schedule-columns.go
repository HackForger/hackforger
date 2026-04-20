// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"strings"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Drop legacy hackathon schedule columns (migrated to phase table in v14b)",
		Upgrade: func(x *xorm.Engine) error {
			// Old dumps pre-date the HackForger module; nothing to drop. Fresh
			// installs get the hackathon table from xorm.Sync() of the current
			// model definition, which already lacks these legacy columns. This
			// migration only matters for instances upgrading from an earlier
			// HackForger build that had the columns.
			exists, err := x.IsTableExist("hackathon")
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}

			columns := []string{
				"registration_start",
				"registration_end",
				"hacking_start",
				"hacking_end",
				"judging_end",
			}
			for _, col := range columns {
				if _, err := x.Exec("ALTER TABLE hackathon DROP COLUMN " + col); err != nil {
					// Tolerate "no such column" — the legacy column may
					// already be absent on instances that never had it.
					if strings.Contains(err.Error(), "no such column") {
						continue
					}
					return err
				}
			}
			return nil
		},
	})
}
