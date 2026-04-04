# P0 Infrastructure (Phase 0) Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the complete data layer (16 tables — spec says 15 but JudgeScore is the 16th), migration files, Feed event type system, HackForgerNotifier skeleton, CRUD query functions in models, router registration scaffolding, frontend directory structure, and i18n keys — everything needed before Phase 1 feature development begins.

**Architecture:** Forgejo's strict layered architecture (routers → services → models → modules). All new code lives in `*/hackforger/` directories. Models register via `init()` + `db.RegisterModel()`. CRUD data-access functions live in the `models/hackforger/` package (following Forgejo convention where `models/issues/issue.go` contains `GetIssueByID`, etc.), while business logic (state machines, transactional operations) lives in `services/hackforger/`. Migrations use Forgejo's filename-based ID system (`v14c_*` prefix). The HackForgerNotifier embeds `NullNotifier` and registers alongside existing notifiers.

**Tech Stack:** Go 1.23+, XORM ORM, go-chi router via `web.Route`, Vue 3 Options API, Tailwind CSS (`tw-` prefix), Fomantic UI.

**Spec corrections:** The spec claims "Hackathon 4 tables" but we need 5 (Hackathon + Track + Registration + Submission + JudgeScore), giving 16 total tables not 15. The spec says cron tasks go in `tasks_extended.go` (injection point ③), but we create a separate `tasks_hackforger.go` and hook into `cron.go` for cleaner separation.

---

## Chunk 1: Data Models (Tables 1-8: Hackathon + Bounty)

### Task 1: Hackathon Model — Core Structs + Registration

**Files:**
- Create: `models/hackforger/hackathon.go`
- Reference: `models/activities/action.go` (ActionType pattern), `models/issues/issue.go` (struct tags, error types)

- [ ] **Step 1: Create `models/hackforger/hackathon.go` with Hackathon struct**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

type HackathonStatus int

const (
	HackathonStatusDraft        HackathonStatus = iota // 0
	HackathonStatusRegistration                        // 1
	HackathonStatusHacking                             // 2
	HackathonStatusJudging                             // 3
	HackathonStatusFinished                            // 4
)

func (s HackathonStatus) String() string {
	switch s {
	case HackathonStatusDraft:
		return "draft"
	case HackathonStatusRegistration:
		return "registration"
	case HackathonStatusHacking:
		return "hacking"
	case HackathonStatusJudging:
		return "judging"
	case HackathonStatusFinished:
		return "finished"
	default:
		return "unknown"
	}
}

type Hackathon struct {
	ID                int64           `xorm:"pk autoincr"`
	OrgID             int64           `xorm:"INDEX NOT NULL"`
	OwnerID           int64           `xorm:"INDEX NOT NULL"`
	Name              string          `xorm:"NOT NULL"`
	Slug              string          `xorm:"UNIQUE NOT NULL"`
	Description       string          `xorm:"TEXT"`
	Status            HackathonStatus `xorm:"NOT NULL DEFAULT 0"`
	MaxTeamSize       int             `xorm:"NOT NULL DEFAULT 5"`
	RegistrationStart timeutil.TimeStamp
	RegistrationEnd   timeutil.TimeStamp
	HackingStart      timeutil.TimeStamp
	HackingEnd        timeutil.TimeStamp
	JudgingEnd        timeutil.TimeStamp
	PrizeSummary      string `xorm:"TEXT"`
	TemplateRepoID    int64  `xorm:"INDEX"`
	CreatedUnix       timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix       timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(Hackathon))
}

// Error types
type ErrHackathonNotExist struct {
	ID   int64
	Slug string
}

func IsErrHackathonNotExist(err error) bool {
	_, ok := err.(ErrHackathonNotExist)
	return ok
}

func (err ErrHackathonNotExist) Error() string {
	return fmt.Sprintf("hackathon does not exist [id: %d, slug: %s]", err.ID, err.Slug)
}

func (err ErrHackathonNotExist) Unwrap() error {
	return util.ErrNotExist
}

type ErrHackathonSlugExists struct {
	Slug string
}

func IsErrHackathonSlugExists(err error) bool {
	_, ok := err.(ErrHackathonSlugExists)
	return ok
}

func (err ErrHackathonSlugExists) Error() string {
	return fmt.Sprintf("hackathon slug already exists [slug: %s]", err.Slug)
}

func (err ErrHackathonSlugExists) Unwrap() error {
	return util.ErrAlreadyExist
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS (no errors)

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/hackathon.go
git commit -m "feat(hackforger): add Hackathon model struct and registration"
```

### Task 2: Hackathon Model — Track, Registration, Submission, JudgeScore

**Files:**
- Create: `models/hackforger/hackathon_track.go`
- Create: `models/hackforger/hackathon_registration.go`
- Create: `models/hackforger/hackathon_submission.go`
- Create: `models/hackforger/hackathon_judge_score.go`

- [ ] **Step 1: Create `models/hackforger/hackathon_track.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type HackathonTrack struct {
	ID          int64  `xorm:"pk autoincr"`
	HackathonID int64  `xorm:"INDEX NOT NULL"`
	Name        string `xorm:"NOT NULL"`
	Description string `xorm:"TEXT"`
	PrizeAmount float64
	PrizeCurrency string `xorm:"VARCHAR(16)"`
	PrizeCredits  int64
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(HackathonTrack))
}
```

- [ ] **Step 2: Create `models/hackforger/hackathon_registration.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type RegistrationStatus int

const (
	RegistrationStatusPending  RegistrationStatus = iota // 0
	RegistrationStatusApproved                           // 1
	RegistrationStatusRejected                           // 2
)

type HackathonRegistration struct {
	ID          int64              `xorm:"pk autoincr"`
	HackathonID int64              `xorm:"INDEX NOT NULL"`
	UserID      int64              `xorm:"INDEX NOT NULL"`
	TeamName    string             `xorm:"NOT NULL"`
	TrackID     int64              `xorm:"INDEX"`
	Status      RegistrationStatus `xorm:"NOT NULL DEFAULT 0"`
	TeamID      int64              `xorm:"INDEX"`
	RepoID      int64              `xorm:"INDEX"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(HackathonRegistration))
}
```

- [ ] **Step 3: Create `models/hackforger/hackathon_submission.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type SubmissionStatus int

const (
	SubmissionStatusDraft     SubmissionStatus = iota // 0
	SubmissionStatusSubmitted                         // 1
)

type HackathonSubmission struct {
	ID             int64            `xorm:"pk autoincr"`
	HackathonID    int64            `xorm:"INDEX NOT NULL"`
	RegistrationID int64            `xorm:"INDEX NOT NULL"`
	UserID         int64            `xorm:"INDEX NOT NULL"`
	TrackID        int64            `xorm:"INDEX"`
	RepoID         int64            `xorm:"INDEX"`
	Title          string           `xorm:"NOT NULL"`
	Description    string           `xorm:"TEXT"`
	DemoURL        string           `xorm:"VARCHAR(2048)"`
	Status         SubmissionStatus `xorm:"NOT NULL DEFAULT 0"`
	TotalScore     float64          `xorm:"NOT NULL DEFAULT 0"`
	Rank           int              `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix    timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix    timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(HackathonSubmission))
}
```

- [ ] **Step 4: Create `models/hackforger/hackathon_judge_score.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

type HackathonJudgeScore struct {
	ID           int64   `xorm:"pk autoincr"`
	HackathonID  int64   `xorm:"INDEX NOT NULL"`
	SubmissionID int64   `xorm:"INDEX NOT NULL"`
	JudgeID      int64   `xorm:"INDEX NOT NULL"`
	Score        float64 `xorm:"NOT NULL DEFAULT 0"`
	Comment      string  `xorm:"TEXT"`
	CreatedUnix  timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(HackathonJudgeScore))
}

type ErrDuplicateScore struct {
	SubmissionID int64
	JudgeID      int64
}

func IsErrDuplicateScore(err error) bool {
	_, ok := err.(ErrDuplicateScore)
	return ok
}

func (err ErrDuplicateScore) Error() string {
	return fmt.Sprintf("duplicate score [submission_id: %d, judge_id: %d]", err.SubmissionID, err.JudgeID)
}

func (err ErrDuplicateScore) Unwrap() error {
	return util.ErrAlreadyExist
}
```

- [ ] **Step 5: Verify all compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 6: Commit**

```bash
git add models/hackforger/hackathon_track.go models/hackforger/hackathon_registration.go models/hackforger/hackathon_submission.go models/hackforger/hackathon_judge_score.go
git commit -m "feat(hackforger): add HackathonTrack, Registration, Submission, JudgeScore models"
```

### Task 3: Bounty Model — Core Structs

**Files:**
- Create: `models/hackforger/bounty.go`
- Create: `models/hackforger/bounty_reward.go`
- Create: `models/hackforger/bounty_application.go`
- Create: `models/hackforger/bounty_winner.go`

- [ ] **Step 1: Create `models/hackforger/bounty.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

type BountyStatus int

const (
	BountyStatusOpen      BountyStatus = iota // 0
	BountyStatusClaimed                       // 1
	BountyStatusInReview                      // 2
	BountyStatusCompleted                     // 3
	BountyStatusPaid                          // 4
	BountyStatusExpired                       // 5
	BountyStatusCancelled                     // 6
)

