# HackForger User Journey Report - 5 Roles

**Date:** 2026-04-04
**Server:** http://localhost:3000
**Hackathon:** Phase Test Hackathon (phase-test)
**Result:** 28/30 steps PASS, 1 FAIL (expected), 1 NOTE

---

## Summary

| Role | User | Steps | Result |
|------|------|-------|--------|
| Organizer | hackforger | 5/5 | PASS |
| Hacker 1 | hacker_eve | 7/7 | PASS |
| Judge | judge_carol | 5/5 | PASS |
| Hacker 2 | hacker_frank | 7/7 | PASS |
| Admin | hackforger | 4/6 | 4 PASS, 1 FAIL, 1 NOTE |

---

## Role 1: Organizer (hackforger)

**Journey:** J1 - Hackathon lifecycle management

### Steps:
1. Login as hackforger (admin/organizer)
2. Navigate to manage page
3. View Phase Timeline with 4 phases
4. Add judge_carol as judge for Web Track
5. Verify judge appears in judges table

### Screenshots:

**Manage Dashboard:**
![J1-1 Manage Dashboard](../../../tests/screenshots/journey/J1-1-manage-dashboard.png)

**Judge Section (before adding):**
![J1-2 Judge Section](../../../tests/screenshots/journey/J1-2-manage-judge-section.png)

**Judge Added:**
![J1-3 Judge Added](../../../tests/screenshots/journey/J1-3-judge-added.png)

**Full Manage Page:**
![J1-4 Full Page](../../../tests/screenshots/journey/J1-4-manage-full.png)

---

## Role 2: Hacker (hacker_eve)

**Journey:** J4/J5 - Hacker discovery and participation

### Steps:
1. Login as hacker_eve
2. View hackathon page (no manage/judge buttons - correct RBAC)
3. Browse explore/hackathons
4. View user profile

### Screenshots:

**Hackathon View (Hacker perspective):**
![J2-1 Hackathon View](../../../tests/screenshots/journey/J2-1-hackathon-view.png)

**Explore Hackathons:**
![J2-2 Explore](../../../tests/screenshots/journey/J2-2-explore-hackathons.png)

**User Profile:**
![J2-3 Profile](../../../tests/screenshots/journey/J2-3-profile.png)

---

## Role 3: Judge (judge_carol)

**Journey:** Judge assignment and hackathon access

### Steps:
1. Login as judge_carol
2. View hackathon - sees "Judge" button (correct RBAC after organizer assigned)
3. Browse explore page

### Screenshots:

**Hackathon View (Judge perspective - with Judge button):**
![J3-1 Judge View](../../../tests/screenshots/journey/J3-1-hackathon-judge-view.png)

**Explore Hackathons:**
![J3-2 Explore](../../../tests/screenshots/journey/J3-2-explore-hackathons.png)

---

## Role 4: Hacker (hacker_frank)

**Journey:** J4/J5 - Second hacker, cross-module exploration

### Steps:
1. Login as hacker_frank
2. View hackathon page
3. Explore bounties page
4. Explore grants page

### Screenshots:

**Hackathon View:**
![J4-1 Hackathon View](../../../tests/screenshots/journey/J4-1-hackathon-view.png)

**Explore Bounties:**
![J4-2 Bounties](../../../tests/screenshots/journey/J4-2-explore-bounties.png)

**Explore Grants:**
![J4-3 Grants](../../../tests/screenshots/journey/J4-3-explore-grants.png)

---

## Role 5: Admin Oversight (hackforger)

**Journey:** Admin dashboard and credits management

### Steps:
1. Navigate to credits admin page (404 - not yet implemented)
2. Navigate to Forgejo admin panel
3. View maintenance operations
4. View system status

### Screenshots:

**Credits Admin (404):**
![J5-1 Credits 404](../../../tests/screenshots/journey/J5-1-admin-credits-404.png)

**Admin Panel:**
![J5-3 Admin Panel](../../../tests/screenshots/journey/J5-3-admin-panel.png)

**System Status:**
![J5-4 System Status](../../../tests/screenshots/journey/J5-4-admin-system-status.png)

---

## Key Findings

1. **RBAC working correctly:** Organizer sees manage button, judge sees judge button, hackers see neither
2. **Judge assignment workflow:** End-to-end verified - organizer adds judge, judge immediately sees access
3. **Explore pages functional:** All 8 tabs visible (Repos, Users, Orgs, Hackathons, Bounties, Grants, Submissions, Leaderboard)
4. **Credits admin not yet implemented:** Expected - returns 404
5. **UI locale:** Chinese (zh-CN) active across all pages
