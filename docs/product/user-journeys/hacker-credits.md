# Journey 4: Credits 流转 + 兑换

### Preconditions
- hacker1 has 1950 Credits (accumulated from J1 + J2 + J3)
- `admin` has configured redeem options
- RedeemOption key pool exists

### Step 4.1: Admin configures RedeemOption

- **Role**: `admin`
- **Operation**: Create a redeem option (e.g., GitHub Copilot 1-month subscription)
- **API**: `PUT /api/v1/hackforger/credits/redeem/options/1` (or POST to create)
  ```json
  {
    "name": "GitHub Copilot - 1 Month",
    "description": "One month of GitHub Copilot Individual subscription",
    "cost": 200,
    "stock": 10,
    "is_active": true
  }
  ```
- **Expected Result**: RedeemOption created, active, cost = 200 Credits
- **Fixture Hint**: `redeem_option(id=1, name="GitHub Copilot - 1 Month", cost=200, stock=10, active=true)`

### Step 4.2: Admin adds keys to the pool

- **Role**: `admin`
- **Operation**: Upload activation keys for the redeem option
- **API**: (Admin key pool management endpoint)
  ```json
  {
    "option_id": 1,
    "keys": ["COPILOT-KEY-001", "COPILOT-KEY-002", "COPILOT-KEY-003"]
  }
  ```
- **Expected Result**: 3 keys added to pool for option #1
- **Fixture Hint**: `redeem_option_key(option_id=1, key="COPILOT-KEY-001", used=false)` x3

### Step 4.3: hacker1 checks balance

- **Role**: `hacker1`
- **API**: `GET /api/v1/hackforger/credits/balance`
- **Web**: `/hackforger/credits` (dashboard)
- **Expected Result**:
  ```json
  { "balance": 1950 }
  ```
- **Fixture Hint**: `credit_account(user=hacker1, balance=1950)`

### Step 4.4: hacker1 views transaction history

- **Role**: `hacker1`
- **API**: `GET /api/v1/hackforger/credits/transactions`
- **Expected Result**: List of all transactions:
  - admin_deposit → (not visible, this was to organizer)
  - deposit +500 (Hackathon 1st place)
  - escrow_release +300 (Bounty #1 reward)
  - escrow_release +150 (Bounty #2 1st place)
  - deposit +1000 (Grant #1 funding)
- **Fixture Hint**: Multiple `credit_transaction` records with correct types and amounts

### Step 4.5: hacker1 browses redeem options

- **Role**: `hacker1`
- **API**: `GET /api/v1/hackforger/credits/redeem/options`
- **Web**: `/hackforger/credits/redeem`
- **Expected Result**:
  ```json
  [
    { "id": 1, "name": "GitHub Copilot - 1 Month", "cost": 200, "stock": 10 }
  ]
  ```

### Step 4.6: hacker1 redeems Credits for reward

- **Role**: `hacker1`
- **Operation**: Redeem 200 Credits for GitHub Copilot subscription
- **API**: `POST /api/v1/hackforger/credits/redeem`
  ```json
  { "option_id": 1 }
  ```
- **Expected Result**:
  - RedeemOrder created with status = `pending`
  - Credits deducted atomically: 1950 - 200 = 1750
  - Transaction type: `redeem`
  - Feed event: `credits_redeemed(42)` — audience: self
- **Fixture Hint**: `redeem_order(id=1, user_id=hacker1, option_id=1, status=pending)`, `credit_transaction(redeem, hacker1, -200)`

### Step 4.7: Admin fulfills the order

- **Role**: `admin`
- **Operation**: Assign a key from pool and fulfill the order
- **API**: `POST /api/v1/hackforger/credits/redeem/orders/1/fulfill`
  ```json
  { "key": "COPILOT-KEY-001" }
  ```
- **Expected Result**:
  - Order status: `pending` → `fulfilled`
  - Key "COPILOT-KEY-001" marked as used
  - hacker1 can view their key in order details
  - Feed event: `order_fulfilled(58)`
- **Fixture Hint**: `redeem_order(id=1, status=fulfilled, key="COPILOT-KEY-001")`, `redeem_option_key(key="COPILOT-KEY-001", used=true)`

### Step 4.8: Admin manual deposit

- **Role**: `admin`
- **Operation**: Manual Credit deposit (e.g., sponsorship bonus)
- **API**: `POST /api/v1/hackforger/credits/admin/deposit`
  ```json
  { "username": "hacker2", "amount": 500, "reason": "Community contribution bonus" }
  ```
- **Expected Result**:
  - hacker2 balance: 400 + 500 = 900
  - Transaction type: `admin_deposit`
- **Fixture Hint**: `credit_transaction(admin_deposit, hacker2, 500)`

### Step 4.9: Admin manual deduct

- **Role**: `admin`
- **Operation**: Deduct Credits (e.g., correction for error)
- **API**: `POST /api/v1/hackforger/credits/admin/deduct`
  ```json
  { "username": "hacker2", "amount": 50, "reason": "Duplicate reward correction" }
  ```
- **Expected Result**:
  - hacker2 balance: 900 - 50 = 850
  - Transaction type: `admin_deduct`
- **Fixture Hint**: `credit_transaction(admin_deduct, hacker2, -50)`

### State Snapshot: After Journey 4

| Entity | State |
|--------|-------|
| hacker1 Credits | 1750 (1950 - 200 redeemed) |
| hacker2 Credits | 850 (400 + 500 - 50) |
| RedeemOrder #1 | `fulfilled` |
| Key COPILOT-KEY-001 | Used |
| Transaction types covered | deposit, withdraw, redeem, admin_deposit, admin_deduct, escrow, escrow_release, escrow_refund |
