# Hackathon Registration: Organization Model Design

> Status: Design proposal. Not implemented in Phase 1.

## Current State (Phase 1)

Registration uses a plain `TeamName string` field. This is disconnected from Forgejo's Organization/Team model.

## Proposed Model

Replace `TeamName` with `OrgID` to align with Forgejo's native Organization model:

```
HackathonRegistration:
  ID, HackathonID, UserID, OrgID (0 = solo), Status, ...
  UNIQUE(HackathonID, UserID)      -- one registration per person
  UNIQUE(HackathonID, OrgID) WHERE OrgID > 0  -- one registration per org
```

### Registration Modes

| Mode | OrgID | Meaning |
|------|-------|---------|
| Solo | 0 | Individual participant |
| Team | org.ID | Organization-based team, doer must be org member |

### Rules

1. **Solo → Team**: Allowed. Update `OrgID` from 0 to org.ID. Original solo record becomes team registration.
2. **Team → Solo**: Forbidden. Team may have other members whose work depends on the registration.
3. **Solo + same person joins via Org**: Auto-merge. Remove solo registration, user participates under the Org.
4. **Person in two Orgs tries to register both**: Reject second. `UNIQUE(HackathonID, UserID)` prevents this.
5. **Two members of same Org register separately**: Second member joins existing Org registration (not a new record).

### Corner Cases

| Case | Resolution |
|------|-----------|
| Member kicked from Org after registration | Registration remains valid (snapshot). Submission requires re-validation of Org membership. |
| Org owner cancels registration | Entire team cancelled (all members). Individual members cannot cancel an Org registration. |
| Org deleted after registration | Registration orphaned. Treat as solo for display purposes. |
| Hackathon MaxTeamSize exceeded | Reject additional Org members beyond limit. |

### Migration Path

1. Add `OrgID int64` column (default 0) to `hackathon_registration`
2. Keep `TeamName` for display (derived from Org name or "Solo")
3. Update registration form: dropdown to select "Solo" or one of user's Orgs
4. Update uniqueness check: per-user AND per-org constraints

### Dependencies

- Forgejo `org_user` table for membership verification
- Forgejo `organization` table for Org name display
- Possibly `team` table if sub-team scoping is needed (future)
