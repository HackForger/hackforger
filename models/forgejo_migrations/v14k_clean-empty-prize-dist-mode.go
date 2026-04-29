// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "Backfill empty prize_dist_mode to 'winner_takes_all'",
		Upgrade:     cleanEmptyPrizeDistMode,
	})
}

func cleanEmptyPrizeDistMode(x *xorm.Engine) error {
	_, err := x.Exec(
		"UPDATE hackathon_track SET prize_dist_mode = 'winner_takes_all' " +
			"WHERE prize_dist_mode = '' OR prize_dist_mode IS NULL",
	)
	return err
}