func (s BountyStatus) String() string {
	switch s {
	case BountyStatusOpen:
		return "open"
	case BountyStatusClaimed:
		return "claimed"
	case BountyStatusInReview:
		return "in_review"
	case BountyStatusCompleted:
		return "completed"
	case BountyStatusPaid:
		return "paid"
	case BountyStatusExpired:
		return "expired"
	case BountyStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

type BountyMode int

const (
	BountyModeExclusive   BountyMode = iota // 0
	BountyModeCompetitive                   // 1
)

type Bounty struct {
	ID          int64        `xorm:"pk autoincr"`
	RepoID      int64        `xorm:"INDEX NOT NULL"`
	IssueID     int64        `xorm:"UNIQUE NOT NULL"`
	PublisherID int64        `xorm:"INDEX NOT NULL"`
	ClaimerID   int64        `xorm:"INDEX"`
	Title       string       `xorm:"NOT NULL"`
	Status      BountyStatus `xorm:"NOT NULL DEFAULT 0"`
	Mode        BountyMode   `xorm:"NOT NULL DEFAULT 0"`
	Deadline    timeutil.TimeStamp
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(Bounty))
}

// Error types
type ErrBountyNotExist struct {
	ID int64
}

func IsErrBountyNotExist(err error) bool {
	_, ok := err.(ErrBountyNotExist)
	return ok
}

func (err ErrBountyNotExist) Error() string {
	return fmt.Sprintf("bounty does not exist [id: %d]", err.ID)
}

func (err ErrBountyNotExist) Unwrap() error {
	return util.ErrNotExist
}

type ErrBountyAlreadyExists struct {
	IssueID int64
}

func IsErrBountyAlreadyExists(err error) bool {
	_, ok := err.(ErrBountyAlreadyExists)
	return ok
}

func (err ErrBountyAlreadyExists) Error() string {
	return fmt.Sprintf("bounty already exists for issue [issue_id: %d]", err.IssueID)
}

func (err ErrBountyAlreadyExists) Unwrap() error {
	return util.ErrAlreadyExist
}
```

- [ ] **Step 2: Create `models/hackforger/bounty_reward.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type RewardType string

const (
	RewardTypeMoney   RewardType = "money"
	RewardTypeCredits RewardType = "credits"
	RewardTypeOther   RewardType = "other"
)

type BountyReward struct {
	ID        int64      `xorm:"pk autoincr"`
	BountyID  int64      `xorm:"INDEX NOT NULL"`
	Type      RewardType `xorm:"VARCHAR(16) NOT NULL"`
	Amount    float64    `xorm:"NOT NULL DEFAULT 0"`
	Currency  string     `xorm:"VARCHAR(16)"`
	Credits   int64      `xorm:"NOT NULL DEFAULT 0"`
	Rank      int        `xorm:"NOT NULL DEFAULT 1"`
	Note      string     `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
}

func init() {
	db.RegisterModel(new(BountyReward))
}
```

- [ ] **Step 3: Create `models/hackforger/bounty_application.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

type ApplicationStatus int

const (
	ApplicationStatusPending  ApplicationStatus = iota // 0
	ApplicationStatusAccepted                          // 1
	ApplicationStatusRejected                          // 2
)

type BountyApplication struct {
	ID        int64             `xorm:"pk autoincr"`
	BountyID  int64             `xorm:"INDEX NOT NULL"`
	UserID    int64             `xorm:"INDEX NOT NULL"`
	Status    ApplicationStatus `xorm:"NOT NULL DEFAULT 0"`
	Message   string            `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(BountyApplication))
}

type ErrAlreadyApplied struct {
	BountyID int64
	UserID   int64
}

func IsErrAlreadyApplied(err error) bool {
	_, ok := err.(ErrAlreadyApplied)
	return ok
}

func (err ErrAlreadyApplied) Error() string {
	return fmt.Sprintf("user already applied [bounty_id: %d, user_id: %d]", err.BountyID, err.UserID)
}

func (err ErrAlreadyApplied) Unwrap() error {
	return util.ErrAlreadyExist
}
```

- [ ] **Step 4: Create `models/hackforger/bounty_winner.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type BountyWinner struct {
	ID        int64  `xorm:"pk autoincr"`
	BountyID  int64  `xorm:"INDEX NOT NULL"`
	UserID    int64  `xorm:"INDEX NOT NULL"`
	Rank      int    `xorm:"NOT NULL DEFAULT 1"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
}

func init() {
	db.RegisterModel(new(BountyWinner))
}
```

- [ ] **Step 5: Verify all compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 6: Commit**

```bash
git add models/hackforger/bounty.go models/hackforger/bounty_reward.go models/hackforger/bounty_application.go models/hackforger/bounty_winner.go
git commit -m "feat(hackforger): add Bounty, BountyReward, BountyApplication, BountyWinner models"
```

### Task 4: Grant Model

**Files:**
- Create: `models/hackforger/grant_round.go`
- Create: `models/hackforger/grant_project.go`

- [ ] **Step 1: Create `models/hackforger/grant_round.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

type GrantRoundStatus int

const (
	GrantRoundStatusSetup      GrantRoundStatus = iota // 0
	GrantRoundStatusOpen                               // 1
	GrantRoundStatusReviewing                          // 2
	GrantRoundStatusFinalized                          // 3
	GrantRoundStatusDistributed                        // 4
)

func (s GrantRoundStatus) String() string {
	switch s {
	case GrantRoundStatusSetup:
		return "setup"
	case GrantRoundStatusOpen:
		return "open"
	case GrantRoundStatusReviewing:
		return "reviewing"
	case GrantRoundStatusFinalized:
		return "finalized"
	case GrantRoundStatusDistributed:
		return "distributed"
	default:
		return "unknown"
	}
}

type GrantRound struct {
	ID            int64            `xorm:"pk autoincr"`
	OrgID         int64            `xorm:"INDEX NOT NULL"`
	OwnerID       int64            `xorm:"INDEX NOT NULL"`
	Name          string           `xorm:"NOT NULL"`
	Slug          string           `xorm:"UNIQUE NOT NULL"`
	Description   string           `xorm:"TEXT"`
	Status        GrantRoundStatus `xorm:"NOT NULL DEFAULT 0"`
	Budget        float64          `xorm:"NOT NULL DEFAULT 0"`
	Currency      string           `xorm:"VARCHAR(16) NOT NULL DEFAULT 'USD'"`
	BudgetCredits int64            `xorm:"NOT NULL DEFAULT 0"`
	Deadline      timeutil.TimeStamp
	CreatedUnix   timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix   timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(GrantRound))
}

type ErrGrantRoundNotExist struct {
	ID int64
}

func IsErrGrantRoundNotExist(err error) bool {
	_, ok := err.(ErrGrantRoundNotExist)
	return ok
}

func (err ErrGrantRoundNotExist) Error() string {
	return fmt.Sprintf("grant round does not exist [id: %d]", err.ID)
}

func (err ErrGrantRoundNotExist) Unwrap() error {
	return util.ErrNotExist
}
```

- [ ] **Step 2: Create `models/hackforger/grant_project.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type GrantProjectStatus int

const (
	GrantProjectStatusPending  GrantProjectStatus = iota // 0
	GrantProjectStatusApproved                           // 1
	GrantProjectStatusRejected                           // 2
)

type GrantProject struct {
	ID           int64              `xorm:"pk autoincr"`
	RoundID      int64              `xorm:"INDEX NOT NULL"`
	UserID       int64              `xorm:"INDEX NOT NULL"`
	RepoID       int64              `xorm:"INDEX"`
	Title        string             `xorm:"NOT NULL"`
	Description  string             `xorm:"TEXT"`
	Status       GrantProjectStatus `xorm:"NOT NULL DEFAULT 0"`
	AwardAmount  float64            `xorm:"NOT NULL DEFAULT 0"`
	AwardCredits int64              `xorm:"NOT NULL DEFAULT 0"`
	CreatedUnix  timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(GrantProject))
}
```

- [ ] **Step 3: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/grant_round.go models/hackforger/grant_project.go
git commit -m "feat(hackforger): add GrantRound, GrantProject models"
```

### Task 5: Credits Model (4 tables)

**Files:**
- Create: `models/hackforger/credit_account.go`
- Create: `models/hackforger/credit_transaction.go`
- Create: `models/hackforger/redeem_option.go`
- Create: `models/hackforger/redeem_order.go`

- [ ] **Step 1: Create `models/hackforger/credit_account.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

type CreditAccount struct {
	ID        int64 `xorm:"pk autoincr"`
	UserID    int64 `xorm:"UNIQUE NOT NULL"`
	Balance   int64 `xorm:"NOT NULL DEFAULT 0"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(CreditAccount))
}

type ErrInsufficientCredits struct {
	UserID  int64
	Balance int64
	Amount  int64
}

func IsErrInsufficientCredits(err error) bool {
	_, ok := err.(ErrInsufficientCredits)
	return ok
}

func (err ErrInsufficientCredits) Error() string {
	return fmt.Sprintf("insufficient credits [user_id: %d, balance: %d, amount: %d]", err.UserID, err.Balance, err.Amount)
}

func (err ErrInsufficientCredits) Unwrap() error {
	return util.ErrInvalidArgument
}
```

- [ ] **Step 2: Create `models/hackforger/credit_transaction.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type TransactionType string

const (
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypeRedeem   TransactionType = "redeem"
	TransactionTypeRefund   TransactionType = "refund"
	TransactionTypeDeduct   TransactionType = "deduct"
)

type CreditTransaction struct {
	ID          int64           `xorm:"pk autoincr"`
	UserID      int64           `xorm:"INDEX NOT NULL"`
	Type        TransactionType `xorm:"VARCHAR(16) NOT NULL"`
	Amount      int64           `xorm:"NOT NULL"`
	Balance     int64           `xorm:"NOT NULL"`
	Reference   string          `xorm:"VARCHAR(255)"`
	Note        string          `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(CreditTransaction))
}
```

- [ ] **Step 3: Create `models/hackforger/redeem_option.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

