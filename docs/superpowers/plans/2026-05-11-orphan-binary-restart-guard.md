# Orphan-Binary Restart Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the "running gitea + missing on-disk binary" state visible at restart time, prevent the restart script from making it worse, and document the phenomenon.

**Architecture:** Two surgical additions to each of the two restart scripts (pre-flight diagnostic + post-build assertion), plus a new explainer doc and a one-line cross-reference in `CLAUDE.md`. No upstream Forgejo Go changes; no shared bash library; no monitoring; no production-side changes.

**Tech Stack:** Bash 3.2 (macOS default), `lsof`, plain Markdown. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-05-11-orphan-binary-restart-guard-design.md`

---

## Pre-flight (do this once before starting)

- [ ] **P0.1: Confirm current working state**

Run from `/Users/h2oslabs/Workspace/hackforger`:

```bash
git status
git log --oneline -5
ls -la scripts/restart-gitea.sh scripts/restart-gitea-test.sh CLAUDE.md
ls -d docs/notes docs/superpowers/specs docs/superpowers/plans
```

Expected: clean tree (or only the spec + plan files we just added), both restart scripts present, target directories all exist.

- [ ] **P0.2: Confirm the current orphan state (this is our "live test fixture")**

```bash
PID_MAIN=$(lsof -iTCP:3000 -sTCP:LISTEN -t)
PID_TEST=$(lsof -iTCP:3001 -sTCP:LISTEN -t)
echo "main pid=$PID_MAIN test pid=$PID_TEST"
ls -la /Users/h2oslabs/Workspace/hackforger/gitea 2>&1
```

Expected: both PIDs non-empty, `ls` reports `No such file or directory`. **Do not run any restart-* script yet** — we want this state preserved as a fixture until Task 1 step 4.

If the binary already exists (someone rebuilt it), the orphan-banner verification in Task 1 step 4 will need a different fixture — see the alternate-path note in that step.

---

## Task 1: Add pre-flight orphan diagnostic to `restart-gitea.sh`

**Files:**
- Modify: `scripts/restart-gitea.sh` (insert after line 15 `cd "$REPO"`)

This is Change A from the spec. The block is purely informational; it runs **before** `make build` so that the cause is visible upfront rather than buried under 30 seconds of compile output.

- [ ] **Step 1: Read the current file in full to lock in the exact context**

```bash
cat -n scripts/restart-gitea.sh
```

Expected: 62 lines. Confirm line 15 is `cd "$REPO"` and line 17 is `echo "[1/4] Building backend (bindata + sqlite tags)..."`. If line numbers have drifted, re-anchor on the literal strings, not on numbers.

- [ ] **Step 2: Document the "failing test" state**

Without invoking the script, write down what current behavior is in orphan state:

> Running `bash scripts/restart-gitea.sh` against an orphan instance prints nothing about the orphan condition — the cause of any prior push failures is invisible. The user sees `[1/4] Building backend ...` and has no way to know they were already broken before this run.

This is the gap Task 1 closes.

- [ ] **Step 3: Insert the pre-flight diagnostic block**

Edit `scripts/restart-gitea.sh`. Find the exact block:

```bash
cd "$REPO"

echo "[1/4] Building backend (bindata + sqlite tags)..."
```

Replace with:

```bash
cd "$REPO"

# Pre-flight: detect orphan-binary state (gitea running but $BINARY deleted on disk).
# Unix unlink-while-open keeps the process alive but git push fails with ENOENT in
# pre-receive. The rebuild below will recover; this banner just makes the cause
# visible before make's 30s of output buries it. See docs/notes/orphan-binary.md.
ORPHAN_PID=$(lsof -iTCP:3000 -sTCP:LISTEN -t 2>/dev/null || true)
if [ -n "$ORPHAN_PID" ] && [ ! -e "$BINARY" ]; then
  echo ""
  echo "  ⚠ Orphan-binary state detected"
  echo "    PID $ORPHAN_PID is listening on :3000 but $BINARY no longer exists on disk."
  echo "    Any git push to this instance has been failing with ENOENT in pre-receive."
  echo "    This restart will recover. See docs/notes/orphan-binary.md"
  echo ""
