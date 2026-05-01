# HackForger Development Guide

## Project Overview
HackForger is a Fork of Forgejo, adding Hackathon, Bounty, Grant, Credits, and Feed modules.
All new code lives in `*/hackforger/` directories, minimizing changes to upstream Forgejo files.

- Upstream: https://codeberg.org/forgejo/forgejo
- Project repo: https://github.com/HackForger/hackforger
- Internal instance: https://hackforger.inside.h2os.cloud
- Primary dev tool: Claude Code

## Architecture Rules
- Forgejo uses strict layered architecture: routers -> services -> models -> modules
- Upper layers may only call lower layers, never the reverse
- CRUD data-access functions live in `models/`, not `services/` (matches Forgejo convention: models/issues/issue.go has GetIssueByID)
- Web explore routes must be inside the existing `/explore` group in web.go to inherit `ignExploreSignIn` middleware
- Use pointer types for optional enum filters in ListOptions (nil = no filter, avoids zero-value ambiguity)
- NotifyWatchers only handles repo watchers; HackForger uses custom PublishHackforgerAction for 4 audience types
- New Go package paths: `forgejo.org/models/hackforger/`, `forgejo.org/services/hackforger/`, etc.
- Database: XORM ORM, define Go struct + tags for auto table creation
- Frontend: Go template SSR + partial Vue 3 component enhancement (not SPA) — see [docs/frontend-dev-guide.md](docs/frontend-dev-guide.md)
- **Vue components must call web routes for actions, NOT `/api/v1/` routes** (session cookie auth vs token auth)

## Directory Structure
- `models/hackforger/` -- Data models (16 tables) + CRUD data-access functions (Get/List/Create/Update/Delete)
- `services/hackforger/` -- Business logic only (state machines, transactional operations like Deposit/Redeem)
- `routers/api/v1/hackforger/` -- REST API
- `routers/web/hackforger/` -- Web page routes
- `templates/hackforger/` -- Go HTML templates
- `modules/hackforger/feed/` -- Feed event type definitions
- `web_src/js/features/hackforger/` -- Vue components

## Naming Conventions
- Go files: snake_case (hackathon.go, bounty_reward.go)
- Go structs: CamelCase (HackathonSubmission, BountyReward)
- API paths: kebab-case (/api/v1/hackforger/grant-rounds)
- Template files: snake_case (judge_panel.tmpl)
- Vue components: PascalCase (BountyPanel.vue)

## Error Handling & i18n
- Service layer returns typed errors (`ErrNoTracks`, `ErrDuplicateRegistration`), NOT `fmt.Errorf("english")`
- Web handlers check error type with `IsErr*()` then call `ctx.Tr()` for user-facing message
- **Never show `err.Error()` to users** without type-checking first
- See [docs/notes/i18n-error-pattern.md](docs/notes/i18n-error-pattern.md) for the pattern

## CSRF / Cross-Origin Protection
- Forgejo uses Go's `net/http.CrossOriginProtection` (NOT traditional CSRF tokens)
- **Do NOT add `{{.CsrfTokenHtml}}` or `_csrf` hidden inputs to templates** — they don't exist in Forgejo
- See [docs/notes/cross-origin-protection.md](docs/notes/cross-origin-protection.md) for full explanation and reverse proxy setup

## Forgejo Actions Token 权限
- `workflow.Dispatch(ctx, inputGetter, repo, doer)` 中的 **doer 决定 `github.token` 的权限**
- Action token 对自身 repo 有内置写权限 (git push, 创建 PR)，但 **merge PR 需要 doer 有 `CanWrite(TypeCode)` 权限**
- **Workflow 中如果需要 merge PR、创建 Release 等管理操作，dispatcher 必须是 repo owner / org admin**，不能用普通用户 (hacker/judge)
- macOS host runner 使用 BSD shell 工具，不要用 GNU 扩展 (`sed \?` 等)，改用 POSIX 语法
- See [docs/notes/forgejo-actions-token-permission.md](docs/notes/forgejo-actions-token-permission.md) for full permission matrix and checklist

## Local Testing
- See [docs/tests/local-testing-guide.md](docs/tests/local-testing-guide.md) for starting HackForger in worktrees, shared database, and common issues
- See [docs/tests/e2e/e2e-lessons-learned.md](docs/tests/e2e/e2e-lessons-learned.md) for common pitfalls (migration mismatch, pr.Issue gotcha, template crashes, Vue auth, feed rendering)
- See [docs/tests/e2e/e2e-testing-guide.md](docs/tests/e2e/e2e-testing-guide.md) for E2E automated testing with agent-browser (localhost:3000, web-first, screenshots)
- See [docs/tests/e2e/templates/](docs/tests/e2e/templates/) for E2E test templates (single feature, regression)
- See [docs/tests/e2e/tasks/](docs/tests/e2e/tasks/) for E2E test tasks (user journey full cycle, feature tests)
- **Key**: always copy `custom/conf/app.ini` from main repo before starting server in a worktree
- **E2E testing**: use `agent-browser` via `http://localhost:3000` (not HTTPS — local proxy blocks Tailscale TLS). Web-first with screenshots in reports.

