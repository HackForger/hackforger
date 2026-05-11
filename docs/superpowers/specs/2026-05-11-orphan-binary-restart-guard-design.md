---
date: 2026-05-11
status: approved
owner: claude + jjkysy
---

# Orphan-Binary Restart Guard

## Problem

On dev Macs, the `gitea` binary at `/Users/h2oslabs/Workspace/hackforger/gitea` can be deleted while the running `gitea` process is still alive (via `make clean`, manual `rm`, an aborted rebuild, or a worktree shuffle). Unix unlink-while-open semantics keep the running process healthy — it serves HTTP fine — but every git push fails because each repo's `hooks/pre-receive.d/gitea` script `exec`s the path:

```
/Users/h2oslabs/Workspace/hackforger/gitea hook --config=… pre-receive
```

…and the path now resolves to nothing. Symptom on the client side:

```
remote: ./hooks/pre-receive.d/gitea: line 3:/Users/h2oslabs/Workspace/hackforger/gitea: No such file or directory
remote: error: hook declined to update refs/heads/…
! [remote rejected] … (pre-receive hook declined)
error: failed to push some refs
```

Production ECS is immune: `deploy/ecs/redeploy.sh` uses `sudo install -m 755 /tmp/gitea-new /opt/hackforger/gitea`, which is an atomic rename — the path always points to a valid inode. Verified empirically 2026-05-11 by an end-to-end push test against `https://www.synnovator.com`.

## Goals

1. The orphan state becomes **visible** at the moment a human is most likely to notice — restart time.
2. The orphan state becomes **recoverable** by a single one-line command the developer already knows.
3. The restart script will **not** make the situation worse (e.g., killing the running process when no replacement binary exists).
4. The phenomenon is documented so future operators (human or Claude) recognize it on sight.

## Non-Goals

- No changes to upstream Forgejo Go code (e.g., teaching hook templates to use `os.Executable()` at exec time). Maintenance cost in our fork outweighs the value for a dev-only failure mode.
- No periodic monitoring / launchd healthchecker. Out of scope for this iteration.
- No production-side changes. Production is already covered by the `install(1)` atomic-replace pattern in `redeploy.sh`.
- No script consolidation/refactor between `restart-gitea.sh` and `restart-gitea-test.sh`. They remain near-duplicates (~70 lines each); the cost of factoring out a shared library exceeds the benefit at this size.

## Design

### Change set

| File | Change | Lines |
|---|---|---|
| `scripts/restart-gitea.sh` | Add (A) pre-flight orphan diagnostic, (B) post-build binary assertion | +~15 |
| `scripts/restart-gitea-test.sh` | Mirror the same two changes | +~15 |
| `docs/notes/orphan-binary.md` | New explainer note (~50 lines) | new |
| `CLAUDE.md` | One-line pointer next to the `restart-gitea.sh` bullet | +1 |

### A. Pre-flight orphan diagnostic

Inserted at the **very top of `main`**, before `make build`. Purely informational; does not abort.

Logic:

1. `PID = lsof -iTCP:$PORT -sTCP:LISTEN -t` — current listener on the script's target port.
2. If `PID` is non-empty AND `! -e $BINARY` on disk → print a banner.

Banner content (each script substitutes its own `$PORT` and `$BINARY`):

```
⚠ Orphan-binary state detected
  PID $PID is listening on :$PORT but $BINARY no longer exists on disk.
  Any git push to this instance has been failing with ENOENT in pre-receive.
  This restart will recover. See docs/notes/orphan-binary.md
```

Implementation note: the existing script computes `PID` only inside step `[2/4]`. The new block needs its own local `PID` lookup at the top so the banner can print before `make build` starts (which itself takes 30s+ and obscures the cause). The variable is shadowed and recomputed in step `[2/4]` — kept as-is to minimize blast radius.

### B. Post-build binary assertion

Inserted **after `make build` returns, before the kill step** (i.e., between current line 18 and current line 20 of both scripts).