fi

echo "[1/4] Building backend (bindata + sqlite tags)..."
```

Style notes:
- Use `[ ! -e "$BINARY" ]` (not `! [ -e ... ]`) for bash 3.2 portability.
- `2>/dev/null || true` on `lsof` prevents an early exit under `set -e` if nothing is listening.
- Variable named `ORPHAN_PID` (not `PID`) to avoid shadowing the `PID` that step `[2/4]` re-derives a few lines below — we keep that block untouched for blast-radius reasons.

- [ ] **Step 4: Syntax-check, then run the script**

```bash
bash -n scripts/restart-gitea.sh && echo "syntax ok"
```

Expected: `syntax ok`.

Then **run the script** against the current orphan fixture:

```bash
bash scripts/restart-gitea.sh
```

Expected output (in order):
1. The new orphan banner (4 indented lines with the warning, before `[1/4]`).
2. `[1/4] Building backend ...` followed by Go build output.
3. `[2/4] ... Stopping main instance (PID <ORPHAN_PID>, binary /Users/h2oslabs/Workspace/hackforger/gitea)`.
4. `[3/4] Starting new instance...`.
5. `[4/4] ... ✓ HTTP 200 on localhost:3000`.

Alternate path (if the orphan fixture was already healed before this step):
- Run `ls /Users/h2oslabs/Workspace/hackforger/gitea`; if the binary now exists, the banner will not fire this run. That's still correct behavior (test 1 from the verification matrix). Move on; we'll re-verify the banner in Task 3 step 5 against the test instance's likely-still-orphan fixture, or by deliberately `rm`'ing the binary while a process holds it open.

- [ ] **Step 5: Confirm the actual recovery worked (push end-to-end on the now-rebuilt instance)**

```bash
ls -la /Users/h2oslabs/Workspace/hackforger/gitea
lsof -iTCP:3000 -sTCP:LISTEN -t
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:3000/
```

Expected: binary present, single PID listening, HTTP 200.

- [ ] **Step 6: Commit**

```bash
git add scripts/restart-gitea.sh
git commit -m "$(cat <<'EOF'
scripts: add orphan-binary pre-flight diagnostic to restart-gitea.sh

Surfaces the "gitea running + on-disk binary deleted" state at the top of
the restart flow so the cause of preceding push failures is visible before
the 30s build step buries it. Informational only — does not alter control
flow. The existing build-first ordering already handles recovery.

See docs/superpowers/specs/2026-05-11-orphan-binary-restart-guard-design.md
EOF
)"
```

---

## Task 2: Add post-build binary assertion to `restart-gitea.sh`

**Files:**
- Modify: `scripts/restart-gitea.sh` (insert between current `make build` line and the `[2/4]` block)

This is Change B from the spec. Defends against a pathological `make build` that exits 0 without producing the binary, which would otherwise let us kill the running gitea and end up with nothing to start.

- [ ] **Step 1: Locate the exact insertion point**

```bash
grep -n 'TAGS=' scripts/restart-gitea.sh
grep -n '\[2/4\]' scripts/restart-gitea.sh
```

Expected: `make build` is on the line above `[2/4]`. The assertion goes between them.

- [ ] **Step 2: Insert the assertion**

Find:

```bash
echo "[1/4] Building backend (bindata + sqlite tags)..."
TAGS="bindata sqlite sqlite_unlock_notify" make build

echo "[2/4] Checking for process on port 3000..."
```

Replace with:

```bash
echo "[1/4] Building backend (bindata + sqlite tags)..."
TAGS="bindata sqlite sqlite_unlock_notify" make build

# Defensive: make build exited 0 but did it actually produce a runnable binary?
# Without this check, a broken Makefile target would let us kill the running
# instance below and leave us with nothing to start.
if [ ! -x "$BINARY" ]; then
  echo "FATAL: make build returned 0 but $BINARY is missing or non-executable." >&2
  echo "       Refusing to kill the running instance — that would leave you with nothing to start." >&2
  exit 1
fi

