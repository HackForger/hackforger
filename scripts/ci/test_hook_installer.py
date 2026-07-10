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
SOURCE_ROOT = SCRIPT_DIR.parent.parent
INSTALLER = SOURCE_ROOT / "scripts/install-public-boundary-hook.sh"
PRE_PUSH = SOURCE_ROOT / "scripts/pre-push-public-boundary.sh"
CHECKER = SCRIPT_DIR / "check_public_repository_boundary.py"
GUARD_POLICY = SCRIPT_DIR / "boundary_guard_policy.py"
POLICY = SCRIPT_DIR / "private-content-markers.txt"
CANONICAL_ORIGIN = "git@github.com:HackForger/hackforger.git"


class HookInstallerTest(unittest.TestCase):
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
        self.write("README.md", "neutral pre-hook worktree\n")
        self.pre_hook = self.commit("pre-hook baseline")
        self.copy_source(INSTALLER, "scripts/install-public-boundary-hook.sh", 0o755)
        self.copy_source(PRE_PUSH, "scripts/pre-push-public-boundary.sh", 0o755)
        self.copy_source(CHECKER, "scripts/ci/check_public_repository_boundary.py", 0o755)
        self.copy_source(GUARD_POLICY, "scripts/ci/boundary_guard_policy.py", 0o644)
        self.copy_source(POLICY, "scripts/ci/private-content-markers.txt", 0o644)
        self.baseline = self.commit("trusted default")
        subprocess.run(
            ["git", "-C", str(self.repo), "push", "-q", "origin", "main"],
            check=True,
            env=self.env,
        )
        subprocess.run(
            ["git", "-C", str(self.remote), "symbolic-ref", "HEAD", "refs/heads/main"],
            check=True,
        )
        self.common_dir = Path(
            subprocess.run(
                ["git", "-C", str(self.repo), "rev-parse", "--path-format=absolute", "--git-common-dir"],
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()
        )

    def tearDown(self) -> None:
        self.temp.cleanup()

    def copy_source(self, source: Path, rel: str, mode: int) -> None:
        target = self.repo / rel
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source, target)
        target.chmod(mode)
        subprocess.run(["git", "-C", str(self.repo), "add", rel], check=True)

    def write(self, rel: str, content: str) -> None:
        target = self.repo / rel
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")
        subprocess.run(["git", "-C", str(self.repo), "add", rel], check=True)

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

    def run_installer(self, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(self.repo / "scripts/install-public-boundary-hook.sh")],
            cwd=self.repo,
            env=env or self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

    def hooks_path(self) -> Path:
        return Path(
            subprocess.run(
                ["git", "-C", str(self.repo), "config", "--get", "core.hooksPath"],
                check=True,
                text=True,
                stdout=subprocess.PIPE,
            ).stdout.strip()
        )

    def test_installs_remote_default_bundle_not_checkout_controlled_hook(self) -> None:
        checkout_hook = self.repo / "scripts/pre-push-public-boundary.sh"
        checkout_hook.write_text("#!/bin/sh\nexit 77\n", encoding="utf-8")
        result = self.run_installer()
        self.assertEqual(0, result.returncode, result.stderr)

        hooks_path = self.hooks_path()
        self.assertTrue(hooks_path.is_absolute())
        self.assertEqual(self.common_dir / "hackforger-boundary-hooks" / self.baseline, hooks_path)
        effective = subprocess.run(
            ["git", "-C", str(self.repo), "rev-parse", "--path-format=absolute", "--git-path", "hooks"],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
        ).stdout.strip()
        self.assertEqual(str(hooks_path), effective)
        self.assertTrue((hooks_path / "pre-push").is_file())
        self.assertTrue((hooks_path / "pre-push").stat().st_mode & 0o111)
        self.assertEqual(0, hooks_path.stat().st_mode & 0o222)
        self.assertEqual(0, (hooks_path / "pre-push").stat().st_mode & 0o222)
        trusted_hook = subprocess.run(
            ["git", "-C", str(self.repo), "show", f"{self.baseline}:scripts/pre-push-public-boundary.sh"],
            check=True,
            stdout=subprocess.PIPE,
        ).stdout
        self.assertEqual(trusted_hook, (hooks_path / "pre-push").read_bytes())
        self.assertNotEqual(checkout_hook.read_bytes(), (hooks_path / "pre-push").read_bytes())

    def test_installer_ignores_git_environment_repository_overlays(self) -> None:
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
        result = self.run_installer(env)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertEqual(self.common_dir / "hackforger-boundary-hooks" / self.baseline, self.hooks_path())

    def test_explicit_canonical_sample_only_hooks_path_can_switch(self) -> None:
        canonical = self.common_dir / "hooks"
        subprocess.run(
            ["git", "-C", str(self.repo), "config", "core.hooksPath", str(canonical)],
            check=True,
        )
        result = self.run_installer()
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertNotEqual(canonical, self.hooks_path())

    def test_installed_hook_blocks_sensitive_push_from_pre_hook_worktree(self) -> None:
        result = self.run_installer()
        self.assertEqual(0, result.returncode, result.stderr)
        legacy = self.base / "legacy-worktree"
        subprocess.run(
            ["git", "-C", str(self.repo), "worktree", "add", "-q", "-b", "legacy", str(legacy), self.pre_hook],
            check=True,
        )
        marker = "synno" + "vator"
        sensitive = legacy / "legacy-note.md"
        sensitive.write_text(f"tenant: {marker}\n", encoding="utf-8")
        subprocess.run(["git", "-C", str(legacy), "add", "legacy-note.md"], check=True)
        subprocess.run(
            [
                "git",
                "-C",
                str(legacy),
                "-c",
                "user.name=Boundary Test",
                "-c",
                "user.email=boundary@example.invalid",
                "commit",
                "-q",
                "-m",
                "legacy sensitive change",
            ],
            check=True,
        )
        push = subprocess.run(
            ["git", "-C", str(legacy), "push", "origin", "legacy"],
            env=self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertNotEqual(0, push.returncode)
        self.assertIn("must contain the installed boundary source commit", push.stderr)
        remote_ref = subprocess.run(
            ["git", "-C", str(self.remote), "show-ref", "--verify", "refs/heads/legacy"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertNotEqual(0, remote_ref.returncode)

    def test_guard_change_fails_closed_until_atomic_reinstall(self) -> None:
        first_install = self.run_installer()
        self.assertEqual(0, first_install.returncode, first_install.stderr)
        old_hooks_path = self.hooks_path()

        checker = self.repo / "scripts/ci/check_public_repository_boundary.py"
        checker.write_text(checker.read_text(encoding="utf-8") + "\n# guard revision\n", encoding="utf-8")
        subprocess.run(["git", "-C", str(self.repo), "add", str(checker.relative_to(self.repo))], check=True)
        guard_commit = self.commit("revise guard")
        subprocess.run(
            ["git", "-C", str(self.repo), "push", "--no-verify", "-q", "origin", "main"],
            check=True,
            env=self.env,
        )
        self.write("neutral.txt", "neutral follow-up\n")
        self.commit("neutral follow-up")
        stale_push = subprocess.run(
            ["git", "-C", str(self.repo), "push", "origin", "main"],
            env=self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertNotEqual(0, stale_push.returncode)
        self.assertIn("installed boundary guard is stale", stale_push.stderr)

        reinstall = self.run_installer()
        self.assertEqual(0, reinstall.returncode, reinstall.stderr)
        self.assertEqual(self.common_dir / "hackforger-boundary-hooks" / guard_commit, self.hooks_path())
        self.assertNotEqual(old_hooks_path, self.hooks_path())
        recovered_push = subprocess.run(
            ["git", "-C", str(self.repo), "push", "origin", "main"],
            env=self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(0, recovered_push.returncode, recovered_push.stderr)

    def test_real_canonical_hook_is_not_replaced(self) -> None:
        real_hook = self.common_dir / "hooks/pre-push"
        real_hook.write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
        real_hook.chmod(0o755)
        result = self.run_installer()
        self.assertEqual(1, result.returncode)
        self.assertIn("canonical hooks directory contains a real hook", result.stderr)
        self.assertEqual("#!/bin/sh\nexit 0\n", real_hook.read_text(encoding="utf-8"))
        configured = subprocess.run(
            ["git", "-C", str(self.repo), "config", "--get", "core.hooksPath"],
            text=True,
            stdout=subprocess.PIPE,
        )
        self.assertEqual(1, configured.returncode)

    def test_conflicting_hooks_path_is_not_replaced(self) -> None:
        other = self.base / "team-hooks"
        other.mkdir()
        subprocess.run(
            ["git", "-C", str(self.repo), "config", "core.hooksPath", str(other)],
            check=True,
        )
        result = self.run_installer()
        self.assertEqual(1, result.returncode)
        self.assertIn("refusing to replace it", result.stderr)
        self.assertEqual(other, self.hooks_path())

    def test_noncanonical_origin_is_rejected(self) -> None:
        subprocess.run(
            ["git", "-C", str(self.repo), "remote", "set-url", "origin", "git@github.com:example/other.git"],
            check=True,
        )
        result = self.run_installer()
        self.assertEqual(1, result.returncode)
        self.assertIn("origin must be the canonical HackForger/hackforger", result.stderr)

    def test_effective_hook_rejects_origin_changed_after_install(self) -> None:
        result = self.run_installer()
        self.assertEqual(0, result.returncode, result.stderr)
        subprocess.run(
            ["git", "-C", str(self.repo), "remote", "set-url", "origin", "https://github.com/example/other.git"],
            check=True,
        )
        self.write("neutral.txt", "neutral update\n")
        head = self.commit("neutral update")
        hook = subprocess.run(
            [str(self.hooks_path() / "pre-push"), "origin", "unused"],
            cwd=self.repo,
            env=self.env,
            input=f"refs/heads/main {head} refs/heads/main {self.baseline}\n",
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(2, hook.returncode)
        self.assertIn("origin must be the canonical HackForger/hackforger", hook.stderr)


if __name__ == "__main__":
    unittest.main()
