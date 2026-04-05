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
			type Phase struct {
				CustomName string `xorm:"VARCHAR(100) NOT NULL DEFAULT ''"`
			}
			return x.Sync(new(Phase))
		},
	})
}
