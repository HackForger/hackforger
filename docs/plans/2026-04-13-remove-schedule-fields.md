# Remove Hackathon Schedule Fields — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the legacy "日程安排" (Schedule) fields from hackathon creation/editing, since all business logic already runs on the Phase system. Milestone auto-creation switches to Phase records.

**Architecture:** The hackathon model has 5 dead timestamp fields (`RegistrationStart`, `RegistrationEnd`, `HackingStart`, `HackingEnd`, `JudgingEnd`) that were migrated to Phase records in v14b. We remove them from templates, router handlers, model struct, and a new DB migration drops the columns. The milestone auto-creation in `CreateTrackWithRepo` switches to querying Phase records instead.

**Tech Stack:** Go (XORM), Go HTML templates, SQLite migration, i18n INI files

---

### Task 1: Remove schedule inputs from hackathon creation template

**Files:**
- Modify: `templates/hackforger/hackathon/new.tmpl:39-63`

**Step 1: Remove the schedule section**

Delete lines 39-63 (the `<h4>Schedule</h4>` heading and all five `datetime-local` inputs). The template should go directly from the `prize_summary` field to the `<h4>Settings</h4>` heading.

Before (lines 38-65):
```html
			</div>

			<h4 class="tw-mt-4 tw-mb-2">{{ctx.Locale.Tr "hackforger.hackathon.field.schedule"}}</h4>
			<div class="two fields">
				...25 lines of schedule inputs...
			</div>

			<h4 class="tw-mt-4 tw-mb-2">{{ctx.Locale.Tr "hackforger.hackathon.field.settings"}}</h4>
```

After:
```html
			</div>

			<h4 class="tw-mt-4 tw-mb-2">{{ctx.Locale.Tr "hackforger.hackathon.field.settings"}}</h4>
```

**Step 2: Verify template renders**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -5`
Expected: compilation succeeds (templates are embedded at build time with bindata tag)

**Step 3: Commit**

```bash
git add templates/hackforger/hackathon/new.tmpl
git commit -m "fix(#51): remove schedule inputs from hackathon creation form"
```

---

### Task 2: Remove schedule section from hackathon manage template

**Files:**
- Modify: `templates/hackforger/hackathon/manage.tmpl:38-46` (preview schedule display)
- Modify: `templates/hackforger/hackathon/manage.tmpl:82-106` (edit schedule inputs)

**Step 1: Remove schedule display from preview mode**

Delete lines 38-46 — the `{{if or .Hackathon.RegistrationStart ...}}` block that shows schedule dates in the preview card. Keep the surrounding `</div>` tags intact.

Before (lines 37-47):
```html
					<div>{{svg "octicon-people" 14}} {{ctx.Locale.Tr "hackforger.hackathon.field.max_team_size"}}: {{.Hackathon.MaxTeamSize}}</div>
				</div>

				{{if or .Hackathon.RegistrationStart .Hackathon.RegistrationEnd .Hackathon.HackingStart .Hackathon.HackingEnd .Hackathon.JudgingEnd}}
				<div class="tw-flex tw-flex-wrap tw-gap-x-6 tw-gap-y-1 tw-text-sm tw-mt-2 tw-text-gray">
					{{if .Hackathon.RegistrationStart}}<div>...</div>{{end}}
					...
				</div>
				{{end}}
			</div>
		</div>
```

After:
```html
					<div>{{svg "octicon-people" 14}} {{ctx.Locale.Tr "hackforger.hackathon.field.max_team_size"}}: {{.Hackathon.MaxTeamSize}}</div>
				</div>
			</div>
		</div>
```

**Step 2: Remove schedule inputs from edit mode**

Delete lines 82-106 — the `<h4>Schedule</h4>` heading and all five `datetime-local` inputs with their pre-filled values. Keep the `prize_summary` textarea above and the `max_team_size` input below.

Before (lines 81-107):
```html
						</div>
						<h4 class="tw-mt-2 tw-mb-2">{{ctx.Locale.Tr "hackforger.hackathon.field.schedule"}}</h4>
						<div class="two fields">
							...24 lines of schedule inputs...
						</div>
						<div class="field">
							<label>{{ctx.Locale.Tr "hackforger.hackathon.field.max_team_size"}}</label>