type RedeemOption struct {
	ID          int64  `xorm:"pk autoincr"`
	Name        string `xorm:"NOT NULL"`
	Description string `xorm:"TEXT"`
	Cost        int64  `xorm:"NOT NULL"`
	Stock       int    `xorm:"NOT NULL DEFAULT -1"`
	IsActive    bool   `xorm:"NOT NULL DEFAULT true"`
	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(RedeemOption))
}

type ErrOutOfStock struct {
	OptionID int64
}

func IsErrOutOfStock(err error) bool {
	_, ok := err.(ErrOutOfStock)
	return ok
}

func (err ErrOutOfStock) Error() string {
	return fmt.Sprintf("redeem option out of stock [option_id: %d]", err.OptionID)
}

func (err ErrOutOfStock) Unwrap() error {
	return util.ErrInvalidArgument
}
```

- [ ] **Step 4: Create `models/hackforger/redeem_order.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusFulfilled OrderStatus = "fulfilled"
	OrderStatusRefunded  OrderStatus = "refunded"
)

type RedeemOrder struct {
	ID          int64       `xorm:"pk autoincr"`
	UserID      int64       `xorm:"INDEX NOT NULL"`
	OptionID    int64       `xorm:"INDEX NOT NULL"`
	Cost        int64       `xorm:"NOT NULL"`
	Status      OrderStatus `xorm:"VARCHAR(16) NOT NULL DEFAULT 'pending'"`
	FulfillNote string      `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(RedeemOrder))
}
```

- [ ] **Step 5: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 6: Commit**

```bash
git add models/hackforger/credit_account.go models/hackforger/credit_transaction.go models/hackforger/redeem_option.go models/hackforger/redeem_order.go
git commit -m "feat(hackforger): add CreditAccount, CreditTransaction, RedeemOption, RedeemOrder models"
```

### Task 6: Reputation Model

**Files:**
- Create: `models/hackforger/reputation.go`

- [ ] **Step 1: Create `models/hackforger/reputation.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

type Reputation struct {
	ID               int64 `xorm:"pk autoincr"`
	UserID           int64 `xorm:"UNIQUE NOT NULL"`
	Score            int64 `xorm:"NOT NULL DEFAULT 0"`
	BountiesCompleted int  `xorm:"NOT NULL DEFAULT 0"`
	HackathonWins    int   `xorm:"NOT NULL DEFAULT 0"`
	GrantsReceived   int   `xorm:"NOT NULL DEFAULT 0"`
	TotalStars       int64 `xorm:"NOT NULL DEFAULT 0"`
	TotalCreditsEarned int64 `xorm:"NOT NULL DEFAULT 0"`
	UpdatedUnix      timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(Reputation))
}
```

- [ ] **Step 2: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/reputation.go
git commit -m "feat(hackforger): add Reputation model"
```

---

## Chunk 2: Feed Event Types + Migration + Notifier Skeleton

### Task 7: Feed Action Types

**Files:**
- Create: `models/hackforger/action_types.go`
- Reference: `models/activities/action.go` (existing ActionType enum, ends at 27)

- [ ] **Step 1: Create `models/hackforger/action_types.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	activities_model "forgejo.org/models/activities"
)

// HackForger action types, starting from 30 to avoid collision with
// Forgejo's built-in ActionType (1-27).
const (
	// User behavior events (30-42)
	ActionHackathonCreated    activities_model.ActionType = 30
	ActionHackathonRegistered activities_model.ActionType = 31
	ActionHackathonSubmitted  activities_model.ActionType = 32
	ActionHackathonScored     activities_model.ActionType = 33
	ActionBountyCreated       activities_model.ActionType = 34
	ActionBountyClaimed       activities_model.ActionType = 35
	ActionBountyDelivered     activities_model.ActionType = 36
	ActionBountyCompleted     activities_model.ActionType = 37
	ActionBountyWinnersSelected activities_model.ActionType = 38
	ActionGrantRoundCreated   activities_model.ActionType = 39
	ActionGrantProjectSubmitted activities_model.ActionType = 40
	ActionGrantAwarded        activities_model.ActionType = 41
	ActionCreditsRedeemed     activities_model.ActionType = 42

	// Entity lifecycle events (50-56)
	ActionHackathonPhaseChanged activities_model.ActionType = 50
	ActionHackathonFinalized    activities_model.ActionType = 51
	ActionBountyExpired         activities_model.ActionType = 52
	ActionBountyCancelled       activities_model.ActionType = 53
	ActionGrantRoundOpened      activities_model.ActionType = 54
	ActionGrantRoundClosed      activities_model.ActionType = 55
	ActionGrantRoundFinalized   activities_model.ActionType = 56

	// Milestone events (60, future)
	ActionMilestone activities_model.ActionType = 60
)

// HackforgerActionTypeName maps HackForger action types to string names.
var HackforgerActionTypeName = map[activities_model.ActionType]string{
	ActionHackathonCreated:      "hackathon_created",
	ActionHackathonRegistered:   "hackathon_registered",
	ActionHackathonSubmitted:    "hackathon_submitted",
	ActionHackathonScored:       "hackathon_scored",
	ActionBountyCreated:         "bounty_created",
	ActionBountyClaimed:         "bounty_claimed",
	ActionBountyDelivered:       "bounty_delivered",
	ActionBountyCompleted:       "bounty_completed",
	ActionBountyWinnersSelected: "bounty_winners_selected",
	ActionGrantRoundCreated:     "grant_round_created",
	ActionGrantProjectSubmitted: "grant_project_submitted",
	ActionGrantAwarded:          "grant_awarded",
	ActionCreditsRedeemed:       "credits_redeemed",
	ActionHackathonPhaseChanged: "hackathon_phase_changed",
	ActionHackathonFinalized:    "hackathon_finalized",
	ActionBountyExpired:         "bounty_expired",
	ActionBountyCancelled:       "bounty_cancelled",
	ActionGrantRoundOpened:      "grant_round_opened",
	ActionGrantRoundClosed:      "grant_round_closed",
	ActionGrantRoundFinalized:   "grant_round_finalized",
	ActionMilestone:             "milestone",
}

// IsHackforgerAction returns true if the ActionType is a HackForger event.
func IsHackforgerAction(at activities_model.ActionType) bool {
	_, ok := HackforgerActionTypeName[at]
	return ok
}

// HackforgerActionContent is the JSON content stored in action.content
// for most HackForger events.
type HackforgerActionContent struct {
	EntityType string `json:"entity_type"`           // "hackathon", "bounty", "grant"
	EntityID   int64  `json:"entity_id"`
	EntityName string `json:"entity_name"`
	EntitySlug string `json:"entity_slug,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// HackforgerPhaseContent extends HackforgerActionContent with status change info.
type HackforgerPhaseContent struct {
	HackforgerActionContent
	OldStatus   string `json:"old_status"`
	NewStatus   string `json:"new_status"`
	StatusLabel string `json:"status_label,omitempty"`
	WinnerName  string `json:"winner_name,omitempty"`
}
```

- [ ] **Step 2: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/action_types.go
git commit -m "feat(hackforger): add Feed action type constants and content structs"
```

### Task 8: Forgejo Migration — Create All 15 HackForger Tables

**Files:**
- Create: `models/forgejo_migrations/v14c_add-hackforger-tables.go`
- Reference: `models/forgejo_migrations/v14b_action-reindexing.go` (migration pattern)

Note: We use `v14c_` prefix so it sorts after the existing `v14b_` migrations. A single migration file creates all 15 tables since they are introduced together.

- [ ] **Step 1: Create `models/forgejo_migrations/v14c_add-hackforger-tables.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add hackforger tables",
		Upgrade:     addHackforgerTables,
	})
}

