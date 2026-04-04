# Forgejo Actions: Token 权限模型

> 所有通过 `workflow.Dispatch()` 触发的 Forgejo Actions 都必须遵循此权限模型。

## 核心概念

Forgejo Actions 中的 `github.token`（即 `${{ github.token }}`）是为**触发者 (trigger user)** 自动生成的临时 token。它的权限继承自触发者在目标 repo 上的实际权限。

```
workflow.Dispatch(ctx, inputGetter, repo, doer)
                                          ^^^^
                                     此 doer 决定 github.token 的身份和权限
```

## 权限矩阵

| 操作 | 所需权限 | `github.token` 内置支持 | 说明 |
|------|---------|:---:|------|
| git clone (公开 repo) | 无 | Y | 任何人 |
| git clone (私有 repo) | Read | Y | token 对自身 repo 有读权限 |
| git push | Write | Y | **action token 对触发 workflow 的 repo 有内置写权限** |
| 创建 PR (API) | Write | Y | 同上 |
| **合并 PR (API)** | **Write + `CanWrite(TypeCode)`** | **取决于 doer** | **需要 doer 在 repo 的 team 中有 Write 权限** |
| 创建 Release (API) | Write | Y | 同上 |
| 删除分支 | Write | Y | 同上 |
| 管理 repo 设置 | Admin | N | 通常不允许 |

### 关键区别：git push vs merge PR

- **git push**: action token 对触发它的 repo 有**内置写权限**，不检查 doer 的 team 权限
- **merge PR**: 走 `IsUserAllowedToMerge()` → 检查 `perm.CanWrite(unit.TypeCode)` → 这依赖 doer 在 org/team 中的角色

```go
// services/pull/merge.go:527
if (p.CanWrite(unit.TypeCode) && pb == nil) || (pb != nil && git_model.IsUserMergeWhitelisted(...)) {
    return true, nil  // 允许 merge
}
return false, nil     // 拒绝 merge → 405 ErrUserNotAllowedToMerge
```

## HackForger 中的权限场景

### Hackathon Org 的角色分布

| 角色 | Org 关系 | Team | repo 写权限 | merge 权限 |
|------|---------|------|:---------:|:---------:|
| 组织者 (hackforger) | Owner | Owners | Y | Y |
| 评委 (judge_carol) | 不在 org | — | N | N |
| 参赛者 (hacker_eve) | Member (注册时加入) | 无 team | N | N |

### 错误示例

```go
// ❌ 错误：用 hacker (提交者) 作为 dispatcher
workflow.Dispatch(ctx, inputGetter, trackRepo, hacker)
// → github.token 身份 = hacker
// → git push 成功 (内置写权限)
// → merge PR 失败: hacker 不在 Writers team → 405
```

### 正确做法

```go
// ✅ 正确：用 repo owner (org admin) 作为 dispatcher
baseRepo.LoadOwner(ctx)
workflow.Dispatch(ctx, inputGetter, baseRepo, baseRepo.Owner)
// → github.token 身份 = org owner
// → git push 成功
// → merge PR 成功: owner 有完整 repo 权限
```

## 规则总结

### 选择 Dispatcher 的原则

1. **只读操作** (clone, fetch, 读 API)：任何用户都行
2. **写入操作** (push, 创建 PR/Issue)：action token 内置支持，任意 repo 成员即可
3. **管理操作** (merge PR, 创建 Release, 修改 repo 设置)：**必须使用有对应权限的用户**
   - 通常是 repo owner 或 org admin
   - 通过 `baseRepo.LoadOwner(ctx)` 获取后传入 `workflow.Dispatch`

### Checklist (新建 workflow 时必查)

- [ ] workflow 中是否调用了 merge PR API？→ dispatcher 必须是 repo owner
- [ ] workflow 中是否创建了 Release？→ dispatcher 必须有 Write 权限
- [ ] workflow 中是否修改了 repo 设置？→ dispatcher 必须是 Admin
- [ ] workflow 中是否只做了 push + 创建 PR？→ 任意用户的 action token 即可
- [ ] macOS runner 上的 shell 命令是否用了 GNU 扩展 (如 `sed` 的 `\?`)？→ 改用 POSIX 兼容语法

### macOS Runner 兼容性

Host 模式 runner 在 macOS 上运行时，shell 工具是 BSD 版本，不是 GNU 版本：

```bash
# ❌ GNU sed (Linux)
echo "http://localhost" | sed 's|https\?://||'   # \? 不被识别

# ✅ POSIX shell (通用)
SERVER="${SERVER#http://}"
SERVER="${SERVER#https://}"
```

## 参考

- Forgejo merge 权限检查: `services/pull/merge.go:517` `IsUserAllowedToMerge()`
- Forgejo PR merge 检查: `services/pull/check.go:68` `CheckPullMergeable()`
- Action token 内置 repo 写权限: Forgejo 源码中 action task token 创建逻辑
- HackForger workflow dispatch: `services/hackforger/hackathon.go` `triggerSubmissionIndexUpdate()`
