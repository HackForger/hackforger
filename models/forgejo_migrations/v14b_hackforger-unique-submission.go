// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "Add unique constraint on hackathon_submission(user_id, track_id)",
		Upgrade:     addUniqueSubmissionUserTrack,
	})
}

func addUniqueSubmissionUserTrack(x *xorm.Engine) error {
	// Old dumps pre-date the HackForger module; nothing to constrain. The
	// hackathon_submission table is created later by xorm.Sync() from the
	// model definition (which already declares the unique index), so this
	// dedupe + index migration only matters for instances upgrading from
	// an earlier HackForger build.
	exists, err := x.IsTableExist("hackathon_submission")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// Remove duplicate submissions before adding unique constraint.
	// Keep the latest submission (highest id) for each user+track pair.
	if _, err := x.Exec(`DELETE FROM hackathon_submission WHERE id NOT IN (
		SELECT MAX(id) FROM hackathon_submission GROUP BY user_id, track_id
	)`); err != nil {
		return err
	}
	_, err = x.Exec("CREATE UNIQUE INDEX IF NOT EXISTS UQE_hackathon_submission_user_track ON hackathon_submission(user_id, track_id)")
	return err
}