```

After:
```html
						</div>
						<div class="field">
							<label>{{ctx.Locale.Tr "hackforger.hackathon.field.max_team_size"}}</label>
```

**Step 3: Commit**

```bash
git add templates/hackforger/hackathon/manage.tmpl
git commit -m "fix(#51): remove schedule display and edit from manage page"
```

---

### Task 3: Remove schedule field parsing from router handlers

**Files:**
- Modify: `routers/web/hackforger/hackathon.go:164-168` (NewHackathonPost)
- Modify: `routers/web/hackforger/hackathon.go:577-581` (UpdateHackathonPost)

**Step 1: Remove schedule fields from NewHackathonPost**

In `NewHackathonPost()` (around line 157), remove the 5 schedule field lines from the `Hackathon` struct literal.

Before (lines 157-169):
```go
	h := &hackforger_model.Hackathon{
		OwnerID:           ctx.Doer.ID,
		Name:              ctx.FormString("name"),
		Slug:              ctx.FormString("slug"),
		Description:       ctx.FormString("description"),
		PrizeSummary:      ctx.FormString("prize_summary"),
		MaxTeamSize:       maxTeamSize,
		RegistrationStart: parseDatetimeLocal(ctx.FormString("registration_start")),
		RegistrationEnd:   parseDatetimeLocal(ctx.FormString("registration_end")),
		HackingStart:      parseDatetimeLocal(ctx.FormString("hacking_start")),
		HackingEnd:        parseDatetimeLocal(ctx.FormString("hacking_end")),
		JudgingEnd:        parseDatetimeLocal(ctx.FormString("judging_end")),
	}
```

After:
```go
	h := &hackforger_model.Hackathon{
		OwnerID:      ctx.Doer.ID,
		Name:         ctx.FormString("name"),
		Slug:         ctx.FormString("slug"),
		Description:  ctx.FormString("description"),
		PrizeSummary: ctx.FormString("prize_summary"),
		MaxTeamSize:  maxTeamSize,
	}
```

**Step 2: Remove schedule fields from UpdateHackathonPost**

In `UpdateHackathonPost()` (around line 577), remove the 5 schedule field assignment lines.

Before (lines 576-581):
```go
	h.PrizeSummary = ctx.FormString("prize_summary")
	h.RegistrationStart = parseDatetimeLocal(ctx.FormString("registration_start"))
	h.RegistrationEnd = parseDatetimeLocal(ctx.FormString("registration_end"))
	h.HackingStart = parseDatetimeLocal(ctx.FormString("hacking_start"))
	h.HackingEnd = parseDatetimeLocal(ctx.FormString("hacking_end"))
	h.JudgingEnd = parseDatetimeLocal(ctx.FormString("judging_end"))
```

After:
```go
	h.PrizeSummary = ctx.FormString("prize_summary")
```

**Step 3: Check if parseDatetimeLocal is still used elsewhere**

Run: `grep -n "parseDatetimeLocal" routers/web/hackforger/hackathon.go`

If it's only used for these schedule fields (and has no other callers), leave it for now — it may be used by other routers. If only referenced in this file and only for these deleted lines, remove the function too.

**Step 4: Verify compilation**

Run: `go build -tags "bindata sqlite sqlite_unlock_notify" ./routers/web/hackforger/`
Expected: compiles without errors

**Step 5: Commit**

```bash
git add routers/web/hackforger/hackathon.go
git commit -m "fix(#51): remove schedule field parsing from hackathon handlers"
```

---

### Task 4: Remove schedule fields from Hackathon model struct

**Files:**
- Modify: `models/hackforger/hackathon.go:52-56`

**Step 1: Remove the 5 timestamp fields**

In the `Hackathon` struct, delete lines 52-56.

Before (lines 50-57):
```go
	IsPublished       bool               `xorm:"NOT NULL DEFAULT false"`
	MaxTeamSize       int                `xorm:"NOT NULL DEFAULT 5"`
	RegistrationStart timeutil.TimeStamp `xorm:""`
	RegistrationEnd   timeutil.TimeStamp `xorm:""`
	HackingStart      timeutil.TimeStamp `xorm:""`
	HackingEnd        timeutil.TimeStamp `xorm:""`
	JudgingEnd        timeutil.TimeStamp `xorm:""`
	PrizeSummary      string             `xorm:"TEXT"`
