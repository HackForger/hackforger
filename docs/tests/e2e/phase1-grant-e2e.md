# Phase 1: Grant + Credits -- E2E Manual Test Prompt

## Prerequisites

- [ ] HackForger server running at http://localhost:3000 (or https://hackforger.inside.h2os.cloud)
- [ ] Admin account available (hackforger / admin1234)
- [ ] A second "hacker" account available (create one if needed)
- [ ] Database migrated (all hackforger tables exist)
- [ ] No pre-existing grant rounds (clean state recommended)

## Test Flow

### A. Organization + Grant Round Setup (Admin)

**A1. Create Organization**

1. Log in as admin.
2. Create a new organization called `test-grants-org`.
3. Verify the org page loads at `/org/test-grants-org`.

**A2. Create Grant Round**

1. Navigate to the grant rounds page (e.g., `/-/hackforger/grants` or org grants page).
2. Click "New Grant Round".
3. Fill in:
   - Name: `Spring 2026 Grants`
   - Slug: `spring-2026`
   - Description: `Test grant round for E2E validation.`
   - Budget: `10000`
   - Currency: `USD`
   - Budget (Credits): `5000`
   - Deadline: a date in the future
   - Organization: `test-grants-org`
4. Submit.
5. **Verify:** Round is created with status **Draft**.
6. **Verify:** Success toast/flash: "Grant round created successfully."

**A3. Open the Round**

1. On the round detail page, click "Open Round".
2. **Verify:** Status changes to **Open**.
3. **Verify:** Feed event appears in global feed for "grant round opened".

### B. Project Submission (Hacker)

**B1. Submit a Project**

1. Log out. Log in as the hacker account.
2. Navigate to the grant round `spring-2026`.
3. Click "Submit Project".
4. Fill in:
   - Title: `Open Source Widget`
   - Description: `A widget that does amazing things.`
   - Repository: (select or enter a repo URL)
5. Submit.
6. **Verify:** Project status is **Pending**.
7. **Verify:** Success flash: "Project submitted successfully."

**B2. Duplicate Submission (Error Case)**

1. While still logged in as the hacker, try to submit another project to the same round.
2. **Verify:** Error message: "You have already submitted a project to this round."

### C. Review + Allocation (Admin)

**C1. Close Applications**

1. Log in as admin.
2. Navigate to the round detail page.
3. Click "Close Applications".
4. **Verify:** Status changes to **Reviewing** (or Review).

**C2. Approve Project**

1. Go to the project list for the round.
2. Click on the hacker's project.
3. Click "Approve".
4. **Verify:** Project status changes to **Approved**.

**C3. Allocate Award**

1. On the approved project, click "Allocate Award".
2. Enter award amount: `2000` (credits).
3. Submit.
4. **Verify:** Award amount is recorded on the project.

**C4. Exceed Budget (Error Case)**

1. Create a second hacker account and submit + approve a second project.
2. Try to allocate `4000` credits (total would be 6000, exceeding 5000 budget).
3. **Verify:** Error indicating budget would be exceeded.

**C5. Finalize Round**

1. Click "Finalize Allocations".
2. **Verify:** Round status changes to **Finalized**.
3. **Verify:** Feed event for "grant round finalized".

### D. Distribution

**D1. Distribute a Project**

1. On the finalized round, click "Distribute" on the approved project.
2. **Verify:** Project status changes to **Funded**.
3. **Verify:** Credits are deposited to the hacker's balance (check credits ledger).
4. **Verify:** Feed event for "grant project funded".

**D2. Auto-Transition to Distributed**

1. Distribute all remaining approved projects (or if only one, it should already trigger).
2. **Verify:** Once all approved projects are distributed, the round status auto-transitions to **Distributed**.
3. **Verify:** Feed event for "grant round distributed".

### E. Credits Lifecycle

**E1. Check Hacker Balance**

1. Log in as the hacker.
2. Navigate to credits overview page.
3. **Verify:** Balance shows `2000` credits (or whatever was awarded).
4. **Verify:** Transaction history shows a deposit entry with reference to the grant project.

**E2. Redeem Option**

