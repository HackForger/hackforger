# HackForger 运维交接文档

> **目的**：让接手运维的同事在 30 分钟内掌握本系统所有"机器在哪、文件在哪、命令是什么"。
> **范围**：日常运维（部署、备份、回滚、故障恢复）。开发流程见 [`docs/notes/gitflow.md`](../notes/gitflow.md)。
> **最后实战验证**：2026-05-02

---

## 1. 架构一图

```
┌─────────────────────────────────────────────────────────────────────────┐
│  公网用户                                                               │
│      │                                                                  │
│      │  https://www.synnovator.com                                      │
│      ▼                                                                  │
│  ┌──────────────────────────────────────────────┐                       │
│  │  生产环境（华为云 ECS, IP 203.119.115.130）    │                       │
│  │  ─────────────────────────────────────       │                       │
│  │  Caddy :80/:443  (Let's Encrypt 自动证书)    │                       │
│  │   ↓ reverse_proxy                            │                       │
│  │  gitea :3000 (loopback)                      │                       │
│  │   ↕                                          │                       │
│  │  PostgreSQL 18 :5432 (loopback)              │                       │
│  │  forgejo-runner v12.9.0 (host mode)          │                       │
│  │  hackforger-backup.timer (每天 04:00)        │                       │
│  └──────────────────────────────────────────────┘                       │
│           │ ▲                                                           │
│           │ │ 1. 部署：redeploy.sh (binary + custom/ + restart)         │
│           │ │ 2. 同步回 Mac：sync-from-prod.sh (DB only)                │
│           │ │ 3. 备份：launchd 04:30 拉 .pgc → ~/Backups/hackforger/    │
│           ▼ │                                                           │
│  ┌──────────────────────────────────────────────┐                       │
│  │  开发机（Mac, h2oslabs workstation）          │                       │
│  │  ─────────────────────────────────────       │                       │
│  │  https://hackforger.inside.h2os.cloud         │                       │
│  │   (内网 Tailscale，Caddy → localhost:3000)    │                       │
│  │  仓库：/Users/h2oslabs/Workspace/hackforger   │                       │
│  │  本地 PG :5432 / gitea :3000 (开发用)        │                       │
│  │  备份：~/Backups/hackforger/                  │                       │
│  └──────────────────────────────────────────────┘                       │
└─────────────────────────────────────────────────────────────────────────┘
```

**关键定位**：
- **生产**就是 `203.119.115.130` / `www.synnovator.com`，**用户访问的就是它**。
- **Mac** 是开发 + 热备：开发在这做，e2e 测试在这做，生产数据每晚备份到这。**Mac 不是生产**。

---

## 2. 远程（生产）资产清单

### 服务器
| 项 | 值 |
|---|---|
| 公网 IP | `203.119.115.130` |
| 内网 IP | `10.0.0.5` |
| 域名 | `www.synnovator.com`（Aliyun DNS 控制台管理） |
| OS | Ubuntu 24.04.1 LTS |
| 配置 | 4 核 / 15 GiB / 80 GB |
| 云厂商 | 华为云 ECS |
| Hostname | `ecs-hackforger` |

### SSH 入口
```bash
ssh hackforger@203.119.115.130           # 用户 hackforger，key auth
# sudo 密码：在 Mac 的 .env 里，key 名 ECS_SUDO_PASS
```

### 关键路径
| 路径 | 内容 |
|---|---|
| `/opt/hackforger/gitea` | gitea binary（部署目标，113 MB ELF） |
| `/opt/hackforger/app.ini.tmpl` | app.ini 模板（bootstrap 用） |
| `/var/lib/hackforger/data/forgejo-repositories/` | git 仓库存储 |
| `/var/lib/hackforger/data/attachments/` | 附件上传 |
| `/var/lib/hackforger/data/avatars/` | 用户头像 |
| `/var/lib/hackforger/data/lfs/` | LFS 对象 |
| `/var/lib/hackforger/custom/conf/app.ini` | gitea 实例配置 |
| `/var/lib/hackforger/custom/templates/` | 模板覆盖（包含 head_navbar.tmpl 等） |
| `/var/lib/hackforger/custom/public/` | landing 页 + 静态资源 |
| `/var/lib/hackforger/log/` | gitea 日志 |
| `/var/lib/hackforger/pg-backups/` | 每晚 `hf-YYYYMMDD-HHMMSSZ.pgc`，保留 14 天 |
| `/var/lib/hackforger/.last-deploy` | 当前部署的 git SHA（preflight gate 用） |
| `/var/lib/postgresql/18/main/` | PG 数据目录 |
| `/etc/postgresql/18/main/postgresql.conf` | PG 主配置 |
| `/etc/caddy/Caddyfile` | Caddy 反代配置 |
| `/etc/forgejo-runner/config.yml` | Action runner 配置 |
| `/etc/systemd/system/{gitea,caddy,forgejo-runner,hackforger-backup}.{service,timer}` | 5 个 systemd 单元 |
| `/etc/sudoers.d/hackforger-backup` | 让 hackforger 读 PG 密码文件 |
| `/usr/local/bin/{caddy,forgejo-runner,hackforger-backup.sh}` | 第三方/自家 binary 和 backup 脚本 |
| `/root/.hackforger-pg-password` | PG `hackforger` role 密码（root only，hackforger 通过 sudoers 可读） |

