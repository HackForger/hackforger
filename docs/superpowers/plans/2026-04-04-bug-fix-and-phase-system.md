# HackForger Bug Fix & Phase System — Implementation Plan Index

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 27 active tickets across Hackathon, Bounty, Grant, Credits, Feed, and UI modules, culminating in a Phase System that introduces event lifecycle management.

**Architecture:** Three waves of independent PRs. Wave 1 fixes bugs using existing Forgejo mechanisms. Wave 2 adds medium features with some upstream injection points. Wave 3 introduces the Phase data model, controller, and UI component.

**Tech Stack:** Go (XORM, Forgejo services/context), Go HTML templates, JavaScript (ES modules, Fomantic UI), Vue 3 (SFC), CSS

**Spec:** `docs/superpowers/specs/2026-04-04-bug-fix-and-phase-system-design.md`

---

## Prerequisites

Before starting any wave, the implementer must locate the HackForger source files. The internal instance (`hackforger.inside.h2os.cloud`) runs compiled code. Source files may be:
- In bindata (compiled templates/JS)
- On a different branch
- Local/uncommitted on the deployment server

If source files don't exist on this branch, create the directory structure:
```
models/hackforger/
services/hackforger/
routers/web/hackforger/
routers/api/v1/hackforger/
templates/hackforger/
modules/hackforger/
web_src/js/features/hackforger/
```

## Wave Plans

Each wave is a separate plan file. Tasks within each wave are grouped by concurrency — tasks in the same group can run as parallel PRs.

- [Wave 1: Quick Fixes](2026-04-04-wave1-quick-fixes-plan.md) — 10 tasks, 3 concurrency groups
- [Wave 2: Medium Features](2026-04-04-wave2-medium-features-plan.md) — 14 tasks, 4 concurrency groups
- [Wave 3: Phase System](2026-04-04-wave3-phase-system-plan.md) — 8 tasks, 3 concurrency groups (sequential dependency chain)

## PR Strategy

- Each task = 1 independent PR targeting `v0.1-dev/hackforger`
- PRs within the same concurrency group can be created and merged in parallel
- Auto-merge enabled for all PRs
- PR titles follow: `fix(hackforger): HF-XXX <description>` or `feat(hackforger): HF-XXX <description>`
- After each wave completes, run E2E testing before starting next wave

## E2E Testing Checkpoints

- **After Wave 1**: Verify all error messages display correctly, JS error resolved, dropdowns work, admin navigation accessible
- **After Wave 2**: Verify Bounty timeline events, notifications, duplicate prevention, search selectors, leaderboard
- **After Wave 3**: Verify Phase timeline UI, auto-advance, edit locking, action gating across all activity types

E2E testing uses agent-browser via `localhost:3000` with screenshots at key checkpoints.
