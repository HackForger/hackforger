# Wave 2: Medium Features

**Parent**: [Bug Fix & Phase System Design](2026-04-04-bug-fix-and-phase-system-design.md)
**Goal**: Add missing features for data integrity, notifications, and UX improvements.

Split into two sub-waves for focused implementation:
- **Wave 2a**: Data integrity + validation (2.5-2.8, 2.10-2.12) -- backend-heavy, can run concurrently
- **Wave 2b**: UI features + notifications (2.1-2.4, 2.9, 2.13-2.14) -- template/frontend-heavy

## Upstream Injection Points

The following items require modifying upstream Forgejo files. Each is a minimal change (1-5 lines) and must be tracked as an injection point per CLAUDE.md:

| Item | Upstream File | Change |
|------|--------------|--------|
| 2.1 | `models/issues/comment.go` | Add `CommentTypeHackforger` constant (1 line) |
| 2.1 | `templates/repo/issue/view_content/comments.tmpl` | Add `{{else if .IsHackforger}}` include block (2 lines) |
| 2.2 | `templates/org/header.tmpl` | Add "Join" button conditional (3 lines) |
| 2.14 | `templates/explore/navbar.tmpl` | Add "Credits" tab link (2 lines) |
| 2.13 | `templates/user/dashboard/feeds.tmpl` | Add HackForger sidebar partial include (2 lines) |

---

## Wave 2a: Data Integrity & Validation

### 2.5 Duplicate Submission Prevention (HF-011)

- Add `UNIQUE(user_id, track_id)` constraint to `hackforger_submission` table
- **Migration**: New migration file in `models/forgejo_migrations/` to add the constraint
- **Service layer**: Check before insert, return `hackforger.error.duplicate_submission` if violated
- Handles both: DB constraint (safety net) + service check (user-friendly error)

### 2.6 Organizer Self-Registration Block (HF-007)

- In `services/hackforger/hackathon.go` registration handler:
  ```
  if hackathon.OwnerID == user.ID → reject
  also check: user is org owner of the hackathon's org → reject
  ```
- Error: `hackforger.error.cannot_register_own_event`

### 2.7 User Search Selector (HF-017, GH #17)

- Replace text input with search dropdown for all user-name input fields in HackForger
- Reuse existing `web_src/js/features/comp/SearchUserBox.js`
- Calls `/api/v1/users/search?q=xxx` for autocomplete
- Affects: judge assignment, member addition, bounty assignee, grant reviewer assignment

### 2.8 Duplicate Activity Block (HF-014)

- On create Hackathon/Bounty/Grant, check if same-name activity exists within the same org **and same activity kind**
- A Hackathon and a Bounty with the same name in the same org is allowed (different kinds)
- If exists: block creation, return `hackforger.error.duplicate_activity_name`
- Check at service layer before DB insert

### 2.10 Judge Signup Validation (HF-013)

- Validate judge signup: user must exist, must not already be a judge for this hackathon, must not be a participant
- Return appropriate i18n error for each case

### 2.11 Bounty Issue Edit Boundary (HF-015)

- When an Issue has an associated Bounty, restrict editing of Bounty-managed fields (amount, status)
- Issue title/description remain editable by Issue author
- Bounty fields only editable through Bounty service endpoints

### 2.12 Grant Budget Read-Only (HF-019)

- "已使用" amount in Grant Round view: render as read-only text, not editable input
- Admin panel reward amount: same treatment
- Template fix only, no backend changes needed

---

## Wave 2b: UI Features & Notifications

### 2.1 Bounty Status in Issue Timeline (HF-016)

Bounty lifecycle events appear in the Issue's comment timeline.

**Approach**:
- Add a single generic `CommentTypeHackforger` constant to upstream `models/issues/comment.go` (one injection point for all HackForger timeline events)
- Use the comment's `Content` field to store a JSON payload with `{"type": "bounty_created", "data": {...}}` for subtype differentiation
- In `services/hackforger/bounty.go`, every status transition calls `issues_model.CreateComment()` with `CommentTypeHackforger`
- Add HackForger-specific comment rendering template at `templates/hackforger/issue_comment.tmpl`, included from upstream comments template via injection point
- All Bounty status changes must go through the service layer to ensure Comment + Feed are both written

**Subtypes**: `bounty_created`, `bounty_applied`, `bounty_accepted`, `bounty_completed`, `bounty_cancelled`

**Visual**: Timeline entries show a colored badge + status text + actor name + timestamp.

### 2.2 "Join Org" Button (HF-023)

Notification-driven flow, no new table needed.

**Flow**:
1. Org page shows "申请加入" button (visible when user is not a member)
2. `POST /:org/join-request` → sends notification to org owner
3. Notification contains: applicant username + link to add-member page with `?username=<applicant>` query param
4. Owner clicks link → add-member page opens with username pre-filled → owner clicks confirm

**Rejection**: No explicit reject action. If the owner ignores the notification, no action is taken. The applicant can re-send the request. This is intentionally simple -- the owner either adds the member or doesn't.

**Implementation**:
- New web route: `POST /:org/join-request` in `routers/web/hackforger/org.go`
- New notification type in HackForger notifier
- Modify add-member page template to read `?username=` and pre-fill the input
- Button visibility: `{{if not .IsOrganizationMember}}` conditional in org header template (injection point)

### 2.3 Grant Approval Notification (HF-018)

- When a Grant application is approved in `services/hackforger/grant.go`, call `PublishHackforgerAction()` which triggers notification
- Notification content: Grant Round name + approved amount + link to grant detail page
- Uses existing Forgejo notification infrastructure via `HackForgerNotifier`

### 2.4 Follow/Watch Notification Wiring (HF-022)

- Hackathon/Bounty/Grant are bound to org/repo. User watches the repo → receives notifications for HackForger events on that repo
- Ensure all `PublishHackforgerAction()` calls in service layer produce events that the notification system picks up
- Verify: `HackForgerNotifier` implementation covers all state changes

### 2.9 Judge Review UI Enrichment (HF-012)

- Judging interface shows: submitter name (with link to profile), linked repo (with link), submission description, track name
- Data already available in submission model, just needs template updates

### 2.13 Dashboard Feed Optimization (HF-024, GH #16)

- Dashboard right sidebar: filter repo/org by HackForger association
- Display categorized sections using i18n keys: `hackforger.dashboard.my_hackathons`, `hackforger.dashboard.my_bounties`, `hackforger.dashboard.my_grants`
- New model function in `models/hackforger/`: `GetUserActivitySummary(userID)` returning counts and recent items per activity kind
- New partial template `templates/hackforger/dashboard_sidebar.tmpl`, included from upstream feeds template via injection point

### 2.14 Credit Leaderboard on Explore (HF-021)

- Add "积分排行" tab to Explore page (`/explore`) via injection point in explore navbar template
- New route: `GET /explore/credits` in `routers/web/hackforger/explore.go`
- New model function: `models/hackforger/credits.GetLeaderboard(page, pageSize)`
- Query top N users by credit balance
- Display: rank, avatar, username, credit amount
- Paginated list, reuse Explore page layout patterns
- New template: `templates/hackforger/explore_credits.tmpl`