func addHackforgerTables(x *xorm.Engine) error {
	// Define all table structs locally (migration-local, not importing models)
	type Hackathon struct {
		ID                int64  `xorm:"pk autoincr"`
		OrgID             int64  `xorm:"INDEX NOT NULL"`
		OwnerID           int64  `xorm:"INDEX NOT NULL"`
		Name              string `xorm:"NOT NULL"`
		Slug              string `xorm:"UNIQUE NOT NULL"`
		Description       string `xorm:"TEXT"`
		Status            int    `xorm:"NOT NULL DEFAULT 0"`
		MaxTeamSize       int    `xorm:"NOT NULL DEFAULT 5"`
		RegistrationStart int64
		RegistrationEnd   int64
		HackingStart      int64
		HackingEnd        int64
		JudgingEnd        int64
		PrizeSummary      string `xorm:"TEXT"`
		TemplateRepoID    int64  `xorm:"INDEX"`
		CreatedUnix       int64  `xorm:"INDEX created"`
		UpdatedUnix       int64  `xorm:"INDEX updated"`
	}

	type HackathonTrack struct {
		ID            int64   `xorm:"pk autoincr"`
		HackathonID   int64   `xorm:"INDEX NOT NULL"`
		Name          string  `xorm:"NOT NULL"`
		Description   string  `xorm:"TEXT"`
		PrizeAmount   float64
		PrizeCurrency string `xorm:"VARCHAR(16)"`
		PrizeCredits  int64
		CreatedUnix   int64 `xorm:"INDEX created"`
	}

	type HackathonRegistration struct {
		ID          int64 `xorm:"pk autoincr"`
		HackathonID int64 `xorm:"INDEX NOT NULL"`
		UserID      int64 `xorm:"INDEX NOT NULL"`
		TeamName    string `xorm:"NOT NULL"`
		TrackID     int64  `xorm:"INDEX"`
		Status      int    `xorm:"NOT NULL DEFAULT 0"`
		TeamID      int64  `xorm:"INDEX"`
		RepoID      int64  `xorm:"INDEX"`
		CreatedUnix int64  `xorm:"INDEX created"`
		UpdatedUnix int64  `xorm:"updated"`
	}

	type HackathonSubmission struct {
		ID             int64   `xorm:"pk autoincr"`
		HackathonID    int64   `xorm:"INDEX NOT NULL"`
		RegistrationID int64   `xorm:"INDEX NOT NULL"`
		UserID         int64   `xorm:"INDEX NOT NULL"`
		TrackID        int64   `xorm:"INDEX"`
		RepoID         int64   `xorm:"INDEX"`
		Title          string  `xorm:"NOT NULL"`
		Description    string  `xorm:"TEXT"`
		DemoURL        string  `xorm:"VARCHAR(2048)"`
		Status         int     `xorm:"NOT NULL DEFAULT 0"`
		TotalScore     float64 `xorm:"NOT NULL DEFAULT 0"`
		Rank           int     `xorm:"NOT NULL DEFAULT 0"`
		CreatedUnix    int64   `xorm:"INDEX created"`
		UpdatedUnix    int64   `xorm:"updated"`
	}

	type HackathonJudgeScore struct {
		ID           int64   `xorm:"pk autoincr"`
		HackathonID  int64   `xorm:"INDEX NOT NULL"`
		SubmissionID int64   `xorm:"INDEX NOT NULL"`
		JudgeID      int64   `xorm:"INDEX NOT NULL"`
		Score        float64 `xorm:"NOT NULL DEFAULT 0"`
		Comment      string  `xorm:"TEXT"`
		CreatedUnix  int64   `xorm:"INDEX created"`
		UpdatedUnix  int64   `xorm:"updated"`
	}

	type Bounty struct {
		ID          int64 `xorm:"pk autoincr"`
		RepoID      int64 `xorm:"INDEX NOT NULL"`
		IssueID     int64 `xorm:"UNIQUE NOT NULL"`
		PublisherID int64 `xorm:"INDEX NOT NULL"`
		ClaimerID   int64 `xorm:"INDEX"`
		Title       string `xorm:"NOT NULL"`
		Status      int    `xorm:"NOT NULL DEFAULT 0"`
		Mode        int    `xorm:"NOT NULL DEFAULT 0"`
		Deadline    int64
		CreatedUnix int64 `xorm:"INDEX created"`
		UpdatedUnix int64 `xorm:"INDEX updated"`
	}

	type BountyReward struct {
		ID        int64   `xorm:"pk autoincr"`
		BountyID  int64   `xorm:"INDEX NOT NULL"`
		Type      string  `xorm:"VARCHAR(16) NOT NULL"`
		Amount    float64 `xorm:"NOT NULL DEFAULT 0"`
		Currency  string  `xorm:"VARCHAR(16)"`
		Credits   int64   `xorm:"NOT NULL DEFAULT 0"`
		Rank      int     `xorm:"NOT NULL DEFAULT 1"`
		Note      string  `xorm:"TEXT"`
		CreatedUnix int64 `xorm:"created"`
	}

	type BountyApplication struct {
		ID        int64  `xorm:"pk autoincr"`
		BountyID  int64  `xorm:"INDEX NOT NULL"`
		UserID    int64  `xorm:"INDEX NOT NULL"`
		Status    int    `xorm:"NOT NULL DEFAULT 0"`
		Message   string `xorm:"TEXT"`
		CreatedUnix int64 `xorm:"INDEX created"`
		UpdatedUnix int64 `xorm:"updated"`
	}

	type BountyWinner struct {
		ID        int64 `xorm:"pk autoincr"`
		BountyID  int64 `xorm:"INDEX NOT NULL"`
		UserID    int64 `xorm:"INDEX NOT NULL"`
		Rank      int   `xorm:"NOT NULL DEFAULT 1"`
		CreatedUnix int64 `xorm:"created"`
	}

	type GrantRound struct {
		ID            int64   `xorm:"pk autoincr"`
		OrgID         int64   `xorm:"INDEX NOT NULL"`
		OwnerID       int64   `xorm:"INDEX NOT NULL"`
		Name          string  `xorm:"NOT NULL"`
		Slug          string  `xorm:"UNIQUE NOT NULL"`
		Description   string  `xorm:"TEXT"`
		Status        int     `xorm:"NOT NULL DEFAULT 0"`
		Budget        float64 `xorm:"NOT NULL DEFAULT 0"`
		Currency      string  `xorm:"VARCHAR(16) NOT NULL DEFAULT 'USD'"`
		BudgetCredits int64   `xorm:"NOT NULL DEFAULT 0"`
		Deadline      int64
		CreatedUnix   int64 `xorm:"INDEX created"`
		UpdatedUnix   int64 `xorm:"INDEX updated"`
	}

	type GrantProject struct {
		ID           int64   `xorm:"pk autoincr"`
		RoundID      int64   `xorm:"INDEX NOT NULL"`
		UserID       int64   `xorm:"INDEX NOT NULL"`
		RepoID       int64   `xorm:"INDEX"`
		Title        string  `xorm:"NOT NULL"`
		Description  string  `xorm:"TEXT"`
		Status       int     `xorm:"NOT NULL DEFAULT 0"`
		AwardAmount  float64 `xorm:"NOT NULL DEFAULT 0"`
		AwardCredits int64   `xorm:"NOT NULL DEFAULT 0"`
		CreatedUnix  int64   `xorm:"INDEX created"`
		UpdatedUnix  int64   `xorm:"updated"`
	}

	type CreditAccount struct {
		ID        int64 `xorm:"pk autoincr"`
		UserID    int64 `xorm:"UNIQUE NOT NULL"`
		Balance   int64 `xorm:"NOT NULL DEFAULT 0"`
		UpdatedUnix int64 `xorm:"updated"`
	}

	type CreditTransaction struct {
		ID          int64  `xorm:"pk autoincr"`
		UserID      int64  `xorm:"INDEX NOT NULL"`
		Type        string `xorm:"VARCHAR(16) NOT NULL"`
		Amount      int64  `xorm:"NOT NULL"`
		Balance     int64  `xorm:"NOT NULL"`
		Reference   string `xorm:"VARCHAR(255)"`
		Note        string `xorm:"TEXT"`
		CreatedUnix int64  `xorm:"INDEX created"`
	}

	type RedeemOption struct {
		ID          int64  `xorm:"pk autoincr"`
		Name        string `xorm:"NOT NULL"`
		Description string `xorm:"TEXT"`
		Cost        int64  `xorm:"NOT NULL"`
		Stock       int    `xorm:"NOT NULL DEFAULT -1"`
		IsActive    bool   `xorm:"NOT NULL DEFAULT true"`
		CreatedUnix int64  `xorm:"created"`
		UpdatedUnix int64  `xorm:"updated"`
	}

	type RedeemOrder struct {
		ID          int64  `xorm:"pk autoincr"`
		UserID      int64  `xorm:"INDEX NOT NULL"`
		OptionID    int64  `xorm:"INDEX NOT NULL"`
		Cost        int64  `xorm:"NOT NULL"`
		Status      string `xorm:"VARCHAR(16) NOT NULL DEFAULT 'pending'"`
		FulfillNote string `xorm:"TEXT"`
		CreatedUnix int64  `xorm:"INDEX created"`
		UpdatedUnix int64  `xorm:"updated"`
	}

	type Reputation struct {
		ID                 int64 `xorm:"pk autoincr"`
		UserID             int64 `xorm:"UNIQUE NOT NULL"`
		Score              int64 `xorm:"NOT NULL DEFAULT 0"`
		BountiesCompleted  int   `xorm:"NOT NULL DEFAULT 0"`
		HackathonWins      int   `xorm:"NOT NULL DEFAULT 0"`
		GrantsReceived     int   `xorm:"NOT NULL DEFAULT 0"`
		TotalStars         int64 `xorm:"NOT NULL DEFAULT 0"`
		TotalCreditsEarned int64 `xorm:"NOT NULL DEFAULT 0"`
		UpdatedUnix        int64 `xorm:"updated"`
	}

	return x.Sync(
		new(Hackathon),
		new(HackathonTrack),
		new(HackathonRegistration),
		new(HackathonSubmission),
		new(HackathonJudgeScore),
		new(Bounty),
		new(BountyReward),
		new(BountyApplication),
		new(BountyWinner),
		new(GrantRound),
		new(GrantProject),
		new(CreditAccount),
		new(CreditTransaction),
		new(RedeemOption),
		new(RedeemOrder),
		new(Reputation),
	)
}
```

- [ ] **Step 2: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/forgejo_migrations/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add models/forgejo_migrations/v14c_add-hackforger-tables.go
git commit -m "feat(hackforger): add migration to create all 15 hackforger tables"
```

### Task 9: HackForgerNotifier Skeleton

**Files:**
- Create: `services/hackforger/notifier.go`
- Modify: `routers/init.go` (injection point ④)
- Reference: `services/feed/action.go` (notifier registration pattern)

- [ ] **Step 1: Create `services/hackforger/notifier.go`**

Note: Forgejo's `activities_model.NotifyWatchers` only supports repo-watcher distribution. HackForger needs 4 audience strategies (Global, OrgMembers, Followers, RepoWatchers) per the spec section 2.7. We implement `PublishHackforgerAction` with direct action insertion to support all strategies. Full audience resolution is a Phase 1 deliverable; the P0 skeleton inserts actions for the actor and optionally as a global event (UserID=0).

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	notify_service "forgejo.org/services/notify"
)

type hackforgerNotifier struct {
	notify_service.NullNotifier
}

var _ notify_service.Notifier = &hackforgerNotifier{}

// Init registers the HackForger notifier.
func Init() error {
	notify_service.RegisterNotifier(&hackforgerNotifier{})
	return nil
}

// AudienceType determines how feed events are distributed.
type AudienceType int

