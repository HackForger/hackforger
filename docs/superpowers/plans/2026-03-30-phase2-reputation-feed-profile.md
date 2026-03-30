# Phase 2: Reputation + Feed + Profile — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Separate HackForger feed into its own table, add reputation system with admin-configurable weights/tiers, and integrate both into Dashboard and Profile pages.

**Architecture:** New `hackforger_action` table replaces writing to Forgejo's `action` table. Reputation scores computed via cron from DB metrics + admin-configurable weights stored in `hackforger_setting` table. Dashboard and Profile get "Community" tabs; Profile gets reputation sidebar card and "Reputation" tab.

**Tech Stack:** Go (XORM ORM), Go HTML templates (SSR), Forgejo's `chi` router, i18n via locale INI files.

**Spec:** `docs/superpowers/specs/2026-03-30-phase2-reputation-feed-profile-design.md`

---

## Chunk 1: Data Layer

### Task 1: HackforgerAction Model

**Files:**
- Create: `models/hackforger/hackforger_action.go`
- Create: `models/hackforger/hackforger_action_test.go`

- [ ] **Step 1: Create the model file with struct + init**

```go
// models/hackforger/hackforger_action.go
package hackforger

import (
	"context"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

// HackforgerAction stores HackForger feed events in a separate table
// from Forgejo's repo-centric action table.
type HackforgerAction struct {
	ID          int64                       `xorm:"pk autoincr"`
	UserID      int64                       `xorm:"NOT NULL INDEX(idx_hfa_user_created)"`
	ActUserID   int64                       `xorm:"NOT NULL INDEX(idx_hfa_actuser_created)"`
	OpType      activities_model.ActionType `xorm:"NOT NULL"`
	EntityType  string                      `xorm:"VARCHAR(20) NOT NULL INDEX(idx_hfa_entity)"`
	EntityID    int64                       `xorm:"NOT NULL DEFAULT 0"`
	EntityName  string                      `xorm:"VARCHAR(255)"`
	EntitySlug  string                      `xorm:"VARCHAR(255)"`
	OrgID       int64                       `xorm:"DEFAULT 0"`
	RepoID      int64                       `xorm:"DEFAULT 0"`
	Content     string                      `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp          `xorm:"INDEX(idx_hfa_user_created) INDEX(idx_hfa_actuser_created) INDEX(idx_hfa_entity) created"`

	ActUser *user_model.User `xorm:"-"`
}

func init() {
	db.RegisterModel(new(HackforgerAction))
}

// GetHackforgerFeedsOptions configures feed queries.
type GetHackforgerFeedsOptions struct {
	db.ListOptions
	UserID        int64
	ActUserID     int64
	EntityType    string
	EntityID      int64
	IncludeGlobal bool
}

// GetHackforgerFeeds returns feed events matching the given options.
func GetHackforgerFeeds(ctx context.Context, opts GetHackforgerFeedsOptions) ([]*HackforgerAction, int64, error) {
	cond := builder.NewCond()

	if opts.UserID > 0 {
		if opts.IncludeGlobal {
			cond = cond.And(builder.Or(
				builder.Eq{"user_id": opts.UserID},
				builder.Eq{"user_id": 0},
			))
		} else {
			cond = cond.And(builder.Eq{"user_id": opts.UserID})
		}
	} else if opts.IncludeGlobal {
		cond = cond.And(builder.Eq{"user_id": 0})
	}

	if opts.ActUserID > 0 {
		cond = cond.And(builder.Eq{"act_user_id": opts.ActUserID})
	}

	if opts.EntityType != "" {
		cond = cond.And(builder.Eq{"entity_type": opts.EntityType})
	}

	if opts.EntityID > 0 {
		cond = cond.And(builder.Eq{"entity_id": opts.EntityID})
	}

	opts.SetDefaultValues()
	sess := db.GetEngine(ctx).Where(cond)
	sess = db.SetSessionPagination(sess, &opts.ListOptions)

	actions := make([]*HackforgerAction, 0, opts.PageSize)
	count, err := sess.Desc("created_unix").FindAndCount(&actions)
	return actions, count, err
}

// GetEntityTimeline returns all events for a specific entity, deduplicated.
func GetEntityTimeline(ctx context.Context, entityType string, entityID int64, opts db.ListOptions) ([]*HackforgerAction, int64, error) {
	cond := builder.Eq{"entity_type": entityType, "entity_id": entityID}

	opts.SetDefaultValues()
	sess := db.GetEngine(ctx).Where(cond).
		GroupBy("act_user_id, op_type, entity_type, entity_id, created_unix")
	sess = db.SetSessionPagination(sess, &opts)

	actions := make([]*HackforgerAction, 0, opts.PageSize)
	count, err := sess.Desc("created_unix").FindAndCount(&actions)
	return actions, count, err
}

