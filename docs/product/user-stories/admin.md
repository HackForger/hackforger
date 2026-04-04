# Admin Stories

## Epic 4: Credits Economy

> **Epic Hypothesis:** We believe that an internal Credits economy connecting hackathon prizes, bounty rewards, and grant funding into a single redeemable balance will increase platform engagement, because participants see tangible value accumulating across all activities.

---

### Story C-003: Admin Fulfills Redeem Order

- **Summary:** Admin assigns a key from the pool and fulfills a pending order

#### Use Case:
- **As a** platform admin
- **I want to** fulfill pending redeem orders by assigning activation keys
- **so that** users receive their rewards promptly

#### Acceptance Criteria:

- **Scenario:** Admin fulfills an order
- **Given:** RedeemOrder #1 is in `pending` status and key pool has available keys
- **When:** I fulfill the order with key "COPILOT-KEY-001"
- **Then:** Order status becomes `fulfilled`, the key is marked as used, the user can view their key, and an `order_fulfilled(58)` feed event is emitted

**Journey ref:** J4 Steps 4.1, 4.2, 4.7

---

### Story C-004: Admin Manual Deposit/Deduct

- **Summary:** Admin manually adjusts a user's Credits balance

#### Use Case:
- **As a** platform admin
- **I want to** deposit or deduct Credits from any user's account with a reason
- **so that** I can handle sponsorship bonuses, error corrections, and special allocations

#### Acceptance Criteria:

- **Scenario:** Admin deposits Credits
- **Given:** `hacker2` has 400 Credits
- **When:** I deposit 500 Credits with reason "Community contribution bonus"
- **Then:** hacker2's balance becomes 900, a `admin_deposit` transaction is recorded

- **Scenario:** Admin deducts Credits
- **Given:** `hacker2` has 900 Credits
- **When:** I deduct 50 Credits with reason "Duplicate reward correction"
- **Then:** hacker2's balance becomes 850, an `admin_deduct` transaction is recorded

**Journey ref:** J4 Steps 4.8, 4.9
