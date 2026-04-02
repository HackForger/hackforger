---
name: hackforger-api
description: Interact with HackForger platform via hackforger-cli. Use when querying or managing hackathons, bounties, grants, credits, or searching the platform.
---

Use `hackforger-cli` to interact with the HackForger instance.

## Authentication
- URL: `$HACKFORGER_URL` (default: https://hackforger.inside.h2os.cloud)
- Token: `$FORGEJO_TOKEN`

## Commands

```bash
# Search
hackforger-cli search query --q "keyword" --scope all

# Hackathons
hackforger-cli hackathon list
hackforger-cli hackathon get <id>

# Bounties
hackforger-cli bounty list
hackforger-cli bounty get <owner> <repo> <id>

# Grants
hackforger-cli grant round-list
hackforger-cli grant round-get <id>

# Credits
hackforger-cli credits balance
hackforger-cli credits transactions

# Feed
hackforger-cli feed list --type global

# Reputation
hackforger-cli reputation get <username>
hackforger-cli reputation leaderboard

# AI Assistant
hackforger-cli assistant chat --query "What bounties are available?"
```

## Output Formats
- `--output json` (default) — JSON output
