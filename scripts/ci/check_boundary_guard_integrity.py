#!/usr/bin/env python3
"""Require boundary-enforcement files to match the trusted default branch."""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

from boundary_guard_policy import is_guarded_path



def tracked_guard_entries(root: Path) -> dict[str, tuple[str, str]]:
    proc = subprocess.run(
        ["git", "-C", str(root), "ls-files", "--stage", "-z"],
        check=True,
        stdout=subprocess.PIPE,
    )
    entries: dict[str, tuple[str, str]] = {}
    for raw in proc.stdout.split(b"\0"):
        if not raw:
            continue
        metadata, path_bytes = raw.split(b"\t", 1)
        mode, oid, stage = metadata.decode("ascii").split()
        path = path_bytes.decode("utf-8", "surrogateescape")
        if stage != "0":
            continue
        if is_guarded_path(path):
            entries[path] = (mode, oid)
    return entries


def changed_paths(trusted_root: Path, candidate_root: Path) -> list[str]:
    trusted = tracked_guard_entries(trusted_root)
    candidate = tracked_guard_entries(candidate_root)
    return sorted(
        path
        for path in trusted.keys() | candidate.keys()
        if trusted.get(path) != candidate.get(path)
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--trusted-root", type=Path, required=True)
    parser.add_argument("--candidate-root", type=Path, required=True)
    args = parser.parse_args()
    changes = changed_paths(args.trusted_root.resolve(), args.candidate_root.resolve())
    if changes:
        print("public boundary guard integrity: FAIL", file=sys.stderr)
        for _path in changes:
            print("GUARD_CHANGE\tguard-path-redacted", file=sys.stderr)
        print(
            "Boundary guard changes require a dedicated security-owner review and ruleset bypass.",
            file=sys.stderr,
        )
        return 1
    print("public boundary guard integrity: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
