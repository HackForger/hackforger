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
	// Remove duplicate submissions before adding unique constraint.
	// Keep the latest submission (highest id) for each user+track pair.
	if _, err := x.Exec(`DELETE FROM hackathon_submission WHERE id NOT IN (
		SELECT MAX(id) FROM hackathon_submission GROUP BY user_id, track_id
	)`); err != nil {
		return err
	}
	_, err := x.Exec("CREATE UNIQUE INDEX IF NOT EXISTS UQE_hackathon_submission_user_track ON hackathon_submission(user_id, track_id)")
	return err
}
