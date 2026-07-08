# HANDOFF: Landing Page S3 时间更新 + 日期驱动按钮

> 交接给部署同事:本文档描述本次落地页修改的内容、服务器状态、以及后续操作。

## 1. 本次修改概要

### 已提交 PR
| PR | 内容 | 状态 |
|---|---|---|
| #192 | S2 排名页按得分重排 + S2 弹窗删 3 栏 | ✅ 已合并到 dev |
| #193 | S3 轮播 banner + S1 排名页重排 + 弹窗删 3 栏 | ✅ 已合并到 dev |
| **(本文档)** | S3 赛程时间更新 + 年度总决赛改为待定 + 日期驱动按钮 | 🔴 **本 PR 待创建** |

### 核心改动(本handoff包含)

1. **S3 赛程时间更新**(依据运营最新时间表):
   - W1: `6月25日-7月2日`(不变)
   - W2: `7月4日-7月11日` → `7月4日-7月23日`
   - W3: `7月13日-7月20日` → `7月24日-8月4日`
   - W4: `7月28日-7月30日` → `8月12日-8月14日`

2. **年度总决赛**: `#7月底/8月初` → `待定`(中英文同步)

3. **中英文赛规弹窗**: 时间表 + S3 Stage Descriptions + DICT 翻译同步更新

4. **日期驱动按钮状态**(新特性):
   - 新增 JS:根据客户端当前日期自动切换按钮(未开始/进行中/已结束)
   - 标记: `data-wave-start` / `data-wave-end` 属性
   - 安全:服务端 Phase 独立校验,客户端仅 UX

5. **文档**: `docs/notes/landing-date-driven-buttons.md` 记录设计模式

## 2. 服务器当前状态

### 已部署(`d55bd411`,PR #192/193 合并内容)
- ✅ S3 轮播 banner 图片(`轮播图3.webp`, `轮播图4.webp`)
- ✅ S3 卡片新日期(`7月24日-8月4日`, `8月12日-8月14日`)
- ✅ S2 排名页按得分重排
- ✅ S1/S2 弹窗删 3 栏(保留:团队/作者、系统项目名、项目简介)
- ✅ 日期驱动按钮 JS(基础版本)

### 未部署(本handoff新增)
- ❌ 中文赛规弹窗**主时间表**更新
- ❌ 年度总决赛改为"待定"(卡片已更新,但赛规弹窗未更新)
- ❌ DICT 翻译条目清理(旧条目仍残留在服务器)
- ❌ 设计模式文档(`docs/notes/landing-date-driven-buttons.md`)

### 服务器验证命令
```bash
# SSH 到服务器
ssh hackforger@203.119.115.130

# 检查部署版本
cat /var/lib/hackforger/.last-deploy
# 预期: d55bd411...(当前) → 部署后变为新 SHA

# 检查文件是否在服务器上
grep "7月24日-8月4日" /var/lib/hackforger/custom/public/assets/landing/index.html
# 预期: 返回 1 行

grep "待定" /var/lib/hackforger/custom/public/assets/landing/index.html
# 预期: 返回多行(部署前也可能有,因为 PR #193 已包含部分"待定")

# 服务健康检查
curl -fsS http://127.0.0.1:3000/api/v1/version
# 预期: 返回版本 JSON
```

## 3. 部署步骤(给部署同事)

### 方式 A:从我的分支部署(推荐)

```bash
# 1. 切到最新代码
git fetch origin
git checkout fix/landing-s3-xuedoushan-banner
git pull origin fix/landing-s3-xuedoushan-banner

# 2. 构建 + 部署(标准流程)
bash deploy/ecs/redeploy.sh

# 3. 验证
curl -fsS https://www.synnovator.com/api/v1/version
```

### 方式 B: 创建新 PR 后合并部署

```bash
# 1. 基于 fix/landing-s3-xuedoushan-banner 创建新 PR
gh pr create --base v0.1-dev/hackforger --head fix/landing-s3-xuedoushan-banner \
  --title "feat(landing): S3 时间更新 + 年度总决赛待定 + 日期驱动按钮" \
  --body "..."

# 2. Review + 合并

# 3. 部署
git checkout v0.1-dev/hackforger
git pull
bash deploy/ecs/redeploy.sh
```

### 方式 C: 不部署(仅合并 PR,暂不上线)

如果运营还需要修改,可以先不部署。代码合并到 dev 后等待后续统一部署。

## 4. 部署后验证清单

```bash
# 1. 服务健康
curl -fsS https://www.synnovator.com/api/v1/version

# 2. 落地页可访问
curl -fsS -o /dev/null -w "%{http_code}\n" https://www.synnovator.com/lingang-2026

# 3. 中文赛规弹窗(主时间表已更新)
ssh hackforger@203.119.115.130 'grep "6月25日至8月4日" /var/lib/hackforger/custom/public/assets/landing/index.html'
# 预期: 至少 1 行

# 4. 年度总决赛(待定)
ssh hackforger@203.119.115.130 'grep "时间待定" /var/lib/hackforger/custom/public/assets/landing/index.html | head -3'
# 预期: 3 行(AI训练营/总决赛/颁奖)

# 5. 按钮状态 JS 存在
ssh hackforger@203.119.115.130 'grep "data-wave-start" /var/lib/hackforger/custom/public/assets/landing/index.html | head -1'
# 预期: 至少 1 行

# 6. 无旧日期残留(应为 0)
ssh hackforger@203.119.115.130 'grep "7月13日-7月20日" /var/lib/hackforger/custom/public/assets/landing/index.html | wc -l'
# 预期: 0 (DICT 中的旧条目是翻译 key,不影响功能,可后续清理)
```

## 5. 回滚方案

如果部署后发现问题:

```bash
# 1. 回滚到上一个已知良好版本
git checkout d55bd411  # PR #192/193 合并版本
bash deploy/ecs/redeploy.sh

# 2. 验证
curl -fsS https://www.synnovator.com/api/v1/version
```

## 6. 文件清单(本次变更)

```
custom/public/assets/landing/index.html  (+55 行, -11 行)
docs/notes/landing-date-driven-buttons.md  (新增, 131 行)
docs/tests/e2e/reports/assets/
  ├── landing-s3-updated.png  (新增截图)
  ├── rules-full-cn.png  (新增截图)
  └── rules-s3-detail.png  (新增 screenshot)
docs/ops/HANDOFF-landing-s3-update.md  (本文档)
```

## 7. 联系方式

- 开发者: daiming (FatNine)
- 分支: `fix/landing-s3-xuedoushan-banner`
- 服务器: `hackforger@203.119.115.130`
- 内部预览: `http://100.64.0.11:3000/lingang-2026`(Tailscale only)