### 服务清单（systemd）
| 服务 | 状态应该是 | 说明 |
|---|---|---|
| `postgresql@18-main.service` | active enabled | PG18 实例 |
| `gitea.service` | active enabled | gitea web :3000（loopback） |
| `caddy.service` | active enabled | 反代 :80/:443 + Let's Encrypt |
| `forgejo-runner.service` | active enabled | Action runner（host mode） |
| `hackforger-backup.timer` | active enabled | 每天 04:00 跑 dump |

一行检查：
```bash
ssh hackforger@203.119.115.130 'for s in postgresql@18-main gitea caddy forgejo-runner hackforger-backup.timer; do printf "%-30s %s\n" "$s" "$(systemctl is-active $s)"; done'
```

### 安全组（华为云控制台）
| 入站端口 | 来源 | 用途 |
|---|---|---|
| `22` TCP | `0.0.0.0/0` | SSH |
| `80` TCP | `0.0.0.0/0` | Caddy（HTTP→HTTPS 跳转 + LE HTTP-01 续证） |
| `443` TCP | `0.0.0.0/0` | Caddy HTTPS |

不要再开其他端口。PG/gitea/runner 都绑 `127.0.0.1`，进不来。

### DNS（阿里云控制台）
- 域名：`synnovator.com`，NS 是 `dns13.hichina.com / dns14.hichina.com`
- A 记录：`www.synnovator.com → 203.119.115.130`，TTL 60s
- 顶级域 `synnovator.com → 47.116.160.70`（**别动**，那是另一台阿里云机器，跑共享 infra，与 HackForger 无关）

---

## 3. 本地（Mac）资产清单

### 主仓库
| 项 | 值 |
|---|---|
| 路径 | `/Users/h2oslabs/Workspace/hackforger` |
| 默认分支 | `v0.1-dev/hackforger` |
| 远端 | `git@github.com:HackForger/hackforger.git` (origin) + Forgejo upstream |
| 凭证文件 | `.env`（**gitignored**，mode 600） |

### Mac 端服务
| 进程 | 启动方式 | 端口 / 路径 |
|---|---|---|
| Caddy（生产域名 inside.h2os.cloud） | launchd `com.h2os.caddy` | :80/:443，配置 `~/.config/caddy/` |
| gitea web | 手动 / `bash scripts/restart-gitea.sh` | :3000，binary 在仓库根 `./gitea` |
| gitea web (test instance) | `bash scripts/restart-gitea-test.sh` | :3001，custom path `/tmp/hackforger-test-custom` |
| Postgres | Homebrew | :5432，DB 名 `hackforger` 和 `hackforger_test` |
| forgejo-runner（生产） | launchd `com.h2os.forgejo-runner` | 注册到本机 :3000 |
| forgejo-runner（test） | 手动 | 注册到本机 :3001，配置 `~/.config/forgejo-runner-test/` |
| 备份拉取（从云端） | launchd `com.h2os.hackforger-backup-pull` | 每天 04:30 跑 `scripts/backup-pull.sh` |

### Mac 备份目录
```
~/Backups/hackforger/
├── db/             # 14 天 .pgc 镜像（从云端 rsync）
├── data/           # 仓库 / 附件 / 头像 镜像
├── custom/         # 云端 custom/ 目录镜像
├── backup.log      # 拉取日志
└── launchd.log     # launchd stdout/stderr
```

