# Attachments API

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/hackforger/attachments` | Upload a platform-level (RepoID=-1) attachment for use in hackathon descriptions, grant project pages, and submission READMEs |

## Why this exists

Forgejo's standard attachment endpoints (`/repos/{owner}/{repo}/issues/{index}/assets`, `/releases/{id}/assets`) all force a repo / issue / release scope and persist `RepoID > 0`. HackForger needs attachments for content that isn't bound to any repo — hackathon descriptions, grant project pitches, etc. — so this endpoint produces an attachment with `RepoID = -1`.

## Usage

```bash
curl -X POST \
  -H "Authorization: token $FORGEJO_TOKEN" \
  -F "file=@design.png" \
  "${FORGEJO_URL:?set FORGEJO_URL first — see SKILL.md Step 0}/api/v1/hackforger/attachments"
# → {"uuid":"a3f8...e2c1"}
```

Reference the returned UUID in markdown like:
```markdown
![architecture diagram](/attachments/a3f8...e2c1)
```

## Errors

| Status | Cause |
|--------|-------|
| 400 | File extension not in `setting.Attachment.AllowedTypes` |
| 404 | Site attachment uploads disabled |
| 401 | No token |
