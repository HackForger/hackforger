// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add HackForger phase 2 tables (hackforger_action, hackforger_setting) and reputation tier",
		Upgrade:     addHackforgerPhase2Tables,
	})
}

type v14gHackforgerAction struct {
	ID          int64              `xorm:"pk autoincr"`
	UserID      int64              `xorm:"NOT NULL INDEX(idx_hfa_user_created)"`
	ActUserID   int64              `xorm:"NOT NULL INDEX(idx_hfa_actuser_created)"`
	OpType      int                `xorm:"NOT NULL"`
	EntityType  string             `xorm:"VARCHAR(20) NOT NULL INDEX(idx_hfa_entity)"`
	EntityID    int64              `xorm:"NOT NULL DEFAULT 0"`
	EntityName  string             `xorm:"VARCHAR(255)"`
	EntitySlug  string             `xorm:"VARCHAR(255)"`
	OrgID       int64              `xorm:"DEFAULT 0"`
	RepoID      int64              `xorm:"DEFAULT 0"`
	Content     string             `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX(idx_hfa_user_created) INDEX(idx_hfa_actuser_created) INDEX(idx_hfa_entity) created"`
}

func (v14gHackforgerAction) TableName() string { return "hackforger_action" }

type v14gHackforgerSetting struct {
	ID    int64  `xorm:"pk autoincr"`
	Key   string `xorm:"VARCHAR(100) UNIQUE NOT NULL"`
	Value string `xorm:"TEXT NOT NULL"`
}

func (v14gHackforgerSetting) TableName() string { return "hackforger_setting" }

// v14gReputation is a delta struct: only the new Tier column is declared so
// that x.Sync adds it to the existing reputation table without touching other
// columns.
type v14gReputation struct {
	Tier string `xorm:"VARCHAR(20) NOT NULL DEFAULT 'Bronze'"`
}

func (v14gReputation) TableName() string { return "reputation" }

func addHackforgerPhase2Tables(x *xorm.Engine) error {
	if err := x.Sync(new(v14gHackforgerAction)); err != nil {
		return err
	}

	if err := x.Sync(new(v14gHackforgerSetting)); err != nil {
		return err
	}

	if err := x.Sync(new(v14gReputation)); err != nil {
		return err
	}

	defaults := []v14gHackforgerSetting{
		{Key: "reputation.weights", Value: `{"stars":1,"bounties_completed":5,"hackathon_wins":10,"grants_received":3,"credits_earned":0.1}`},
		{Key: "reputation.tiers", Value: `[{"name":"Bronze","min":0},{"name":"Silver","min":50},{"name":"Gold","min":200},{"name":"Diamond","min":500}]`},
	}
	for _, s := range defaults {
		has, err := x.Where("`key` = ?", s.Key).Exist(new(v14gHackforgerSetting))
		if err != nil {
			return err
		}
		if !has {
			if _, err := x.Insert(&s); err != nil {
				return err
			}
		}
	}
	return nil
}
