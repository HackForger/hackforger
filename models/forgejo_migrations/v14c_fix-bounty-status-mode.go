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
	return nil
}
