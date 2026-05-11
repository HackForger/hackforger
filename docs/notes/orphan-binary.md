# Orphan-Binary State on Dev Macs

When you see this on `git push` to `localhost:3000` (or `:3001`):

```
remote: ./hooks/pre-receive.d/gitea: line 3:/Users/h2oslabs/Workspace/hackforger/gitea: No such file or directory
remote: error: hook declined to update refs/heads/...
! [remote rejected] ... (pre-receive hook declined)
```

…the dev instance is in **orphan-binary state**. This note explains what it means, how to fix it, and why production is immune.

## What it means

The `gitea` binary at `/Users/h2oslabs/Workspace/hackforger/gitea` was deleted while the `gitea web` process was still running. Common causes:

- `make clean` while the server was up
- A manual `rm gitea`
- A rebuild that crashed mid-way (rare)
- Switching between worktrees that reuse the path

Unix `unlink()` only removes the **directory entry** that names the inode. The inode itself stays alive as long as some process still holds it open. `gitea web` mapped the binary into its address space at startup (`txt` segment in `lsof`), so it keeps that inode alive and continues serving HTTP normally. But hooks are short-lived child processes — they invoke `exec()` on the **path** `/Users/.../gitea`, which the kernel resolves by name. The name no longer exists, so `exec()` fails with `ENOENT`. Every git push fails. HTTP traffic looks fine.

## Self-diagnosis

```bash
PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t)
lsof -p "$PID" | awk '/txt.*\/gitea$/{print $NF}'
ls -la /Users/h2oslabs/Workspace/hackforger/gitea
```

If `lsof` prints a path and `ls` says `No such file or directory` on that same path → orphan state.

(On macOS, `lsof` does **not** append `(deleted)` to the path the way Linux does — you have to compare against the filesystem yourself.)

## Fix

```bash
bash scripts/restart-gitea.sh        # for the main instance (port 3000)
bash scripts/restart-gitea-test.sh   # for the test instance (port 3001)
```

The restart script builds first, asserts the binary exists, then kills + restarts. After a clean run, pushes work again.

If you run a patched (post-2026-05-11) version of the script, you'll see a banner before the build telling you the orphan state was detected — that's expected and means the script is doing its job.

## Why production isn't affected

Production ECS uses `sudo install -m 755 /tmp/gitea-new /opt/hackforger/gitea` in `deploy/ecs/redeploy.sh:113`. `install(1)` is an atomic rename — the path always points to a valid inode. The old running process holds the old inode (same `unlink-while-open` mechanism!), but the path resolves to the new inode for any hook fork. There's no window during which the path is empty, so prod hooks never `ENOENT`.

## Prevention

- Don't `rm gitea` while `gitea web` is running. If you must, run `bash scripts/restart-gitea.sh` immediately afterwards.
- `make clean` will likely remove the binary — restart afterwards.
- If unsure, check with the self-diagnosis snippet above.

## See also

- `scripts/restart-gitea.sh` and `scripts/restart-gitea-test.sh` — the recovery commands
- `deploy/ecs/redeploy.sh:113` — the prod atomic-replace pattern this dev guard approximates
- `docs/superpowers/specs/2026-05-11-orphan-binary-restart-guard-design.md` — the design behind the patch
