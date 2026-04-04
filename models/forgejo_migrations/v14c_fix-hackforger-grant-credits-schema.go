// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "fix HackForger grant+credits table schemas to match Go models",
		Upgrade:     fixHackforgerGrantCreditsSchema,
	})
}

// --- Corrected structs matching Go models exactly ---

type v14cFixGrantRound struct {
	ID            int64              `xorm:"pk autoincr"`
	OrgID         int64              `xorm:"INDEX NOT NULL"`
	OwnerID       int64              `xorm:"INDEX NOT NULL"`
	Name          string             `xorm:"NOT NULL"`
	Slug          string             `xorm:"UNIQUE NOT NULL"`
	Description   string             `xorm:"TEXT"`
	Status        int                `xorm:"NOT NULL DEFAULT 0"`
	Budget        float64            `xorm:"NOT NULL DEFAULT 0"`
	Currency      string             `xorm:"VARCHAR(16) NOT NULL DEFAULT 'USD'"`
	BudgetCredits int64              `xorm:"NOT NULL DEFAULT 0"`
	Deadline      timeutil.TimeStamp `xorm:""`
	CreatedUnix   timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix   timeutil.TimeStamp `xorm:"INDEX updated"`
}

func (v14cFixGrantRound) TableName() string { return "grant_round" }

type v14cFixGrantProject struct {
	ID           int64              `xorm:"pk autoincr"`
	RoundID      int64              `xorm:"INDEX NOT NULL"`
	UserID       int64              `xorm:"INDEX NOT NULL"`
	RepoID       int64              `xorm:"INDEX"`
	Title        string             `xorm:"NOT NULL"`
	Description  string             `xorm:"TEXT"`
	Status       int                `xorm:"NOT NULL DEFAULT 0"`
	AwardAmount  float64            `xorm:"NOT NULL DEFAULT 0"`
	AwardCredits int64              `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix  timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}

func (v14cFixGrantProject) TableName() string { return "grant_project" }

type v14cFixCreditAccount struct {
	ID          int64              `xorm:"pk autoincr"`
	UserID      int64              `xorm:"UNIQUE NOT NULL"`
	Balance     int64              `xorm:"NOT NULL DEFAULT 0"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func (v14cFixCreditAccount) TableName() string { return "credit_account" }

type v14cFixCreditTransaction struct {
	ID          int64              `xorm:"pk autoincr"`
	UserID      int64              `xorm:"INDEX NOT NULL"`
	Type        string             `xorm:"VARCHAR(16) NOT NULL"`
	Amount      int64              `xorm:"NOT NULL"`
	Balance     int64              `xorm:"NOT NULL"`
	Reference   string             `xorm:"VARCHAR(255)"`
	Note        string             `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func (v14cFixCreditTransaction) TableName() string { return "credit_transaction" }

type v14cFixRedeemOption struct {
	ID          int64              `xorm:"pk autoincr"`
	Name        string             `xorm:"NOT NULL"`
	Description string             `xorm:"TEXT"`
	Cost        int64              `xorm:"NOT NULL"`
	Stock       int                `xorm:"NOT NULL DEFAULT -1"`
	IsActive    bool               `xorm:"NOT NULL DEFAULT true"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func (v14cFixRedeemOption) TableName() string { return "redeem_option" }

type v14cFixRedeemOrder struct {
	ID          int64              `xorm:"pk autoincr"`
	UserID      int64              `xorm:"INDEX NOT NULL"`
	OptionID    int64              `xorm:"INDEX NOT NULL"`
	Cost        int64              `xorm:"NOT NULL"`
	Status      string             `xorm:"VARCHAR(16) NOT NULL DEFAULT 'pending'"`
	FulfillNote string             `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func (v14cFixRedeemOrder) TableName() string { return "redeem_order" }

func fixHackforgerGrantCreditsSchema(x *xorm.Engine) error {
	// Pre-production: safe to drop and recreate. Order matters due to
	// potential foreign-key-like references (grant_project -> grant_round).
	tables := []string{
		"grant_project",
		"grant_round",
		"credit_transaction",
		"credit_account",
		"redeem_order",
		"redeem_option",
	}
	for _, t := range tables {
		if _, err := x.Exec("DROP TABLE IF EXISTS " + t); err != nil {
			return err
		}
	}

	return x.Sync(
		new(v14cFixGrantRound),
		new(v14cFixGrantProject),
		new(v14cFixCreditAccount),
		new(v14cFixCreditTransaction),
		new(v14cFixRedeemOption),
		new(v14cFixRedeemOrder),
	)
}
