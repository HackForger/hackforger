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
			// PG-safe: is_unique is BOOLEAN in PG; integer literal 0 raises a type error.
			// Use parameterized boolean (works in both PG and SQLite).
			// See docs/notes/pg-migration-pitfalls.md.
			_, err := x.Exec("UPDATE phase_type SET is_unique = ? WHERE activity_kind = ? AND key = ?", false, "hackathon", "judging")
			return err
		},
	})
}
