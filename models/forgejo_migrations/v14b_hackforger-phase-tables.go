// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "Create hackforger_phase_type and hackforger_phase tables with seed data",
		Upgrade: func(x *xorm.Engine) error {
			// XORM derives table names from struct names (snake_case).
			// PhaseType → phase_type, Phase → phase (matching existing HackForger convention)
			type PhaseType struct {
				ID              int64  `xorm:"pk autoincr"`
				ActivityKind    string `xorm:"VARCHAR(20) NOT NULL"`
				Key             string `xorm:"VARCHAR(50) NOT NULL"`
				DisplayNameI18n string `xorm:"VARCHAR(100) NOT NULL"`
				IsUnique        bool   `xorm:"NOT NULL DEFAULT true"`
				AllowedActions  string `xorm:"TEXT NOT NULL"`
				DefaultOrder    int    `xorm:"NOT NULL DEFAULT 0"`
				CreatedUnix     int64  `xorm:"created"`
			}

			if err := x.Sync(new(PhaseType)); err != nil {
				return err
			}

			type Phase struct {
				ID           int64  `xorm:"pk autoincr"`
				PhaseTypeID  int64  `xorm:"NOT NULL"`
				ActivityKind string `xorm:"VARCHAR(20) NOT NULL"`
				ActivityID   int64  `xorm:"NOT NULL"`
				SortOrder    int    `xorm:"NOT NULL"`
				StartTime    int64  `xorm:"NOT NULL"`
				EndTime      int64  `xorm:"NOT NULL"`
				CreatedUnix  int64  `xorm:"created"`
				UpdatedUnix  int64  `xorm:"updated"`
			}

			if err := x.Sync(new(Phase)); err != nil {
				return err
			}

			// Create index (table name is "phase" per XORM convention)
			if _, err := x.Exec("CREATE INDEX IF NOT EXISTS IDX_phase_activity ON phase(activity_kind, activity_id, sort_order)"); err != nil {
				return err
			}

			// Seed default phase types
			seeds := []PhaseType{
				// Hackathon phases
				{ActivityKind: "hackathon", Key: "registration", DisplayNameI18n: "hackforger.phase.hackathon.registration", IsUnique: true, AllowedActions: `["register","form_team"]`, DefaultOrder: 1},
				{ActivityKind: "hackathon", Key: "development", DisplayNameI18n: "hackforger.phase.hackathon.development", IsUnique: false, AllowedActions: `["submit_work"]`, DefaultOrder: 2},
				{ActivityKind: "hackathon", Key: "judging", DisplayNameI18n: "hackforger.phase.hackathon.judging", IsUnique: true, AllowedActions: `["score"]`, DefaultOrder: 3},
				{ActivityKind: "hackathon", Key: "results", DisplayNameI18n: "hackforger.phase.hackathon.results", IsUnique: true, AllowedActions: `["publish_results"]`, DefaultOrder: 4},
				// Bounty phases
				{ActivityKind: "bounty", Key: "open", DisplayNameI18n: "hackforger.phase.bounty.open", IsUnique: true, AllowedActions: `["apply","claim"]`, DefaultOrder: 1},
				{ActivityKind: "bounty", Key: "in_progress", DisplayNameI18n: "hackforger.phase.bounty.in_progress", IsUnique: true, AllowedActions: `["submit_pr"]`, DefaultOrder: 2},
				{ActivityKind: "bounty", Key: "review", DisplayNameI18n: "hackforger.phase.bounty.review", IsUnique: true, AllowedActions: `["review"]`, DefaultOrder: 3},
				{ActivityKind: "bounty", Key: "closed", DisplayNameI18n: "hackforger.phase.bounty.closed", IsUnique: true, AllowedActions: `["settle"]`, DefaultOrder: 4},
				// Grant phases
				{ActivityKind: "grant", Key: "application", DisplayNameI18n: "hackforger.phase.grant.application", IsUnique: true, AllowedActions: `["apply"]`, DefaultOrder: 1},
				{ActivityKind: "grant", Key: "review", DisplayNameI18n: "hackforger.phase.grant.review", IsUnique: true, AllowedActions: `["review","approve"]`, DefaultOrder: 2},
				{ActivityKind: "grant", Key: "disbursement", DisplayNameI18n: "hackforger.phase.grant.disbursement", IsUnique: true, AllowedActions: `["disburse"]`, DefaultOrder: 3},
			}

			for i := range seeds {
				if _, err := x.Insert(&seeds[i]); err != nil {
					return err
				}
			}

			return nil
		},
	})
}
