# External content deployment

This directory defines the public, business-neutral contract for publishing
runtime content that is owned by a separate private repository. It contains no
real deployment target, customer domain, infrastructure topology, or secret.

The currently supported mount is `landing`, mapped to
`custom/public/assets/landing`. The mapping is fixed in reviewed public code;
callers cannot choose an arbitrary remote destination.

## Private repository contract

Keep the content tree and its manifest outside this repository, for example:

```text
private-content/
├── overlays/hackforger/landing/
│   ├── index.html
│   └── assets/
├── overlays/hackforger/SHA256SUMS
└── deploy/hackforger/production.env
```

Generate the manifest after every content change:

```bash
bash /path/to/hackforger/deploy/content/generate-manifest.sh \
  overlays/hackforger/landing \
  overlays/hackforger/SHA256SUMS
```

The manifest contains one lowercase SHA-256 digest, two spaces, and one path
relative to the content root per line. It must exactly cover both the regular
files and the directory structure implied by those paths. Missing, extra,
duplicate, unsafe, symlinked, FIFO, socket, device, and Git LFS pointer content
fails validation. `COPYFILE_DISABLE=1` is set for archive creation and
extraction, and the finished archive is re-extracted and checked against the
manifest before upload; this prevents macOS AppleDouble members from silently
changing the release.

## Configuration

Copy `config.example` into the private repository and replace every example
value there. Pass the private file explicitly with `--config`. The publisher
does not accept target, URL, or root overrides on the command line and has no
production defaults.

`REQUIRE_CLEAN_SOURCE=true` requires all of the following before even a dry run:

- source, manifest, and deployment config are regular files in the same Git repository;
- every source file, the manifest, and config are tracked regular blobs in `HEAD`;
- every working file is byte-for-byte equal to its `HEAD` blob;
- the repository is clean, non-shallow, on a branch, and has no graft or
  replacement refs;
- raw `HEAD` equals the branch tip returned by an actual
  `git ls-remote --exit-code` query against explicit command-line trust
  anchors; and
- no source file is a Git LFS pointer.

For SSH publishing, `--provenance-remote` must be exactly
`git@github.com:OWNER/REPO.git`, and `--provenance-ref` must be a valid
`refs/heads/*` ref matching the checked-out branch. These anchors are CLI
arguments so a modified private config or local Git remote cannot silently
redirect verification. Local fixture mode may instead name an explicit,
absolute, canonical bare repository path.

Raw Git commands run with replacement-object processing, environment-provided
repository overrides, global/system config, remote helpers, and the file
protocol disabled for SSH provenance. The publisher queries the branch, fetches
it into a fresh bare repository, enumerates the complete source subtree, and
materializes source, manifest, and config directly from that fetched commit.
Only regular executable or non-executable blobs are accepted. The complete
remote subtree must exactly match the manifest and the local working input, so
sparse checkout or `skip-worktree` cannot hide an extra remote file. The deploy
archive is built from the authoritative materialized source rather than the
working tree, eliminating a source-tree packaging race. The remote branch tip
is queried again under the deployment lock immediately before activation.

SSH transport always requires this protection. Local transport may disable it
only for disposable fixture testing. `ALLOW_INITIAL_INSTALL=false` prevents an
accidental publish to a target where the live mount is absent.

`SMOKE_ASSET` is a required, URL-safe path distinct from `index.html`. It must
be nonempty and present in the manifest. SSH activation verifies both the root
HTML and this static asset against their manifest checksums.

The SSH target must own the configured root and provide Bash, Python 3, curl,
tar, and SHA-256 tooling. SSH configuration and credentials stay outside this
repository.

## Publish

The default is a read-only plan:

```bash
bash deploy/content/publish-overlay.sh \
  --config /path/to/private/deploy/hackforger/production.env \
  --mount landing \
  --source /path/to/private/overlays/hackforger/landing \
  --manifest /path/to/private/overlays/hackforger/SHA256SUMS \
  --provenance-remote git@github.com:OWNER/PRIVATE-REPO.git \
  --provenance-ref refs/heads/main
```

After reviewing the target, release digest, file count, and current state, add
the explicit mutation flag:

```bash
bash deploy/content/publish-overlay.sh ... --apply
```

An apply performs these steps:

1. Revalidates exact tree coverage, checksums, inode types, remotely fetched Git
   provenance, and the re-extracted authoritative archive.
2. Asks the remote transaction helper to allocate an unpredictable, deployment-
   user-owned `0700` directory under `/tmp`, uploads three fixed-basename inputs,
   and verifies their ownership, inode type, link count, mode and hashes before
   acquiring the deployment lock. The private directory is removed after use.
3. Acquires the authoritative remote deployment lock.
4. For SSH, probes Linux `renameat2(RENAME_EXCHANGE)` on the live filesystem.
   A missing Python helper, unsupported kernel/filesystem, or failed probe stops
   before snapshot or live mutation. There is no non-atomic SSH fallback.
5. While still holding the lock, creates a versioned remote tar+manifest
   snapshot, transfers it to persistent local storage, re-extracts it, validates
   exact contents and hash, durably flushes both copies, and only then writes an
   acknowledgement back under the lock.
6. Extracts the new archive into a unique directory beside live and verifies it.
7. Atomically exchanges the fixed live and staging directory names with
   `RENAME_EXCHANGE`; the live path is never absent.
8. Verifies the new live tree and, over SSH, fetches both `/` and
   `/assets/landing/$SMOKE_ASSET` with identity encoding and compares hashes.
9. Preserves the former live tree in the remote backup and writes the independent
   `REMOTE_ROOT/.last-content-deploy` marker only after every check passes.

Every mutating phase is recorded by atomic replacement of a fsynced `PHASE`
journal inside the versioned remote backup. Content trees, backup files,
markers, renamed directory parents, and lock creation/removal are flushed before
the corresponding phase can be acknowledged. A nonterminal journal from a
crashed transaction blocks a later lock acquisition and names the exact backup
that must be inspected. An orderly failure before activation first records the
terminal `aborted` phase and only then durably releases the lock, so a corrected
publish can retry without bypassing crash detection.

Activation or smoke failure uses the same exchange primitive to restore the old
tree, restores the previous marker, and verifies both before reporting success.
If the exchange helper reports an error after the kernel may already have
swapped the directory names, the transaction compares both paths against the
incoming and previous manifests. A proven swap is treated as activated and is
rolled back; a proven non-swap follows pre-activation cleanup; any other state
is marked for manual recovery with the lock retained.
If any recovery action or verification fails, exit status `75` is returned, the
deployment lock is deliberately retained, and the remote backup receives a
`MANUAL_RECOVERY` file containing exact live/recovery paths. The publisher never
claims restoration in that state.

`TRANSPORT=local` uses a clearly labelled `local-test` three-rename simulation
because macOS does not provide Linux `renameat2`. It exists only for fixtures and
is never selected for SSH/production. The publisher does not compile HackForger,
replace its binary, restart a service, or change the application's `.last-deploy`
marker.

Run the local fixture suite with:

```bash
bash deploy/content/tests/run.sh
```

The suite covers remotely fetched Git provenance, explicit trust anchors,
local-dot and replacement-ref rejection, ignored/untracked and LFS rejection,
FIFO rejection on both sides,
exact cross-platform archive contents, lock-before-backup ordering, durable
phase journals, capability-probe fail-closed behavior, post-syscall exchange
errors, verified rollback, rollback-failure lock retention, and the required
second smoke asset. It also verifies that complete authoritative subtree
enumeration defeats sparse-checkout and `skip-worktree` omissions.
