# Embedded workflow YAML drifts after distribution

A property of Forgejo Actions (and GitHub Actions) that bites you the
first time. Not specific to HackForger, but worth knowing because we
embed workflow YAML in Go code and write it into downstream repos at
creation time.

## The mechanism

Forgejo Actions reads workflow definitions from each repo's own
`.forgejo/workflows/*.yml` files. There is no central "template
registry" that workflow runs consult — each repo owns its workflow as a
plain committed file.

If your service writes a workflow file into a downstream repo as a
side-effect of some action (e.g. "create track repo" → installs an
auto-index workflow), then from that moment on:

- The Go source string and the file in the downstream repo are
  **decoupled**.
- Future dispatches against that repo execute the file the repo
  carries, not the latest source.
- Editing the Go source only affects repos created **after** the edit.

## Concrete example (HackForger)

`services/hackforger/hackathon.go` defines `submissionIndexWorkflow`
as a Go const. `CreateTrackWithRepo` writes that string to
`.forgejo/workflows/update-submission-index.yml` in every new track
repo. Today (May 2026) only this one workflow is distributed this
way.

| Event | Effect on the YAML in downstream track repos |
| --- | --- |
| Edit the Go const, no rebuild | nothing happens — distributed copies unchanged |
| Edit + rebuild + restart server | new tracks get new YAML; existing tracks keep old YAML |
| Existing track dispatches a run after the change | runs against its **old** YAML |
| Backfill script overwrites the file in each track repo | now in sync; next dispatch uses new YAML |

## When this is harmless

Cosmetic edits — step name, comments, log noise suppression. Old and
new YAML produce equivalent observable behavior. Skip the backfill;
let the old copies live out their lives.

PR #132 (this one) is an example: rename a step, fix a docstring, add
`credential.helper ""`. Nothing breaks for old tracks.

## When this hurts

Anything that changes **observable behavior or contracts**:

- API path the workflow calls changed (`/api/v1/foo` → `/api/v1/bar`)
- Input schema changed (renamed/added/removed `inputs.*`)
- Bug in the workflow logic itself (jq expression, git command, etc.)
- Required env var added

In these cases, **existing track repos will keep failing or producing
wrong output** until their YAML files are overwritten. The Go-side
`triggerSubmissionIndexUpdate` may also need a feature flag if it
dispatches with new inputs that old YAML doesn't understand.

## How to backfill

One-off Go script or admin endpoint:

```go
tracks, _ := hackforger_model.ListAllTracksWithRepo(ctx)
for _, track := range tracks {
    repo, _ := repo_model.GetRepositoryByID(ctx, track.RepoID)
    _, _ = files_service.ChangeRepoFiles(ctx, repo, sysUser, &files_service.ChangeRepoFilesOptions{
        OldBranch: repo.DefaultBranch,
        NewBranch: repo.DefaultBranch,
        Message:   "Sync update-submission-index.yml to latest template",
        Files: []*files_service.ChangeRepoFile{{
            Operation:     "update",
            TreePath:      ".forgejo/workflows/update-submission-index.yml",
            ContentReader: strings.NewReader(submissionIndexWorkflow),
        }},
    })
}
```

Each track repo gets one new commit on its default branch.

## Avoiding the problem long-term

If a workflow is going to evolve frequently, switch from embedded YAML
to a **reusable workflow** kept in a central repo. Each track repo
then carries only a one-line stub:

```yaml
# .forgejo/workflows/update-submission-index.yml
on: workflow_dispatch
jobs:
  call:
    uses: hackforger-ops/workflows/.forgejo/workflows/update-index.yml@v1
    with: { hackathon_id: '${{ inputs.hackathon_id }}', ... }
```

Updating the central workflow auto-applies to all consumers (subject
to the `@v1` pin — bump the tag for breaking changes).

Trade-off: introduces a hard dependency on the central repo being
reachable, plus the auth/permission story for cross-repo `uses:`.
Worth it only if you actually expect frequent edits.

## Lesson summary

- Inline embedded YAML = "ship and forget". Cheap, but distributed
  copies drift forever.
- Reusable workflow = central control. More moving parts, but edits
  propagate.
- For HackForger today (one workflow, low edit frequency), inline is
  the right call. Re-evaluate if `submissionIndexWorkflow` starts
  changing more than once per quarter.
