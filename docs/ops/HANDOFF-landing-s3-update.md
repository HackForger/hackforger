# HANDOFF: Landing Page S3 时间更新 + 日期驱动按钮

> 交接给部署同事:本次落地页修改已全部完成并部署到生产环境。本文件说明已完成的变更和后续可选操作。

## 1. 已完成的变更(已部署到生产)

### 1.1 核心功能变更(全部已部署并验证通过)

| 变更 | 验证方式 | 状态 |
|---|---|---|
| S3 W2 日期: `7月11日` → `7月23日` | `grep "7月24日-8月4日" /var/lib/hackforger/custom/public/assets/landing/index.html` → 1 行 | ✅ |
| S3 W3 日期: `7月13-20日` → `7月24日-8月4日` | 同上 | ✅ |
| S3 W4 日期: `7月28-30日` → `8月12-14日` | `grep "8月12日-8月14日"` → 3 行 | ✅ |
| 年度总决赛: `7月底/8月初` → `待定` | `grep "时间待定"` → 8 行 | ✅ |
| 中文赛规弹窗: 主时间表更新 | `grep "6月25日至8月4日"` → 3 行 | ✅ |
| 英文赛规弹窗: Schedule + S3 描述 | `grep "Jun 25 – Aug 4"` → 多行 | ✅ |
| 按钮状态 JS: 根据日期自动切换 | `grep "data-wave-start"` → 8 行 | ✅ |
| S2 排名页: 按得分重排 | S2 #1 = CrossAccess AI(一等奖, 80.4分) | ✅ |
| S1 排名页: 按得分重排(去掉2支未获奖队伍) | S1 #1 = MiniGame Studio(一等奖, 80.4分) | ✅ |
| S1/S2 弹窗: 删除3栏(保留团队/作者、系统项目名、项目简介) | 弹窗只显示3行 | ✅ |

### 1.2 代码提交记录

| PR | 内容 | 状态 |
|---|---|---|
| #191 | S1/S2 决赛展示页 + banner 基础 | ✅ 已合并部署 |
| #192 | S2 排名页按得分重排 + S2 弹窗删 3 栏 | ✅ 已合并部署 |
| #193 | S3 时间更新 + 年度总决赛待定 + 按钮状态 + 中文赛规修复 + 设计模式文档 | ✅ 已合并部署 |

### 1.3 设计模式文档

- `docs/notes/landing-date-driven-buttons.md` — 记录日期驱动按钮的实现模式和安全考量,方便未来赛事复用

## 2. 生产服务器验证(已执行并通过)

```bash
# SSH 到服务器
ssh hackforger@203.119.115.130

# 所有验证命令返回预期结果(详见"已完成的变更"表)

# 服务健康
curl -fsS http://127.0.0.1:3000/api/v1/version
# 返回: {"version":"..."} (正常)

# 生产域名可访问
curl -fsS -o /dev/null -w "%{http_code}\n" https://www.synnovator.com/lingang-2026
# 返回: 200
```

## 3. 后续可选操作

### 3.1 静态页资源移除(待创建 PR)

**目的**: 把落地页/排名页从 git 仓库移除,减少仓库体积。部署继续通过 rsync 完成。

**计划**:
1. `git rm -r --cached custom/public/assets/landing/` — 取消跟踪,本地文件保留
2. 更新 `.gitignore` 移除 `!/custom/public/assets/landing/` 例外规则
3. 创建 PR → 合并

**影响评估**:
- ✅ 部署不受影响(rsync 读文件系统,不依赖 git 跟踪)
- ✅ 本地文件保留,继续可编辑
- ⚠️ 需要确保其他开发者知道 landing 文件不再从 git 获取(需单独拷贝或从服务器拉取)

### 3.2 清理废弃 DICT 条目(可选)

生产服务器 DICT 中仍有 1 条废弃翻译:
- `"7月13日-7月20日": "Jul 13 – Jul 20"`(旧 W3 日期)

**影响**: 无功能影响(该 key 不再被任何元素引用)。可在下次部署时清理。

### 3.3 新赛事复用指南

当举办新赛事(如 2027 年)时:

1. **时间更新**: 修改卡片中的 `<span>` 日期 + `data-wave-start/end` 属性
2. **赛规弹窗**: 修改 `rule-stage-03` md 字符串 + 中英文 DICT
3. **按钮状态**: JS 自动根据新日期切换(无需改代码)
4. **参考文档**: `docs/notes/landing-date-driven-buttons.md`

## 4. 回滚方案

如果发现问题需要回滚:

```bash
# 1. 回滚到 PR #192/193 合并前的状态(不包含任何本次变更)
git checkout f0300494  # PR #191 后的稳定版本
bash deploy/ecs/redeploy.sh

# 2. 验证
curl -fsS https://www.synnovator.com/api/v1/version
```

## 5. 服务器信息

| 项目 | 值 |
|---|
| 服务器 | `hackforger@203.119.115.130` |
| 静态文件路径 | `/var/lib/hackforger/custom/public/` |
| Landing 路径 | `/var/lib/hackforger/custom/public/assets/landing/` |
| 二进制路径 | `/opt/hackforger/gitea` |
| 部署脚本 | `bash deploy/ecs/redeploy.sh` |
| Tailscale 预览 | `http://100.64.0.11:3000/lingang-2026` |

## 6. 文件清单(本次变更涉及)

```
修改:
  custom/public/assets/landing/index.html
  custom/public/assets/landing/s1-ranking.html
  custom/public/assets/landing/s2-ranking.html

新增(已入库):
  docs/notes/landing-date-driven-buttons.md
  docs/tests/e2e/reports/assets/*.png (多张截图)

新增(未入库,本交接文档):
  docs/ops/HANDOFF-landing-s3-update.md
```

## 7. 联系方式

- 开发者: daiming (FatNine)
- 分支: `fix/landing-s3-xuedoushan-banner` 和 `fix/landing-s2-ranking-reorder` 均已合并到 dev
- 设计模式文档: `docs/notes/landing-date-driven-buttons.md`
