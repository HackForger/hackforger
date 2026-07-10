#!/usr/bin/env python3
from __future__ import annotations

import subprocess
import tempfile
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
CHECKER = SCRIPT_DIR / "check_boundary_guard_integrity.py"


class BoundaryGuardIntegrityTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.trusted = self.root / "trusted"
        self.candidate = self.root / "candidate"
        for repo in (self.trusted, self.candidate):
            subprocess.run(["git", "init", "-q", str(repo)], check=True)
            self.write(repo, "CODEOWNERS", "* @example/reviewer\n")
            self.write(repo, ".github/workflows/boundary.yml", "name: boundary\n")
            self.write(repo, "scripts/check-public-repository-boundary.sh", "exit 0\n")
            self.write(repo, "scripts/ci/checker.py", "print('checked')\n")

    def tearDown(self) -> None:
        self.temp.cleanup()

    @staticmethod
    def write(repo: Path, rel: str, content: str) -> None:
        path = repo / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
        subprocess.run(["git", "-C", str(repo), "add", rel], check=True)

    def run_checker(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                "python3",
                str(CHECKER),
                "--trusted-root",
                str(self.trusted),
                "--candidate-root",
                str(self.candidate),
            ],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

    def test_identical_guard_and_unrelated_changes_pass(self) -> None:
        self.write(self.candidate, "README.md", "candidate documentation\n")
        self.assertEqual(0, self.run_checker().returncode)

    def test_modified_deleted_or_added_guard_file_fails(self) -> None:
        self.write(self.candidate, "scripts/ci/checker.py", "print('disabled')\n")
        result = self.run_checker()
        self.assertIn("GUARD_CHANGE\tguard-path-redacted", result.stderr)

        subprocess.run(
            ["git", "-C", str(self.candidate), "rm", "-f", "CODEOWNERS"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        self.write(self.candidate, ".github/workflows/spoof.yml", "name: spoof\n")
        address = "8" + ".8.8.8"
        self.write(
            self.candidate,
            f".github/workflows/prod-{address}.yml",
            "name: sensitive metadata\n",
        )
        self.write(self.candidate, ".github/CODEOWNERS", "* @attacker\n")
        self.write(self.candidate, ".github/dependabot.yml", "version: 2\n")
        self.write(self.candidate, ".github/dependabot.yaml", "version: 2\n")
        self.write(self.candidate, ".githooks/pre-push", "exit 0\n")
        self.write(self.candidate, "scripts/pre-push-public-boundary.sh", "exit 0\n")
        self.write(self.candidate, "docs/CODEOWNERS", "* @attacker\n")
        result = self.run_checker()
        self.assertGreaterEqual(
            result.stderr.count("GUARD_CHANGE\tguard-path-redacted"),
            8,
        )
        self.assertNotIn("spoof.yml", result.stderr)
        self.assertNotIn("dependabot.yml", result.stderr)
        self.assertNotIn(address, result.stderr)


if __name__ == "__main__":
    unittest.main()
