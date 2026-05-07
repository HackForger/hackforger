# Orgs API (HackForger extensions)

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/hackforger/orgs/{org}/join-request` | Notify the org's owners that the authenticated user wants to join |

## Semantics

This is **fire-and-forget**. No `join_request` row is persisted; the operation only publishes a `ActionOrgJoinRequest` notification visible to the org's owners. Hence the `202 Accepted` response — there is no resource to GET later, only a side-effecting notification.

To **accept** or **reject** a join request, use Forgejo's standard team membership API:

| Forgejo standard | Use for |
|------------------|---------|
| `PUT /teams/{id}/members/{username}` | Add the user to a team (effectively accepts) |
| `DELETE /teams/{id}/members/{username}` | Remove (or reject by inaction) |

## Usage

```bash
curl -X POST \
  -H "Authorization: token $FORGEJO_TOKEN" \
  "${FORGEJO_URL:?set FORGEJO_URL first — see SKILL.md Step 0}/api/v1/hackforger/orgs/web3-innovation/join-request"
# → 202 {"status":"notified"}
```

## Errors

| Status | Cause |
|--------|-------|
| 422 | The authenticated user is already a member of the org |
| 404 | Org does not exist |
| 401 | No token |