```

After:
```go
	IsPublished  bool   `xorm:"NOT NULL DEFAULT false"`
	MaxTeamSize  int    `xorm:"NOT NULL DEFAULT 5"`
	PrizeSummary string `xorm:"TEXT"`
```

**Step 2: Check if `timeutil` import is still needed**

If `CreatedUnix` and `UpdatedUnix` still use `timeutil.TimeStamp`, the import stays. (They do — leave the import.)

**Step 3: Verify compilation**

Run: `go build -tags "bindata sqlite sqlite_unlock_notify" ./models/hackforger/`
Expected: compiles without errors

**Step 4: Commit**

```bash
git add models/hackforger/hackathon.go
git commit -m "fix(#51): remove legacy schedule timestamp fields from Hackathon model"
```

---

### Task 5: Update milestone auto-creation to use Phase records

**Files:**
- Modify: `services/hackforger/hackathon.go:271-287`

**Step 1: Replace milestone creation logic**

Replace the hardcoded milestone array (lines 271-287) with Phase-based lookups. The function already has `ctx` and `h` in scope.

Before (lines 271-287):
```go
	// Auto-create phase milestones on the track repo
	milestones := []struct {
		Name     string
		Deadline timeutil.TimeStamp
	}{
		{"Registration", h.RegistrationEnd},
		{"Hacking", h.HackingEnd},
		{"Judging", h.JudgingEnd},
		{"Results", 0}, // closed when finalized
	}
	for _, ms := range milestones {
		_ = issues_model.NewMilestone(ctx, &issues_model.Milestone{
			RepoID:       repo.ID,
			Name:         ms.Name,
			DeadlineUnix: ms.Deadline,
		})
	}
```

After:
```go
	// Auto-create phase milestones on the track repo from Phase records.
	// Each phase becomes a milestone; deadline = phase EndTime.
	phases, phaseErr := hackforger_model.GetPhasesByActivity(ctx, "hackathon", h.ID)
	if phaseErr == nil {
		for _, p := range phases {
			name := p.CustomName
			if name == "" && p.PhaseType != nil {
				name = p.PhaseType.Key // "registration", "development", "judging", "results"
			}
			var deadline timeutil.TimeStamp
			if p.EndTime > 0 {
				deadline = timeutil.TimeStamp(p.EndTime)
			}
			_ = issues_model.NewMilestone(ctx, &issues_model.Milestone{
				RepoID:       repo.ID,
				Name:         name,
				DeadlineUnix: deadline,
			})
		}
	}
