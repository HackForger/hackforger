#!/usr/bin/env python3
from __future__ import annotations

import os
import shlex
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
CHECKER = SCRIPT_DIR / "check_public_repository_boundary.py"
POLICY = SCRIPT_DIR / "private-content-markers.txt"
GUARD_POLICY = SCRIPT_DIR / "boundary_guard_policy.py"
PRE_PUSH = SCRIPT_DIR.parent / "pre-push-public-boundary.sh"
CANONICAL_ORIGIN = "git@github.com:HackForger/hackforger.git"


class PrePushBoundaryTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.base = Path(self.temp.name)
        self.repo = self.base / "repo"
        self.remote = self.base / "origin.git"
        subprocess.run(["git", "init", "-q", "--bare", str(self.remote)], check=True)
        subprocess.run(["git", "init", "-q", str(self.repo)], check=True)
        subprocess.run(["git", "-C", str(self.repo), "checkout", "-q", "-b", "main"], check=True)
        subprocess.run(["git", "-C", str(self.repo), "remote", "add", "origin", CANONICAL_ORIGIN], check=True)
        fake_ssh = self.base / "fake-ssh"
        fake_ssh.write_text(
            "#!/bin/sh\n"
            "case \"$*\" in\n"
            f"  *git-upload-pack*) exec git-upload-pack {shlex.quote(str(self.remote))} ;;\n"
            f"  *git-receive-pack*) exec git-receive-pack {shlex.quote(str(self.remote))} ;;\n"
            "  *) exit 97 ;;\n"
            "esac\n",
            encoding="utf-8",
        )
        fake_ssh.chmod(0o755)
        self.env = os.environ.copy()
        self.env["GIT_SSH_COMMAND"] = str(fake_ssh)
        target_ci = self.repo / "scripts/ci"
        target_ci.mkdir(parents=True)
        shutil.copy2(CHECKER, target_ci / CHECKER.name)
        shutil.copy2(POLICY, target_ci / POLICY.name)
        shutil.copy2(GUARD_POLICY, target_ci / GUARD_POLICY.name)
        shutil.copy2(PRE_PUSH, self.repo / "scripts" / PRE_PUSH.name)
        subprocess.run(
            [
                "git",
                "-C",
                str(self.repo),
                "add",
                "scripts/pre-push-public-boundary.sh",
                "scripts/ci/check_public_repository_boundary.py",
                "scripts/ci/boundary_guard_policy.py",
                "scripts/ci/private-content-markers.txt",
            ],
            check=True,
        )
        self.write("README.md", "neutral baseline\n")
        self.baseline = self.commit("baseline")
        subprocess.run(
            ["git", "-C", str(self.repo), "push", "-q", "-u", "origin", "main"],
            check=True,
            env=self.env,
        )
        subprocess.run(
            ["git", "-C", str(self.remote), "symbolic-ref", "HEAD", "refs/heads/main"],
            check=True,
        )
        self.bundle = self.base / "hooks" / self.baseline
        self.bundle.mkdir(parents=True)
        shutil.copy2(CHECKER, self.bundle / "check_public_repository_boundary.py")
        shutil.copy2(POLICY, self.bundle / "private-content-markers.txt")
        shutil.copy2(GUARD_POLICY, self.bundle / "boundary_guard_policy.py")
        shutil.copy2(PRE_PUSH, self.bundle / "pre-push")
        (self.bundle / "source-commit").write_text(f"{self.baseline}\n", encoding="ascii")
        (self.bundle / "check_public_repository_boundary.py").chmod(0o555)
        (self.bundle / "pre-push").chmod(0o555)

    def tearDown(self) -> None:
        self.temp.cleanup()

    def write(self, rel: str, content: str) -> None:
        path = self.repo / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
        subprocess.run(["git", "-C", str(self.repo), "add", "-f", rel], check=True)

    def commit(self, message: str) -> str:
        subprocess.run(
            [
                "git",
                "-C",
                str(self.repo),
                "-c",
                "user.name=Boundary Test",
                "-c",
                "user.email=boundary@example.invalid",
                "commit",
                "-q",
                "-m",
                message,
            ],
            check=True,
        )
        return subprocess.run(
            ["git", "-C", str(self.repo), "rev-parse", "HEAD"],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
        ).stdout.strip()

    def test_outgoing_add_then_delete_is_rejected_despite_replace_ref(self) -> None:
        marker = "synno" + "vator"
        self.write("temporary.md", f"tenant: {marker}\n")
        sensitive = self.commit("temporary content")
        subprocess.run(
            ["git", "-C", str(self.repo), "rm", "temporary.md"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        head = self.commit("remove temporary content")
        tree = subprocess.run(
            ["git", "-C", str(self.repo), "rev-parse", f"{head}^{{tree}}"],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
        ).stdout.strip()
        replacement = subprocess.run(
            [
                "git",
                "-C",
                str(self.repo),
                "-c",
                "user.name=Boundary Test",
                "-c",
                "user.email=boundary@example.invalid",
                "commit-tree",
                tree,
                "-p",
                self.baseline,
                "-m",
                "neutral replacement",
            ],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
        ).stdout.strip()
        subprocess.run(
            ["git", "-C", str(self.repo), "replace", head, replacement],
            check=True,
        )
        result = subprocess.run(
            [str(self.bundle / "pre-push"), "origin", str(self.remote)],
            cwd=self.repo,
            env=self.env,
            input=f"refs/heads/main {head} refs/heads/main {self.baseline}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(1, result.returncode)
        self.assertIn(
            f"HISTORY_BUSINESS_MARKER\t{sensitive[:12]}:temporary.md",
            result.stderr,
        )
        self.assertNotIn(marker, result.stderr.lower())

    def test_git_environment_overlays_cannot_redirect_history_scan(self) -> None:
        marker = "synno" + "vator"
        self.write("temporary.md", f"tenant: {marker}\n")
        sensitive = self.commit("temporary content")
        subprocess.run(
            ["git", "-C", str(self.repo), "rm", "temporary.md"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        head = self.commit("remove temporary content")
        decoy = self.base / "decoy"
        subprocess.run(["git", "init", "-q", str(decoy)], check=True)
        env = self.env.copy()
        env.update(
            {
                "GIT_DIR": str(decoy / ".git"),
                "GIT_WORK_TREE": str(decoy),
                "GIT_IMPLICIT_WORK_TREE": "0",
                "GIT_OBJECT_DIRECTORY": str(decoy / ".git/objects"),
                "GIT_ALTERNATE_OBJECT_DIRECTORIES": str(decoy / ".git/objects"),
                "GIT_INDEX_FILE": str(decoy / ".git/index"),
                "GIT_NAMESPACE": "decoy",
                "GIT_NO_REPLACE_OBJECTS": "0",
                "GIT_CONFIG_COUNT": "1",
                "GIT_CONFIG_KEY_0": "remote.origin.url",
                "GIT_CONFIG_VALUE_0": "https://github.com/example/other.git",
                "GIT_CONFIG_GLOBAL": str(decoy / "config"),
                "GIT_EXEC_PATH": str(decoy),
                "GIT_INTERNAL_SUPER_PREFIX": "decoy/",
            }
        )
        result = subprocess.run(
            [str(self.bundle / "pre-push"), "origin", str(self.remote)],
            cwd=self.repo,
            env=env,
            input=f"refs/heads/main {head} refs/heads/main {self.baseline}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(1, result.returncode, result.stderr)
        self.assertIn(
            f"HISTORY_BUSINESS_MARKER\t{sensitive[:12]}:temporary.md",
            result.stderr,
        )
        self.assertNotIn(marker, result.stderr.lower())

    def test_legacy_graft_cannot_hide_sensitive_add_then_delete(self) -> None:
        marker = "synno" + "vator"
        self.write("temporary.md", f"tenant: {marker}\n")
        self.commit("temporary content")
        subprocess.run(
            ["git", "-C", str(self.repo), "rm", "temporary.md"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        head = self.commit("remove temporary content")
        grafts = Path(
            subprocess.run(
                [
                    "git",
                    "-C",
                    str(self.repo),
                    "rev-parse",
                    "--path-format=absolute",
                    "--git-path",
                    "info/grafts",
                ],
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()
        )
        grafts.parent.mkdir(parents=True, exist_ok=True)
        grafts.write_text(f"{head} {self.baseline}\n", encoding="ascii")
        result = subprocess.run(
            [str(self.bundle / "pre-push"), "origin", str(self.remote)],
            cwd=self.repo,
            env=self.env,
            input=f"refs/heads/main {head} refs/heads/main {self.baseline}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(2, result.returncode)
        self.assertIn("legacy Git grafts are not allowed", result.stderr)
        self.assertNotIn(marker, result.stderr.lower())

    def test_shallow_boundary_cannot_hide_sensitive_add_then_delete(self) -> None:
        marker = "synno" + "vator"
        self.write("temporary.md", f"tenant: {marker}\n")
        self.commit("temporary content")
        subprocess.run(
            ["git", "-C", str(self.repo), "rm", "temporary.md"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        head = self.commit("remove temporary content")
        shallow = Path(
            subprocess.run(
                [
                    "git",
                    "-C",
                    str(self.repo),
                    "rev-parse",
                    "--path-format=absolute",
                    "--git-path",
                    "shallow",
                ],
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()
        )
        shallow.write_text(f"{head}\n", encoding="ascii")
        result = subprocess.run(
            [str(self.bundle / "pre-push"), "origin", str(self.remote)],
            cwd=self.repo,
            env=self.env,
            input=f"refs/heads/main {head} refs/heads/main {self.baseline}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(2, result.returncode)
        self.assertIn("shallow repositories are not allowed", result.stderr)
        self.assertNotIn(marker, result.stderr.lower())

    def test_annotated_tag_is_rejected_before_metadata_can_escape(self) -> None:
        marker = "synno" + "vator"
        subprocess.run(
            [
                "git",
                "-C",
                str(self.repo),
                "-c",
                "user.name=Boundary Test",
                "-c",
                "user.email=boundary@example.invalid",
                "tag",
                "-a",
                "release-test",
                "-m",
                f"tenant: {marker}",
            ],
            check=True,
        )
        tag_oid = subprocess.run(
            ["git", "-C", str(self.repo), "rev-parse", "refs/tags/release-test"],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
        ).stdout.strip()
        result = subprocess.run(
            [str(self.bundle / "pre-push"), "origin", str(self.remote)],
            cwd=self.repo,
            env=self.env,
            input=f"refs/tags/release-test {tag_oid} refs/tags/release-test {'0' * 40}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(2, result.returncode)
        self.assertIn("annotated tags or non-commit refs require dedicated security review", result.stderr)
        self.assertNotIn(marker, result.stderr.lower())

    def test_sensitive_ref_name_is_rejected_even_without_new_commits(self) -> None:
        address = "8" + ".8.8.8"
        host = "api.corp." + "internal"
        ref_name = f"refs/heads/prod-{address}-{host}"
        result = subprocess.run(
            [str(self.bundle / "pre-push"), "origin", str(self.remote)],
            cwd=self.repo,
            env=self.env,
            input=f"{ref_name} {self.baseline} {ref_name} {self.baseline}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(1, result.returncode)
        self.assertIn("REF_INTERNAL_HOST\tref-name-redacted", result.stderr)
        self.assertIn("REF_NON_DOCUMENTATION_IP\tref-name-redacted", result.stderr)
        self.assertNotIn(address, result.stderr)
        self.assertNotIn(host, result.stderr)

    def test_ref_deletion_skips_annotated_object_and_sensitive_name(self) -> None:
        address = "8" + ".8.8.8"
        host = "api.corp." + "internal"
        ref_name = f"refs/tags/prod-{address}-{host}"
        subprocess.run(
            [
                "git",
                "-C",
                str(self.repo),
                "-c",
                "user.name=Boundary Test",
                "-c",
                "user.email=boundary@example.invalid",
                "tag",
                "-a",
                "delete-me",
                "-m",
                "neutral tag",
            ],
            check=True,
        )
        tag_oid = subprocess.run(
            ["git", "-C", str(self.repo), "rev-parse", "refs/tags/delete-me"],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
        ).stdout.strip()
        grafts = Path(
            subprocess.run(
                ["git", "-C", str(self.repo), "rev-parse", "--path-format=absolute", "--git-path", "info/grafts"],
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()
        )
        grafts.parent.mkdir(parents=True, exist_ok=True)
        grafts.write_text(f"{self.baseline} {self.baseline}\n", encoding="ascii")
        shallow = Path(
            subprocess.run(
                ["git", "-C", str(self.repo), "rev-parse", "--path-format=absolute", "--git-path", "shallow"],
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()
        )
        shallow.write_text(f"{self.baseline}\n", encoding="ascii")
        result = subprocess.run(
            [str(self.bundle / "pre-push"), "origin", str(self.remote)],
            cwd=self.repo,
            env=self.env,
            input=f"(delete) {'0' * 40} {ref_name} {tag_oid}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("PASS (ref deletions only)", result.stdout)
        self.assertNotIn(address, result.stderr)
        self.assertNotIn(host, result.stderr)

    def test_clean_update_to_existing_default_ref_passes(self) -> None:
        self.write("README.md", "neutral baseline\nneutral update\n")
        head = self.commit("neutral update")
        result = subprocess.run(
            [str(self.bundle / "pre-push"), "origin", str(self.remote)],
            cwd=self.repo,
            env=self.env,
            input=f"refs/heads/main {head} refs/heads/main {self.baseline}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("public repository boundary: PASS", result.stdout)

    def test_checkout_baseline_environment_override_is_ignored(self) -> None:
        self.write("README.md", "neutral baseline\nneutral update\n")
        head = self.commit("neutral update")
        env = self.env.copy()
        env["HACKFORGER_BOUNDARY_BASELINE_REF"] = "refs/heads/not-a-trusted-baseline"
        result = subprocess.run(
            [str(self.bundle / "pre-push"), "origin", str(self.remote)],
            cwd=self.repo,
            env=env,
            input=f"refs/heads/main {head} refs/heads/main {self.baseline}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("public repository boundary: PASS", result.stdout)


if __name__ == "__main__":
    unittest.main()
