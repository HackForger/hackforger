# Phase Timeline UX Enhancements Design

**Date:** 2026-04-05
**Status:** Approved
**Scope:** 4 UX improvements to the PhaseTimeline Vue component and supporting backend

---

## 1. Dropdown: Show All Types, Disable Already-Added Unique Types

**Problem:** `availableTypes` computed property completely hides unique phase types that are already added. Users don't know these types exist.

**Design:**
- Replace `availableTypes` with `allTypesWithState` computed that annotates each phaseType with `{ disabled, reason }`
- Unique types already added: `disabled: true`, reason: i18n string "already added"
- Dropdown renders disabled options in gray with parenthetical explanation
- Non-unique types (development) always remain enabled

**Files:** `PhaseTimeline.vue` only

## 2. Phase Rename (custom_name Field)

**Problem:** Phase instances have no custom name. All phases of the same type show identical names (e.g. two "Development" phases are indistinguishable).

**Design:**
- Add `CustomName string` field to Phase struct (`xorm:"VARCHAR(100)" json:"custom_name"`)
- DB migration: `ALTER TABLE phase ADD COLUMN custom_name VARCHAR(100) NOT NULL DEFAULT ''`
- Display logic: `custom_name || display_name` (Vue side)
- UI: pencil icon next to phase name. Click opens inline input, blur/Enter sends PUT to save.
- Backend: extend `ManagePhasesUpdate` request struct with `CustomName` field
- Model: add `custom_name` to `UpdatePhase` Cols list

**Files:** Phase model, migration, phase.go router, PhaseTimeline.vue

## 3. Date Picker Simplification + Timezone Fix

**Problem:** Two issues:
1. `toISOString().slice(0,16)` outputs UTC, but `<input type="datetime-local">` interprets as local time. UTC+8 users see times shifted by -8 hours.
2. Hour/minute granularity is unnecessary. Phases should be date-level.

**Design:**
- Replace `<input type="datetime-local">` with `<input type="date">`
- Remove hour/minute from UI entirely
- Storage convention: start = `YYYY-MM-DDT00:00` local, end = `YYYY-MM-DDT23:59` local
- New helpers:
  - `toDateLocal(unix)` → `YYYY-MM-DD` string using local Date methods
  - `fromDateLocal(str, isEnd)` → unix timestamp. If `isEnd`, appends `T23:59`, else `T00:00`
- Delete `toDatetimeLocal` and `fromDatetimeLocal`

**Files:** `PhaseTimeline.vue` only

## 4. Drag-and-Drop Reorder

**Problem:** No reordering UI. `sort_order` only increments on add.

**Design:**
- Import `vuedraggable` (SortableJS already in Forgejo's npm dependencies)
- Add drag handle (grip icon `⠿`) to each phase row's left side
- Locked phases: handle hidden, not draggable (via `filter` option)
- On drag end: collect new order, POST to `/manage/phases/reorder`
- Backend: new `ManagePhasesReorder` handler accepts `[{id, sort_order}]` array
- Model: new `BatchUpdatePhaseSortOrder` function
- Route: `m.Post("/phases/reorder", ...)` in web.go

**Files:** Phase model, phase.go router, web.go route, PhaseTimeline.vue

---

## Summary

| # | Enhancement | Backend | Frontend | Migration |
|---|------------|---------|----------|-----------|
| 1 | Dropdown disabled states | - | Vue | - |
| 2 | Phase rename | Model + Router | Vue | Yes |
| 3 | Date picker + TZ fix | - | Vue | - |
| 4 | Drag reorder | Model + Router + Route | Vue | - |