```bash
[ -x "$BINARY" ] || {
  echo "FATAL: make build returned 0 but $BINARY is missing or non-executable." >&2
  echo "       Refusing to kill the running instance — that would leave you with nothing to start." >&2
  exit 1
}
```

Defends against pathological cases where the Makefile target succeeds without producing the expected output (e.g., target name changed, output dir moved, partial build, surface-area regression). Without this check, the script would proceed to kill the running gitea and then fail to start anything — exactly the "make it worse" failure we want to prevent.

### C. `docs/notes/orphan-binary.md`

New markdown note covering:

1. **What it looks like** — exact client-side error string, server-log signature.
2. **Why it happens** — one-paragraph explanation of unlink-while-open: directory entry deleted, but the kernel keeps the inode alive because the running process still has it mapped (`txt` segment). HTTP keeps working (no path resolution). `exec()` of the path fails (path resolution required).
3. **Self-diagnosis** — `lsof -p <pid> | awk '/txt.*\/gitea$/{print $NF}'` returns the path; compare with `ls`. If `ls` says "No such file or directory" while `lsof` shows the path, you're in this state.
4. **Fix** — `bash scripts/restart-gitea.sh` (and the test-instance variant). The script rebuilds and restarts atomically; the new prod-style behavior is in place.
5. **Why production is unaffected** — one paragraph on `redeploy.sh`'s `install(1)` atomic replace. Cross-reference `deploy/ecs/redeploy.sh:113`.
6. **Prevention** — don't manually `rm gitea` or `make clean` while gitea is running; if you do, immediately run the restart script.

### D. `CLAUDE.md` pointer

Append a single trailing sentence to the existing `restart-gitea.sh` bullet (≈ line 84 of CLAUDE.md, under **Common Commands**):

> If you ever see `pre-receive ... No such file or directory` on push, you're in the orphan-binary state — see [docs/notes/orphan-binary.md](docs/notes/orphan-binary.md).

## Error Handling

| Failure | Behavior |
|---|---|
| Pre-flight detects orphan state | Banner printed; script continues (banner is advisory, the rebuild is the fix) |
| `make build` fails | Existing `set -euo pipefail` aborts before kill. Unchanged. |
| `make build` exit 0 but binary missing | New assertion B catches it; abort with explicit message; running gitea **not** killed |
| Port held by a foreign binary | Existing safety check unchanged — still refuses to kill |
| Nothing on the port | Unchanged: skip kill, build + start |

## Verification Matrix

| # | Setup | Expected |
|---|---|---|
| 1 | Healthy: binary present, gitea running | No banner; full clean restart; HTTP 200 |
| 2 | Orphan: `rm $BINARY`, gitea still listening | Banner appears at top; `make build` rebuilds binary; kill + restart succeeds; HTTP 200 |
| 3 | Sabotaged build: stub `make build` to `exit 0` without producing binary | Assertion B fires; exit 1 with explicit message; **running gitea untouched**; port-3000 unchanged |
| 4 | No process on port, binary present | No banner; build + start; HTTP 200 |
| 5 | No process on port, binary missing | Banner does **not** fire (no running PID to be orphaned); `make build` rebuilds; start; HTTP 200 |

Manual verification on a developer Mac is sufficient — no unit test harness for these scripts.

## Open Questions

None. The design is intentionally minimal and additive; existing script behavior is preserved.

## Reference

- Root-cause debug session (2026-05-11): identified the unlink-while-open mechanism via `lsof` inode comparison between PID 67188 (port 3000) and PID 49461 (port 3001) — two different inodes for the same missing path.
- Production verification (2026-05-11): end-to-end push test on `https://www.synnovator.com` succeeded with exit 0. Throwaway repo `SynNovator/precv-repro-*` created, pushed, and deleted (HTTP 204) — see conversation transcript.
- Related: `deploy/ecs/redeploy.sh:113` (the atomic-install pattern this design aims to approximate in dev).
