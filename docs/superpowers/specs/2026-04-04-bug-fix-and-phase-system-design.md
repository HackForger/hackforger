# HackForger Bug Fix & Phase System Design

**Date**: 2026-04-04
**Status**: Approved
**Scope**: 28 tickets across 6 modules, organized into 3 waves (HF-027 closed, excluded from active work)

## Overview

This spec addresses all bugs and feature requests discovered during E2E testing and tracked in GitHub Issues. The work is split into three waves ordered by complexity and dependency:

- **Wave 1** -- Quick Fixes: Error handling audit, JS bugs, UI fixes
- **Wave 2** -- Medium Features: Timeline events, notifications, data integrity guards
- **Wave 3** -- Phase System: Core architectural change introducing event lifecycle management

Each wave is detailed in a sub-file:

- [Wave 1: Quick Fixes](2026-04-04-wave1-quick-fixes.md)
- [Wave 2: Medium Features](2026-04-04-wave2-medium-features.md)
- [Wave 3: Phase System](2026-04-04-wave3-phase-system.md)

## Ticket Index

| ID | Module | Type | Summary | Wave |
|----|--------|------|---------|------|
| HF-001 | Hackathon/Phase | Bug | Phase Control UI missing i18n | 1 |
| HF-002 | Hackathon/Phase | Bug | No error when publishing without tracks | 1 |
| HF-003 | Hackathon/Phase | Bug | Past phases still editable | 3 |
| HF-004 | Hackathon/Phase | Bug | Cancel without confirmation dialog | 1 |
| HF-005 | Hackathon/Phase | Feature | Auto phase advance by time | 3 |
| HF-006 | Hackathon/Phase | Feature | Custom phase stages | 3 |
| HF-007 | Hackathon/Phase | Feature | Block organizer self-registration | 2 |
| HF-008 | Hackathon/Submit | Bug | Track dropdown truncated (GH #19) | 1 |
| HF-009 | Hackathon/Judge | Bug | Scoring submit not working | 1 |
| HF-010 | Hackathon/Judge | Bug | Results preview blank | 1 |
| HF-011 | Hackathon/Submit | Feature | Block duplicate submission per track | 2 |
| HF-012 | Hackathon/Judge | Feature | Richer judge review UI | 2 |
| HF-013 | Hackathon/Judge | Feature | Judge signup validation | 2 |
| HF-014 | Hackathon | Feature | Block duplicate activity creation | 2 |
| HF-015 | Bounty | Bug | Bounty should not allow editing linked Issue | 2 |
| HF-016 | Bounty | Feature | Bounty status in Issue timeline | 2 |
| HF-017 | Bounty/General | Feature | User search selector (GH #17) | 2 |
| HF-018 | Grant | Feature | Grant approval notification | 2 |
| HF-019 | Grant | Feature | Budget/reward amount read-only | 2 |
| HF-020 | Credits | Bug | Credits page has no navigation entry (GH #18) | 1 |
| HF-021 | Credits | Feature | Credit leaderboard on Explore | 2 |
| HF-022 | Feed/Notif | Feature | Follow/Watch triggers notifications | 2 |
| HF-023 | Feed/Notif | Feature | "Join Org" button (notification-driven) | 2 |
| HF-024 | Feed/Notif | Feature | Dashboard feed optimization (GH #16) | 2 |
| HF-025 | UI | Bug | JS null error in repo-issue-pr-status.js | 1 |
| HF-026 | UI | Bug | Remove "display name" field | 1 |
| HF-027 | Infra | Bug | Workflow auto-merge token (GH #10, closed) | -- |
| HF-028 | Admin | Feature | Admin panel HackForger navigation (GH #18) | 1 |

## Cross-Cutting Concerns

### Error Handling (applies to all waves)
- All flash errors must use `ctx.Tr("hackforger.error.xxx")` with i18n keys
- Web routes: `ctx.Flash.Error(ctx.Tr(...), true)` + render current page
- API routes: `ctx.Error(status, title, err)` with meaningful message
- Vue/JS: `showErrorToast(msg)` from `web_src/js/modules/toast.js`
- No silent failures -- every error path must produce user-visible feedback

### PR Strategy
- Each fix/feature is an independent PR with auto-merge
- Enables granular rollback if issues arise
- Concurrent work on independent tasks across waves
