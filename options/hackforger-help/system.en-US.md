# HackForger System Guide

**HackForger** is a business-neutral Forgejo distribution for collaborative programs. It combines Git collaboration with extension modules for events, bounties, grants, credits, feeds, and related workflows. This page is a capability index — find the feature you need, then follow the link to go deeper.

## Core Git Collaboration Capabilities

These capabilities are inherited from Forgejo. Usage is identical to Forgejo / Gitea / GitHub:

* **Repositories** — Create, Fork, branches, tags, protection rules, template repositories, mirror sync
* **Pull Requests / Merge Requests** — Code review, conflict resolution, checks, auto-merge
* **Issues** — Labels, milestones, project boards, automation, lock/pin
* **Wiki / Project Docs** — Built-in repository Wiki, custom home page
* **Forgejo Actions / Workflows** — `.forgejo/workflows/*.yml` CI/CD, self-hosted runner support
* **Full-text Search** — Multi-dimensional search across code, Issues, Commits, and Wiki
* **Webhooks / External Integrations** — Push code/Issue events to Slack, WeCom, external CI, and more
* **SSH / HTTPS Access** — Personal Access Token (PAT), SSH key management
* **Organizations / Teams** — Multi-level permissions, shared team repositories

If you are familiar with GitHub, the workflows here are nearly identical. For detailed usage documentation, refer to the upstream [Forgejo Docs](https://forgejo.org/docs/latest/).

## HackForger Platform Extension Modules

These are the reusable collaboration modules HackForger adds on top of Forgejo:

| Module | Description | Access Path |
|--------|-------------|-------------|
| Hackathon | Event creation, registration, tracks, judges, scoring, leaderboard | `/explore/hackathons` |
| Bounty | Issue linking, apply/claim, escrow, multi-party prize pool | `/explore/bounties` |
| Grant | Funding rounds, project applications, review, allocation | `/explore/grants` |
| Credits | Platform-wide credits ledger, redemption store, order history | `/credits` |
| Submissions | Aggregated browsing of submissions across all activities | `/explore/submissions` |
| Feed | Activity/Bounty/Grant/Credits event stream | `/` dashboard |
| Reputation | User participation metrics, leaderboard, tier badges | `/explore/reputation` |

## Developer Resources

* **API Reference** — Visit `/api/swagger` for the full OpenAPI documentation; HackForger-specific endpoints start with `/api/v1/hackforger/`
* **CLI Tool** — `hackforger-cli` (repository root, Go implementation, for bulk creation and management of activities)
* **AI Agent Integration** — Authorize via PAT to let an AI agent operate the platform API on your behalf

## Need Help?

* Platform feature questions — Read the "Platform Guide" section on this page
* Git / repository questions — Refer to [Forgejo Docs](https://forgejo.org/docs/latest/)
* Found a bug — Open an issue in the [HackForger GitHub repository](https://github.com/HackForger/hackforger/issues)
