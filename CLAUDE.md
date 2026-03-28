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
- New Go package paths: `forgejo.org/models/hackforger/`, `forgejo.org/services/hackforger/`, etc.
- Database: XORM ORM, define Go struct + tags for auto table creation
- Frontend: Go template SSR + partial Vue 3 component enhancement (not SPA)

## Directory Structure
- `models/hackforger/` -- Data models (15 tables)
- `services/hackforger/` -- Business logic
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

## Common Commands
- `make backend` -- Compile backend
- `make frontend` -- Compile frontend
- `go test ./models/hackforger/... -v` -- Run model tests
- `go test ./services/hackforger/... -v` -- Run service tests
- `./gitea web` -- Start server (http://localhost:3000)

## Important Constraints
- Use `gh` CLI for GitHub operations (not `tea` -- that's for Codeberg/Forgejo)
- Do not modify upstream Forgejo files unless listed in the 11 injection points (see implementation-plan-draft.md section 1.2)
- All state changes must call PublishHackforgerAction to write Feed events
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

## Git Remotes
- `origin` -- git@github.com:HackForger/hackforger.git (our repo)
- `upstream` -- https://codeberg.org/forgejo/forgejo.git (Forgejo upstream, read-only)