const (
	AudienceGlobal      AudienceType = iota // UserID=0, visible to everyone
	AudienceFollowers                       // Visible to ActUser's followers
	AudienceOrgMembers                      // Visible to org members
	AudienceRepoWatchers                    // Visible to repo watchers
)

// HackforgerActionOpts holds the parameters for publishing a HackForger feed event.
type HackforgerActionOpts struct {
	ActUserID    int64
	OpType       activities_model.ActionType
	RepoID       int64
	Content      any
	AudienceType AudienceType
	OrgID        int64 // used when AudienceType == AudienceOrgMembers
}

// PublishHackforgerAction writes HackForger events to the action table.
// Unlike Forgejo's NotifyWatchers (which only handles repo watchers),
// this supports 4 audience strategies per the spec.
//
// P0 skeleton: inserts for actor + global (UserID=0) only.
// Phase 1: adds full audience resolution (followers, org members, repo watchers).
func PublishHackforgerAction(ctx context.Context, opts *HackforgerActionOpts) error {
	contentBytes, err := json.Marshal(opts.Content)
	if err != nil {
		return err
	}
	contentStr := string(contentBytes)
	now := timeutil.TimeStampNow()

	// Always insert the actor's own action record
	actorAction := &activities_model.Action{
		ActUserID:   opts.ActUserID,
		UserID:      opts.ActUserID,
		OpType:      opts.OpType,
		RepoID:      opts.RepoID,
		Content:     contentStr,
		CreatedUnix: now,
	}
	if _, err := db.GetEngine(ctx).Insert(actorAction); err != nil {
		log.Error("PublishHackforgerAction (actor): %v", err)
		return err
	}

	// For global events, also insert a UserID=0 record
	if opts.AudienceType == AudienceGlobal {
		globalAction := &activities_model.Action{
			ActUserID:   opts.ActUserID,
			UserID:      0,
			OpType:      opts.OpType,
			RepoID:      opts.RepoID,
			Content:     contentStr,
			CreatedUnix: now,
		}
		if _, err := db.GetEngine(ctx).Insert(globalAction); err != nil {
			log.Error("PublishHackforgerAction (global): %v", err)
			return err
		}
	}

	// Phase 1 TODO: resolve audience based on opts.AudienceType
	// - AudienceFollowers: query user_follow table for ActUserID's followers
	// - AudienceOrgMembers: query org_user table for opts.OrgID members
	// - AudienceRepoWatchers: query watch table for opts.RepoID watchers

	return nil
}

// MergePullRequest is called when a PR is merged. We check if the PR's
// issue has a linked Bounty and trigger status transitions if so.
// This is a skeleton — full logic added in Phase 1.
func (n *hackforgerNotifier) MergePullRequest(ctx context.Context, doer *user_model.User, pr *issues_model.PullRequest) {
	// Phase 1: Check if PR's Issue has a Bounty, and if doer is the Claimer,
	// transition to InReview status.
	_ = doer
	_ = pr
	_ = ctx
	_ = hackforger_model.ActionBountyDelivered // ensure import is used
}
```

- [ ] **Step 2: Register in `routers/init.go`**

Add import and `mustInit` call alongside existing notifier inits.

In `routers/init.go`, add import:
```go
hackforger_service "forgejo.org/services/hackforger"
```

Add after `mustInit(feed_service.Init)` (around line 121):
```go
mustInit(hackforger_service.Init)
```

- [ ] **Step 3: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./...`
Expected: SUCCESS (full project build to catch import cycles)

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/notifier.go routers/init.go
git commit -m "feat(hackforger): add HackForgerNotifier skeleton and register in init"
```

---

## Chunk 3: CRUD Data Access (in models) + Business Logic Services

Note: Following Forgejo's layered architecture, CRUD data-access functions (Get, List, Create, Update, Delete) live in `models/hackforger/`. The `services/hackforger/` package is reserved for business logic that orchestrates across models (state machines, transactional operations like Deposit/Redeem). This matches how `models/issues/issue.go` contains `GetIssueByID` while `services/issue/` handles higher-level workflows.

### Task 10: Hackathon CRUD (models layer)

**Files:**
- Modify: `models/hackforger/hackathon.go` (append CRUD functions)

- [ ] **Step 1: Add CRUD functions to `models/hackforger/hackathon.go`**

Append after the error types:

```go
// CreateHackathon inserts a new Hackathon.
func CreateHackathon(ctx context.Context, h *Hackathon) error {
	_, err := db.GetEngine(ctx).Insert(h)
	return err
}

// GetHackathonByID returns a Hackathon by its ID.
func GetHackathonByID(ctx context.Context, id int64) (*Hackathon, error) {
	h := &Hackathon{ID: id}
	has, err := db.GetEngine(ctx).Get(h)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrHackathonNotExist{ID: id}
	}
	return h, nil
}

// GetHackathonBySlug returns a Hackathon by its slug.
func GetHackathonBySlug(ctx context.Context, slug string) (*Hackathon, error) {
	h := &Hackathon{Slug: slug}
	has, err := db.GetEngine(ctx).Get(h)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrHackathonNotExist{Slug: slug}
	}
	return h, nil
}

// ListHackathonsOptions holds options for listing hackathons.
type ListHackathonsOptions struct {
	db.ListOptions
	Status *HackathonStatus // pointer: nil means no filter
	OrgID  int64
	Query  string
}

// ListHackathons returns hackathons matching the given options.
func ListHackathons(ctx context.Context, opts *ListHackathonsOptions) ([]*Hackathon, int64, error) {
	sess := db.GetEngine(ctx).NewSession()
	defer sess.Close()

	if opts.Status != nil {
		sess.Where("status = ?", *opts.Status)
	}
	if opts.OrgID > 0 {
		sess.And("org_id = ?", opts.OrgID)
	}
	if opts.Query != "" {
		sess.And("name LIKE ?", "%"+opts.Query+"%")
	}

	var hackathons []*Hackathon
	count, err := sess.OrderBy("created_unix DESC").
		Limit(opts.PageSize, (opts.Page-1)*opts.PageSize).
		FindAndCount(&hackathons)
	return hackathons, count, err
}

// UpdateHackathon updates a Hackathon record.
func UpdateHackathon(ctx context.Context, h *Hackathon) error {
	_, err := db.GetEngine(ctx).ID(h.ID).AllCols().Update(h)
	return err
}

// DeleteHackathon deletes a Hackathon (only Draft status).
func DeleteHackathon(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).
		Where("status = ?", HackathonStatusDraft).
		Delete(new(Hackathon))
	return err
}
```

Also add `"context"` to the import block.

- [ ] **Step 2: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/hackathon.go
git commit -m "feat(hackforger): add Hackathon CRUD functions in models layer"
```

### Task 11: Bounty CRUD (models layer)

**Files:**
- Modify: `models/hackforger/bounty.go` (append CRUD functions)

- [ ] **Step 1: Add CRUD functions to `models/hackforger/bounty.go`**

Append after the error types:

```go
// CreateBounty inserts a new Bounty. Returns ErrBountyAlreadyExists if
// the Issue already has a Bounty.
func CreateBounty(ctx context.Context, b *Bounty) error {
	existing := &Bounty{IssueID: b.IssueID}
	has, err := db.GetEngine(ctx).Get(existing)
	if err != nil {
		return err
	}
	if has {
		return ErrBountyAlreadyExists{IssueID: b.IssueID}
	}
	_, err = db.GetEngine(ctx).Insert(b)
	return err
}

// GetBountyByID returns a Bounty by ID.
func GetBountyByID(ctx context.Context, id int64) (*Bounty, error) {
	b := &Bounty{ID: id}
	has, err := db.GetEngine(ctx).Get(b)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrBountyNotExist{ID: id}
	}
	return b, nil
}

// GetBountyByIssueID returns a Bounty linked to the given Issue.
func GetBountyByIssueID(ctx context.Context, issueID int64) (*Bounty, error) {
	b := &Bounty{IssueID: issueID}
	has, err := db.GetEngine(ctx).Get(b)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrBountyNotExist{}
	}
	return b, nil
}

// ListBountiesOptions holds options for listing bounties.
type ListBountiesOptions struct {
	db.ListOptions
	RepoID int64
	Status *BountyStatus // pointer: nil means no filter
}

// ListBounties returns bounties matching the given options.
func ListBounties(ctx context.Context, opts *ListBountiesOptions) ([]*Bounty, int64, error) {
	sess := db.GetEngine(ctx).NewSession()
	defer sess.Close()

	if opts.RepoID > 0 {
		sess.Where("repo_id = ?", opts.RepoID)
	}
	if opts.Status != nil {
		sess.And("status = ?", *opts.Status)
	}

	var bounties []*Bounty
	count, err := sess.OrderBy("created_unix DESC").
		Limit(opts.PageSize, (opts.Page-1)*opts.PageSize).
		FindAndCount(&bounties)
	return bounties, count, err
}

// UpdateBounty updates a Bounty record.
func UpdateBounty(ctx context.Context, b *Bounty) error {
	_, err := db.GetEngine(ctx).ID(b.ID).AllCols().Update(b)
	return err
}
```

Also add `"context"` to the import block.

- [ ] **Step 2: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/bounty.go
git commit -m "feat(hackforger): add Bounty CRUD functions in models layer"
```

### Task 12: Grants CRUD (models layer)

**Files:**
- Modify: `models/hackforger/grant_round.go` (append CRUD)
- Modify: `models/hackforger/grant_project.go` (append CRUD)

- [ ] **Step 1: Add CRUD to `models/hackforger/grant_round.go`**

```go
// CreateGrantRound inserts a new Grant Round.
func CreateGrantRound(ctx context.Context, r *GrantRound) error {
	_, err := db.GetEngine(ctx).Insert(r)
	return err
}

