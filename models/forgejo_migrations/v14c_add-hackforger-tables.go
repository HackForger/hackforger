// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add HackForger tables (hackathon, bounty, grant, credits, reputation)",
		Upgrade:     addHackforgerTables,
	})
}

// --- Hackathon tables (5) ---

type v14cHackathon struct {
	ID               int64  `xorm:"pk autoincr"`
	Name             string `xorm:"NOT NULL"`
	Slug             string `xorm:"UNIQUE NOT NULL"`
	Description      string `xorm:"TEXT"`
	OrgID            int64  `xorm:"INDEX NOT NULL"`
	CreatorID        int64  `xorm:"NOT NULL"`
	Status           int    `xorm:"DEFAULT 0 INDEX"`     // 0=Draft,1=Registration,2=Hacking,3=Judging,4=Finished
	RegistrationEnd  int64  `xorm:"DEFAULT 0"`           // unix timestamp
	HackingEnd       int64  `xorm:"DEFAULT 0"`           // unix timestamp
	JudgingEnd       int64  `xorm:"DEFAULT 0"`           // unix timestamp
	MaxTeamSize      int    `xorm:"DEFAULT 5"`
	MinTeamSize      int    `xorm:"DEFAULT 1"`
	BannerURL        string `xorm:"TEXT"`
	Rules            string `xorm:"TEXT"`
	CustomFields     string `xorm:"TEXT"`                 // JSON
	TemplateRepoID   int64  `xorm:"DEFAULT 0"`
	CreatedUnix      int64  `xorm:"created"`
	UpdatedUnix      int64  `xorm:"updated"`
}

func (v14cHackathon) TableName() string { return "hackathon" }

type v14cHackathonTrack struct {
	ID           int64  `xorm:"pk autoincr"`
	HackathonID  int64  `xorm:"INDEX NOT NULL"`
	Name         string `xorm:"NOT NULL"`
	Description  string `xorm:"TEXT"`
	PrizeDesc    string `xorm:"TEXT"`
	SortOrder    int    `xorm:"DEFAULT 0"`
	CreatedUnix  int64  `xorm:"created"`
}

func (v14cHackathonTrack) TableName() string { return "hackathon_track" }

type v14cHackathonRegistration struct {
	ID           int64  `xorm:"pk autoincr"`
	HackathonID  int64  `xorm:"INDEX NOT NULL"`
	UserID       int64  `xorm:"INDEX NOT NULL"`
	TeamID       int64  `xorm:"DEFAULT 0"`           // org team ID, 0 = solo
	Status       int    `xorm:"DEFAULT 0"`           // 0=Pending,1=Approved,2=Rejected
	Note         string `xorm:"TEXT"`
	CreatedUnix  int64  `xorm:"created"`
	UpdatedUnix  int64  `xorm:"updated"`
}

func (v14cHackathonRegistration) TableName() string { return "hackathon_registration" }

type v14cHackathonSubmission struct {
	ID           int64   `xorm:"pk autoincr"`
	HackathonID  int64   `xorm:"INDEX NOT NULL"`
	TrackID      int64   `xorm:"INDEX DEFAULT 0"`
	UserID       int64   `xorm:"INDEX NOT NULL"`
	TeamID       int64   `xorm:"DEFAULT 0"`
	RepoID       int64   `xorm:"DEFAULT 0"`
	Title        string  `xorm:"NOT NULL"`
	Description  string  `xorm:"TEXT"`
	DemoURL      string  `xorm:"TEXT"`
	Status       int     `xorm:"DEFAULT 0"`           // 0=Draft,1=Submitted,2=Disqualified
	TotalScore   float64 `xorm:"DEFAULT 0"`
	Rank         int     `xorm:"DEFAULT 0"`
	CreatedUnix  int64   `xorm:"created"`
	UpdatedUnix  int64   `xorm:"updated"`
}

func (v14cHackathonSubmission) TableName() string { return "hackathon_submission" }

type v14cHackathonJudgeScore struct {
	ID           int64   `xorm:"pk autoincr"`
	HackathonID  int64   `xorm:"INDEX NOT NULL"`
	SubmissionID int64   `xorm:"INDEX NOT NULL"`
	JudgeID      int64   `xorm:"INDEX NOT NULL"`
	Score        float64 `xorm:"DEFAULT 0"`
	Comment      string  `xorm:"TEXT"`
	Criteria     string  `xorm:"TEXT"`                 // JSON: per-criterion scores
	CreatedUnix  int64   `xorm:"created"`
	UpdatedUnix  int64   `xorm:"updated"`
}

func (v14cHackathonJudgeScore) TableName() string { return "hackathon_judge_score" }

// --- Bounty tables (4) ---