```

**Step 2: Check that `hackforger_model` is already imported**

Run: `grep 'hackforger_model' services/hackforger/hackathon.go | head -3`
Expected: import alias already present (it's used elsewhere in the file)

**Step 3: Remove unused `timeutil` import if no longer referenced**

Check: `grep 'timeutil\.' services/hackforger/hackathon.go`
If the only usage was in the milestone struct literal and `timeutil.TimeStamp` is still used in the replacement code, the import stays.

**Step 4: Verify compilation**

Run: `go build -tags "bindata sqlite sqlite_unlock_notify" ./services/hackforger/`
Expected: compiles without errors

**Step 5: Commit**

```bash
git add services/hackforger/hackathon.go
git commit -m "fix(#51): derive milestone deadlines from Phase records instead of schedule fields"
```

---

### Task 6: Remove schedule-related i18n keys

**Files:**
- Modify: `options/locale/locale_en-US.ini:4075-4080`
- Modify: `options/locale/locale_zh-CN.ini:4083-4088`

**Step 1: Remove 6 keys from en-US locale**

Delete these lines from `locale_en-US.ini` (around line 4075):
```ini
hackathon.field.schedule = Schedule
hackathon.field.registration_start = Registration Opens
hackathon.field.registration_end = Registration Closes
hackathon.field.hacking_start = Hacking Starts
hackathon.field.hacking_end = Hacking Ends
hackathon.field.judging_end = Judging Ends
```

**Step 2: Remove 6 keys from zh-CN locale**

Delete these lines from `locale_zh-CN.ini` (around line 4083):
```ini
hackathon.field.schedule = 日程安排
hackathon.field.registration_start = 报名开始
hackathon.field.registration_end = 报名截止
hackathon.field.hacking_start = 开发开始
hackathon.field.hacking_end = 开发截止
hackathon.field.judging_end = 评审截止
```

**Step 3: Commit**

```bash
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "fix(#51): remove unused schedule i18n keys from both locales"
```

---

### Task 7: Add database migration to drop schedule columns

**Files:**
- Create: `models/forgejo_migrations/v14i_drop-hackathon-schedule-columns.go`

**Step 1: Create migration file**

Use the naming convention from existing files (v14i is the next available letter after v14h).

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Drop legacy hackathon schedule columns (migrated to phase table in v14b)",
		Upgrade: func(x *xorm.Engine) error {
			columns := []string{
				"registration_start",
				"registration_end",
				"hacking_start",
				"hacking_end",
				"judging_end",
			}
			for _, col := range columns {
				if _, err := x.Exec("ALTER TABLE hackathon DROP COLUMN " + col); err != nil {
					return err
				}
			}
			return nil
		},
	})
}
```

**Step 2: Verify compilation**

Run: `go build -tags "bindata sqlite sqlite_unlock_notify" ./models/forgejo_migrations/`
Expected: compiles without errors

**Step 3: Commit**

```bash
git add models/forgejo_migrations/v14i_drop-hackathon-schedule-columns.go
git commit -m "fix(#51): add migration to drop legacy schedule columns from hackathon table"
```

---

### Task 8: Update test fixtures

**Files:**
- Modify: `models/fixtures/hackathon.yml`

**Step 1: Remove schedule fields from fixture data**

Remove all references to `registration_start`, `registration_end`, `hacking_start`, `hacking_end`, `judging_end` from the YAML fixture. These fields:
- `id: 1` has `judging_end: 1735689600`
- `id: 3` has `registration_start: 1672578000`, `registration_end: 1735689600`
- `id: 4` has `hacking_start: 1672578000`, `hacking_end: 1735689600`

Also rename `status` to `status_cache` in fixtures (the column was renamed in v14b migration).

**Step 2: Run model tests**

Run: `go test -tags "bindata sqlite sqlite_unlock_notify" ./models/hackforger/... -v -count=1 2>&1 | tail -20`
Expected: all tests pass

**Step 3: Run service tests**

Run: `go test -tags "bindata sqlite sqlite_unlock_notify" ./services/hackforger/... -v -count=1 2>&1 | tail -20`
Expected: all tests pass

**Step 4: Commit**

```bash
git add models/fixtures/hackathon.yml
git commit -m "fix(#51): update hackathon fixtures to remove schedule fields"
```

---

### Task 9: Full build verification

**Step 1: Build frontend (templates changed)**

Run: `make frontend 2>&1 | tail -5`
Expected: build succeeds

**Step 2: Build backend**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -5`
Expected: build succeeds

**Step 3: Run all hackforger tests**

Run: `go test -tags "bindata sqlite sqlite_unlock_notify" ./models/hackforger/... ./services/hackforger/... ./routers/web/hackforger/... -count=1 2>&1 | tail -20`
Expected: all tests pass

**Step 4: Commit (if any fixups needed)**

Only if previous tasks introduced issues caught here.

---

### Task 10: Manual E2E verification

Test on local instance (`http://localhost:3000`) using agent-browser:

1. **Create hackathon** — verify the schedule section is gone from the creation form
2. **Manage hackathon** — verify no schedule display in preview, no schedule inputs in edit mode
3. **Add phases** — verify Phase Timeline still works correctly
4. **Create a track** — verify milestones are created from Phase data (check the track repo's milestones)
5. **Screenshot** each step and save to `tests/screenshots/`

Write E2E report to `docs/tests/e2e/reports/issue-51-remove-schedule-report.md`.
