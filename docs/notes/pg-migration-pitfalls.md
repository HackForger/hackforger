# PostgreSQL migration pitfalls (READ before writing new HackForger migrations)

HackForger 默认 backing store 自 2026-04-30 起切换为 **PostgreSQL** 用于生产，dev 仍可用 SQLite。

历史上多数 HackForger migrations 是在 SQLite 下开发测试的，存在 **PG 不兼容的 SQL 写法**，在 PG 下表现为：

- 部分 migration 标记为成功，但 seed 数据未持久化（query 静默失败 + 错误处理吞没）
- 部分 migration 在 PG 下硬错（PRAGMA / boolean 比较）

本文档记录已确认的 pitfalls + 写新 migration 的 checklist。

---

## 已发现的具体 PG 不兼容问题（2026-05-01 audit）

### Pitfall 1：Backtick identifier quoting

PG 用双引号 `"key"` 引用标识符；MySQL/SQLite 也接受 backtick `` `key` ``。**XORM 的 `Where(rawSQL)` 不会自动转换 dialect**，所以原始 backtick 字符串在 PG 下直接 SQL 语法错误。

**已发现实例：**

- `models/forgejo_migrations/v14g_hackforger-phase2-tables.go:71`
  ```go
  has, err := x.Where("`key` = ?", s.Key).Exist(new(v14gHackforgerSetting))
  ```
  → PG 下 query error，导致 `reputation.weights` / `reputation.tiers` seed inserts 不执行
  → 修复：`x.Where("\"key\" = ?", s.Key)` 或改用 XORM struct-based query（`.Where(builder.Eq{"key": s.Key})`）

注：Forgejo 上游 v14a* migrations 也用 backticks，但他们用的是 `x.Table("\`...\`")` + 链式 API，XORM 在那里能正确处理 dialect。**只有 raw SQL 字符串里的 backtick 出问题**。

### Pitfall 2：Boolean as integer

PG 对 BOOLEAN 列严格区分 `TRUE`/`FALSE`，不接受 `0`/`1`；SQLite 把 boolean 当 INTEGER 存。

**已发现实例：**

- `models/forgejo_migrations/v14j_allow-multiple-judging-phases.go:14`
  ```go
  _, err := x.Exec("UPDATE phase_type SET is_unique = 0 WHERE activity_kind = 'hackathon' AND key = 'judging'")
  ```
  → PG 下 type error: `column "is_unique" is of type boolean but expression is of type integer`
  → 修复：`SET is_unique = false`

注：v14b_hackforger-phase-integration.go:40 的 `is_published = 1` **不算 bug**——该列在 v14b 显式声明为 `INTEGER NOT NULL DEFAULT 0`，赋整数正确。

### Pitfall 3：SQLite-specific PRAGMA

`PRAGMA table_info(...)` 是 SQLite 专属命令，PG 用 `information_schema.columns` 或 `pg_attribute`。

**已发现实例：**

- `models/forgejo_migrations/v14h_hackforger-judge-system.go:82`
  ```go
  tableInfo, err := x.QueryString("PRAGMA table_info(hackathon_submission)")
  ```
  → PG 下 syntax error
  → 修复：用 cross-dialect 检测，如 `x.IsTableExist(...)` + `x.IsColumnExist(...)`，或者根据 `setting.Database.Type` 分支

### Pitfall 4：In-function struct + Insert seed pattern (suspected)

`v14b_hackforger-phase-tables.go` 的 11 行 phase_type seed inserts 在 PG 下未持久化，但代码里没明显的 backtick / boolean=int 问题。怀疑：

- XORM 对**函数内 anonymous struct**（每次 migration 调用都重新定义类型）在 PG dialect 下的元信息缓存有问题
- 或者 BOOLEAN 列的 default + struct 字段有冲突

**未根因**——临时 workaround：见 `RESTORE.md` 的「Post-migrate manual seed」章节，重建 PG 后跑手工 SQL 补 seed。

---

## 写新 migration 的 checklist

写新 `models/forgejo_migrations/v14*.go` 时按这个清单 self-review：

- [ ] **不用 backtick**：raw SQL 里的 identifier 用双引号 `"col"`，或者用 XORM struct-based query API（`builder.Eq{...}`）
- [ ] **boolean 用 boolean 字面量**：`SET col = true/false`，不要 `= 1/0`
- [ ] **不用 PRAGMA / SQLite-specific 函数**：用 XORM 的跨 dialect API（`IsTableExist`, `IsColumnExist`）
- [ ] **seed insert 后必须验证持久化**：写完 migration，**到 PG 里 SELECT 一次** 看 row 真的进去了
- [ ] **PG + SQLite 双 driver 跑**：本地至少两个 dialect 各跑一次，对比 row count
- [ ] **Migration 标记成功 ≠ 数据真的进去了**：错误处理可能静默吞掉 query error，看 logs 也看 DB 状态

---

## 跨 dialect 通用模式（推荐）