type v14cBounty struct {
	ID           int64   `xorm:"pk autoincr"`
	RepoID       int64   `xorm:"INDEX NOT NULL"`
	IssueID      int64   `xorm:"UNIQUE NOT NULL"`
	CreatorID    int64   `xorm:"NOT NULL"`
	Mode         int     `xorm:"DEFAULT 0"`           // 0=Exclusive,1=Competitive
	Status       int     `xorm:"DEFAULT 0 INDEX"`     // 0=Open,1=Claimed,2=InReview,3=Completed,4=Paid,5=Expired,6=Cancelled
	ClaimerID    int64   `xorm:"DEFAULT 0"`           // exclusive mode: the accepted claimer
	MaxWinners   int     `xorm:"DEFAULT 1"`           // competitive mode: max winners
	Deadline     int64   `xorm:"DEFAULT 0"`           // unix timestamp
	CreatedUnix  int64   `xorm:"created"`
	UpdatedUnix  int64   `xorm:"updated"`
}

func (v14cBounty) TableName() string { return "bounty" }

type v14cBountyReward struct {
	ID          int64   `xorm:"pk autoincr"`
	BountyID    int64   `xorm:"INDEX NOT NULL"`
	RewardType  string  `xorm:"NOT NULL"`             // "cash", "credits", "swag", "other"
	Currency    string  `xorm:"DEFAULT ''"`           // e.g. "USD", "CNY", "credits"
	Amount      float64 `xorm:"DEFAULT 0"`
	Description string  `xorm:"TEXT"`
	CreatedUnix int64   `xorm:"created"`
}

func (v14cBountyReward) TableName() string { return "bounty_reward" }

type v14cBountyApplication struct {
	ID          int64  `xorm:"pk autoincr"`
	BountyID    int64  `xorm:"INDEX NOT NULL"`
	UserID      int64  `xorm:"INDEX NOT NULL"`
	Status      int    `xorm:"DEFAULT 0"`             // 0=Pending,1=Accepted,2=Rejected
	Note        string `xorm:"TEXT"`
	CreatedUnix int64  `xorm:"created"`
	UpdatedUnix int64  `xorm:"updated"`
}

func (v14cBountyApplication) TableName() string { return "bounty_application" }

type v14cBountyWinner struct {
	ID          int64   `xorm:"pk autoincr"`
	BountyID    int64   `xorm:"INDEX NOT NULL"`
	UserID      int64   `xorm:"INDEX NOT NULL"`
	Rank        int     `xorm:"DEFAULT 0"`
	RewardID    int64   `xorm:"DEFAULT 0"`            // linked bounty_reward
	Amount      float64 `xorm:"DEFAULT 0"`            // actual amount awarded
	CreatedUnix int64   `xorm:"created"`
}

func (v14cBountyWinner) TableName() string { return "bounty_winner" }

// --- Grant tables (2) ---

type v14cGrantRound struct {
	ID           int64   `xorm:"pk autoincr"`
	Name         string  `xorm:"NOT NULL"`
	Slug         string  `xorm:"UNIQUE NOT NULL"`
	Description  string  `xorm:"TEXT"`
	OrgID        int64   `xorm:"INDEX NOT NULL"`
	CreatorID    int64   `xorm:"NOT NULL"`
	Status       int     `xorm:"DEFAULT 0 INDEX"`     // 0=Draft,1=Open,2=Reviewing,3=Finalized,4=Distributed
	TotalBudget  float64 `xorm:"DEFAULT 0"`
	Currency     string  `xorm:"DEFAULT ''"`
	Deadline     int64   `xorm:"DEFAULT 0"`           // unix timestamp
	CreatedUnix  int64   `xorm:"created"`
	UpdatedUnix  int64   `xorm:"updated"`
}

func (v14cGrantRound) TableName() string { return "grant_round" }

type v14cGrantProject struct {
	ID           int64   `xorm:"pk autoincr"`
	RoundID      int64   `xorm:"INDEX NOT NULL"`
	RepoID       int64   `xorm:"DEFAULT 0"`
	UserID       int64   `xorm:"INDEX NOT NULL"`       // applicant
	Title        string  `xorm:"NOT NULL"`
	Description  string  `xorm:"TEXT"`
	RequestedAmt float64 `xorm:"DEFAULT 0"`
	AwardedAmt   float64 `xorm:"DEFAULT 0"`
	Status       int     `xorm:"DEFAULT 0"`            // 0=Pending,1=Approved,2=Rejected,3=Awarded
	StarCount    int64   `xorm:"DEFAULT 0"`            // cached star count as signal
	ReviewNote   string  `xorm:"TEXT"`
	CreatedUnix  int64   `xorm:"created"`
	UpdatedUnix  int64   `xorm:"updated"`
}

