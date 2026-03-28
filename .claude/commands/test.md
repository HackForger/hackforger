---
description: Run HackForger tests
allowed-tools: Bash, Read, Grep
---
Run tests. Arguments: module name or "all"

If "$ARGUMENTS" is "all":
  Run `go test ./models/hackforger/... ./services/hackforger/... -v -count=1`

If "$ARGUMENTS" is a module name (bounty, hackathon, grants, credits, reputation, feed):
  Run `go test ./models/hackforger/... ./services/hackforger/... -v -run "Test.*${ARGUMENTS}" -count=1`

If "$ARGUMENTS" is "e2e":
  Run `go test ./tests/integration/... -v -tags='sqlite sqlite_unlock_notify' -run TestE2E`

Report results with pass/fail summary.
