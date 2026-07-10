"""Canonical path policy for self-protected public-boundary enforcement."""

from __future__ import annotations


GUARDED_FILES = {
    ".github/CODEOWNERS",
    ".github/dependabot.yaml",
    ".github/dependabot.yml",
    ".agents/skills/hackforger-development",
    ".claude/skills/hackforger-development",
    "AGENTS.md",
    "CODEOWNERS",
    "docs/CODEOWNERS",
    "scripts/check-public-repository-boundary.sh",
    "scripts/install-public-boundary-hook.sh",
    "scripts/pre-push-public-boundary.sh",
    "skills/hackforger-development/SKILL.md",
}
GUARDED_PREFIXES = (".githooks/", ".github/workflows/", "scripts/ci/")


def is_guarded_path(path: str) -> bool:
    return path in GUARDED_FILES or path.startswith(GUARDED_PREFIXES)