### .env 里有什么
```bash
PGHOST=localhost
PGPORT=5432
PGDATABASE=hackforger
PGUSER=hackforger
PGPASSWORD=...                      # 本地 PG hackforger role 密码
HACKFORGER_ADMIN_PASSWORD=...       # gitea 管理员密码（生产 + Mac 都一样，跟随同步）
HACKFORGER_TEST_ADMIN_PASSWORD=...  # 测试实例 :3001 的管理员密码
ECS_SUDO_PASS=...                   # 远程 hackforger 用户 sudo 密码
FORGEJO_TOKEN=...                   # Mac 实例的 admin API token
GITHUB_TOKEN=...                    # gh CLI 用
```

Mode 必须 `600`：`chmod 600 .env`

---

## 4. 三个最常用操作

### 4.1 部署新版本到生产
```bash
cd /Users/h2oslabs/Workspace/hackforger
git checkout v0.1-dev/hackforger && git pull
bash deploy/ecs/redeploy.sh
```

`redeploy.sh` 自动做：
1. **preflight 检查**（确认所有要部署的 commit 都有签字的 smoke-test 报告，详见 [`docs/notes/gitflow.md`](../notes/gitflow.md)）
2. 用 docker 交叉编译 linux/amd64 binary（5–8 分钟，qemu 仿真）
3. **rsync `custom/templates/` + `custom/public/`** 到云端 ⚠️ 这一步**绝对不能省** —— 否则模板/landing 改动不会生效
4. scp binary → install → restart gitea
5. 更新 `/var/lib/hackforger/.last-deploy` 标记
6. 公网烟测（API + navbar 文案 + HTTPS 200）

紧急 hotfix 跳过 preflight：
```bash
bash deploy/ecs/redeploy.sh --skip-preflight
# 用了之后必须事后补 smoke-test 报告
```

### 4.2 把生产数据拉回 Mac（用于本地复现 / 调试）
```bash
bash scripts/sync-from-prod.sh --confirm
```

会：触发云端 fresh `pg_dump` → 拉回 Mac → 停 Mac gitea → restore → 重启 Mac gitea。**Mac 上的本地 DB 改动会被覆盖**。

只同步 DB，不同步 git 仓库 / 附件。后者要单独：
```bash
rsync -az --delete hackforger@203.119.115.130:/var/lib/hackforger/data/ ~/Backups/hackforger/data/
# 然后手动复制需要的部分到 ./data/
```

### 4.3 看每晚备份是否还在跑
```bash
# Mac 端
ls -lt ~/Backups/hackforger/db/ | head -5     # 最新一条应在 24h 以内
tail -20 ~/Backups/hackforger/backup.log
launchctl print "gui/$UID/com.h2os.hackforger-backup-pull" | grep -E "last_exit|run_count"

# 云端
ssh hackforger@203.119.115.130 'systemctl list-timers hackforger-backup.timer'
ssh hackforger@203.119.115.130 'ls -lh /var/lib/hackforger/pg-backups/ | tail -5'
```

预期：每天 04:00 云端 dump，04:30 Mac 拉。两端各保留 14 天。

---

## 5. 故障恢复

### 5.1 生产 502 / gitea 挂了
```bash
ssh hackforger@203.119.115.130 'systemctl status gitea caddy --no-pager; journalctl -u gitea -n 50 --no-pager'
# 多数情况下重启即可：
ssh hackforger@203.119.115.130 "echo \$ECS_SUDO_PASS | sudo -S systemctl restart gitea"
```

### 5.2 Caddy TLS 证书过期 / 拉不到
LE 证书 90 天有效，Caddy 自动续期。如果挂了：
```bash
ssh hackforger@203.119.115.130 'journalctl -u caddy -n 100 --no-pager | grep -i "acme\|cert\|error"'
# 通常原因：80 端口被防火墙挡了 → 检查华为云安全组
nc -z 203.119.115.130 80   # 应回 OPEN
```

### 5.3 部署后 navbar / landing 没更新
**99% 是 `custom/` 没 rsync**。重新跑：
```bash
bash deploy/ecs/redeploy.sh
```
（不是 `scp gitea && systemctl restart` —— 那种老办法只换 binary，custom/ 不动）

### 5.4 PG 容量满
```bash
ssh hackforger@203.119.115.130 'df -h /var/lib/postgresql/'
# 如果接近 80%，先看是不是 pg_backups/ 该清的没清：
ssh hackforger@203.119.115.130 'du -sh /var/lib/hackforger/pg-backups/'
# 备份脚本应该自动 14 天轮转，如果失效手动清：
ssh hackforger@203.119.115.130 'find /var/lib/hackforger/pg-backups/ -mtime +14 -delete'
```