// GetGrantRoundByID returns a Grant Round by ID.
func GetGrantRoundByID(ctx context.Context, id int64) (*GrantRound, error) {
	r := &GrantRound{ID: id}
	has, err := db.GetEngine(ctx).Get(r)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrGrantRoundNotExist{ID: id}
	}
	return r, nil
}

// ListGrantRoundsOptions holds options for listing grant rounds.
type ListGrantRoundsOptions struct {
	db.ListOptions
	Status *GrantRoundStatus // pointer: nil means no filter
	OrgID  int64
}

// ListGrantRounds returns grant rounds matching the given options.
func ListGrantRounds(ctx context.Context, opts *ListGrantRoundsOptions) ([]*GrantRound, int64, error) {
	sess := db.GetEngine(ctx).NewSession()
	defer sess.Close()

	if opts.Status != nil {
		sess.Where("status = ?", *opts.Status)
	}
	if opts.OrgID > 0 {
		sess.And("org_id = ?", opts.OrgID)
	}

	var rounds []*GrantRound
	count, err := sess.OrderBy("created_unix DESC").
		Limit(opts.PageSize, (opts.Page-1)*opts.PageSize).
		FindAndCount(&rounds)
	return rounds, count, err
}

// UpdateGrantRound updates a Grant Round.
func UpdateGrantRound(ctx context.Context, r *GrantRound) error {
	_, err := db.GetEngine(ctx).ID(r.ID).AllCols().Update(r)
	return err
}
```

- [ ] **Step 2: Add CRUD to `models/hackforger/grant_project.go`**

```go
// CreateGrantProject inserts a new Grant Project application.
func CreateGrantProject(ctx context.Context, p *GrantProject) error {
	_, err := db.GetEngine(ctx).Insert(p)
	return err
}

// GetGrantProjectByID returns a Grant Project by ID.
func GetGrantProjectByID(ctx context.Context, id int64) (*GrantProject, error) {
	p := &GrantProject{ID: id}
	has, err := db.GetEngine(ctx).Get(p)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return p, nil
}

// ListGrantProjectsByRound returns all projects for a given round.
func ListGrantProjectsByRound(ctx context.Context, roundID int64) ([]*GrantProject, error) {
	var projects []*GrantProject
	err := db.GetEngine(ctx).Where("round_id = ?", roundID).
		OrderBy("created_unix DESC").
		Find(&projects)
	return projects, err
}

// UpdateGrantProject updates a Grant Project.
func UpdateGrantProject(ctx context.Context, p *GrantProject) error {
	_, err := db.GetEngine(ctx).ID(p.ID).AllCols().Update(p)
	return err
}
```

- [ ] **Step 3: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/grant_round.go models/hackforger/grant_project.go
git commit -m "feat(hackforger): add Grants CRUD functions in models layer"
```

### Task 13: Credits Business Logic Service

**Files:**
- Create: `services/hackforger/credits.go`

Note: Credits Deposit/Redeem require `db.WithTx` transactions spanning multiple tables — this is business logic, correctly placed in the services layer. Simple reads (GetOrCreateCreditAccount, ListTransactions, etc.) also live here since they're tightly coupled to the transactional operations.

- [ ] **Step 1: Create `services/hackforger/credits.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
)

// GetOrCreateCreditAccount returns the credit account for a user,
// creating one with zero balance if it doesn't exist.
func GetOrCreateCreditAccount(ctx context.Context, userID int64) (*hackforger_model.CreditAccount, error) {
	acct := &hackforger_model.CreditAccount{UserID: userID}
	has, err := db.GetEngine(ctx).Where("user_id = ?", userID).Get(acct)
	if err != nil {
		return nil, err
	}
	if !has {
		acct = &hackforger_model.CreditAccount{UserID: userID, Balance: 0}
		if _, err := db.GetEngine(ctx).Insert(acct); err != nil {
			return nil, err
		}
	}
	return acct, nil
}

// Deposit adds credits to a user's account within a transaction.
func Deposit(ctx context.Context, userID int64, amount int64, reference, note string) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		acct, err := GetOrCreateCreditAccount(ctx, userID)
		if err != nil {
			return err
		}

		acct.Balance += amount
		if _, err := db.GetEngine(ctx).ID(acct.ID).Cols("balance").Update(acct); err != nil {
			return err
		}

		tx := &hackforger_model.CreditTransaction{
			UserID:    userID,
			Type:      hackforger_model.TransactionTypeDeposit,
			Amount:    amount,
			Balance:   acct.Balance,
			Reference: reference,
			Note:      note,
		}
		_, err = db.GetEngine(ctx).Insert(tx)
		return err
	})
}

// Redeem deducts credits and creates an order within a transaction.
func Redeem(ctx context.Context, userID int64, optionID int64) (*hackforger_model.RedeemOrder, error) {
	var order *hackforger_model.RedeemOrder

	err := db.WithTx(ctx, func(ctx context.Context) error {
		// Get option
		option := &hackforger_model.RedeemOption{ID: optionID}
		has, err := db.GetEngine(ctx).Get(option)
		if err != nil {
			return err
		}
		if !has {
			return hackforger_model.ErrOutOfStock{OptionID: optionID}
		}

		// Check stock
		if option.Stock == 0 {
			return hackforger_model.ErrOutOfStock{OptionID: optionID}
		}

		// Get account
		acct, err := GetOrCreateCreditAccount(ctx, userID)
		if err != nil {
			return err
		}

		// Check balance
		if acct.Balance < option.Cost {
			return hackforger_model.ErrInsufficientCredits{
				UserID:  userID,
				Balance: acct.Balance,
				Amount:  option.Cost,
			}
		}

		// Deduct balance
		acct.Balance -= option.Cost
		if _, err := db.GetEngine(ctx).ID(acct.ID).Cols("balance").Update(acct); err != nil {
			return err
		}

		// Deduct stock (if not unlimited)
		if option.Stock > 0 {
			option.Stock--
			if _, err := db.GetEngine(ctx).ID(option.ID).Cols("stock").Update(option); err != nil {
				return err
			}
		}

		// Create transaction
		tx := &hackforger_model.CreditTransaction{
			UserID:    userID,
			Type:      hackforger_model.TransactionTypeRedeem,
			Amount:    -option.Cost,
			Balance:   acct.Balance,
			Reference: option.Name,
		}
		if _, err := db.GetEngine(ctx).Insert(tx); err != nil {
			return err
		}

		// Create order
		order = &hackforger_model.RedeemOrder{
			UserID:   userID,
			OptionID: optionID,
			Cost:     option.Cost,
			Status:   hackforger_model.OrderStatusPending,
		}
		_, err = db.GetEngine(ctx).Insert(order)
		return err
	})

	return order, err
}

// ListTransactions returns credit transactions for a user.
func ListTransactions(ctx context.Context, userID int64, opts db.ListOptions) ([]*hackforger_model.CreditTransaction, int64, error) {
	sess := db.GetEngine(ctx).Where("user_id = ?", userID)
	var txns []*hackforger_model.CreditTransaction
	count, err := sess.OrderBy("created_unix DESC").
		Limit(opts.PageSize, (opts.Page-1)*opts.PageSize).
		FindAndCount(&txns)
	return txns, count, err
}

// ListRedeemOptions returns all active redeem options.
func ListRedeemOptions(ctx context.Context) ([]*hackforger_model.RedeemOption, error) {
	var options []*hackforger_model.RedeemOption
	err := db.GetEngine(ctx).Where("is_active = ?", true).Find(&options)
	return options, err
}

// ListOrders returns redeem orders for a user.
func ListOrders(ctx context.Context, userID int64, opts db.ListOptions) ([]*hackforger_model.RedeemOrder, int64, error) {
	sess := db.GetEngine(ctx).Where("user_id = ?", userID)
	var orders []*hackforger_model.RedeemOrder
	count, err := sess.OrderBy("created_unix DESC").
		Limit(opts.PageSize, (opts.Page-1)*opts.PageSize).
		FindAndCount(&orders)
	return orders, count, err
}
```

- [ ] **Step 2: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./services/hackforger/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add services/hackforger/credits.go
git commit -m "feat(hackforger): add Credits service with transactional Deposit/Redeem"
```

### Task 14: Reputation CRUD (models layer)

**Files:**
- Modify: `models/hackforger/reputation.go` (append CRUD functions)

- [ ] **Step 1: Add CRUD functions to `models/hackforger/reputation.go`**

```go
// GetOrCreateReputation returns the reputation for a user.
func GetOrCreateReputation(ctx context.Context, userID int64) (*Reputation, error) {
	rep := &Reputation{UserID: userID}
	has, err := db.GetEngine(ctx).Where("user_id = ?", userID).Get(rep)
	if err != nil {
		return nil, err
	}
	if !has {
		rep = &Reputation{UserID: userID}
		if _, err := db.GetEngine(ctx).Insert(rep); err != nil {
			return nil, err
		}
	}
	return rep, nil
}

// UpdateReputation updates a reputation record.
func UpdateReputation(ctx context.Context, rep *Reputation) error {
	_, err := db.GetEngine(ctx).ID(rep.ID).AllCols().Update(rep)
	return err
}

// ReputationLeaderboard returns top users by reputation score.
func ReputationLeaderboard(ctx context.Context, limit int) ([]*Reputation, error) {
	var reps []*Reputation
	err := db.GetEngine(ctx).OrderBy("score DESC").Limit(limit).Find(&reps)
	return reps, err
}
```

Also add `"context"` to the import block.

- [ ] **Step 2: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./models/hackforger/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/reputation.go
git commit -m "feat(hackforger): add Reputation CRUD functions in models layer"
```

---

## Chunk 4: Router Registration + Frontend Scaffolding + i18n

### Task 15: API Router Scaffolding

