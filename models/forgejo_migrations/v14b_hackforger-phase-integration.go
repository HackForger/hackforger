// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"time"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Rename hackathon.status to status_cache, add is_published, migrate time fields to phase records",
		Upgrade: func(x *xorm.Engine) error {
			// 1. Rename status -> status_cache
			if _, err := x.Exec("ALTER TABLE hackathon RENAME COLUMN status TO status_cache"); err != nil {
				return err
			}

			// 2. Add is_published column
			if _, err := x.Exec("ALTER TABLE hackathon ADD COLUMN is_published INTEGER NOT NULL DEFAULT 0"); err != nil {
				return err
			}

			// 3. Set is_published = true for non-Draft hackathons
			if _, err := x.Exec("UPDATE hackathon SET is_published = 1 WHERE status_cache > 0"); err != nil {
				return err
			}

			// 4. Look up phase_type IDs for hackathon kinds
			type ptRow struct {
				ID  int64  `xorm:"id"`
				Key string `xorm:"key"`
			}
			var pts []ptRow
			if err := x.SQL("SELECT id, key FROM phase_type WHERE activity_kind = 'hackathon'").Find(&pts); err != nil {
				return err
			}
			ptMap := make(map[string]int64)
			for _, pt := range pts {
				ptMap[pt.Key] = pt.ID
			}

			// 5. For each published hackathon, generate Phase records from time fields
			type hackRow struct {
				ID                int64 `xorm:"id"`
				StatusCache       int   `xorm:"status_cache"`
				RegistrationStart int64 `xorm:"registration_start"`
				RegistrationEnd   int64 `xorm:"registration_end"`
				HackingStart      int64 `xorm:"hacking_start"`
				HackingEnd        int64 `xorm:"hacking_end"`
				JudgingEnd        int64 `xorm:"judging_end"`
				CreatedUnix       int64 `xorm:"created_unix"`
			}
			var hacks []hackRow
			if err := x.SQL("SELECT id, status_cache, registration_start, registration_end, hacking_start, hacking_end, judging_end, created_unix FROM hackathon WHERE status_cache > 0").Find(&hacks); err != nil {
				return err
			}

			now := time.Now().Unix()
			for _, h := range hacks {
				order := 1
				// Helper: use value or reasonable default
				orDefault := func(val, fallback int64) int64 {
					if val > 0 {
						return val
					}
					return fallback
				}

				// Registration phase
				regStart := orDefault(h.RegistrationStart, h.CreatedUnix)
				regEnd := orDefault(h.RegistrationEnd, regStart+7*86400) // 7 days default
				if ptID, ok := ptMap["registration"]; ok && regEnd > regStart {
					if _, err := x.Exec(
						"INSERT INTO phase (phase_type_id, activity_kind, activity_id, sort_order, start_time, end_time, created_unix, updated_unix) VALUES (?, 'hackathon', ?, ?, ?, ?, ?, ?)",
						ptID, h.ID, order, regStart, regEnd, now, now,
					); err != nil {
						return err
					}
					order++
				}

				// Development phase
				devStart := orDefault(h.HackingStart, regEnd)
				devEnd := orDefault(h.HackingEnd, devStart+14*86400) // 14 days default
				if ptID, ok := ptMap["development"]; ok && devEnd > devStart {
					if _, err := x.Exec(
						"INSERT INTO phase (phase_type_id, activity_kind, activity_id, sort_order, start_time, end_time, created_unix, updated_unix) VALUES (?, 'hackathon', ?, ?, ?, ?, ?, ?)",
						ptID, h.ID, order, devStart, devEnd, now, now,
					); err != nil {
						return err
					}
					order++
				}

				// Judging phase
				judgStart := devEnd
				judgEnd := orDefault(h.JudgingEnd, judgStart+7*86400) // 7 days default
				if ptID, ok := ptMap["judging"]; ok && judgEnd > judgStart {
					if _, err := x.Exec(
						"INSERT INTO phase (phase_type_id, activity_kind, activity_id, sort_order, start_time, end_time, created_unix, updated_unix) VALUES (?, 'hackathon', ?, ?, ?, ?, ?, ?)",
						ptID, h.ID, order, judgStart, judgEnd, now, now,
					); err != nil {
						return err
					}
					order++
				}

				// Results phase
				resStart := judgEnd
				resEnd := resStart + 30*86400 // 30 days default
				if ptID, ok := ptMap["results"]; ok {
					if _, err := x.Exec(
						"INSERT INTO phase (phase_type_id, activity_kind, activity_id, sort_order, start_time, end_time, created_unix, updated_unix) VALUES (?, 'hackathon', ?, ?, ?, ?, ?, ?)",
						ptID, h.ID, order, resStart, resEnd, now, now,
					); err != nil {
						return err
					}
				}
			}

			// NOTE: Old time columns (registration_start, etc.) are intentionally NOT dropped.
			// They will be removed in a future migration after validation.
			return nil
		},
	})
}
