# Orphan-Binary State on Dev Macs

When you see this on `git push` to `localhost:3000` (or `:3001`):

```
remote: ./hooks/pre-receive.d/gitea: line 3: <repo>/gitea: No such file or directory
remote: error: hook declined to update refs/heads/...
! [remote rejected] ... (pre-receive hook declined)
```

…the dev instance is in **orphan-binary state**. This note explains what it means, how to fix it, and how atomic replacement prevents the same gap in managed releases.

## What it means

The `gitea` binary at `<repo>/gitea` was deleted while the `gitea web` process was still running. Common causes:

- `make clean` while the server was up
- A manual `rm gitea`
- A rebuild that crashed mid-way (rare)
- Switching between worktrees that reuse the path

Unix `unlink()` only removes the **directory entry** that names the inode. The inode itself stays alive as long as some process still holds it open. `gitea web` mapped the binary into its address space at startup (`txt` segment in `lsof`), so it keeps that inode alive and continues serving HTTP normally. But hooks are short-lived child processes — they invoke `exec()` on the **path** `<repo>/gitea`, which the kernel resolves by name. The name no longer exists, so `exec()` fails with `ENOENT`. Every git push fails. HTTP traffic looks fine.

## Self-diagnosis

```bash
PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t)
lsof -p "$PID" | awk '/txt.*\/gitea$/{print $NF}'
ls -la "$(git rev-parse --show-toplevel)/gitea"
```

If `lsof` prints a path and `ls` says `No such file or directory` on that same path → orphan state.

(On macOS, `lsof` does **not** append `(deleted)` to the path the way Linux does — you have to compare against the filesystem yourself.)

## Fix

```bash
bash scripts/restart-gitea.sh        # for the main instance (port 3000)
bash scripts/restart-gitea-test.sh   # for the test instance (port 3001)
```

The restart script builds first, asserts the binary exists, then kills + restarts. After a clean run, pushes work again.

If your `scripts/restart-gitea.sh` prints an `⚠ Orphan-binary state detected` banner before `[1/4] Building backend`, that's the diagnostic doing its job — it just confirmed why your last push failed, and the build below will fix it.

## Why managed releases use atomic replacement

A release process should install a new binary through an atomic replacement on
the same filesystem. The path then always resolves to either the complete old
inode or the complete new inode. The running process may keep the old inode
open, while new hook processes resolve the new one; there is no missing-path
window that can produce `ENOENT`.

## Prevention

- Don't `rm gitea` while `gitea web` is running. If you must, run `bash scripts/restart-gitea.sh` immediately afterwards.
- `make clean` removes the binary (Makefile target `clean-no-bindata` deletes `$(EXECUTABLE)`) — restart afterwards.
- If unsure, check with the self-diagnosis snippet above.

## See also

- `scripts/restart-gitea.sh` and `scripts/restart-gitea-test.sh` — the recovery commands
- your environment's reviewed release runbook — it should use same-filesystem atomic replacement
