// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "fix bounty status and mode values for Phase 1",
		Upgrade:     fixBountyStatusMode,
	})
}

func fixBountyStatusMode(x *xorm.Engine) error {
	// Remap old Cancelled(3) → new Cancelled(6)
	if _, err := x.Exec("UPDATE bounty SET status = 6 WHERE status = 3"); err != nil {
		return err
	}
	// Remove any Invitation mode(2) rows
	if _, err := x.Exec("DELETE FROM bounty WHERE mode = 2"); err != nil {
		return err
	}

	// Fix column name mismatches between P0 migration and Go models.
	// These use ALTER TABLE ... RENAME COLUMN (SQLite 3.25+, all MySQL/PostgreSQL).
	// Errors are ignored because the column may already have the correct name
	// when running from scratch or re-running migrations.

	// bounty table: creator_id → publisher_id, add title column
	_, _ = x.Exec("ALTER TABLE bounty RENAME COLUMN creator_id TO publisher_id")
	_, _ = x.Exec("ALTER TABLE bounty ADD COLUMN title TEXT NOT NULL DEFAULT ''")

	// bounty_reward table: reward_type → type, description → note, add credits + rank columns
	_, _ = x.Exec("ALTER TABLE bounty_reward RENAME COLUMN reward_type TO type")
	_, _ = x.Exec("ALTER TABLE bounty_reward RENAME COLUMN description TO note")
	_, _ = x.Exec("ALTER TABLE bounty_reward ADD COLUMN credits INTEGER NOT NULL DEFAULT 0")
	_, _ = x.Exec("ALTER TABLE bounty_reward ADD COLUMN rank INTEGER NOT NULL DEFAULT 1")

	// bounty_application table: note → message
	_, _ = x.Exec("ALTER TABLE bounty_application RENAME COLUMN note TO message")

	return nil
}