### 5.5 完全重建生产实例（灾难恢复）
1. 在华为云开新 ECS，挂 SSH key
2. 给 Mac 加上新 IP 的 ssh 入口
3. 在新机器跑 `bash /tmp/ecs/ecs-bootstrap.sh`（先 scp `deploy/ecs/` 整个目录过去）
4. 跑 `bash deploy/ecs/migrate-data.sh --confirm` 从 Mac 灌数据回去
5. Aliyun DNS 把 `www.synnovator.com` A 记录改到新 IP
6. 全程参考 [`deploy/ecs/cutover-runbook.md`](../../deploy/ecs/cutover-runbook.md)

---

## 6. 凭证 / 安全

| 凭证 | 在哪 | 怎么换 |
|---|---|---|
| Mac `.env` 全部 | `/Users/h2oslabs/Workspace/hackforger/.env`（mode 600，gitignored） | 直接编辑 |
| 云端 `hackforger` 用户 sudo 密码 | OS 级，与 Mac `.env` 的 `ECS_SUDO_PASS` 同步 | `ssh ... 'passwd'` 后更新 .env |
| 云端 PG `hackforger` role 密码 | `/root/.hackforger-pg-password`（mode 600） | `ALTER ROLE hackforger PASSWORD '...'` 后更新文件 + `app.ini` |
| 云端 SSH 公钥 | `/home/hackforger/.ssh/authorized_keys` | 加 / 删行 |
| Logto OAuth 密钥 | Logto 控制台，gitea 这边在 DB 的 `login_source` 表 | 改了要重启 gitea |
| TLS 证书 | Caddy 自动管理（`/var/lib/caddy/.local/share/caddy/`） | 自动续期，不用动 |

⚠️ **不要 commit `.env`** —— `.gitignore` 已经保护，但人工 `git add .env` 会绕过。如果不小心提交了，立刻 `git rm --cached .env && git commit && git push --force`，然后**轮换所有凭证**（已经被 GitHub indexer 看到了）。

---

## 7. 重要文件 / 目录索引

| 你想做的事 | 看哪里 |
|---|---|
| 改部署脚本 | `deploy/ecs/*.sh` |
| 改 systemd 单元 | `deploy/ecs/systemd/*` |
| 改 Caddy 配置 | `deploy/ecs/caddy/Caddyfile.tmpl` |
| 改 app.ini 模板 | `deploy/ecs/app.ini.tmpl`（注意：改了模板要 bootstrap 时才会生成新文件，正在跑的实例的 `/var/lib/hackforger/custom/conf/app.ini` 不会自动更新） |
| 改 gitflow 流程 | `docs/notes/gitflow.md` |
| 看历史背景 | `docs/superpowers/specs/2026-05-01-prod-cloud-migration-design.md`（设计文档） + `docs/superpowers/plans/2026-05-01-prod-cloud-migration.md`（实施计划） |
| 写 smoke-test 报告 | `docs/tests/e2e/reports/`，模板看现有的 `2026-05-02-pr-137-oauth-copy.md` |
| Forgejo 上游文档 | https://forgejo.org/docs/ |

---

## 8. 别动这些

- **47.116.160.70**（阿里云）：跑共享 infra（traefik、私有 docker registry、oneauth、keycloak、syntrust 等）。**HackForger 不在这里**，但 `synnovator.com` 顶级域指向它。
- **CDN / WAF**（如果上了）：本系统目前**没用** CDN，Caddy 直接面对公网。如果未来加了，要更新 Caddy 配置和健康检查脚本。
- **Mac 上的 Caddy launchd `com.h2os.caddy`**：不要 unload，会断 inside.h2os.cloud 内网域名。

---

## 9. 谁能帮你

- 业务相关 / 紧急生产事故：先 ping 项目维护者
- 域名 / DNS / Aliyun 控制台：找运维（持有 Aliyun 账号的人）
- 华为云 ECS / 安全组：找运维（持有华为云账号的人）
- Logto 后台：找认证组
- GitHub repo 权限：仓库 admin

---

## 10. 一句话兜底

如果你只记得一句话：
**"任何对生产的代码改动都跑 `bash deploy/ecs/redeploy.sh`，永远不要手动 scp + restart"**。
那个脚本是这次首次上线踩过坑后总结出来的，包含了所有必要的步骤（特别是 `custom/` rsync）。
