// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add custom_name column to phase table",
		Upgrade: func(x *xorm.Engine) error {
			// If the phase table doesn't exist yet (fresh install / old dump
			// pre-dating the HackForger phase system), skip — `phase-tables`
			// migration will create the table with custom_name already in
			// place. This migration only does work for live instances that
			// already had a `phase` table from an earlier HackForger build.
			exists, err := x.IsTableExist("phase")
			if err != nil {
				return err
			}
			if !exists {
				return nil
			}
			type Phase struct {
				CustomName string `xorm:"VARCHAR(100) NOT NULL DEFAULT ''"`
			}
			return x.Sync(new(Phase))
		},
	})
}