echo "[2/4] Checking for process on port 3000..."
```

- [ ] **Step 3: Syntax-check + happy-path re-run**

```bash
bash -n scripts/restart-gitea.sh && echo "syntax ok"
bash scripts/restart-gitea.sh
```

Expected: `syntax ok`, then a clean run (no banner this time — we're healthy), HTTP 200 at the end. The assertion silently passes because `make build` produces a real binary.

- [ ] **Step 4: Sabotaged-build verification (optional but recommended)**

This is the test that exercises the assertion. Simulate a Makefile that lies:

```bash
# Save the real binary, simulate "build produced nothing"
mv /Users/h2oslabs/Workspace/hackforger/gitea /tmp/gitea.real

# Stub PATH so `make build` is a no-op
mkdir -p /tmp/stub
cat > /tmp/stub/make <<'STUB'
#!/bin/bash
echo "stub make: pretending to build $@"
exit 0
STUB
chmod +x /tmp/stub/make

# Run the script with the stub in front
PATH=/tmp/stub:$PATH bash scripts/restart-gitea.sh
RC=$?
echo "exit code: $RC"

# Verify: assertion fired, no kill happened
lsof -iTCP:3000 -sTCP:LISTEN -t
```

Expected:
- Script prints `FATAL: make build returned 0 but ... is missing` and exits 1.
- `lsof` still shows the original PID — **the running gitea was NOT killed**.
- Exit code: 1.

Cleanup:

```bash
mv /tmp/gitea.real /Users/h2oslabs/Workspace/hackforger/gitea
rm -rf /tmp/stub
ls -la /Users/h2oslabs/Workspace/hackforger/gitea  # confirm restored
```

- [ ] **Step 5: Commit**

```bash
git add scripts/restart-gitea.sh
git commit -m "$(cat <<'EOF'
scripts: add post-build binary assertion to restart-gitea.sh

If `make build` exits 0 without producing $BINARY (e.g., regressed Makefile
target, partial build, surface-area change), abort before the kill step so
we don't leave the host with nothing to start. Refuses to kill the running
instance in that case.

See docs/superpowers/specs/2026-05-11-orphan-binary-restart-guard-design.md
EOF
)"
```

---

## Task 3: Mirror both changes to `restart-gitea-test.sh`

**Files:**
- Modify: `scripts/restart-gitea-test.sh`

The test-instance script is a near-duplicate of the main one. Both new blocks are mirrored verbatim with `:3000` replaced by `:$PORT` (the test script parametrises the port).

- [ ] **Step 1: Read the current test script**

```bash
cat -n scripts/restart-gitea-test.sh
```

Expected: 70 lines. Confirm it uses `$PORT` (set to 3001 at the top) rather than a hardcoded port number in the lsof block.

- [ ] **Step 2: Insert pre-flight diagnostic (mirror of Task 1)**

Find:

```bash
cd "$REPO"

echo "[1/4] Building backend (bindata + sqlite tags)..."
```

Replace with:

```bash
cd "$REPO"

# Pre-flight: detect orphan-binary state. See docs/notes/orphan-binary.md.
ORPHAN_PID=$(lsof -iTCP:$PORT -sTCP:LISTEN -t 2>/dev/null || true)
if [ -n "$ORPHAN_PID" ] && [ ! -e "$BINARY" ]; then
  echo ""
  echo "  ⚠ Orphan-binary state detected"
  echo "    PID $ORPHAN_PID is listening on :$PORT but $BINARY no longer exists on disk."
  echo "    Any git push to this instance has been failing with ENOENT in pre-receive."
  echo "    This restart will recover. See docs/notes/orphan-binary.md"
  echo ""
fi

echo "[1/4] Building backend (bindata + sqlite tags)..."
```

- [ ] **Step 3: Insert post-build assertion (mirror of Task 2)**

Find:

```bash
echo "[1/4] Building backend (bindata + sqlite tags)..."
TAGS="bindata sqlite sqlite_unlock_notify" make build

echo "[2/4] Checking for process on port $PORT..."
```

Replace with:

```bash
echo "[1/4] Building backend (bindata + sqlite tags)..."
TAGS="bindata sqlite sqlite_unlock_notify" make build

