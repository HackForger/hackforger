# Phase 8: Credits 兑换

> **前置条件**: 完成 `00-setup.md` + `01-hackathon-lifecycle.md` + `02-bounty-collab.md` + `03-grant-funding.md` + `04-submission-judging.md`

本文件覆盖积分总览、交易历史查看、兑换奖品、Admin 发货、Admin 手动充值/扣除。
环境/账号/Session 信息请参考 [00-setup.md](./00-setup.md)。

---

## Phase 8: Credits 兑换

**目标**: 用户查看积分总览和交易历史，兑换奖品，Admin 发货。
**角色**: hacker_eve, hackforger (Admin)
**来源**: J4 Steps 4.3-4.9

### Step 8.1: Hacker1 查看积分总览
- **角色**: hacker_eve session
- **UI 路径**: 顶部导航栏用户菜单 → 点击"积分" → `/credits`
- **操作**: 查看余额和交易历史
- **验证**:
  - 余额显示当前累计积分
  - 交易历史完整，包含：
    - `admin_deposit`（Phase 4 充值，如有）
    - `escrow`（Bounty escrow，如有）
    - `deposit`（Grant 资助 +300）
    - `deposit`（Hackathon 奖金 +XXX）
- **截图**: `screenshots/full-cycle/p8-01-credits-overview.png`

### Step 8.2: Hacker1 浏览兑换选项
- **角色**: hacker_eve session
- **UI 路径**: 积分概览页 `/credits` → 兑换选项区域
- **操作**: 浏览可用的兑换选项列表
- **验证**:
  - 显示 "GPU 算力 - 100 小时"，价格 200 Credits，库存 10
- **截图**: `screenshots/full-cycle/p8-02-redeem-options.png`

### Step 8.3: Hacker1 兑换算力
- **角色**: hacker_eve session
- **UI 路径**: 积分概览页 → 点击 "GPU 算力 - 100 小时" 的"兑换"按钮 → 进入兑换确认页 `/credits/redeem/{id}`
- **操作**: 确认兑换信息无误，点击"确认兑换"按钮
- **验证**:
  - RedeemOrder 创建，状态 = `pending`
  - 积分扣除 200
  - 交易类型：`redeem`
  - Feed 事件: `credits_redeemed(42)` -- 仅自己可见
- **截图**: `screenshots/full-cycle/p8-03-redeem-confirmed.png`

### Step 8.4: Admin Fulfill 订单
- **角色**: hackforger session
- **UI 路径**: 站点管理 → 积分管理 → 订单管理 `/-/admin/credits/orders`
- **操作**: 找到 hacker_eve 的待处理订单，点击"发货"(Fulfill) 按钮（POST `/-/admin/credits/orders/{oid}/fulfill`）
- **验证**:
  - 订单状态：`pending` -> `fulfilled`
  - 密钥 "GPU-KEY-001" 自动分配并标记为已使用
  - hacker_eve 可在订单详情中查看密钥
  - Feed 事件: `order_fulfilled(58)`
- **截图**: `screenshots/full-cycle/p8-04-order-fulfilled.png`

### Step 8.5: Hacker1 查看订单和密钥
- **角色**: hacker_eve session
- **UI 路径**: 积分概览页 `/credits` → 点击"我的订单" → `/credits/orders`
- **操作**: 查看已兑换的订单详情
- **验证**:
  - 订单状态 = `fulfilled`
  - 显示密钥 "GPU-KEY-001"
- **截图**: `screenshots/full-cycle/p8-05-order-detail.png`

### Step 8.6: Admin 手动充值演示（给 Hacker2）
- **角色**: hackforger session
- **UI 路径**: 站点管理 → 积分管理 `/-/admin/credits`
- **操作**: 在"手动充值"区域填写：用户名 = `hacker_frank`，金额 = `500`，原因 = `Community contribution bonus`，点击"充值"
- **验证**:
  - hacker_frank 余额增加 500
  - 交易类型：`admin_deposit`
- **截图**: `screenshots/full-cycle/p8-06-admin-deposit-hacker2.png`

### Step 8.7: Admin 手动扣除演示
- **角色**: hackforger session
- **UI 路径**: 站点管理 → 积分管理 `/-/admin/credits` → "手动扣除"区域
- **操作**: 填写：用户名 = `hacker_frank`，金额 = `50`，原因 = `Duplicate reward correction`，点击"扣除"
- **验证**:
  - hacker_frank 余额减少 50
  - 交易类型：`admin_deduct`
- **截图**: `screenshots/full-cycle/p8-07-admin-deduct.png`

### Phase 8 状态快照

| 实体 | 状态 |
|------|------|
| RedeemOrder #1 (hacker_eve) | `fulfilled` |
| Key GPU-KEY-001 | 已使用 |
| hacker_eve Credits | 余额 - 200（兑换）|
| hacker_frank Credits | 余额 + 500 - 50（Admin 充值/扣除）|

---

## 本阶段结果

| 步骤 | 状态 | 截图 |
|------|------|------|
| Step 8.1: Hacker1 查看积分总览 | PASS/FAIL | screenshots/full-cycle/p8-01-credits-overview.png |
| Step 8.2: 浏览兑换选项 | PASS/FAIL | screenshots/full-cycle/p8-02-redeem-options.png |
| Step 8.3: 兑换算力 | PASS/FAIL | screenshots/full-cycle/p8-03-redeem-confirmed.png |
| Step 8.4: Admin Fulfill 订单 | PASS/FAIL | screenshots/full-cycle/p8-04-order-fulfilled.png |
| Step 8.5: 查看订单和密钥 | PASS/FAIL | screenshots/full-cycle/p8-05-order-detail.png |
| Step 8.6: Admin 手动充值 | PASS/FAIL | screenshots/full-cycle/p8-06-admin-deposit-hacker2.png |
| Step 8.7: Admin 手动扣除 | PASS/FAIL | screenshots/full-cycle/p8-07-admin-deduct.png |