## Common Commands
- `TAGS="bindata sqlite sqlite_unlock_notify" make backend` -- Compile backend (bindata embeds templates, sqlite enables SQLite3)
- `make frontend` -- Compile frontend (required after JS/Vue changes)
- `go test ./models/hackforger/... -v` -- Run model tests
- `go test ./services/hackforger/... -v` -- Run service tests
- `./gitea web` -- Start server (http://localhost:3000)
- `bash scripts/restart-gitea.sh` -- **After merging a PR**: one-command atomic rebuild + restart of the main instance. Does: build → stop-if-binary-path-matches → start → verify HTTP 200. Refuses to kill port-3000 processes whose binary path isn't the main repo's `gitea`, so it's safe to automate.
- `bash scripts/restart-gitea-test.sh` -- Same flow for the **test instance** (port 3001, `--custom-path /tmp/hackforger-test-custom`, log at `/tmp/gitea-test.log`). Use after editing test app.ini or rebuilding while it's up.
- Restart server manually (fallback): kill old process, remove LevelDB lock (`rm -f data/queues/common/LOCK`), then start

## Important Constraints
- Use `gh` CLI for GitHub operations (not `tea` -- that's for Codeberg/Forgejo)
- Do not modify upstream Forgejo files unless listed in the 11 injection points (see implementation-plan-draft.md section 1.2)
- All state changes must call PublishHackforgerAction to write Feed events
- **i18n**: All user-facing text MUST have both `locale_en-US.ini` and `locale_zh-CN.ini` entries under the `[hackforger]` section. Never add keys to only one locale file.
- Credits Deposit/Redeem must use db.WithTx transactions
- Internal HackForger instance: https://hackforger.inside.h2os.cloud
- API base path: https://hackforger.inside.h2os.cloud/api/v1/hackforger/

## CI/CD
- Primary CI: GitHub Actions (`.github/workflows/`)
- Self-hosted instance CI: Forgejo Actions (`.forgejo/workflows/`) -- kept for self-deployed HackForger instances
- Do not create Forgejo Actions for the GitHub-hosted repo

## Environment Variables
- `GITHUB_TOKEN` -- For gh CLI and GitHub API
- `FORGEJO_TOKEN` -- For self-hosted HackForger instance API
- `FORGEJO_URL` -- https://hackforger.inside.h2os.cloud
- `HACKFORGER_ADMIN_PASSWORD` -- `hackforger` admin password on the **production** instance (port 3000)
- `HACKFORGER_TEST_ADMIN_PASSWORD` -- `hackforger` admin password on the **test** instance (port 3001) — different from prod
- `PGPASSWORD` -- Postgres password for the `hackforger` DB user (covers both `hackforger` and `hackforger_test` databases)
- All of the above live in `.env` (gitignored). Source it with `set -a; . .env; set +a` if a tool needs them in the shell.

## Internal Instance (Production)
- URL: https://hackforger.inside.h2os.cloud (Caddy → localhost:3000)
- Login: hackforger / `$HACKFORGER_ADMIN_PASSWORD`
- DB: `hackforger` on Postgres
- Caddy reverse proxy: managed by launchd (com.h2os.caddy), do NOT restart or unload
- Check Caddy status: `launchctl list com.h2os.caddy`
- Caddy config: ~/.config/caddy/ (Caddyfile, env, run.sh)
- ⚠️ If Caddy config reload is needed, MUST confirm with developer first: `caddy reload --config ~/.config/caddy/Caddyfile`
- Default branch: v0.1-dev/hackforger

## Test Instance
- URL: http://localhost:3001 (no Caddy, direct)
- Custom path: `/tmp/hackforger-test-custom` (contains `conf/app.ini`)
- Work path: `/tmp/hackforger-test-data`
- DB: `hackforger_test` on Postgres (separate from prod's `hackforger`)
- Login: hackforger / `$HACKFORGER_TEST_ADMIN_PASSWORD` (**different from prod**)
- Restart: `bash scripts/restart-gitea-test.sh` (rebuild + safe restart)
- Use for: E2E tests, schema/migration experiments, anything you don't want hitting prod data
- ⚠️ Do NOT share `WORK_PATH` with the prod instance (LevelDB lock conflict)
- ⚠️ Action runners are bound to one instance via `.runner` file — a runner registered against port 3000 will NOT pick up jobs dispatched on 3001. For test E2E that involves Actions, register a separate runner with its own config dir.

## Action Runners (local dev machine)
- **Production runner**: `~/.config/forgejo-runner/` — daemon under launchd `com.h2os.forgejo-runner`, auto-starts at login
- **Test runner**: `~/.config/forgejo-runner-test/` — registered against port 3001 but NOT under launchd; start manually only when running E2E
  - Start:  `cd ~/.config/forgejo-runner-test && nohup ~/.local/bin/forgejo-runner daemon > runner.log 2>&1 & disown`
  - Stop:   `pkill -f 'forgejo-runner daemon$'` (matches the no-flag invocation; spares the prod daemon which uses `--config config.yml`)
- Both runners use `host` mode labels (`ubuntu-latest:host`, `macos-arm64:host`) — jobs run directly on the mac, no Docker required. Beware: `runs-on: ubuntu-latest` in YAML actually runs against macOS BSD shell tools.

## Git Remotes
- `origin` -- git@github.com:HackForger/hackforger.git (our repo)
- `upstream` -- https://codeberg.org/forgejo/forgejo.git (Forgejo upstream, read-only)