# Defensive: make build exited 0 but did it actually produce a runnable binary?
if [ ! -x "$BINARY" ]; then
  echo "FATAL: make build returned 0 but $BINARY is missing or non-executable." >&2
  echo "       Refusing to kill the running instance — that would leave you with nothing to start." >&2
  exit 1
fi

echo "[2/4] Checking for process on port $PORT..."
```

- [ ] **Step 4: Syntax-check**

```bash
bash -n scripts/restart-gitea-test.sh && echo "syntax ok"
```

Expected: `syntax ok`.

- [ ] **Step 5: Run against the test instance (likely still orphan)**

```bash
lsof -iTCP:3001 -sTCP:LISTEN -t
ls -la /Users/h2oslabs/Workspace/hackforger/gitea
```

If both show: PID listening AND binary present → instance is already healthy; running the script should print no banner and complete cleanly.

If PID listening AND binary missing → orphan fixture exists; banner should fire.

Actually run it:

```bash
bash scripts/restart-gitea-test.sh
```

Expected: banner if-and-only-if orphan, then a clean build + restart, HTTP 200 on `:3001`.

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:3001/
```

Expected: `200`.

- [ ] **Step 6: Commit**

```bash
git add scripts/restart-gitea-test.sh
git commit -m "$(cat <<'EOF'
scripts: mirror orphan-binary guards to restart-gitea-test.sh

Adds the same pre-flight orphan diagnostic and post-build binary assertion
that restart-gitea.sh got, parametrised on $PORT for the test instance.

See docs/superpowers/specs/2026-05-11-orphan-binary-restart-guard-design.md
EOF
)"
```

---

## Task 4: Write `docs/notes/orphan-binary.md`

**Files:**
- Create: `docs/notes/orphan-binary.md`

The explainer note. ~50 lines, six sections per spec section C.

- [ ] **Step 1: Confirm `docs/notes/` exists and pick a neighbour-style for tone**

```bash
ls docs/notes/ | head -5
head -30 docs/notes/i18n-error-pattern.md 2>/dev/null || head -30 docs/notes/cross-origin-protection.md 2>/dev/null
```

Expected: directory exists, neighbouring notes use short titles, plain Markdown, mix of zh-CN/en-US text — match whatever style is current.

- [ ] **Step 2: Create the note**

Write to `docs/notes/orphan-binary.md`. Note: the outer fence below uses **four** backticks so the inner three-backtick fences in the file content are preserved verbatim.

````markdown
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
````

- [ ] **Step 3: Render-check**

```bash
ls -la docs/notes/orphan-binary.md
wc -l docs/notes/orphan-binary.md
```

Expected: file exists, between 45 and 65 lines.

- [ ] **Step 4: Commit**

```bash
git add docs/notes/orphan-binary.md
git commit -m "$(cat <<'EOF'
docs: add note on orphan-binary state on dev Macs

Explains the Unix unlink-while-open mechanism that lets gitea keep serving
HTTP while git push fails with ENOENT, how to self-diagnose, how to fix
(restart-gitea.sh), and why production ECS isn't affected (atomic install
in redeploy.sh).

See docs/superpowers/specs/2026-05-11-orphan-binary-restart-guard-design.md
EOF
)"
```

---

## Task 5: Add pointer to `CLAUDE.md`

**Files:**
- Modify: `CLAUDE.md` (append a sentence to the `restart-gitea.sh` bullet under **Common Commands**)

- [ ] **Step 1: Find the exact bullet**

```bash
grep -n 'restart-gitea.sh' CLAUDE.md
```

Expected: at least one hit; find the bullet under "Common Commands" that begins with `bash scripts/restart-gitea.sh -- **After merging a PR**:`.

- [ ] **Step 2: Append the cross-reference**

Locate the exact line:

```
- `bash scripts/restart-gitea.sh` -- **After merging a PR**: one-command atomic rebuild + restart of the main instance. Does: build → stop-if-binary-path-matches → start → verify HTTP 200. Refuses to kill port-3000 processes whose binary path isn't the main repo's `gitea`, so it's safe to automate.
```

Replace with:

