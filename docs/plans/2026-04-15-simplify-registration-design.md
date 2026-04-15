# Simplify Hackathon Registration Flow

**Date:** 2026-04-15
**Issue:** #55
**Status:** Approved by tester

## Problem

Current registration requires users to choose "individual vs team (organization)" from a dropdown. This causes confusion because teams may form or dissolve after registration. The tester wants all registration to be repo-based — the repo owner determines individual vs team.

## Design

### Core Principle

Registration = selecting a repo. The repo owner (user or org) determines participation type.

### Phase Behavior

| Phase | Repo Selection | Behavior |
|-------|---------------|----------|
| Registration | Any repo the user owns (personal or org) | Creates Registration record with RepoID |
| Development | Only repos already registered | Updates Submission on existing Registration |

### Registration Form (replaces current)

```
[Select Repository ▾]          [Select Track ▾]
  my-project                     数字文化赛道
  my-org/team-project            数字营销赛道
  + 创建新仓库                    ...

[Project Title    ]
[Description      ]
[Demo URL         ]

[Submit / 报名]
```

- "创建新仓库" redirects to `/repo/create?redirect_to=/hackathon/{slug}` with the new repo auto-selected on return
- Repo dropdown includes both personal repos and repos from orgs where user is a member

### Data Flow

**Registration phase POST:**
1. User selects repo + track + fills project info
2. Handler creates `HackathonRegistration` with `RepoID`, `TrackID`
3. `OrgID` = repo.OwnerID if repo owner is org, else 0
4. `TeamName` = repo owner name
5. Simultaneously creates `HackathonSubmission` linked to this registration

**Development phase POST:**
1. Handler loads user's existing Registration(s) for this hackathon
2. Repo dropdown only shows repos from those Registrations
3. Updates the linked Submission (title, description, demo URL)

### Model Changes

No schema migration needed. Existing fields are reused:
- `HackathonRegistration.RepoID` — already exists, currently optional → becomes required
- `HackathonRegistration.OrgID` — auto-derived from repo owner
- `HackathonRegistration.TeamName` — auto-derived from repo owner name
- `HackathonRegistration.TrackID` — already exists

### Files to Change

| Layer | File | Change |
|-------|------|--------|
| Template | `templates/hackforger/hackathon/view.tmpl` | Replace individual/team dropdown with repo selector + track selector + project fields |
| Handler | `routers/web/hackforger/hackathon.go` `RegisterPost` | Remove org_id form param, add repo_id + track + project fields, derive OrgID/TeamName from repo |
| Handler | `routers/web/hackforger/hackathon.go` view handler | Load user repos for dropdown instead of user orgs |
| Handler | `routers/web/hackforger/hackathon.go` `SubmitPost` | Filter repos to registered-only during development phase |
| i18n | Both locale files | Update registration-related keys |

### Questions (answered)

1. **Multiple repos per user?** — One registration per user per hackathon (existing UNIQUE constraint)
2. **Track selection at registration?** — Yes, select track at registration time
3. **Auto-select new repo?** — Yes, via URL query param `?repo_id=N`