**Files:**
- Create: `routers/api/v1/hackforger/hackathon.go`
- Create: `routers/api/v1/hackforger/bounty.go`
- Create: `routers/api/v1/hackforger/grants.go`
- Create: `routers/api/v1/hackforger/credits.go`
- Create: `routers/api/v1/hackforger/feed.go`
- Modify: `routers/api/v1/api.go` (injection point ②)

- [ ] **Step 1: Create stub handler files**

Each file exports a single placeholder handler for now. Full implementations come in Phase 1.

`routers/api/v1/hackforger/hackathon.go`:
```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
)

// ListHackathons returns a list of hackathons.
func ListHackathons(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, []any{})
}
```

`routers/api/v1/hackforger/bounty.go`:
```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
)

// ListBounties returns a list of bounties.
func ListBounties(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, []any{})
}
```

`routers/api/v1/hackforger/grants.go`:
```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
)

// ListGrantRounds returns a list of grant rounds.
func ListGrantRounds(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, []any{})
}
```

`routers/api/v1/hackforger/credits.go`:
```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
)

// GetBalance returns the user's credit balance.
func GetBalance(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, map[string]int64{"balance": 0})
}
```

`routers/api/v1/hackforger/feed.go`:
```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	"forgejo.org/services/context"
)

// GetFeed returns the hackforger activity feed.
func GetFeed(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, map[string]any{
		"items":       []any{},
		"total_count": 0,
	})
}
```

- [ ] **Step 2: Register routes in `routers/api/v1/api.go`**

Add import at the top of `api.go`:
```go
hackforger_api "forgejo.org/routers/api/v1/hackforger"
```

Find the location inside `Routes()` where other top-level groups are registered (near the end of the function), and add:
```go
// HackForger API routes
m.Group("/hackforger", func() {
	m.Get("/hackathons", hackforger_api.ListHackathons)
	m.Get("/bounties", hackforger_api.ListBounties)
	m.Get("/grants/rounds", hackforger_api.ListGrantRounds)
	m.Get("/credits/balance", hackforger_api.GetBalance)
	m.Get("/feed", hackforger_api.GetFeed)
})
```

- [ ] **Step 3: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./routers/...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add routers/api/v1/hackforger/ routers/api/v1/api.go
git commit -m "feat(hackforger): add API router scaffolding with stub handlers"
```

### Task 16: Web Router Scaffolding

**Files:**
- Create: `routers/web/hackforger/hackathon.go`
- Create: `templates/hackforger/explore.tmpl`
- Modify: `routers/web/web.go` (injection point ①)

- [ ] **Step 1: Create web handler stub**

`routers/web/hackforger/hackathon.go`:
```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"forgejo.org/services/context"
)

const tplExplore = "hackforger/explore"

// ExploreHackathons renders the hackathon explore page.
func ExploreHackathons(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.hackathons")
	ctx.Data["PageIsExploreHackathons"] = true
	ctx.HTML(200, tplExplore)
}

// ExploreBounties renders the bounty explore page.
func ExploreBounties(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.bounties")
	ctx.Data["PageIsExploreBounties"] = true
	ctx.HTML(200, tplExplore)
}

// ExploreGrants renders the grants explore page.
func ExploreGrants(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.explore.grants")
	ctx.Data["PageIsExploreGrants"] = true
	ctx.HTML(200, tplExplore)
}
```

- [ ] **Step 2: Create minimal explore template**

`templates/hackforger/explore.tmpl`:
```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content explore">
	{{template "explore/navbar" .}}
	<div class="ui container">
		<h2>{{.Title}}</h2>
		<p>{{ctx.Locale.Tr "hackforger.explore.coming_soon"}}</p>
	</div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 3: Register routes in `routers/web/web.go`**

Add import:
```go
hackforger_web "forgejo.org/routers/web/hackforger"
```

Add the 3 explore routes **inside the existing** `m.Group("/explore", func() { ... }, ignExploreSignIn)` block (at line ~487 of `web.go`), after the existing `/organizations` route:
```go
m.Get("/hackathons", hackforger_web.ExploreHackathons)
m.Get("/bounties", hackforger_web.ExploreBounties)
m.Get("/grants", hackforger_web.ExploreGrants)
```

Important: Do NOT create a separate `/explore` group — the HackForger routes must be inside the existing group to inherit the `ignExploreSignIn` middleware, ensuring consistent auth behavior with other explore pages.

- [ ] **Step 4: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./routers/...`
Expected: SUCCESS

- [ ] **Step 5: Commit**

```bash
git add routers/web/hackforger/ routers/web/web.go templates/hackforger/explore.tmpl
git commit -m "feat(hackforger): add web router scaffolding with explore pages"
```

### Task 17: Frontend Directory Scaffolding

**Files:**
- Create: `web_src/js/features/hackforger/init.js`
- Modify: `web_src/js/index.js` (injection point ⑨)

- [ ] **Step 1: Create `web_src/js/features/hackforger/init.js`**

```js
// HackForger frontend module entry point.
// Lazy-loads Vue components when their mount elements are present.

export function initHackforger() {
  // BountyPanel
  const bountyEl = document.getElementById('hackforger-bounty-panel');
  if (bountyEl) {
    (async () => {
      const {default: BountyPanel} = await import(
        /* webpackChunkName: "hackforger-bounty" */
        '../../components/hackforger/BountyPanel.vue'
      );
      const {createApp} = await import('vue');
      createApp(BountyPanel, {
        bountyId: bountyEl.getAttribute('data-bounty-id'),
      }).mount(bountyEl);
    })();
  }
}
```

- [ ] **Step 2: Create placeholder Vue component**

Create `web_src/js/components/hackforger/BountyPanel.vue`:
```vue
<template>
  <div class="hackforger-bounty-panel">
    <!-- Placeholder: full implementation in Phase 1 -->
  </div>
</template>

<script>
export default {
  name: 'BountyPanel',
  props: {
    bountyId: {
      type: String,
      default: '',
    },
  },
};
</script>
```

- [ ] **Step 3: Add import in `web_src/js/index.js`**

Add import at top of file:
```js
import {initHackforger} from './features/hackforger/init.js';
```

Add call inside the `onDomReady()` block:
```js
initHackforger();
```

- [ ] **Step 4: Verify frontend builds**

Run: `cd /Users/h2oslabs/Workspace/hackforger && make frontend`
Expected: SUCCESS (or at least no import/syntax errors)

- [ ] **Step 5: Commit**

```bash
git add web_src/js/features/hackforger/ web_src/js/components/hackforger/ web_src/js/index.js
git commit -m "feat(hackforger): add frontend directory scaffolding with lazy-load init"
```

### Task 18: i18n Keys

**Files:**
- Modify: `options/locale/locale_en-US.ini`

- [ ] **Step 1: Add HackForger i18n section**

Append to the end of `options/locale/locale_en-US.ini`:
```ini

[hackforger]
explore.hackathons = Hackathons
explore.bounties = Bounties
explore.grants = Grants
explore.coming_soon = Coming soon...

hackathon.new = New Hackathon
hackathon.status.draft = Draft
hackathon.status.registration = Registration
hackathon.status.hacking = Hacking
hackathon.status.judging = Judging
hackathon.status.finished = Finished

bounty.status.open = Open
bounty.status.claimed = Claimed
bounty.status.in_review = In Review
bounty.status.completed = Completed
bounty.status.paid = Paid
bounty.status.expired = Expired
bounty.status.cancelled = Cancelled
bounty.mode.exclusive = Exclusive
bounty.mode.competitive = Competitive

grants.round.status.setup = Setup
grants.round.status.open = Open
grants.round.status.reviewing = Reviewing
grants.round.status.finalized = Finalized
grants.round.status.distributed = Distributed

credits.balance = Balance
credits.transactions = Transactions
credits.redeem = Redeem
credits.redeem.options = Redeem Options
credits.redeem.orders = Orders

reputation.score = Reputation Score
reputation.leaderboard = Leaderboard

feed.following = Following
feed.global = Global
```

- [ ] **Step 2: Add cron task i18n keys**

These must exist in the `[admin]` section for cron task registration. Find the `[admin]` section and add:
```ini
dashboard.hackforger_hackathon_status = Check hackathon phase transitions
dashboard.hackforger_bounty_expiry = Expire stale bounties
dashboard.hackforger_reputation_recalc = Recalculate reputation scores
dashboard.hackforger_grant_deadline = Check grant round deadlines
```

- [ ] **Step 3: Verify the locale file is valid**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add options/locale/locale_en-US.ini
git commit -m "feat(hackforger): add i18n keys for all hackforger modules"
```

### Task 19: Cron Task Registration

**Files:**
- Create: `services/cron/tasks_hackforger.go`
- Modify: `services/cron/cron.go` (injection point ③)

- [ ] **Step 1: Create `services/cron/tasks_hackforger.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package cron

import (
	"context"

	user_model "forgejo.org/models/user"
)

func registerHackforgerHackathonStatus() {
	RegisterTaskFatal("hackforger_hackathon_status", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@every 5m",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		// Phase 1: call hackforger_service.CheckHackathonTransitions(ctx)
		return nil
	})
}

func registerHackforgerBountyExpiry() {
	RegisterTaskFatal("hackforger_bounty_expiry", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@every 5m",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		// Phase 1: call hackforger_service.ExpireOldBounties(ctx)
		return nil
	})
}

func registerHackforgerReputationRecalc() {
	RegisterTaskFatal("hackforger_reputation_recalc", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@every 1h",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		// Phase 1: call hackforger_service.RecalculateAllReputations(ctx)
		return nil
	})
}

func registerHackforgerGrantDeadline() {
	RegisterTaskFatal("hackforger_grant_deadline", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@every 1h",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		// Phase 1: call hackforger_service.CheckGrantDeadlines(ctx)
		return nil
	})
}

func initHackforgerTasks() {
	registerHackforgerHackathonStatus()
	registerHackforgerBountyExpiry()
	registerHackforgerReputationRecalc()
	registerHackforgerGrantDeadline()
}
```

