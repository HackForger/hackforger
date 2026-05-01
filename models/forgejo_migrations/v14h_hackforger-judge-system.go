// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"context"

	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add judge criteria tables, drop fork_repo_id from submissions",
		Upgrade:     addJudgeSystemTables,
	})
}

// New tables

type v14hHackathonJudgeCriteria struct {
	ID          int64              `xorm:"pk autoincr"`
	HackathonID int64             `xorm:"INDEX NOT NULL"`
	Name        string            `xorm:"NOT NULL"`
	Description string            `xorm:"TEXT"`
	MaxScore    float64           `xorm:"NOT NULL DEFAULT 10"`
	Weight      float64           `xorm:"NOT NULL DEFAULT 25"`
	SortOrder   int               `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func (v14hHackathonJudgeCriteria) TableName() string { return "hackathon_judge_criteria" }

type v14hHackathonTrackCriteria struct {
	ID         int64   `xorm:"pk autoincr"`
	TrackID    int64   `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CriteriaID int64  `xorm:"UNIQUE(s) INDEX NOT NULL"`
	Enabled    bool    `xorm:"NOT NULL DEFAULT true"`
	Weight     float64 `xorm:"NOT NULL DEFAULT 0"`
}

func (v14hHackathonTrackCriteria) TableName() string { return "hackathon_track_criteria" }

// Delta structs for existing table modifications

type v14hHackathonJudge struct {
	TrackID int64 `xorm:"INDEX DEFAULT 0"`
}

func (v14hHackathonJudge) TableName() string { return "hackathon_judge" }

type v14hHackathonJudgeScore struct {
	CriteriaID int64 `xorm:"INDEX DEFAULT 0"`
}

func (v14hHackathonJudgeScore) TableName() string { return "hackathon_judge_score" }

func addJudgeSystemTables(x *xorm.Engine) error {
	// Create new tables
	if err := x.Sync(new(v14hHackathonJudgeCriteria)); err != nil {
		return err
	}
	if err := x.Sync(new(v14hHackathonTrackCriteria)); err != nil {
		return err
	}

	// Add new columns to existing tables
	if err := x.Sync(new(v14hHackathonJudge)); err != nil {
		return err
	}
	if err := x.Sync(new(v14hHackathonJudgeScore)); err != nil {
		return err
	}

	// Drop fork_repo_id from hackathon_submission
	// SQLite requires dropping the index first
	if _, err := x.Exec("DROP INDEX IF EXISTS IDX_hackathon_submission_fork_repo_id"); err != nil {
		return err
	}
	// Check if column exists before dropping (idempotent).
	// PG-safe: PRAGMA is SQLite-only and would syntax-error in PG. Use XORM's
	// dialect-agnostic IsColumnExist instead. See docs/notes/pg-migration-pitfalls.md.
	hasCol, err := x.Dialect().IsColumnExist(x.DB(), context.Background(), "hackathon_submission", "fork_repo_id")
	if err != nil {
		return err
	}
	if hasCol {
		if _, err := x.Exec("ALTER TABLE hackathon_submission DROP COLUMN fork_repo_id"); err != nil {
			return err
		}
	}

	return nil
}
