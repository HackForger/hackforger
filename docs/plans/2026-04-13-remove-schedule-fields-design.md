# Remove Hackathon Schedule Fields (Issue #51)

## Problem
Hackathon creation has two overlapping time mechanisms: legacy "日程安排" (Schedule) fields on the `hackathon` table and the Phase system (`hackforger_phase` table). The schedule fields don't control any business logic — status, permissions, and transitions are all driven by Phase records. This confuses users and creates data inconsistency.

## Decision
Remove the schedule fields entirely. Adapt milestone auto-creation to derive deadlines from Phase records instead.

## Changes

### Template
- `hackathon/new.tmpl` — remove schedule date inputs
- `hackathon/manage.tmpl` — remove schedule edit section and preview display

### Router
- `routers/web/hackforger/hackathon.go` — remove schedule field parsing from `NewHackathonPost()` and `UpdateHackathonPost()`

### Service
- `services/hackforger/hackathon.go` — milestone auto-creation reads Phase `EndTime` instead of schedule fields:
  - `PhaseTypeRegistration` EndTime → Registration milestone
  - Last `PhaseTypeDevelopment` EndTime → Development milestone
  - Last `PhaseTypeJudging` EndTime → Judging milestone
  - Skip milestone if corresponding phase doesn't exist

### Model
- `models/hackforger/hackathon.go` — remove 5 timestamp fields (`RegistrationStart`, `RegistrationEnd`, `HackingStart`, `HackingEnd`, `JudgingEnd`)
- New migration — DROP the 5 columns (completing the cleanup deferred in v14b)

### i18n
- Remove schedule-related keys from `locale_en-US.ini` and `locale_zh-CN.ini`
