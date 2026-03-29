# Local Testing Guide

## Starting HackForger in a Git Worktree

Git worktrees only contain tracked files. The `custom/` directory is gitignored, so instance config (database path, domain, secrets) is missing in new worktrees.

### Before starting the server

```bash
# 1. Copy config from main repo
mkdir -p custom/conf
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini

# 2. Build backend (embeds templates + assets)
TAGS="bindata sqlite sqlite_unlock_notify" make build

# 3. Build frontend (if JS/Vue files changed)
make frontend

# 4. Regenerate bindata (if frontend was rebuilt)
TAGS="bindata sqlite sqlite_unlock_notify" make build

# 5. Remove stale LevelDB lock
rm -f data/queues/common/LOCK

# 6. Start server
./gitea web
```

### Shared database

All worktrees share the same SQLite database at:
```
/Users/h2oslabs/Workspace/hackforger/data/forgejo.db
```

This is configured in `custom/conf/app.ini`. The database contains test users, orgs, and repos created during P0 E2E testing.

### Access

- URL: https://hackforger.inside.h2os.cloud/ (via Caddy reverse proxy on localhost:3000)
- Login: `hackforger` / `admin1234`
- Caddy must be running: `launchctl list | grep caddy`
- If Caddy is down: `launchctl load ~/Library/LaunchAgents/com.h2os.caddy.plist`

### Common issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| Initial setup wizard appears | Missing `custom/conf/app.ini` | Copy from main repo (step 1 above) |
| "Failed to load asset files" | Stale bindata after frontend rebuild | Run `make build` again to re-embed assets |
| "LOCK: resource temporarily unavailable" | Stale LevelDB lock from previous crash | `rm -f data/queues/common/LOCK` |
| Port 3000 already in use | Another instance running | `lsof -i :3000` then `kill <PID>` |
| HTTPS not working | Caddy not running | `launchctl load ~/Library/LaunchAgents/com.h2os.caddy.plist` |