- [ ] **Step 2: Register in `services/cron/cron.go`**

Add `initHackforgerTasks()` call after `initActionsTasks()` inside `NewContext()`:
```go
initHackforgerTasks()
```

- [ ] **Step 3: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./services/cron/...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add services/cron/tasks_hackforger.go services/cron/cron.go
git commit -m "feat(hackforger): register 4 cron tasks (hackathon/bounty/reputation/grant)"
```

---

## Chunk 5: Package Init, Test Infrastructure, Verification + Tests

### Task 20: Package init.go + Test Import Registration

**Files:**
- Create: `models/hackforger/init.go`
- Modify: `modules/testimport/import.go`

- [ ] **Step 1: Create `models/hackforger/init.go`**

This file serves as the package documentation entry point. Individual model files register themselves via their own `init()` functions.

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Package hackforger contains the data models for HackForger's
// Hackathon, Bounty, Grant, Credits, and Reputation modules.
// Each model file registers its struct via db.RegisterModel() in init().
package hackforger
```

- [ ] **Step 2: Add blank import in `modules/testimport/import.go`**

This is critical — without it, integration tests won't register HackForger tables and will fail.

Add to the import block:
```go
_ "forgejo.org/models/hackforger"
```

- [ ] **Step 3: Verify compile**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go build ./...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/init.go modules/testimport/import.go
git commit -m "feat(hackforger): add package init.go and testimport registration"
```

### Task 21: Create Minimal Test Fixture YAML Files

**Files:**
- Create: `models/fixtures/hackforger_hackathon.yml` (and 15 more)

- [ ] **Step 1: Create empty fixture YAML files for all 16 tables**

Each file is initially empty (just a YAML comment). These will be populated with test data in Phase 1. The files must exist for the test infrastructure to recognize the tables.

```bash
cd /Users/h2oslabs/Workspace/hackforger
for table in hackathon hackathon_track hackathon_registration hackathon_submission hackathon_judge_score bounty bounty_reward bounty_application bounty_winner grant_round grant_project credit_account credit_transaction redeem_option redeem_order reputation; do
  echo "# HackForger test fixture: $table" > "models/fixtures/$table.yml"
done
```

- [ ] **Step 2: Commit**

```bash
git add models/fixtures/*.yml
git commit -m "feat(hackforger): add empty test fixture YAML files for all 16 tables"
```

### Task 22: Full Build Verification

- [ ] **Step 1: Run full backend build**

Run: `cd /Users/h2oslabs/Workspace/hackforger && make backend`
Expected: SUCCESS — binary compiles with all new code

- [ ] **Step 2: Run full frontend build**

Run: `cd /Users/h2oslabs/Workspace/hackforger && make frontend`
Expected: SUCCESS

- [ ] **Step 3: Verify migration registers correctly**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go test ./models/forgejo_migrations/ -run TestMigrations -v`
Expected: If there's a migration registration test, it should pass. If not, at minimum `go build` confirms the init() was picked up.

### Task 23: Model Layer Unit Tests

**Files:**
- Create: `models/hackforger/hackathon_test.go`
- Create: `models/hackforger/bounty_test.go`
- Create: `models/hackforger/credits_test.go`

Note: These tests verify struct registration and basic XORM operations. They require the Forgejo test infrastructure (`tests/integration/` framework or `models/unittest/`). Check how existing model tests are structured before writing.

- [ ] **Step 1: Research existing model test patterns**

Run: `ls models/issues/*_test.go | head -5` and read one to understand the test setup pattern (e.g., `TestMain`, `unittest.PrepareTestEnv`).

- [ ] **Step 2: Create `models/hackforger/hackathon_test.go`**

The exact test setup depends on the patterns found in Step 1. At minimum, test that the structs can be inserted and queried:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

// Tests will follow the pattern discovered in Step 1.
// Key test cases from test-plan-draft.md:
//
// TestCreateHackathon — normal create, verify ID/Slug/Status/CreatedUnix
// TestCreateHackathon_DuplicateSlug — duplicate slug → ErrHackathonSlugExists
// TestGetHackathonByID — read correct
// TestGetHackathonBySlug — read correct
// TestListHackathons_FilterByStatus — filter by status
```

- [ ] **Step 3: Write and run tests based on discovered patterns**

Run: `cd /Users/h2oslabs/Workspace/hackforger && go test ./models/hackforger/... -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/*_test.go
git commit -m "test(hackforger): add model layer unit tests for Hackathon, Bounty, Credits"
```

### Task 24: Final Phase 0 Verification

- [ ] **Step 1: Verify all 15 model structs registered**

Run: `cd /Users/h2oslabs/Workspace/hackforger && grep -r "db.RegisterModel" models/hackforger/ | wc -l`
Expected: 15 (one per table: Hackathon, HackathonTrack, HackathonRegistration, HackathonSubmission, HackathonJudgeScore, Bounty, BountyReward, BountyApplication, BountyWinner, GrantRound, GrantProject, CreditAccount, CreditTransaction, RedeemOption, RedeemOrder, Reputation)

Actually 16 — 15 tables + Reputation = 16 `RegisterModel` calls. The implementation plan says "15 tables" but Reputation is the 16th struct. Verify: Hackathon(1) + Track(2) + Registration(3) + Submission(4) + JudgeScore(5) + Bounty(6) + BountyReward(7) + BountyApplication(8) + BountyWinner(9) + GrantRound(10) + GrantProject(11) + CreditAccount(12) + CreditTransaction(13) + RedeemOption(14) + RedeemOrder(15) + Reputation(16) = **16 RegisterModel calls**.

- [ ] **Step 2: Verify directory structure matches plan**

Run:
```bash
find models/hackforger services/hackforger routers/api/v1/hackforger routers/web/hackforger templates/hackforger web_src/js/features/hackforger web_src/js/components/hackforger -type f | sort
```
Expected: All planned files exist

- [ ] **Step 3: Verify injection points**

Check that all 4 P0 injection points are in place:
1. `routers/api/v1/api.go` — contains `/hackforger` group
2. `routers/web/web.go` — contains hackforger routes
3. `routers/init.go` — contains `hackforger_service.Init`
4. `services/cron/cron.go` — contains `initHackforgerTasks()`

Run:
```bash
grep -n "hackforger" routers/api/v1/api.go routers/web/web.go routers/init.go services/cron/cron.go
```
Expected: Each file shows the injection

- [ ] **Step 4: Final commit if any fixups needed**

```bash
git status
# If clean: Phase 0 complete
# If changes: git add ... && git commit -m "fix(hackforger): phase 0 fixups"
```

---

---

## Chunk 6: Manual E2E Verification

### Task 25: Generate E2E Prompt and Report Template

**Files:**
- Create: `docs/tests/e2e/p0-infrastructure-e2e-prompt.md`
- Create: `docs/tests/e2e/p0-infrastructure-e2e-report.md`

- [ ] **Step 1: Create E2E prompt guide**

See `docs/tests/e2e/p0-infrastructure-e2e-prompt.md` (already generated with the plan).

- [ ] **Step 2: Create E2E report template**

See `docs/tests/e2e/p0-infrastructure-e2e-report.md` (already generated with the plan).

- [ ] **Step 3: Hand off to human for manual verification**

Notify the user that P0 implementation is complete and ready for manual E2E testing. The user should follow the prompt guide, fill in the report, and flag any issues.

- [ ] **Step 4: Commit**

```bash
git add docs/tests/e2e/p0-infrastructure-e2e-prompt.md docs/tests/e2e/p0-infrastructure-e2e-report.md
git commit -m "docs(hackforger): add P0 manual E2E test prompt and report template"
```

---

## Summary

| Chunk | Tasks | Files Created | Files Modified | Commits |
|-------|-------|--------------|----------------|---------|
| 1: Data Models | 1-6 | 12 model files | — | 6 |
| 2: Feed + Migration + Notifier | 7-9 | 3 files | `routers/init.go` | 3 |
| 3: CRUD (models) + Credits Service | 10-14 | 1 service file | 5 model files (append CRUD) | 5 |
| 4: Routers + Frontend + i18n | 15-19 | 10+ files | 4 injection points | 5 |
| 5: Init + Fixtures + Verification + Tests | 20-24 | init.go + 16 fixtures + tests | `modules/testimport/import.go` | 4-5 |
| 6: Manual E2E Verification | 25 | 2 doc files | — | 1 |
| **Total** | **25 tasks** | **~47 files** | **~7 files** | **~24 commits** |

### Review fixes applied (from code-reviewer):
- **CRUD in models layer**: Data access functions (Get/List/Create/Update/Delete) moved from `services/` to `models/hackforger/`, matching Forgejo convention
- **Custom audience distribution**: `PublishHackforgerAction` uses direct action insertion, not `NotifyWatchers` (which only handles repo watchers)
- **Test infrastructure**: Added `modules/testimport/import.go` blank import (critical for integration tests)
- **Status filter zero-value**: Changed to pointer types (`*HackathonStatus`, `*BountyStatus`, `*GrantRoundStatus`) so nil = no filter
- **Error semantics**: `ErrInsufficientCredits` and `ErrOutOfStock` now wrap `util.ErrInvalidArgument` instead of `util.ErrPermissionDenied`
- **Explore routes**: Added inside existing `/explore` group (inherits `ignExploreSignIn` middleware)
- **Package init.go**: Added `models/hackforger/init.go` as package doc
- **Test fixtures**: Added task for empty YAML fixture files to unblock Phase 1 integration tests
- **Table count**: Clarified 16 tables (spec says 15 but JudgeScore is the 16th)