### 安全的"如果不存在则插入"

```go
// ❌ NOT PG-safe
has, err := x.Where("`key` = ?", k).Exist(new(MySetting))

// ✅ PG-safe
has, err := x.Where("\"key\" = ?", k).Exist(new(MySetting))

// ✅ PG-safe + dialect-agnostic
has, err := x.Where(builder.Eq{"key": k}).Exist(new(MySetting))
```

### 安全的 boolean update

```go
// ❌ NOT PG-safe
x.Exec("UPDATE foo SET is_unique = 0 WHERE ...")

// ✅ PG + SQLite 都接受
x.Exec("UPDATE foo SET is_unique = false WHERE ...")
```

### 安全的列存在检测

```go
// ❌ SQLite-only
tableInfo, err := x.QueryString("PRAGMA table_info(my_table)")

// ✅ Cross-dialect
hasCol, err := x.IsColumnExist("my_table", "my_col")
```

---

## 历史背景 / 为什么之前没发现

HackForger 历来用 SQLite 开发 + 内部测试。首次迁到 PG 是 2026-04-30 prod-init clean-slate 的一部分（详见 `docs/superpowers/specs/2026-04-30-prod-init-clean-slate-design.md`）。

迁完才发现：
- `phase_type` 11 行 seed 没了 → manage UI 显示空，admin 创建 phase 失败
- `hackforger_setting.reputation.{weights,tiers}` 2 行 seed 没了 → reputation 模块靠 runtime fallback 维持功能，但 admin UI 无法编辑权重

两者都通过手工 SQL 补回（见 `RESTORE.md`）。本文档为未来 PG 重建 / 测试 DB 初始化的人提个醒，避免重蹈覆辙。

---

## ⚠ 关键发现：fresh DB 上 migrations 的 Upgrade 函数**永不执行**

排查 phase_type seed bug 时调研 Forgejo migration runner 源码（`models/forgejo_migrations/migrate.go:151-179`），发现一个根本性机制：

```go
if len(inDBMigrationIDs) == 0 && freshDB {
    // During startup on a new, empty database, and during integration tests, we rely
    // only on `SyncAllTables` to create the DB schema. No migrations can be applied
    // because `SyncAllTables` occurs later in the initialization cycle. We mark all
    // migrations as complete up to this point and only run future migrations.
    for _, migration := range orderedMigrations {
        err := recordMigrationComplete(x, migration)  // ← only records, no Upgrade()
        ...
    }
}
```

### 这意味着

**任何 migration 的 `Upgrade` 函数对 fresh DB（空 `forgejo_version` 表）都不执行。** 所有 v14a~v14k 的 Upgrade 在新 PG / 新 SQLite 实例启动时都被**跳过**，但 `forgejo_migration` 表里却被 mark 成 complete。Schema 由 `SyncAllTables` 单独创建（仅建表，不运行 Upgrade 里的 seed/data 操作）。

### 影响

任何依赖 migration `Upgrade` 函数做 **seed 数据 / 反查初始化** 的场景，**对 fresh DB 都失效**：

- v14b 的 11 行 phase_type seed inserts → fresh DB 上不跑 → 表空
- v14g 的 2 行 hackforger_setting seed inserts → fresh DB 上不跑 → 表空
- v14j 的 `UPDATE phase_type SET is_unique = 0 ...` → fresh DB 上不跑（且就算跑也是 PG 类型错）

### 为什么之前 SQLite 上有 seed？

历史原因——HackForger 早期开发实例可能在 fresh-DB-skip 逻辑被引入 Forgejo 之前就跑过 migration（那时 Upgrade 函数是会执行的）。或者旧版 Forgejo 没有 freshDB 短路。一旦 seed 被插入，后续所有 sync 都保留它，所以从来没暴露过。

### 这对源码 fix 的实际意义

本文档下面列出的 fix（v14g backtick → builder.Eq 等）**只能影响"已经有部分 migration 历史的 DB 升级"路径**——即从某版已有 v14a..v14k 但缺 v14g 的 DB upgrade 上来。

**对 fresh PG / fresh SQLite，源码 fix 无效。** 真正的 fix 路径只有两种：
1. **手工 seed**（`RESTORE.md` 的 Post-migrate manual seed 节）—— 当前生产用的
2. **加 post-`SyncAllTables` 初始化 hook**（Forgejo 上游层面的改动，工作量大；目前不做）

源码 fix 的价值是 **代码质量 / 上游正确性 / 防止未来 freshDB 行为改变后再踩**，不是实际解决 fresh DB 的 seed 问题。

---

## 相关

- `RESTORE.md` 的「Post-migrate manual seed」节（`.claude/worktrees/dev/data-snapshot/RESTORE.md`）
- Spec: `docs/superpowers/specs/2026-04-30-prod-init-clean-slate-design.md`
- Plan: `docs/superpowers/plans/2026-04-30-prod-init-clean-slate.md`
