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
	_, err := x.Exec("CREATE UNIQUE INDEX IF NOT EXISTS UQE_hackathon_submission_user_track ON hackathon_submission(user_id, track_id)")
	return err
}
