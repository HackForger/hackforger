# Phase 0: 平台准备 + 环境设置

> **前置条件**: 无（本文件是所有其他文件的前置）

本文件包含测试环境配置、账号设置、多用户 Session 初始化，以及 Admin 平台基础设施准备。

---

## 测试概述

本测试将 5 个用户旅程融合为一个连贯的叙事，模拟真实的平台使用场景：
一个 Hackathon 从创建到完结的完整生命周期中，自然地涉及 Bounty 协作、
Grant 资助、Credits 兑换和社交互动。

**测试规模**: 11 个阶段（Phase 0-10），约 80+ 步骤
**涉及角色**: Admin/Organizer, Hacker1, Hacker2, Judge1, Judge2
**v0.1 限制**: 仅使用积分（Credits）作为奖励，法币功能关闭

---

## 测试环境

- **服务器**: `http://localhost:3000`（worktree 构建的最新 binary）
- **数据库**: 每轮测试前清空 HackForger 数据（见 `e2e-testing-guide.md` 的 Database Cleanup）
- **截图保存**: `screenshots/full-cycle/`（相对于本文件目录）

## 测试账号与角色映射

| 角色 | Session 名 | 用户名 | 密码 | 本次旅程中的职责 |
|------|-----------|--------|------|----------------|
| Admin/Organizer | hackforger | hackforger | admin1234 | 平台管理、积分充值、兑换选项配置、创建 Hackathon/Bounty/Grant |
| Judge 1 | judge_carol | judge_carol | admin1234 | Hackathon 评委 |
| Judge 2 | judge_dave | judge_dave | admin1234 | Hackathon 评委 |
| Hacker 1 | hacker_eve | hacker_eve | admin1234 | 活跃参赛者、Bounty 发起者/认领者、Grant 申请者 |
| Hacker 2 | hacker_frank | hacker_frank | admin1234 | 第二参赛者、Bounty 协作者、组队伙伴 |

> **密码重置**: 测试开始前，对所有非 admin 账号通过 admin API 执行密码重置（见 e2e-testing-guide.md）。

## 多用户 Session 设置

使用 `agent-browser --session <name>` 为每个角色创建独立浏览器会话：

```bash
# 登录所有用户（一次性完成）
for pair in "hackforger hackforger" "judge_carol judge_carol" "judge_dave judge_dave" "hacker_eve hacker_eve" "hacker_frank hacker_frank"; do
  set -- $pair
  session=$1 user=$2
  agent-browser --session $session open "http://localhost:3000/user/login"
  agent-browser --session $session snapshot -i
  agent-browser --session $session fill @e13 "$user"
  agent-browser --session $session fill @e14 "admin1234"
  agent-browser --session $session click @e17
  agent-browser --session $session wait --load networkidle
done
```

---

## Phase 0: 平台准备

**目标**: Admin 配置平台基础设施，确保积分体系可用。
**角色**: hackforger (Admin)

### Step 0.1: Admin 配置兑换选项 (RedeemOption)
- **角色**: hackforger session
- **UI 路径**: 顶部导航栏 "站点管理" → 左侧菜单 "积分管理" → 点击"兑换选项" Tab → `/-/admin/credits/options`
- **操作**: 点击"创建兑换选项"，填写表单：
  - 名称 = `GPU 算力 - 100 小时`
  - 描述 = `100 小时 A100 GPU 算力`
  - 价格 = `200` Credits
  - 库存 = `10`
  - 激活 = `true`
  - 点击"创建"
- **验证**:
  - 兑换选项创建成功，出现在列表中
  - 显示价格 200、库存 10、状态为已激活
- **截图**: `screenshots/full-cycle/p0-01-redeem-option-created.png`

### Step 0.2: Admin 添加 Key Pool
- **角色**: hackforger session
- **UI 路径**: 兑换选项列表 → 点击 "GPU 算力 - 100 小时" → 进入 Key 管理页面 `/-/admin/credits/options/{id}/keys`
- **操作**: 点击"添加密钥"，输入：
  - Keys = `GPU-KEY-001`, `GPU-KEY-002`, `GPU-KEY-003`（每行一个或批量输入）
  - 点击"添加"
- **验证**:
  - 3 个密钥添加成功，显示在 Key 列表中
  - 状态均为"未使用"
- **截图**: `screenshots/full-cycle/p0-02-keys-added.png`

### Step 0.3: Admin 为 Organizer 充值积分（Hackathon 奖金池）
- **角色**: hackforger session
- **UI 路径**: 顶部导航栏 "站点管理" → 左侧菜单 "积分管理" → `/-/admin/credits`
- **操作**: 在"手动充值"区域填写：
  - 用户名 = `hackforger`（Organizer 与 Admin 同账号）
  - 金额 = `1000`
  - 原因 = `Hackathon prize pool`
  - 点击"充值"按钮
- **验证**:
  - 页面显示充值成功提示
  - hackforger 余额 = 1000
  - 交易类型 = `admin_deposit`
- **截图**: `screenshots/full-cycle/p0-03-credits-deposit-1000.png`

### Phase 0 状态快照

| 实体 | 状态 |
|------|------|
| 兑换选项 "GPU 算力 - 100 小时" | 已创建，200 Credits，库存 10 |
| Key Pool | 3 个密钥（GPU-KEY-001/002/003），未使用 |
| hackforger (Admin/Organizer) Credits | 1000 |
| 其他用户 Credits | 0 |

---

## 本阶段结果

| 步骤 | 状态 | 截图 |
|------|------|------|
| Step 0.1: Admin 配置兑换选项 | PASS/FAIL | screenshots/full-cycle/p0-01-redeem-option-created.png |
| Step 0.2: Admin 添加 Key Pool | PASS/FAIL | screenshots/full-cycle/p0-02-keys-added.png |
| Step 0.3: Admin 为 Organizer 充值 | PASS/FAIL | screenshots/full-cycle/p0-03-credits-deposit-1000.png |