func (v14cGrantProject) TableName() string { return "grant_project" }

// --- Credits tables (4) ---

type v14cCreditAccount struct {
	ID          int64   `xorm:"pk autoincr"`
	UserID      int64   `xorm:"UNIQUE NOT NULL"`
	Balance     float64 `xorm:"DEFAULT 0"`
	TotalEarned float64 `xorm:"DEFAULT 0"`
	TotalSpent  float64 `xorm:"DEFAULT 0"`
	CreatedUnix int64   `xorm:"created"`
	UpdatedUnix int64   `xorm:"updated"`
}

func (v14cCreditAccount) TableName() string { return "credit_account" }

type v14cCreditTransaction struct {
	ID          int64   `xorm:"pk autoincr"`
	AccountID   int64   `xorm:"INDEX NOT NULL"`
	UserID      int64   `xorm:"INDEX NOT NULL"`
	Type        int     `xorm:"NOT NULL"`              // 0=Deposit,1=Withdraw,2=Redeem
	Amount      float64 `xorm:"NOT NULL"`
	Balance     float64 `xorm:"NOT NULL"`              // balance after transaction
	Reason      string  `xorm:"NOT NULL"`              // e.g. "bounty_completed", "hackathon_prize", "redeem"
	RefType     string  `xorm:"DEFAULT ''"`            // "bounty", "hackathon", "grant", "redeem_order", "admin"
	RefID       int64   `xorm:"DEFAULT 0"`
	Note        string  `xorm:"TEXT"`
	CreatedUnix int64   `xorm:"created"`
}

func (v14cCreditTransaction) TableName() string { return "credit_transaction" }

type v14cRedeemOption struct {
	ID          int64   `xorm:"pk autoincr"`
	Name        string  `xorm:"NOT NULL"`
	Description string  `xorm:"TEXT"`
	Category    string  `xorm:"DEFAULT ''"`            // e.g. "compute", "license", "swag"
	Cost        float64 `xorm:"NOT NULL"`              // credits cost
	Stock       int     `xorm:"DEFAULT -1"`            // -1 = unlimited
	IsActive    bool    `xorm:"DEFAULT true"`
	CreatedUnix int64   `xorm:"created"`
	UpdatedUnix int64   `xorm:"updated"`
}

func (v14cRedeemOption) TableName() string { return "redeem_option" }

type v14cRedeemOrder struct {
	ID          int64   `xorm:"pk autoincr"`
	UserID      int64   `xorm:"INDEX NOT NULL"`
	OptionID    int64   `xorm:"INDEX NOT NULL"`
	Cost        float64 `xorm:"NOT NULL"`
	Status      int     `xorm:"DEFAULT 0"`             // 0=Pending,1=Fulfilled,2=Cancelled
	Note        string  `xorm:"TEXT"`
	CreatedUnix int64   `xorm:"created"`
	UpdatedUnix int64   `xorm:"updated"`
}

func (v14cRedeemOrder) TableName() string { return "redeem_order" }

// --- Reputation table (1) ---

type v14cReputation struct {
	ID              int64   `xorm:"pk autoincr"`
	UserID          int64   `xorm:"UNIQUE NOT NULL"`
	Score           float64 `xorm:"DEFAULT 0"`
	BountyCount     int     `xorm:"DEFAULT 0"`
	BountyScore     float64 `xorm:"DEFAULT 0"`
	HackathonCount  int     `xorm:"DEFAULT 0"`
	HackathonScore  float64 `xorm:"DEFAULT 0"`
	GrantCount      int     `xorm:"DEFAULT 0"`
	GrantScore      float64 `xorm:"DEFAULT 0"`
	StarCount       int64   `xorm:"DEFAULT 0"`
	RecalculatedUnix int64  `xorm:"DEFAULT 0"`
	CreatedUnix     int64   `xorm:"created"`
	UpdatedUnix     int64   `xorm:"updated"`
}

func (v14cReputation) TableName() string { return "reputation" }

func addHackforgerTables(x *xorm.Engine) error {
	return x.Sync(
		new(v14cHackathon),
		new(v14cHackathonTrack),
		new(v14cHackathonRegistration),
		new(v14cHackathonSubmission),
		new(v14cHackathonJudgeScore),
		new(v14cBounty),
		new(v14cBountyReward),
		new(v14cBountyApplication),
		new(v14cBountyWinner),
		new(v14cGrantRound),
		new(v14cGrantProject),
		new(v14cCreditAccount),
		new(v14cCreditTransaction),
		new(v14cRedeemOption),
		new(v14cRedeemOrder),
		new(v14cReputation),
	)
}