```
- `bash scripts/restart-gitea.sh` -- **After merging a PR**: one-command atomic rebuild + restart of the main instance. Does: build → stop-if-binary-path-matches → start → verify HTTP 200. Refuses to kill port-3000 processes whose binary path isn't the main repo's `gitea`, so it's safe to automate. If you ever see `pre-receive ... No such file or directory` on push, you're in orphan-binary state — see [docs/notes/orphan-binary.md](docs/notes/orphan-binary.md).
```

- [ ] **Step 3: Sanity check**

```bash
grep -n 'orphan-binary.md' CLAUDE.md
```

Expected: exactly one hit, on the modified bullet line.

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "$(cat <<'EOF'
docs: cross-reference orphan-binary note from CLAUDE.md restart bullet

So future operators (human or Claude) reading CLAUDE.md know where to look
when they hit the pre-receive ENOENT symptom.

See docs/superpowers/specs/2026-05-11-orphan-binary-restart-guard-design.md
EOF
)"
```

---

## Task 6: End-to-end verification on the restored instance

This is **not a code change**, just a fresh-eyes confirmation that the whole thing works after all five previous tasks are committed.

- [ ] **Step 1: Confirm clean tree and recent history**

```bash
git status
git log --oneline -7
```

Expected: clean tree, last 5 commits match the work above (script A, script B, mirror, doc, CLAUDE.md). Plus the design-spec commit if it was committed separately.

- [ ] **Step 2: End-to-end push test against the now-healthy main instance**

Mirror of the prod verification we did during brainstorming, but pointed at `localhost:3000`:

```bash
set -a; . .env; set +a
TS=$(date +%s)
REPO="precv-local-$TS"

curl -sS -X POST "http://localhost:3000/api/v1/user/repos" \
  -H "Authorization: token ${FORGEJO_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"${REPO}\",\"auto_init\":true,\"private\":true}" \
  -o /tmp/r.json -w "HTTP=%{http_code}\n"
```

If HTTP=201, proceed:

```bash
WORK="/tmp/${REPO}"
# FORGEJO_TOKEN is for the dev instance — confirm the local token works here
OWNER=$(python3 -c "import json; print(json.load(open('/tmp/r.json'))['owner']['login'])")
AUTH_URL="http://${OWNER}:${FORGEJO_TOKEN}@localhost:3000/${OWNER}/${REPO}.git"

git clone "$AUTH_URL" "$WORK"
cd "$WORK"
git config user.email ci@example.com
git config user.name ci-smoke
git checkout -b precv-test
echo "hello $(date)" > smoke.txt
git add smoke.txt
git commit -m "precv smoke"
git push -v origin precv-test
RC=$?
cd /
echo "push exit code: $RC"

curl -sS -X DELETE "http://localhost:3000/api/v1/repos/${OWNER}/${REPO}" \
  -H "Authorization: token ${FORGEJO_TOKEN}" \
  -o /tmp/del.json -w "delete HTTP=%{http_code}\n"
rm -rf "$WORK"
```

Expected:
- `push exit code: 0`
- `delete HTTP=204`
- The push output contains `* [new branch] precv-test -> precv-test` and the gitea-side "Create a new pull request" hint (proof the hook ran cleanly).

If push fails with `ENOENT` again: the orphan state somehow re-occurred between Task 1 step 5 and here. Re-run `bash scripts/restart-gitea.sh` and try the test again; investigate root cause separately.

- [ ] **Step 3: No commit for this task** — verification only.

---

## What "done" looks like

After all 6 tasks:

1. `scripts/restart-gitea.sh` has two new defensive blocks (orphan diagnostic + post-build assertion).
2. `scripts/restart-gitea-test.sh` has the same two blocks, parametrised on `$PORT`.
3. `docs/notes/orphan-binary.md` exists and explains the phenomenon end-to-end.
4. `CLAUDE.md` cross-references the new note from the `restart-gitea.sh` bullet.
5. End-to-end git push against `localhost:3000` succeeds with exit 0.
6. Five commits land on the current branch, each scoped to one task.

No changes to production. No changes to upstream Forgejo Go code. No new monitoring. No script consolidation.