// LoadActUsers bulk-loads ActUser for a slice of HackforgerActions.
func LoadActUsers(ctx context.Context, actions []*HackforgerAction) error {
	if len(actions) == 0 {
		return nil
	}

	userIDs := make([]int64, 0, len(actions))
	seen := make(map[int64]bool)
	for _, a := range actions {
		if !seen[a.ActUserID] {
			userIDs = append(userIDs, a.ActUserID)
			seen[a.ActUserID] = true
		}
	}

	userMap := make(map[int64]*user_model.User)
	if len(userIDs) > 0 {
		var users []*user_model.User
		if err := db.GetEngine(ctx).In("id", userIDs).Find(&users); err != nil {
			return err
		}
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	for _, a := range actions {
		a.ActUser = userMap[a.ActUserID]
	}
	return nil
}
```

- [ ] **Step 2: Write tests**

```go
// models/hackforger/hackforger_action_test.go
package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHackforgerFeeds_Global(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Insert a global event (UserID=0)
	action := &hackforger_model.HackforgerAction{
		UserID:     0,
		ActUserID:  2,
		OpType:     30, // hackathon_created
		EntityType: "hackathon",
		EntityID:   1,
		EntityName: "Test Hackathon",
	}
	_, err := db.GetEngine(db.DefaultContext).Insert(action)
	require.NoError(t, err)

	// Query with IncludeGlobal should find it
	feeds, count, err := hackforger_model.GetHackforgerFeeds(db.DefaultContext, hackforger_model.GetHackforgerFeedsOptions{
		UserID:        99, // any user
		IncludeGlobal: true,
		ListOptions:   db.ListOptions{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.True(t, count >= 1)
	assert.True(t, len(feeds) >= 1)
}

func TestGetHackforgerFeeds_ByActUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	action := &hackforger_model.HackforgerAction{
		UserID:     5,
		ActUserID:  2,
		OpType:     34, // bounty_created
		EntityType: "bounty",
		EntityID:   10,
		EntityName: "Fix login bug",
	}
	_, err := db.GetEngine(db.DefaultContext).Insert(action)
	require.NoError(t, err)

	feeds, count, err := hackforger_model.GetHackforgerFeeds(db.DefaultContext, hackforger_model.GetHackforgerFeedsOptions{
		ActUserID:   2,
		ListOptions: db.ListOptions{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	assert.True(t, count >= 1)
	found := false
	for _, f := range feeds {
		if f.EntityName == "Fix login bug" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestLoadActUsers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	actions := []*hackforger_model.HackforgerAction{
		{ActUserID: 1},
		{ActUserID: 2},
		{ActUserID: 1}, // duplicate
	}
	err := hackforger_model.LoadActUsers(db.DefaultContext, actions)
	require.NoError(t, err)
	assert.NotNil(t, actions[0].ActUser)
	assert.NotNil(t, actions[1].ActUser)
	assert.Equal(t, actions[0].ActUser, actions[2].ActUser)
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./models/hackforger/... -run TestGetHackforgerFeeds -v && go test ./models/hackforger/... -run TestLoadActUsers -v`

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/hackforger_action.go models/hackforger/hackforger_action_test.go
git commit -m "feat(feed): add HackforgerAction model with query functions and tests"
```

---

### Task 2: HackforgerSetting Model

**Files:**
- Create: `models/hackforger/setting.go`
- Create: `models/hackforger/setting_test.go`

- [ ] **Step 1: Create the model file**

```go
// models/hackforger/setting.go
package hackforger

import (
	"context"

	"forgejo.org/models/db"
)

// HackforgerSetting stores HackForger-specific configuration in the database.
type HackforgerSetting struct {
	ID    int64  `xorm:"pk autoincr"`
	Key   string `xorm:"VARCHAR(100) UNIQUE NOT NULL"`
	Value string `xorm:"TEXT NOT NULL"`
}

func init() {
	db.RegisterModel(new(HackforgerSetting))
}

// GetSetting returns the value for a key, or empty string + error if not found.
func GetSetting(ctx context.Context, key string) (string, error) {
	s := new(HackforgerSetting)
	has, err := db.GetEngine(ctx).Where("`key` = ?", key).Get(s)
	if err != nil {
		return "", err
	}
	if !has {
		return "", nil
	}
	return s.Value, nil
}

// GetSettingWithDefault returns the value for a key, or defaultValue if not found.
func GetSettingWithDefault(ctx context.Context, key, defaultValue string) string {
	val, err := GetSetting(ctx, key)
	if err != nil || val == "" {
		return defaultValue
	}
	return val
}

// SetSetting creates or updates a setting.
func SetSetting(ctx context.Context, key, value string) error {
	s := new(HackforgerSetting)
	has, err := db.GetEngine(ctx).Where("`key` = ?", key).Get(s)
	if err != nil {
		return err
	}
	if has {
		s.Value = value
		_, err = db.GetEngine(ctx).ID(s.ID).Cols("value").Update(s)
		return err
	}
	s = &HackforgerSetting{Key: key, Value: value}
	_, err = db.GetEngine(ctx).Insert(s)
	return err
}
```

- [ ] **Step 2: Write tests**

```go
// models/hackforger/setting_test.go
package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetting_GetSetDefault(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// GetSettingWithDefault returns default when key missing
	val := hackforger_model.GetSettingWithDefault(db.DefaultContext, "test.key", "fallback")
	assert.Equal(t, "fallback", val)

	// SetSetting creates
	require.NoError(t, hackforger_model.SetSetting(db.DefaultContext, "test.key", "hello"))
	val, err := hackforger_model.GetSetting(db.DefaultContext, "test.key")
	require.NoError(t, err)
	assert.Equal(t, "hello", val)

	// SetSetting updates
	require.NoError(t, hackforger_model.SetSetting(db.DefaultContext, "test.key", "world"))
	val, err = hackforger_model.GetSetting(db.DefaultContext, "test.key")
	require.NoError(t, err)
	assert.Equal(t, "world", val)
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./models/hackforger/... -run TestSetting -v`

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/setting.go models/hackforger/setting_test.go
git commit -m "feat(settings): add HackforgerSetting model for admin-configurable settings"
```

---

### Task 3: Reputation Model Update

**Files:**
- Modify: `models/hackforger/reputation.go` (add `Tier` field)

- [ ] **Step 1: Add Tier field to Reputation struct**

In `models/hackforger/reputation.go`, add after `TotalCreditsEarned`:

```go
Tier               string             `xorm:"VARCHAR(20) NOT NULL DEFAULT 'Bronze'"`
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./models/hackforger/...`

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/reputation.go
git commit -m "feat(reputation): add Tier field to Reputation model"
```

---

### Task 4: Migration

**Files:**
- Create: `models/forgejo_migrations/v14g_hackforger-phase2-tables.go`

- [ ] **Step 1: Create migration file**

```go
// models/forgejo_migrations/v14g_hackforger-phase2-tables.go
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

func addHackforgerPhase2Tables(x *xorm.Engine) error {
	// Must explicitly Sync new tables — db.RegisterModel only runs at
	// engine init, not during migration execution.
	type HackforgerAction struct {
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
	if err := x.Sync(new(HackforgerAction)); err != nil {
		return err
	}

	type HackforgerSetting struct {
		ID    int64  `xorm:"pk autoincr"`
		Key   string `xorm:"VARCHAR(100) UNIQUE NOT NULL"`
		Value string `xorm:"TEXT NOT NULL"`
	}
	if err := x.Sync(new(HackforgerSetting)); err != nil {
		return err
	}

	// Add tier column to existing reputation table
	type Reputation struct {
		Tier string `xorm:"VARCHAR(20) NOT NULL DEFAULT 'Bronze'"`
	}
	if err := x.Sync(new(Reputation)); err != nil {
		return err
	}

	// Seed default settings
	defaults := []HackforgerSetting{
		{Key: "reputation.weights", Value: `{"stars":1,"bounties_completed":5,"hackathon_wins":10,"grants_received":3,"credits_earned":0.1}`},
		{Key: "reputation.tiers", Value: `[{"name":"Bronze","min":0},{"name":"Silver","min":50},{"name":"Gold","min":200},{"name":"Diamond","min":500}]`},
	}
	for _, s := range defaults {
		has, err := x.Where("`key` = ?", s.Key).Exist(new(HackforgerSetting))
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
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./models/forgejo_migrations/...`

- [ ] **Step 3: Commit**

```bash
git add models/forgejo_migrations/v14g_hackforger-phase2-tables.go
git commit -m "feat(migration): add Phase 2 tables and seed default reputation settings"
```

---

### Task 5: i18n Keys (All)

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 1: Add all i18n keys to both locale files**

Add under the existing `[hackforger]` section in both files.

**English (`locale_en-US.ini`):**
```ini
; Feed tabs
feed.code = Code Activity
feed.community = Community
feed.community_empty = No community activity yet. Explore hackathons, bounties, and grants to get started!

; Profile tabs
profile.community = Community
profile.reputation = Reputation

; Feed event messages
feed.hackathon_created = created hackathon <a href="%[2]s">%[1]s</a>
feed.hackathon_registered = registered for <a href="%[2]s">%[1]s</a>
feed.hackathon_submitted = submitted to <a href="%[2]s">%[1]s</a>
feed.hackathon_scored = scored a submission in <a href="%[2]s">%[1]s</a>
feed.hackathon_phase_changed = changed phase of <a href="%[2]s">%[1]s</a>
feed.hackathon_finalized = finalized <a href="%[2]s">%[1]s</a>
feed.bounty_created = created a bounty in <a href="%[2]s">%[1]s</a>
feed.bounty_claimed = claimed a bounty in <a href="%[2]s">%[1]s</a>
feed.bounty_delivered = delivered a bounty in <a href="%[2]s">%[1]s</a>
feed.bounty_completed = completed a bounty in <a href="%[2]s">%[1]s</a>
feed.bounty_paid = paid a bounty in <a href="%[2]s">%[1]s</a>
feed.bounty_winners_selected = selected bounty winners in <a href="%[2]s">%[1]s</a>
feed.bounty_expired = bounty expired in <a href="%[2]s">%[1]s</a>
feed.bounty_cancelled = cancelled a bounty in <a href="%[2]s">%[1]s</a>
feed.grant_round_created = created grant round <a href="%[2]s">%[1]s</a>
feed.grant_project_submitted = submitted a project to <a href="%[2]s">%[1]s</a>
feed.grant_awarded = received a grant from <a href="%[2]s">%[1]s</a>
feed.grant_round_opened = opened grant round <a href="%[2]s">%[1]s</a>
feed.grant_round_closed = closed grant round <a href="%[2]s">%[1]s</a>
feed.grant_round_finalized = finalized grant round <a href="%[2]s">%[1]s</a>
feed.grant_round_cancelled = cancelled grant round <a href="%[2]s">%[1]s</a>
feed.credits_redeemed = redeemed credits

; Reputation
reputation.score = Reputation Score
reputation.tier = Tier
reputation.bounties_completed = Bounties Completed
reputation.hackathon_wins = Hackathon Wins
reputation.grants_received = Grants Received
reputation.total_stars = Total Stars
reputation.credits_earned = Credits Earned
reputation.leaderboard = Reputation Leaderboard
reputation.rank = Rank
reputation.view_leaderboard = View Leaderboard

; Admin reputation
admin.reputation = Reputation Settings
admin.reputation.weights = Score Weights
admin.reputation.tiers = Tier Thresholds
admin.reputation.recalc = Recalculate All
admin.reputation.recalc_success = Reputation recalculation triggered successfully
admin.reputation.save_success = Reputation settings saved
```

**Chinese (`locale_zh-CN.ini`):** Same keys with Chinese translations:
```ini
feed.code = 代码动态
feed.community = 社区动态
feed.community_empty = 暂无社区动态。去探索 Hackathon、悬赏和资助吧！

profile.community = 社区
profile.reputation = 声誉

feed.hackathon_created = 创建了 Hackathon <a href="%[2]s">%[1]s</a>
feed.hackathon_registered = 报名了 <a href="%[2]s">%[1]s</a>
feed.hackathon_submitted = 提交了作品到 <a href="%[2]s">%[1]s</a>
feed.hackathon_scored = 评审了 <a href="%[2]s">%[1]s</a> 的提交
feed.hackathon_phase_changed = 更改了 <a href="%[2]s">%[1]s</a> 的阶段
feed.hackathon_finalized = 公布了 <a href="%[2]s">%[1]s</a> 的结果
feed.bounty_created = 在 <a href="%[2]s">%[1]s</a> 发布了悬赏
feed.bounty_claimed = 认领了 <a href="%[2]s">%[1]s</a> 的悬赏
feed.bounty_delivered = 交付了 <a href="%[2]s">%[1]s</a> 的悬赏
feed.bounty_completed = 完成了 <a href="%[2]s">%[1]s</a> 的悬赏
feed.bounty_paid = 支付了 <a href="%[2]s">%[1]s</a> 的悬赏
feed.bounty_winners_selected = 选出了 <a href="%[2]s">%[1]s</a> 的获奖者
feed.bounty_expired = <a href="%[2]s">%[1]s</a> 的悬赏已过期
feed.bounty_cancelled = 取消了 <a href="%[2]s">%[1]s</a> 的悬赏
feed.grant_round_created = 创建了资助轮次 <a href="%[2]s">%[1]s</a>
feed.grant_project_submitted = 提交了项目到 <a href="%[2]s">%[1]s</a>
feed.grant_awarded = 获得了 <a href="%[2]s">%[1]s</a> 的资助
feed.grant_round_opened = 开放了资助轮次 <a href="%[2]s">%[1]s</a>
feed.grant_round_closed = 关闭了资助轮次 <a href="%[2]s">%[1]s</a>
feed.grant_round_finalized = 完结了资助轮次 <a href="%[2]s">%[1]s</a>
feed.grant_round_cancelled = 取消了资助轮次 <a href="%[2]s">%[1]s</a>
feed.credits_redeemed = 兑换了积分

reputation.score = 声誉分
reputation.tier = 等级
reputation.bounties_completed = 完成的悬赏
reputation.hackathon_wins = Hackathon 获奖
reputation.grants_received = 获得的资助
reputation.total_stars = 收获的 Star
reputation.credits_earned = 累计获得积分
reputation.leaderboard = 声誉排行榜
reputation.rank = 排名
reputation.view_leaderboard = 查看排行榜

admin.reputation = 声誉设置
admin.reputation.weights = 评分权重
admin.reputation.tiers = 等级阈值
admin.reputation.recalc = 重新计算
admin.reputation.recalc_success = 已触发声誉重新计算
admin.reputation.save_success = 声誉设置已保存
```

- [ ] **Step 2: Verify no syntax errors in locale files**

Run: `go build ./...` (locale files are embedded at build time with `bindata` tag; compilation verifies syntax)

- [ ] **Step 3: Commit**

```bash
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "i18n: add Phase 2 locale keys for feed tabs, reputation, and admin"
```

---

## Chunk 2: Feed System Refactor

### Task 6: Notifier Refactor

**Files:**
- Modify: `services/hackforger/notifier.go`

- [ ] **Step 1: Fix AudienceType to use bit flags**

Replace the `AudienceType` constants (lines 33-40):

```go
// Before:
// AudienceGlobal       AudienceType = iota
// AudienceFollowers
// AudienceOrgMembers
// AudienceRepoWatchers

// After:
const (
	AudienceGlobal       AudienceType = 1 << iota // 1
	AudienceFollowers                               // 2
	AudienceOrgMembers                              // 4
	AudienceRepoWatchers                            // 8
)
```

- [ ] **Step 2: Update HackforgerActionOpts with entity fields**

Replace the struct definition:

```go
type HackforgerActionOpts struct {
	ActUserID    int64
	OpType       activities_model.ActionType
	EntityType   string
	EntityID     int64
	EntityName   string
	EntitySlug   string
	RepoID       int64
	Content      any
	AudienceType AudienceType
	OrgID        int64
}
```

- [ ] **Step 3: Rewrite PublishHackforgerAction to write to hackforger_action table**

Replace the entire function body. Key changes:
- Write to `hackforger_model.HackforgerAction` instead of `activities_model.Action`
- Use `if opts.AudienceType&AudienceX != 0` flag checks instead of `switch`
- Always insert actor record; additionally insert per-audience records

```go
func PublishHackforgerAction(ctx context.Context, opts *HackforgerActionOpts) error {
	contentStr := ""
	if opts.Content != nil {
		contentBytes, err := json.Marshal(opts.Content)
		if err != nil {
			return err
		}
		contentStr = string(contentBytes)
	}
	now := timeutil.TimeStampNow()

	// Helper to insert a single record
	insert := func(userID int64) {
		a := &hackforger_model.HackforgerAction{
			UserID:      userID,
			ActUserID:   opts.ActUserID,
			OpType:      opts.OpType,
			EntityType:  opts.EntityType,
			EntityID:    opts.EntityID,
			EntityName:  opts.EntityName,
			EntitySlug:  opts.EntitySlug,
			OrgID:       opts.OrgID,
			RepoID:      opts.RepoID,
			Content:     contentStr,
			CreatedUnix: now,
		}
		if _, err := db.GetEngine(ctx).Insert(a); err != nil {
			log.Error("PublishHackforgerAction (user_id=%d): %v", userID, err)
		}
	}

	// Actor's own record
	insert(opts.ActUserID)

	// Global audience
	if opts.AudienceType&AudienceGlobal != 0 {
		insert(0)
	}

	// Followers audience
	if opts.AudienceType&AudienceFollowers != 0 {
		var followerIDs []int64
		if err := db.GetEngine(ctx).Table("follow").
			Where("follow_id = ?", opts.ActUserID).
			Cols("user_id").Find(&followerIDs); err != nil {
			log.Error("PublishHackforgerAction (followers query): %v", err)
		} else {
			for _, uid := range followerIDs {
				if uid != opts.ActUserID {
					insert(uid)
				}
			}
		}
	}

	// Org members audience
	if opts.AudienceType&AudienceOrgMembers != 0 && opts.OrgID > 0 {
		var memberIDs []int64
		if err := db.GetEngine(ctx).Table("org_user").
			Where("org_id = ?", opts.OrgID).
			Cols("uid").Find(&memberIDs); err != nil {
			log.Error("PublishHackforgerAction (org members query): %v", err)
		} else {
			for _, uid := range memberIDs {
				if uid != opts.ActUserID {
					insert(uid)
				}
			}
		}
	}

	// Repo watchers audience
	if opts.AudienceType&AudienceRepoWatchers != 0 && opts.RepoID > 0 {
		var watcherIDs []int64
		if err := db.GetEngine(ctx).Table("watch").
			Where("repo_id = ? AND mode != 2", opts.RepoID).
			Cols("user_id").Find(&watcherIDs); err != nil {
			log.Error("PublishHackforgerAction (watchers query): %v", err)
		} else {
			for _, uid := range watcherIDs {
				if uid != opts.ActUserID {
					insert(uid)
				}
			}
		}
	}

	return nil
}
```

- [ ] **Step 4: Remove old imports for `activities_model.Action` that are no longer needed**

Update import block — keep `activities_model` (still used for ActionType), remove unused imports. Remove the `MergePullRequest` method's unused variable placeholders.

- [ ] **Step 5: Verify compilation**

Run: `go build ./services/hackforger/...`

Expected: Compilation errors in callers (bounty.go, hackathon.go, grants.go, credits.go) because `HackforgerActionOpts` now requires new fields. This is expected — Task 7 fixes callers.

- [ ] **Step 6: Commit**

```bash
git add services/hackforger/notifier.go
git commit -m "refactor(feed): rewrite notifier to write hackforger_action table with bit-flag audiences"
```

---

### Task 7: Update All Existing Callers

**Files:**
- Modify: `services/hackforger/bounty.go` (all `PublishHackforgerAction` calls)
- Modify: `services/hackforger/hackathon.go` (all calls)
- Modify: `services/hackforger/grants.go` (all calls, including `publishGrantEvent` helper)
- Modify: `services/hackforger/credits.go` (all calls)

- [ ] **Step 1: Update each caller to pass EntityType/EntityID/EntityName/EntitySlug**

The pattern for each caller:
- Extract entity fields from the existing `Content` struct (they're already there)
- Pass them as top-level opts fields

**Example transformation for hackathon.go `publishPhaseChange`:**

Before:
```go
_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
    ActUserID:    doerID,
    OpType:       hackforger_model.ActionHackathonPhaseChanged,
    AudienceType: AudienceOrgMembers,
    OrgID:        h.OrgID,
    Content: hackforger_model.HackforgerPhaseContent{...},
})
```

After:
```go
_ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
    ActUserID:    doerID,
    OpType:       hackforger_model.ActionHackathonPhaseChanged,
    EntityType:   "hackathon",
    EntityID:     h.ID,
    EntityName:   h.Name,
    EntitySlug:   h.Slug,
    AudienceType: AudienceOrgMembers,
    OrgID:        h.OrgID,
    Content: hackforger_model.HackforgerPhaseContent{...},
})
```

Apply this pattern to ALL callers. Search for `PublishHackforgerAction` in each file and add the entity fields.

**IMPORTANT — Audit `AudienceType: 0` usages:** After the bit-flag change, `AudienceType: 0` means "no audience at all" (not even actor record from the global check). Search for any caller that uses `AudienceType: 0` or omits `AudienceType` (Go zero value). If the intent was "actor only, no audience distribution", change to explicitly using `AudienceType: AudienceFollowers` (actor's followers) or remove the field and ensure the actor record is still created in `PublishHackforgerAction`.

**For grants.go `publishGrantEvent` helper:** Add entity fields as parameters or extract from `round` parameter:
```go
func publishGrantEvent(ctx context.Context, actUserID int64, opType activities_model.ActionType, round *hackforger_model.GrantRound, audience AudienceType) error {
    // ... existing content creation ...
    return PublishHackforgerAction(ctx, &HackforgerActionOpts{
        ActUserID:    actUserID,
        OpType:       opType,
        EntityType:   "grant",
        EntityID:     round.ID,
        EntityName:   round.Name,
        EntitySlug:   round.Slug,
        Content:      content,
        AudienceType: audience,
        OrgID:        round.OrgID,
    })
}
```

- [ ] **Step 2: Verify full compilation**

Run: `go build ./services/hackforger/...`
Expected: PASS — all callers now supply the required fields.

- [ ] **Step 3: Run existing service tests to verify no regressions**

Run: `go test ./services/hackforger/... -v`

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/bounty.go services/hackforger/hackathon.go services/hackforger/grants.go services/hackforger/credits.go
git commit -m "refactor(feed): update all notifier callers with entity fields for hackforger_action"
```

---

### Task 8: Feed API Update

**Files:**
- Modify: `routers/api/v1/hackforger/feed.go`

- [ ] **Step 1: Rewrite GetFeed to query hackforger_action**

Replace the entire `GetFeed` function body to use `hackforger_model.GetHackforgerFeeds` instead of directly querying the `action` table. Add `entity_type`, `entity_id`, `user_id` query params.

Key changes:
- Remove direct `action` table query
- Use `GetHackforgerFeeds` / `GetEntityTimeline`
- Keep the same `feedItem` JSON response format
- Use `LoadActUsers` for user data

- [ ] **Step 2: Verify compilation**

Run: `go build ./routers/api/v1/hackforger/...`

- [ ] **Step 3: Commit**

```bash
git add routers/api/v1/hackforger/feed.go
git commit -m "refactor(api): update Feed API to query hackforger_action table"
```

---

### Task 9: Template Cleanup — Remove HackForger branches from feeds.tmpl

**Files:**
- Modify: `templates/user/dashboard/feeds.tmpl`

- [ ] **Step 1: Remove all 22 HackForger `.InActions` branches**

Remove the entire HackForger block — from `{{else if .GetOpType.InActions "hackforger_bounty_created"}}` through the last `{{else if .GetOpType.InActions "hackforger_credits_redeemed"}}` and its closing content. Search for the marker `hackforger_bounty_created` to find the start, and remove all consecutive `hackforger_*` branches. These events will now be rendered by the Community tab's `community_feeds.tmpl`.

Keep all Forgejo-native action rendering (commits, PRs, issues, etc.) intact.

- [ ] **Step 2: Verify template renders (build with bindata tag)**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`

- [ ] **Step 3: Commit**

```bash
git add templates/user/dashboard/feeds.tmpl
git commit -m "cleanup(feed): remove HackForger event branches from Forgejo feeds template"
```

---

## Chunk 3: Reputation System

### Task 10: Reputation Service

**Files:**
- Create: `services/hackforger/reputation.go`
- Create: `services/hackforger/reputation_test.go`

- [ ] **Step 1: Create reputation service**

```go
// services/hackforger/reputation.go
package hackforger

import (
	"context"
	"math"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
)

// Default weights and tiers (used when settings table has no values).
// Exported via DefaultWeightsJSON() / DefaultTiersJSON() for admin handler.
const (
	defaultWeightsJSON = `{"stars":1,"bounties_completed":5,"hackathon_wins":10,"grants_received":3,"credits_earned":0.1}`
	defaultTiersJSON   = `[{"name":"Bronze","min":0},{"name":"Silver","min":50},{"name":"Gold","min":200},{"name":"Diamond","min":500}]`
)

func DefaultWeightsJSON() string { return defaultWeightsJSON }
func DefaultTiersJSON() string   { return defaultTiersJSON }

type ReputationWeights struct {
	Stars             float64 `json:"stars"`
	BountiesCompleted float64 `json:"bounties_completed"`
	HackathonWins     float64 `json:"hackathon_wins"`
	GrantsReceived    float64 `json:"grants_received"`
	CreditsEarned     float64 `json:"credits_earned"`
}

type ReputationTier struct {
	Name string  `json:"name"`
	Min  float64 `json:"min"`
}

func GetReputationWeights(ctx context.Context) (*ReputationWeights, error) {
	raw := hackforger_model.GetSettingWithDefault(ctx, "reputation.weights", defaultWeightsJSON)
	var w ReputationWeights
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		return nil, err
	}
	return &w, nil
}

func GetReputationTiers(ctx context.Context) ([]ReputationTier, error) {
	raw := hackforger_model.GetSettingWithDefault(ctx, "reputation.tiers", defaultTiersJSON)
	var tiers []ReputationTier
	if err := json.Unmarshal([]byte(raw), &tiers); err != nil {
		return nil, err
	}
	return tiers, nil
}

func deriveTier(score int64, tiers []ReputationTier) string {
	tier := "Bronze"
	for _, t := range tiers {
		if float64(score) >= t.Min {
			tier = t.Name
		}
	}
	return tier
}

// RecalculateReputation recomputes a single user's reputation.
func RecalculateReputation(ctx context.Context, userID int64) error {
	rep, err := hackforger_model.GetOrCreateReputation(ctx, userID)
	if err != nil {
		return err
	}

	e := db.GetEngine(ctx)

	// Count metrics (use model constants, not magic integers)
	bountiesCompleted, _ := e.Table("bounty").
		Where("claimer_id = ? AND status IN (?, ?)", userID,
			hackforger_model.BountyStatusCompleted, hackforger_model.BountyStatusPaid).Count()
	hackathonWins, _ := e.Table("hackathon_submission").
		Where("user_id = ? AND rank = 1", userID).Count()
	grantsReceived, _ := e.Table("grant_project").
		Where("user_id = ? AND status IN (?, ?)", userID,
			hackforger_model.GrantProjectStatusApproved, hackforger_model.GrantProjectStatusFunded).Count()

	var totalStars int64
	_, _ = e.SQL("SELECT COALESCE(SUM(num_stars), 0) FROM repository WHERE owner_id = ?", userID).Get(&totalStars)

	var totalCreditsEarned int64
	_, _ = e.SQL("SELECT COALESCE(SUM(amount), 0) FROM credit_transaction WHERE user_id = ? AND type IN ('deposit', 'reward')", userID).Get(&totalCreditsEarned)

	// Load weights and compute score
	weights, err := GetReputationWeights(ctx)
	if err != nil {
		return err
	}

	scoreF := float64(totalStars)*weights.Stars +
		float64(bountiesCompleted)*weights.BountiesCompleted +
		float64(hackathonWins)*weights.HackathonWins +
		float64(grantsReceived)*weights.GrantsReceived +
		float64(totalCreditsEarned)*weights.CreditsEarned
	score := int64(math.Round(scoreF))

	// Derive tier
	tiers, err := GetReputationTiers(ctx)
	if err != nil {
		return err
	}

	rep.Score = score
	rep.BountiesCompleted = int(bountiesCompleted)
	rep.HackathonWins = int(hackathonWins)
	rep.GrantsReceived = int(grantsReceived)
	rep.TotalStars = totalStars
	rep.TotalCreditsEarned = totalCreditsEarned
	rep.Tier = deriveTier(score, tiers)

	return hackforger_model.UpdateReputation(ctx, rep)
}

// RecalculateAllReputations recalculates all users with reputation records.
func RecalculateAllReputations(ctx context.Context) error {
	var reps []*hackforger_model.Reputation
	if err := db.GetEngine(ctx).Find(&reps); err != nil {
		return err
	}
	for _, rep := range reps {
		if err := RecalculateReputation(ctx, rep.UserID); err != nil {
			log.Error("RecalculateReputation(user=%d): %v", rep.UserID, err)
		}
	}
	return nil
}
```

- [ ] **Step 2: Write tests**

```go
// services/hackforger/reputation_test.go
package hackforger

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveTier(t *testing.T) {
	tiers := []ReputationTier{
		{Name: "Bronze", Min: 0},
		{Name: "Silver", Min: 50},
		{Name: "Gold", Min: 200},
		{Name: "Diamond", Min: 500},
	}
	assert.Equal(t, "Bronze", deriveTier(0, tiers))
	assert.Equal(t, "Bronze", deriveTier(49, tiers))
	assert.Equal(t, "Silver", deriveTier(50, tiers))
	assert.Equal(t, "Gold", deriveTier(200, tiers))
	assert.Equal(t, "Diamond", deriveTier(500, tiers))
	assert.Equal(t, "Diamond", deriveTier(9999, tiers))
}

func TestRecalculateReputation(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User 1 exists in test fixtures.
	err := RecalculateReputation(db.DefaultContext, 1)
	require.NoError(t, err)

	rep, err := hackforger_model.GetOrCreateReputation(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.True(t, rep.Score >= 0, "score should be non-negative")
	assert.NotEmpty(t, rep.Tier)
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./services/hackforger/... -run TestDeriveTier -v && go test ./services/hackforger/... -run TestRecalculateReputation -v`

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/reputation.go services/hackforger/reputation_test.go
git commit -m "feat(reputation): add reputation calculation service with configurable weights"
```

---

### Task 11: Cron Task

**Files:**
- Modify: `services/cron/tasks_hackforger.go` (file already exists with 4 task stubs)

- [ ] **Step 1: Fill in the existing `registerHackforgerReputationRecalc` stub**

The file `services/cron/tasks_hackforger.go` already has a skeleton for `registerHackforgerReputationRecalc`. Update only this function's body to call the actual service:

```go
func registerHackforgerReputationRecalc() {
	RegisterTaskFatal("hackforger_reputation_recalc", &BaseConfig{
		Enabled:    true,
		RunAtStart: false,
		Schedule:   "@every 1h",
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		return hackforger_service.RecalculateAllReputations(ctx)
	})
}
```

Add the import for `hackforger_service "forgejo.org/services/hackforger"` if not present. Do NOT modify the other 3 task registrations (`hackathonStatus`, `bountyExpiry`, `grantDeadline`).

- [ ] **Step 2: Verify compilation**

Run: `go build ./services/cron/...`

- [ ] **Step 3: Commit**

```bash
git add services/cron/tasks_hackforger.go
git commit -m "feat(cron): implement reputation recalculation cron task (hourly)"
```

---

### Task 12: Reputation API Endpoints

**Files:**
- Create: `routers/api/v1/hackforger/reputation.go`
- Modify: `routers/api/v1/api.go` (add route registration)

- [ ] **Step 1: Create API handler file**

```go
// routers/api/v1/hackforger/reputation.go
package hackforger

import (
	"net/http"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// GetUserReputation returns reputation data for a user.
func GetUserReputation(ctx *context.APIContext) {
	username := ctx.PathParam("username")
	u, err := user_model.GetUserByName(ctx, username)
	if err != nil {
		ctx.NotFound("GetUserByName", err)
		return
	}
	rep, err := hackforger_model.GetOrCreateReputation(ctx, u.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetOrCreateReputation", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]any{
		"user_id":            u.ID,
		"username":           u.Name,
		"score":              rep.Score,
		"tier":               rep.Tier,
		"bounties_completed": rep.BountiesCompleted,
		"hackathon_wins":     rep.HackathonWins,
		"grants_received":    rep.GrantsReceived,
		"total_stars":        rep.TotalStars,
		"credits_earned":     rep.TotalCreditsEarned,
	})
}

// GetReputationLeaderboard returns top users by reputation score.
func GetReputationLeaderboard(ctx *context.APIContext) {
	limit := ctx.FormInt("limit")
	if limit < 1 || limit > 100 {
		limit = 50
	}
	records, err := hackforger_model.ReputationLeaderboard(ctx, limit)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ReputationLeaderboard", err)
		return
	}

	// Batch-load usernames (avoid N+1 queries)
	userIDs := make([]int64, 0, len(records))
	for _, r := range records {
		userIDs = append(userIDs, r.UserID)
	}
	userMap := make(map[int64]*user_model.User)
	if len(userIDs) > 0 {
		var users []*user_model.User
		if err := db.GetEngine(ctx).In("id", userIDs).Find(&users); err == nil {
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
	}

	items := make([]map[string]any, 0, len(records))
	for _, r := range records {
		username := ""
		if u, ok := userMap[r.UserID]; ok {
			username = u.Name
		}
		items = append(items, map[string]any{
			"user_id":  r.UserID,
			"username": username,
			"score":    r.Score,
			"tier":     r.Tier,
		})
	}
	ctx.JSON(http.StatusOK, items)
}

// AdminRecalculateReputation triggers recalculation for a single user.
func AdminRecalculateReputation(ctx *context.APIContext) {
	username := ctx.PathParam("username")
	u, err := user_model.GetUserByName(ctx, username)
	if err != nil {
		ctx.NotFound("GetUserByName", err)
		return
	}
	if err := hackforger_service.RecalculateReputation(ctx, u.ID); err != nil {
		ctx.Error(http.StatusInternalServerError, "RecalculateReputation", err)
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
```

- [ ] **Step 2: Register routes in api.go**

In `routers/api/v1/api.go`, inside the `/hackforger` group (around line 1800), add:

```go
// Reputation routes
m.Group("/reputation", func() {
    m.Get("/users/{username}", hackforger_api.GetUserReputation)
    m.Get("/leaderboard", hackforger_api.GetReputationLeaderboard)
    m.Post("/recalculate/{username}", reqToken(), reqSiteAdmin(), hackforger_api.AdminRecalculateReputation)
})
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./routers/api/v1/...`

- [ ] **Step 4: Commit**

```bash
git add routers/api/v1/hackforger/reputation.go routers/api/v1/api.go
git commit -m "feat(api): add reputation API endpoints (user, leaderboard, admin recalc)"
```

---

### Task 13: Admin Web Routes + Template

**Files:**
- Create: `routers/web/hackforger/reputation.go`
- Create: `templates/admin/hackforger/reputation.tmpl`
- Modify: `routers/web/web.go` (add admin routes)

- [ ] **Step 1: Create admin web handler**

```go
// routers/web/hackforger/reputation.go
package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/services/context"
	hackforger_service "forgejo.org/services/hackforger"
)

// AdminReputation renders the reputation settings page.
func AdminReputation(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("hackforger.admin.reputation")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminHackforgerReputation"] = true

	weights := hackforger_model.GetSettingWithDefault(ctx, "reputation.weights", hackforger_service.DefaultWeightsJSON())
	tiers := hackforger_model.GetSettingWithDefault(ctx, "reputation.tiers", hackforger_service.DefaultTiersJSON())
	ctx.Data["Weights"] = weights
	ctx.Data["Tiers"] = tiers

	ctx.HTML(http.StatusOK, "admin/hackforger/reputation")
}

// AdminReputationPost saves reputation settings.
func AdminReputationPost(ctx *context.Context) {
	weights := ctx.FormString("weights")
	tiers := ctx.FormString("tiers")

	if weights != "" {
		if err := hackforger_model.SetSetting(ctx, "reputation.weights", weights); err != nil {
			ctx.ServerError("SetSetting weights", err)
			return
		}
	}
	if tiers != "" {
		if err := hackforger_model.SetSetting(ctx, "reputation.tiers", tiers); err != nil {
			ctx.ServerError("SetSetting tiers", err)
			return
		}
	}

	ctx.Flash.Success(ctx.Tr("hackforger.admin.reputation.save_success"))
	ctx.Redirect(ctx.Req.URL.Path)
}

// AdminReputationRecalc triggers a full recalculation.
func AdminReputationRecalc(ctx *context.Context) {
	// Use graceful context — HTTP request context is canceled after response.
	go hackforger_service.RecalculateAllReputations(graceful.GetManager().HammerContext())
	ctx.Flash.Success(ctx.Tr("hackforger.admin.reputation.recalc_success"))
	ctx.Redirect("/admin/hackforger/reputation")
}
```

**Note:** Export `defaultWeightsJSON` and `defaultTiersJSON` from `services/hackforger/reputation.go` as `DefaultWeightsJSON()` and `DefaultTiersJSON()` functions.

- [ ] **Step 2: Create admin template**

Create `templates/admin/hackforger/reputation.tmpl` — a form page with textarea fields for weights JSON and tiers JSON, plus a "Recalculate All" button. Follow the existing admin page pattern (e.g., `templates/admin/config.tmpl` for layout reference).

- [ ] **Step 3: Register routes in web.go**

Inside the admin group in `web.go` (after the HackForger Admin Credits block around line 923):

```go
// ***** START: HackForger Admin Reputation *****
m.Group("/hackforger/reputation", func() {
    m.Get("", hackforger_web.AdminReputation)
    m.Post("", hackforger_web.AdminReputationPost)
    m.Post("/recalc", hackforger_web.AdminReputationRecalc)
})
// ***** END: HackForger Admin Reputation *****
```

- [ ] **Step 4: Verify compilation + build**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`

- [ ] **Step 5: Commit**

```bash
git add routers/web/hackforger/reputation.go templates/admin/hackforger/reputation.tmpl routers/web/web.go services/hackforger/reputation.go
git commit -m "feat(admin): add reputation settings admin page with weight/tier config"
```

---

## Chunk 4: Frontend Integration

### Task 14: Template Helper Functions

**Files:**
- Modify: `modules/templates/helper.go`

- [ ] **Step 1: Add HackforgerEntityURL and HackforgerActionIcon to NewFuncMap()**

In the `NewFuncMap()` return map, add:

```go
"HackforgerEntityURL": func(entityType, slug string) string {
    switch entityType {
    case "hackathon":
        return setting.AppSubURL + "/hackathon/" + url.PathEscape(slug)
    case "grant":
        return setting.AppSubURL + "/grants/" + url.PathEscape(slug)
    default:
        return ""
    }
},
"HackforgerActionIcon": func(opType int) string {
    switch {
    case opType >= 30 && opType <= 33: return "octicon-rocket"     // hackathon
    case opType >= 34 && opType <= 38: return "octicon-gift"       // bounty
    case opType == 43:                 return "octicon-gift"       // bounty_paid
    case opType >= 39 && opType <= 41: return "octicon-heart"      // grant
    case opType == 42:                 return "octicon-credit-card" // credits
    case opType >= 50 && opType <= 51: return "octicon-rocket"     // hackathon lifecycle
    case opType >= 52 && opType <= 53: return "octicon-gift"       // bounty lifecycle
    case opType >= 54 && opType <= 57: return "octicon-heart"      // grant lifecycle
    default:                           return "octicon-pulse"
    }
},
```

Add `"net/url"` to imports if not already present.

- [ ] **Step 2: Verify compilation**

Run: `go build ./modules/templates/...`

- [ ] **Step 3: Commit**

```bash
git add modules/templates/helper.go
git commit -m "feat(templates): add HackforgerEntityURL and HackforgerActionIcon helpers"
```

---

### Task 15: Community Feeds Template

**Files:**
- Create: `templates/hackforger/feed/community_feeds.tmpl`

- [ ] **Step 1: Create the community feeds template**

```html
<div id="community-feed" class="flex-list">
{{range .HackforgerFeeds}}
	<div class="flex-item">
		<div class="flex-item-leading">
			{{ctx.AvatarUtils.Avatar .ActUser 32}}
		</div>
		<div class="flex-item-main tw-gap-2">
			<div>
				{{if .ActUser}}
					<a href="{{AppSubUrl}}/{{(.ActUser.Name) | PathEscape}}" title="{{.ActUser.GetDisplayName}}">{{.ActUser.GetDisplayName}}</a>
				{{end}}
				{{if eq .OpType 30}}{{ctx.Locale.Tr "hackforger.feed.hackathon_created" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 31}}{{ctx.Locale.Tr "hackforger.feed.hackathon_registered" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 32}}{{ctx.Locale.Tr "hackforger.feed.hackathon_submitted" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 33}}{{ctx.Locale.Tr "hackforger.feed.hackathon_scored" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 34}}{{ctx.Locale.Tr "hackforger.feed.bounty_created" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 35}}{{ctx.Locale.Tr "hackforger.feed.bounty_claimed" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 36}}{{ctx.Locale.Tr "hackforger.feed.bounty_delivered" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 37}}{{ctx.Locale.Tr "hackforger.feed.bounty_completed" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 38}}{{ctx.Locale.Tr "hackforger.feed.bounty_winners_selected" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 39}}{{ctx.Locale.Tr "hackforger.feed.grant_round_created" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 40}}{{ctx.Locale.Tr "hackforger.feed.grant_project_submitted" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 41}}{{ctx.Locale.Tr "hackforger.feed.grant_awarded" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 42}}{{ctx.Locale.Tr "hackforger.feed.credits_redeemed"}}
				{{else if eq .OpType 43}}{{ctx.Locale.Tr "hackforger.feed.bounty_paid" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 50}}{{ctx.Locale.Tr "hackforger.feed.hackathon_phase_changed" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 51}}{{ctx.Locale.Tr "hackforger.feed.hackathon_finalized" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 52}}{{ctx.Locale.Tr "hackforger.feed.bounty_expired" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 53}}{{ctx.Locale.Tr "hackforger.feed.bounty_cancelled" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 54}}{{ctx.Locale.Tr "hackforger.feed.grant_round_opened" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 55}}{{ctx.Locale.Tr "hackforger.feed.grant_round_closed" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 56}}{{ctx.Locale.Tr "hackforger.feed.grant_round_finalized" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{else if eq .OpType 57}}{{ctx.Locale.Tr "hackforger.feed.grant_round_cancelled" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
				{{end}}
				{{DateUtils.TimeSince .CreatedUnix}}
			</div>
		</div>
		<div class="flex-item-trailing">
			{{svg (HackforgerActionIcon .OpType) 32 "text grey tw-mr-1"}}
		</div>
	</div>
{{end}}
{{template "base/paginate" .}}
</div>
```

- [ ] **Step 2: Build to verify template is valid**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`

- [ ] **Step 3: Commit**

```bash
git add templates/hackforger/feed/community_feeds.tmpl
git commit -m "feat(feed): add community feeds template for HackForger events"
```

---

### Task 16: Dashboard Community Tab

**Files:**
- Modify: `templates/user/dashboard/dashboard.tmpl`
- Modify: `routers/web/user/home.go`

- [ ] **Step 1: Add tab bar to dashboard.tmpl**

Replace the feed section (lines 7-16) with the tab bar + conditional content from spec section 4.1.

- [ ] **Step 2: Add community feed query to Dashboard() in home.go**

Add `?feed=` param handling and the community query branch from spec section 4.2. Import `hackforger_model`.

- [ ] **Step 3: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`

- [ ] **Step 4: Commit**

```bash
git add templates/user/dashboard/dashboard.tmpl routers/web/user/home.go
git commit -m "feat(dashboard): add Community tab with HackForger feed"
```

---

### Task 17: Reputation Templates (Card + Detail + Leaderboard)

**Files:**
- Create: `templates/hackforger/reputation/card.tmpl`
- Create: `templates/hackforger/reputation/detail.tmpl`
- Create: `templates/hackforger/reputation/leaderboard.tmpl`

- [ ] **Step 1: Create reputation sidebar card**

`templates/hackforger/reputation/card.tmpl` — compact card showing tier badge, score, top metrics. Follow Forgejo's card styling patterns.

- [ ] **Step 2: Create reputation detail page**

`templates/hackforger/reputation/detail.tmpl` — full metrics table, rank, link to leaderboard.

- [ ] **Step 3: Create leaderboard page**

`templates/hackforger/reputation/leaderboard.tmpl` — table of top users with score/tier. Include explore page wrapper (base/head, base/footer).

- [ ] **Step 4: Build**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`

- [ ] **Step 5: Commit**

```bash
git add templates/hackforger/reputation/card.tmpl templates/hackforger/reputation/detail.tmpl templates/hackforger/reputation/leaderboard.tmpl
git commit -m "feat(reputation): add card, detail, and leaderboard templates"
```

---

### Task 18: Profile Integration

**Files:**
- Modify: `templates/user/overview/header.tmpl` (add 2 tab links)
- Modify: `templates/user/profile.tmpl` (add tab content)
- Modify: `routers/web/user/profile.go` (add tab data loading)
- Modify: `routers/web/web.go` (add leaderboard route)

- [ ] **Step 1: Add Community + Reputation tabs to profile header**

In `templates/user/overview/header.tmpl`, after the "activity" tab link (around line 43), add the two new tab links from spec section 5.2.

- [ ] **Step 2: Add tab content to profile.tmpl**

In `templates/user/profile.tmpl`, add `community` and `reputation` tab content branches before the `{{else}}` default case (around line 61). See spec section 5.3.

- [ ] **Step 3: Add reputation sidebar card injection**

In `templates/user/profile.tmpl`, add `{{if .Reputation}}{{template "hackforger/reputation/card" .}}{{end}}` after `{{template "shared/user/profile_big_avatar" .}}` (line 8).

- [ ] **Step 4: Add tab data loading in profile.go**

In `routers/web/user/profile.go` `prepareUserProfileTabData()`:
- Before the `switch tab` block, load reputation unconditionally
- Add `case "community"` and `case "reputation"` branches
- See spec section 5.3 for exact code

- [ ] **Step 5: Add leaderboard route in web.go**

Inside the explore group (around line 520), add:
```go
m.Get("/reputation", hackforger_web.ExploreReputation)
```

Add `ExploreReputation` handler in `routers/web/hackforger/reputation.go` that loads leaderboard data and renders the template.

- [ ] **Step 6: Build and verify**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`

- [ ] **Step 7: Commit**

```bash
git add templates/user/overview/header.tmpl templates/user/profile.tmpl routers/web/user/profile.go routers/web/web.go routers/web/hackforger/reputation.go
git commit -m "feat(profile): add Community tab, Reputation tab, sidebar card, and leaderboard"
```

---

## Chunk 5: E2E Test Prompt

### Task 19: E2E Test Prompt Document

**Files:**
- Create: `docs/tests/e2e/phase2-reputation-feed-profile.md`

- [ ] **Step 1: Write E2E test prompt with test scenarios**

Cover:
1. Dashboard: Code tab shows only Git events; Community tab shows HackForger events
2. Profile: Activity tab (Git only), Community tab (HackForger), Reputation tab (score + tier)
3. Reputation sidebar card visible on all profile tabs
4. Admin: Reputation settings page — save weights, trigger recalc
5. API: `/api/v1/hackforger/feed?type=global`, `/api/v1/hackforger/reputation/users/{name}`, `/api/v1/hackforger/reputation/leaderboard`
6. Feed events: Create a hackathon → verify event appears in Community tab
7. Explore: `/explore/reputation` shows leaderboard

Include expected results and E2E report template (Round format matching existing docs).

- [ ] **Step 2: Commit**

```bash
git add docs/tests/e2e/phase2-reputation-feed-profile.md
git commit -m "docs: add Phase 2 E2E test prompt for reputation + feed + profile"
```

---

## Summary

| Chunk | Tasks | Key Deliverables |
|-------|-------|-----------------|
| 1: Data Layer | 1-5 | Models, migration, i18n |
| 2: Feed Refactor | 6-9 | Notifier rewrite, caller updates, API, template cleanup |
| 3: Reputation | 10-13 | Service, cron, API, admin page |
| 4: Frontend | 14-18 | Templates, dashboard tab, profile tabs, leaderboard |
| 5: E2E | 19 | Test prompt document |

**Total: 19 tasks, ~11 new files, ~11 modified Forgejo files.**