1. Navigate to redeem options page.
2. If a redeem option exists (e.g., "Sticker Pack" for 100 credits), click "Redeem".
3. **Verify:** Confirmation dialog: 'Are you sure you want to redeem "Sticker Pack" for 100 credits?'
4. Confirm.
5. **Verify:** Success message: "Successfully redeemed!"
6. **Verify:** Balance decreased by 100.
7. **Verify:** A new order appears in "My Orders" with status **Pending**.

**E3. Insufficient Credits (Error Case)**

1. Try to redeem an option that costs more than the remaining balance.
2. **Verify:** Error: "Insufficient credits."

### F. Credits Administration (Admin)

**F1. Deposit Credits**

1. Log in as admin.
2. Navigate to Credits Administration.
3. Deposit `500` credits to the hacker's user ID with reference "bonus" and note "E2E test deposit".
4. **Verify:** Hacker's balance increases by 500.

**F2. Deduct Credits**

1. Deduct `100` credits from the hacker with reference "correction" and note "E2E test deduction".
2. **Verify:** Hacker's balance decreases by 100.

**F3. Fulfill Order**

1. Navigate to "Manage Orders".
2. Find the hacker's pending order from step E2.
3. Click "Fulfill" and enter a fulfillment note: "Shipped via mail."
4. **Verify:** Order status changes to **Fulfilled**.

**F4. Cancel Order**

1. Create another order (redeem as hacker, then switch to admin).
2. Click "Cancel" on the new order.
3. **Verify:** Order status changes to **Cancelled**.
4. **Verify:** Credits are refunded to the hacker's balance.

### G. Feed Verification

Review the global feed (`/-/hackforger/feed` or explore feed) and confirm events were recorded for:

- [ ] Grant round created
- [ ] Grant round opened
- [ ] Project submitted
- [ ] Grant round closed for review
- [ ] Project approved
- [ ] Grant round finalized
- [ ] Project funded / distributed
- [ ] Grant round fully distributed
- [ ] Credits deposited (admin action)
- [ ] Credits redeemed
- [ ] Order fulfilled

### H. Export

**H1. CSV Export**

1. As admin, navigate to the finalized/distributed round.
2. Click "Export CSV".
3. **Verify:** A CSV file downloads containing project titles, applicant usernames, statuses, and award amounts.

---

## Report Template

Copy the checklist below and fill in pass/fail for each item.

| # | Test Step | Result | Notes |
|---|-----------|--------|-------|
| A1 | Create organization | [ ] Pass / [ ] Fail | |
| A2 | Create grant round (Draft) | [ ] Pass / [ ] Fail | |
| A3 | Open round | [ ] Pass / [ ] Fail | |
| B1 | Submit project (Pending) | [ ] Pass / [ ] Fail | |
| B2 | Duplicate submission error | [ ] Pass / [ ] Fail | |
| C1 | Close applications (Reviewing) | [ ] Pass / [ ] Fail | |
| C2 | Approve project | [ ] Pass / [ ] Fail | |
| C3 | Allocate award | [ ] Pass / [ ] Fail | |
| C4 | Exceed budget error | [ ] Pass / [ ] Fail | |
| C5 | Finalize round | [ ] Pass / [ ] Fail | |
| D1 | Distribute project (credits deposited) | [ ] Pass / [ ] Fail | |
| D2 | Round auto-transitions to Distributed | [ ] Pass / [ ] Fail | |
| E1 | Hacker balance correct | [ ] Pass / [ ] Fail | |
| E2 | Redeem option + order created | [ ] Pass / [ ] Fail | |
| E3 | Insufficient credits error | [ ] Pass / [ ] Fail | |
| F1 | Admin deposit credits | [ ] Pass / [ ] Fail | |
| F2 | Admin deduct credits | [ ] Pass / [ ] Fail | |
| F3 | Fulfill order | [ ] Pass / [ ] Fail | |
| F4 | Cancel order + refund | [ ] Pass / [ ] Fail | |
| G | Feed events recorded | [ ] Pass / [ ] Fail | |
| H1 | CSV export | [ ] Pass / [ ] Fail | |

**Tester:** _______________
**Date:** _______________
**Server URL:** _______________
**Commit/Branch:** _______________
