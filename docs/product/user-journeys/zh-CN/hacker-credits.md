## 旅程 4：Credits 流转 + 兑换

### 前置条件
- hacker1 拥有 1950 Credits（来自 J1 + J2 + J3 的累积）
- `admin` 已配置兑换选项
- RedeemOption 密钥池已存在

### 步骤 4.1：Admin 配置 RedeemOption

- **角色**：`admin`
- **操作**：创建一个兑换选项（如 GitHub Copilot 1 个月订阅）
- **API**: `PUT /api/v1/hackforger/credits/redeem/options/1`（或 POST 创建）
  ```json
  {
    "name": "GitHub Copilot - 1 Month",
    "description": "One month of GitHub Copilot Individual subscription",
    "cost": 200,
    "stock": 10,
    "is_active": true
  }
  ```
- **预期结果**：RedeemOption 已创建，激活状态，费用 = 200 Credits
- **Fixture Hint**: `redeem_option(id=1, name="GitHub Copilot - 1 Month", cost=200, stock=10, active=true)`

### 步骤 4.2：Admin 向密钥池添加密钥

- **角色**：`admin`
- **操作**：上传兑换选项的激活密钥
- **API**：（Admin 密钥池管理端点）
  ```json
  {
    "option_id": 1,
    "keys": ["COPILOT-KEY-001", "COPILOT-KEY-002", "COPILOT-KEY-003"]
  }
  ```
- **预期结果**：3 个密钥已添加到选项 #1 的密钥池
- **Fixture Hint**: `redeem_option_key(option_id=1, key="COPILOT-KEY-001", used=false)` x3

### 步骤 4.3：hacker1 查询余额

- **角色**：`hacker1`
- **API**: `GET /api/v1/hackforger/credits/balance`
- **Web**: `/hackforger/credits`（仪表板）
- **预期结果**：
  ```json
  { "balance": 1950 }
  ```
- **Fixture Hint**: `credit_account(user=hacker1, balance=1950)`

### 步骤 4.4：hacker1 查看交易历史

- **角色**：`hacker1`
- **API**: `GET /api/v1/hackforger/credits/transactions`
- **预期结果**：所有交易列表：
  - admin_deposit →（不可见，这是充值给 organizer 的）
  - deposit +500（Hackathon 第 1 名）
  - escrow_release +300（Bounty #1 奖励）
  - escrow_release +150（Bounty #2 第 1 名）
  - deposit +1000（Grant #1 资助）
- **Fixture Hint**: Multiple `credit_transaction` records with correct types and amounts

### 步骤 4.5：hacker1 浏览兑换选项

- **角色**：`hacker1`
- **API**: `GET /api/v1/hackforger/credits/redeem/options`
- **Web**: `/hackforger/credits/redeem`
- **预期结果**：
  ```json
  [
    { "id": 1, "name": "GitHub Copilot - 1 Month", "cost": 200, "stock": 10 }
  ]
  ```

### 步骤 4.6：hacker1 用 Credits 兑换奖励

- **角色**：`hacker1`
- **操作**：兑换 200 Credits 获取 GitHub Copilot 订阅
- **API**: `POST /api/v1/hackforger/credits/redeem`
  ```json
  { "option_id": 1 }
  ```
- **预期结果**：
  - RedeemOrder 已创建，状态 = `pending`
  - Credits 原子扣除：1950 - 200 = 1750
  - 交易类型：`redeem`
  - Feed 事件：`credits_redeemed(42)` — 受众：自己
- **Fixture Hint**: `redeem_order(id=1, user_id=hacker1, option_id=1, status=pending)`, `credit_transaction(redeem, hacker1, -200)`

### 步骤 4.7：Admin 履行订单

- **角色**：`admin`
- **操作**：从密钥池分配密钥并履行订单
- **API**: `POST /api/v1/hackforger/credits/redeem/orders/1/fulfill`
  ```json
  { "key": "COPILOT-KEY-001" }
  ```
- **预期结果**：
  - 订单状态：`pending` → `fulfilled`
  - 密钥 "COPILOT-KEY-001" 标记为已使用
  - hacker1 可在订单详情中查看其密钥
  - Feed 事件：`order_fulfilled(58)`
- **Fixture Hint**: `redeem_order(id=1, status=fulfilled, key="COPILOT-KEY-001")`, `redeem_option_key(key="COPILOT-KEY-001", used=true)`

### 步骤 4.8：Admin 手动充值

- **角色**：`admin`
- **操作**：手动 Credits 充值（如赞助奖励）
- **API**: `POST /api/v1/hackforger/credits/admin/deposit`
  ```json
  { "username": "hacker2", "amount": 500, "reason": "Community contribution bonus" }
  ```
- **预期结果**：
  - hacker2 余额：400 + 500 = 900
  - 交易类型：`admin_deposit`
- **Fixture Hint**: `credit_transaction(admin_deposit, hacker2, 500)`

### 步骤 4.9：Admin 手动扣款

- **角色**：`admin`
- **操作**：扣除 Credits（如纠正错误）
- **API**: `POST /api/v1/hackforger/credits/admin/deduct`
  ```json
  { "username": "hacker2", "amount": 50, "reason": "Duplicate reward correction" }
  ```
- **预期结果**：
  - hacker2 余额：900 - 50 = 850
  - 交易类型：`admin_deduct`
- **Fixture Hint**: `credit_transaction(admin_deduct, hacker2, -50)`

### 状态快照：旅程 4 结束后

| 实体 | 状态 |
|------|------|
| hacker1 Credits | 1750 (1950 - 200 已兑换) |
| hacker2 Credits | 850 (400 + 500 - 50) |
| RedeemOrder #1 | `fulfilled` |
| 密钥 COPILOT-KEY-001 | 已使用 |
| 已覆盖的交易类型 | deposit, withdraw, redeem, admin_deposit, admin_deduct, escrow, escrow_release, escrow_refund |

---
