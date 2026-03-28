---
description: Test a HackForger API endpoint against the internal instance
allowed-tools: Bash
---
Test an API endpoint on the internal HackForger instance.

Usage: /api-test METHOD /path [JSON body]
Example: /api-test GET /hackforger/hackathons
Example: /api-test POST /hackforger/hackathons '{"name":"test"}'

Execute:
```bash
curl -s -X $METHOD \
  -H "Authorization: token $FORGEJO_TOKEN" \
  -H "Content-Type: application/json" \
  ${BODY:+-d "$BODY"} \
  "https://hackforger.inside.h2os.cloud/api/v1$PATH" | jq .
```

If FORGEJO_TOKEN is not set, warn the user to set it first.
