#!/usr/bin/env python3
from __future__ import annotations

import subprocess
import tempfile
import unittest
import os
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
CHECKER = SCRIPT_DIR / "check_public_repository_boundary.py"
POLICY = SCRIPT_DIR / "private-content-markers.txt"


class BoundaryCheckerTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        subprocess.run(["git", "init", "-q", str(self.root)], check=True)

    def tearDown(self) -> None:
        self.temp.cleanup()

    def write(self, path: str, content: str = "neutral\n") -> None:
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")
        subprocess.run(["git", "-C", str(self.root), "add", "-f", path], check=True)

    def write_bytes(self, path: str, content: bytes) -> None:
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(content)
        subprocess.run(["git", "-C", str(self.root), "add", "-f", path], check=True)

    def commit(self, message: str, *, author_name: str = "Boundary Test") -> str:
        subprocess.run(
            [
                "git",
                "-C",
                str(self.root),
                "-c",
                f"user.name={author_name}",
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
            ["git", "-C", str(self.root), "rev-parse", "HEAD"],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
        ).stdout.strip()

    @staticmethod
    def real_ipv6() -> str:
        return "2606" + ":4700" + ":4700" + ":" + ":1111"

    def run_checker(
        self,
        baseline: Path | None = None,
        baseline_ref: str | None = None,
        history_base: str | None = None,
        history_head: str | None = None,
        ref_name: str | None = None,
    ) -> subprocess.CompletedProcess[str]:
        command = ["python3", str(CHECKER), "--root", str(self.root), "--policy", str(POLICY)]
        if baseline_ref is not None:
            command.extend(["--baseline-ref", baseline_ref])
        else:
            baseline = baseline or self.root
            command.extend(["--baseline-root", str(baseline)])
        if history_base is not None or history_head is not None:
            self.assertIsNotNone(history_base)
            self.assertIsNotNone(history_head)
            command.extend(["--history-base-ref", history_base, "--history-head-ref", history_head])
        if ref_name is not None:
            command.extend(["--ref-name", ref_name])
        return subprocess.run(
            command,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

    def test_neutral_content_passes(self) -> None:
        self.write("docs/guide.md", "See https://example.invalid and 203.0.113.10\n")
        self.assertEqual(0, self.run_checker().returncode)

    def test_business_marker_fails_without_echoing_content(self) -> None:
        marker = "synno" + "vator"
        self.write("README.md", f"tenant: {marker}\n")
        result = self.run_checker()
        self.assertEqual(1, result.returncode)
        self.assertIn("BUSINESS_MARKER\tREADME.md", result.stderr)
        self.assertNotIn(marker, result.stderr.lower())

        path = "docs/" + marker.title() + "-runbook.md"
        self.write(path)
        result = self.run_checker()
        self.assertIn("BUSINESS_MARKER\tpath-redacted", result.stderr)
        self.assertIn("PATH_BUSINESS_MARKER\tpath-redacted", result.stderr)
        self.assertNotIn(marker, result.stderr.lower())

    def test_sensitive_file_name_metadata_fails_without_echo(self) -> None:
        address = "8" + ".8.8.8"
        host = "api.corp." + "internal"
        material = "gh" + "p_" + "a" * 24
        path = f"docs/release-{address}-{host}-{material}.md"
        self.write(path, f"host={host}\n")
        result = self.run_checker()
        self.assertIn("PATH_NON_DOCUMENTATION_IP\tpath-redacted", result.stderr)
        self.assertIn("PATH_INTERNAL_HOST\tpath-redacted", result.stderr)
        self.assertIn("PATH_SECRET_MATERIAL\tpath-redacted", result.stderr)
        self.assertNotIn(address, result.stderr)
        self.assertNotIn(host, result.stderr)
        self.assertNotIn(material, result.stderr)

    def test_ip_before_filename_extension_and_sentence_period_fails(self) -> None:
        address = "8" + ".8.8.8"
        path = f"docs/prod-{address}.md"
        self.write(path)
        result = self.run_checker()
        self.assertIn("PATH_NON_DOCUMENTATION_IP\tpath-redacted", result.stderr)
        self.assertNotIn(address, result.stderr)

        subprocess.run(
            ["git", "-C", str(self.root), "rm", "-f", path],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        self.write("docs/host.md", f"host={address}.\n")
        result = self.run_checker()
        self.assertIn("NON_DOCUMENTATION_IP\tdocs/host.md", result.stderr)

    def test_five_component_numeric_sequence_is_not_an_ipv4_submatch(self) -> None:
        self.write("docs/version.md", "version=1.8.8.8.8\n")
        self.assertEqual(0, self.run_checker().returncode)

    def test_real_ipv6_content_path_and_ref_fail_without_echo(self) -> None:
        address = self.real_ipv6()
        self.write("README.md", f"host=[{address}]\n")
        result = self.run_checker()
        self.assertIn("NON_DOCUMENTATION_IPV6\tREADME.md", result.stderr)
        self.assertNotIn(address, result.stderr)

        path = f"docs/prod-[{address}].md"
        self.write(path)
        result = self.run_checker()
        self.assertIn("PATH_NON_DOCUMENTATION_IPV6\tpath-redacted", result.stderr)
        self.assertNotIn(address, result.stderr)

        result = self.run_checker(ref_name=f"refs/heads/prod-[{address}]")
        self.assertIn("REF_NON_DOCUMENTATION_IPV6\tref-name-redacted", result.stderr)
        self.assertNotIn(address, result.stderr)

    def test_documentation_and_loopback_ipv6_addresses_pass(self) -> None:
        self.write(
            "docs/network.md",
            "unspecified=::\nloopback=::1\ndocumentation=2001:db8::1234\n",
        )
        self.assertEqual(0, self.run_checker().returncode)

    def test_intermediate_ipv6_content_is_scanned_after_deletion(self) -> None:
        self.write("README.md", "neutral baseline\n")
        base = self.commit("baseline")
        address = self.real_ipv6()
        self.write("temporary.md", f"host=[{address}]\n")
        sensitive_commit = self.commit("temporary network metadata")
        subprocess.run(
            ["git", "-C", str(self.root), "rm", "temporary.md"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        head = self.commit("remove temporary metadata")
        result = self.run_checker(history_base=base, history_head=head)
        self.assertIn(
            f"HISTORY_NON_DOCUMENTATION_IPV6\t{sensitive_commit[:12]}:temporary.md",
            result.stderr,
        )
        self.assertNotIn(address, result.stderr)

    def test_ipv6_in_raw_commit_metadata_is_scanned(self) -> None:
        self.write("README.md", "neutral baseline\n")
        base = self.commit("baseline")
        address = self.real_ipv6()
        self.write("README.md", "neutral update\n")
        sensitive_commit = self.commit("neutral message", author_name=f"operator [{address}]")
        result = self.run_checker(history_base=base, history_head=sensitive_commit)
        self.assertIn(
            f"HISTORY_NON_DOCUMENTATION_IPV6\t{sensitive_commit[:12]}:COMMIT_METADATA",
            result.stderr,
        )
        self.assertNotIn(address, result.stderr)

    def test_business_marker_in_ref_name_fails(self) -> None:
        self.write("README.md")
        marker = "synno" + "vator"
        result = self.run_checker(ref_name=f"refs/heads/chore/{marker}-content")
        self.assertIn("REF_BUSINESS_MARKER", result.stderr)
        self.assertNotIn(marker, result.stderr.lower())

    def test_infrastructure_facts_in_ref_name_fail_without_echo(self) -> None:
        self.write("README.md")
        address = "8" + ".8.8.8"
        host = "api.corp." + "internal"
        ref_name = f"refs/heads/prod-{address}-{host}"
        result = self.run_checker(ref_name=ref_name)
        self.assertIn("REF_INTERNAL_HOST\tref-name-redacted", result.stderr)
        self.assertIn("REF_NON_DOCUMENTATION_IP\tref-name-redacted", result.stderr)
        self.assertNotIn(address, result.stderr)
        self.assertNotIn(host, result.stderr)

    def test_staged_blob_is_scanned_when_worktree_bytes_differ(self) -> None:
        marker = "synno" + "vator"
        self.write("README.md", f"tenant: {marker}\n")
        (self.root / "README.md").write_text("neutral worktree\n", encoding="utf-8")
        result = self.run_checker()
        self.assertIn("BUSINESS_MARKER\tREADME.md", result.stderr)
        self.assertNotIn(marker, result.stderr.lower())

    def test_intermediate_commit_is_scanned_after_file_is_deleted(self) -> None:
        self.write("README.md", "neutral baseline\n")
        base = self.commit("baseline")
        marker = "synno" + "vator"
        self.write("temporary.md", f"tenant: {marker}\n")
        sensitive_commit = self.commit("temporary content")
        subprocess.run(
            ["git", "-C", str(self.root), "rm", "temporary.md"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        head = self.commit("remove temporary content")
        result = self.run_checker(history_base=base, history_head=head)
        self.assertIn(
            f"HISTORY_BUSINESS_MARKER\t{sensitive_commit[:12]}:temporary.md",
            result.stderr,
        )

    def test_deleting_legacy_forbidden_path_is_allowed(self) -> None:
        self.write("docs/ops/legacy-runbook.md", "legacy public content\n")
        base = self.commit("legacy baseline")
        subprocess.run(
            ["git", "-C", str(self.root), "rm", "docs/ops/legacy-runbook.md"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        head = self.commit("remove legacy runbook")
        result = self.run_checker(history_base=base, history_head=head)
        self.assertEqual(0, result.returncode, result.stderr)

    def test_sensitive_deleted_file_name_is_scanned_without_echo(self) -> None:
        self.write("README.md", "neutral baseline\n")
        base = self.commit("baseline")
        address = "8" + ".8.8.8"
        host = "api.corp." + "internal"
        material = "gh" + "p_" + "a" * 24
        path = f"docs/release-{address}-{host}-{material}.md"
        self.write(path)
        sensitive_commit = self.commit("temporary path metadata")
        subprocess.run(
            ["git", "-C", str(self.root), "rm", path],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        head = self.commit("remove temporary path")
        result = self.run_checker(history_base=base, history_head=head)
        self.assertIn(
            f"HISTORY_PATH_NON_DOCUMENTATION_IP\t{sensitive_commit[:12]}:path-redacted",
            result.stderr,
        )
        self.assertIn(
            f"HISTORY_PATH_INTERNAL_HOST\t{sensitive_commit[:12]}:path-redacted",
            result.stderr,
        )
        self.assertIn(
            f"HISTORY_PATH_SECRET_MATERIAL\t{sensitive_commit[:12]}:path-redacted",
            result.stderr,
        )
        self.assertNotIn(address, result.stderr)
        self.assertNotIn(host, result.stderr)
        self.assertNotIn(material, result.stderr)

    def test_raw_commit_metadata_is_scanned_without_echo(self) -> None:
        self.write("README.md", "neutral baseline\n")
        base = self.commit("baseline")
        marker = "synno" + "vator"
        self.write("README.md", "neutral update\n")
        sensitive_commit = self.commit("neutral message", author_name=marker)
        result = self.run_checker(history_base=base, history_head=sensitive_commit)
        self.assertIn(
            f"HISTORY_BUSINESS_MARKER\t{sensitive_commit[:12]}:COMMIT_METADATA",
            result.stderr,
        )
        self.assertNotIn(marker, result.stderr.lower())

    def test_guard_modify_then_restore_is_rejected_from_history(self) -> None:
        self.write(".github/workflows/boundary.yml", "name: trusted\n")
        base = self.commit("baseline")
        self.write(".github/workflows/boundary.yml", "name: disabled\n")
        modified = self.commit("temporarily modify guard")
        self.write(".github/workflows/boundary.yml", "name: trusted\n")
        restored = self.commit("restore guard")
        result = self.run_checker(history_base=base, history_head=restored)
        self.assertIn(
            f"HISTORY_GUARD_CHANGE\t{modified[:12]}:guard-path-redacted",
            result.stderr,
        )

    def test_merge_of_current_trusted_guard_version_is_allowed(self) -> None:
        self.write(".github/workflows/boundary.yml", "name: trusted-v1\n")
        old_base = self.commit("old trusted baseline")
        subprocess.run(
            ["git", "-C", str(self.root), "checkout", "-q", "-b", "feature"],
            check=True,
        )
        self.write("README.md", "feature work\n")
        self.commit("feature work")

        subprocess.run(
            ["git", "-C", str(self.root), "checkout", "-q", "-b", "trusted", old_base],
            check=True,
        )
        self.write(".github/workflows/boundary.yml", "name: trusted-v2\n")
        target_base = self.commit("audited guard update")

        subprocess.run(
            ["git", "-C", str(self.root), "checkout", "-q", "feature"],
            check=True,
        )
        subprocess.run(
            [
                "git",
                "-C",
                str(self.root),
                "-c",
                "user.name=Boundary Test",
                "-c",
                "user.email=boundary@example.invalid",
                "merge",
                "-q",
                "--no-ff",
                "trusted",
                "-m",
                "merge current trusted policy",
            ],
            check=True,
        )
        head = subprocess.run(
            ["git", "-C", str(self.root), "rev-parse", "HEAD"],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
        ).stdout.strip()
        result = self.run_checker(history_base=target_base, history_head=head)
        self.assertNotIn("HISTORY_GUARD_CHANGE", result.stderr)

    def test_secret_material_fails_outside_scoped_directories(self) -> None:
        material = "gh" + "p_" + "a" * 24
        self.write("README.md", f"credential: {material}\n")
        result = self.run_checker()
        self.assertIn("SECRET_MATERIAL\tREADME.md", result.stderr)
        self.assertNotIn(material, result.stderr)

        webhook = "https://hooks." + "slack.com/services/T000/B000/secret-part"
        self.write("README.md", webhook)
        result = self.run_checker()
        self.assertIn("SECRET_MATERIAL\tREADME.md", result.stderr)
        self.assertNotIn(webhook, result.stderr)

    def test_private_paths_fail(self) -> None:
        self.write("custom/public/assets/landing/index.html")
        self.assertIn("BOUNDARY_PATH", self.run_checker().stderr)

        self.write("Docs/Ops/runbook.md")
        self.assertIn("BOUNDARY_PATH\tDocs/Ops/runbook.md", self.run_checker().stderr)

        self.write(".worktrees/runtime/data.bin")
        self.assertIn("BOUNDARY_PATH\t.worktrees/runtime/data.bin", self.run_checker().stderr)

        self.write("scripts/cleanup-invalid-hackathons.sql")
        self.assertIn(
            "BOUNDARY_PATH\tscripts/cleanup-invalid-hackathons.sql",
            self.run_checker().stderr,
        )

    def test_project_owned_opaque_and_generated_files_fail(self) -> None:
        self.write("docs/guide.md", "neutral\x00hidden\n")
        result = self.run_checker()
        self.assertIn("UNREVIEWED_BINARY\tdocs/guide.md", result.stderr)

        self.write("scripts/ci/__pycache__/checker.pyc", "opaque\x00bytes")
        result = self.run_checker()
        self.assertIn("GENERATED_ARTIFACT\tscripts/ci/__pycache__/checker.pyc", result.stderr)
        self.assertIn("UNREVIEWED_BINARY\tscripts/ci/__pycache__/checker.pyc", result.stderr)

    def test_lfs_pointer_fails_in_any_directory(self) -> None:
        self.write(
            "public/assets/customer.dat",
            "\n".join(
                (
                    "version https://git-lfs.github.com/spec/v1",
                    "oid sha256:" + "a" * 64,
                    "size 1234",
                    "",
                )
            ),
        )
        self.assertIn(
            "LFS_POINTER\tpublic/assets/customer.dat",
            self.run_checker().stderr,
        )

    def test_changed_opaque_file_outside_owned_tree_requires_review(self) -> None:
        with tempfile.TemporaryDirectory() as baseline_dir:
            baseline = Path(baseline_dir)
            target = baseline / "modules/example/fixture.bin"
            target.parent.mkdir(parents=True)
            target.write_bytes(b"baseline\x00bytes")
            self.write("modules/example/fixture.bin", "baseline\x00bytes")
            self.assertEqual(0, self.run_checker(baseline).returncode)

            self.write("modules/example/fixture.bin", "changed\x00bytes")
            result = self.run_checker(baseline)
            self.assertIn("UNREVIEWED_BINARY\tmodules/example/fixture.bin", result.stderr)

    def test_delayed_nul_and_invalid_utf8_are_opaque(self) -> None:
        with tempfile.TemporaryDirectory() as baseline_dir:
            baseline = Path(baseline_dir)
            self.write_bytes("assets/customer.data", b"A" * 8192 + b"\x00private")
            result = self.run_checker(baseline)
            self.assertIn("UNREVIEWED_BINARY\tassets/customer.data", result.stderr)

            self.write_bytes("assets/customer.data", b"A" * 8192 + b"\xffprivate")
            result = self.run_checker(baseline)
            self.assertIn("UNREVIEWED_BINARY\tassets/customer.data", result.stderr)

    def test_git_ref_baseline_detects_changed_opaque_file(self) -> None:
        self.write("modules/example/fixture.bin", "baseline\x00bytes")
        subprocess.run(
            [
                "git",
                "-C",
                str(self.root),
                "-c",
                "user.name=Boundary Test",
                "-c",
                "user.email=boundary@example.invalid",
                "commit",
                "-q",
                "-m",
                "baseline",
            ],
            check=True,
        )
        baseline_ref = subprocess.run(
            ["git", "-C", str(self.root), "rev-parse", "HEAD"],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
        ).stdout.strip()
        self.write("modules/example/fixture.bin", "changed\x00bytes")
        result = self.run_checker(baseline_ref=baseline_ref)
        self.assertIn("UNREVIEWED_BINARY\tmodules/example/fixture.bin", result.stderr)

    def test_cli_requires_an_opaque_file_baseline(self) -> None:
        self.write("README.md")
        result = subprocess.run(
            ["python3", str(CHECKER), "--root", str(self.root), "--policy", str(POLICY)],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(2, result.returncode)
        self.assertIn("one of the arguments --baseline-root --baseline-ref is required", result.stderr)

    def test_personal_path_and_real_ip_fail(self) -> None:
        personal = "/" + "Users/alice/work"
        address = "8" + ".8.8.8"
        self.write("deploy/example.md", f"root={personal}\nhost={address}\n")
        result = self.run_checker()
        self.assertIn("PERSONAL_PATH", result.stderr)
        self.assertIn("NON_DOCUMENTATION_IP", result.stderr)

        self.write("deploy/example.md", "root=/" + "Users/alice\n")
        self.assertIn("PERSONAL_PATH", self.run_checker().stderr)

        self.write("deploy/example.md", "root=/" + "users/alice/work\n")
        self.assertIn("PERSONAL_PATH", self.run_checker().stderr)

    def test_internal_host_fails_but_locale_key_passes(self) -> None:
        host = "api.corp." + "internal"
        self.write("docs/host.md", f"host={host}\n")
        result = self.run_checker()
        self.assertIn("INTERNAL_HOST\tdocs/host.md", result.stderr)

        locale_key = "hackathon.error." + "internal"
        short_locale_key = "desc." + "internal"
        self.write(
            "options/locale/test.ini",
            f"{locale_key} = Generic error.\n"
            f"{short_locale_key} = Internal\n",
        )
        subprocess.run(
            ["git", "-C", str(self.root), "rm", "-f", "docs/host.md"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        self.assertEqual(0, self.run_checker().returncode)

        disguised = "prod.error." + "internal"
        self.write("docs/host.md", f"url=https://{disguised}\n")
        self.assertIn("INTERNAL_HOST\tdocs/host.md", self.run_checker().stderr)

        self.write("docs/host.md", f"host={disguised}\n")
        self.assertIn("INTERNAL_HOST\tdocs/host.md", self.run_checker().stderr)

        self.write("README.md", f"{disguised}\n")
        self.assertIn("INTERNAL_HOST\tREADME.md", self.run_checker().stderr)

        self.write("README.md", f"Connect to `{disguised}`\n")
        self.assertIn("INTERNAL_HOST\tREADME.md", self.run_checker().stderr)

        self.write(
            "routers/example.go",
            f'ctx.Locale.TrString("{locale_key}")\n',
        )
        subprocess.run(
            [
                "git",
                "-C",
                str(self.root),
                "rm",
                "-f",
                "docs/host.md",
                "README.md",
            ],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        self.assertEqual(0, self.run_checker().returncode)

    def test_internal_like_values_outside_locales_still_fail(self) -> None:
        suffix = "internal"
        self.write(
            "docs/locale-keys.md",
            f"desc.{suffix} = Internal\n"
            f"bounty.error.{suffix} = Generic error.\n",
        )
        self.write("modules/example.go", f"return q.{suffix}.Load()\n")
        result = self.run_checker()
        self.assertIn("INTERNAL_HOST\tdocs/locale-keys.md", result.stderr)
        self.assertIn("INTERNAL_HOST\tmodules/example.go", result.stderr)

        self.write("docs/host.md", f"cert=/etc/db.{suffix}.pem\n")
        self.assertIn("INTERNAL_HOST\tdocs/host.md", self.run_checker().stderr)

    def test_single_label_internal_host_and_sensitive_path_parts_fail(self) -> None:
        host = "db." + "internal"
        self.write("README.md", f"host={host}\n")
        self.assertIn("INTERNAL_HOST\tREADME.md", self.run_checker().stderr)

        subprocess.run(
            ["git", "-C", str(self.root), "rm", "-f", "README.md"],
            check=True,
            stdout=subprocess.DEVNULL,
        )
        self.write("prod.error." + "internal", "neutral\n")
        result = self.run_checker()
        self.assertIn("PRIVATE_FILE\tpath-redacted", result.stderr)
        self.assertIn("PATH_INTERNAL_HOST\tpath-redacted", result.stderr)

        self.write("docs/operator.local/runbook.md", "neutral\n")
        self.assertIn("LOCAL_FILE", self.run_checker().stderr)

        self.write("private-content/runbook.md", "neutral\n")
        self.assertIn("PRIVATE_FILE", self.run_checker().stderr)

    def test_literal_credential_fails(self) -> None:
        literal = "pass" + "word: hunter2"
        self.write("README.md", literal)
        result = self.run_checker()
        self.assertIn("LITERAL_CREDENTIAL", result.stderr)
        self.assertNotIn("hunter2", result.stderr)

        self.write("custom/conf/app.ini", literal)
        self.assertIn("LITERAL_CREDENTIAL\tcustom/conf/app.ini", self.run_checker().stderr)

    def test_runtime_screenshot_fails(self) -> None:
        self.write("docs/tests/e2e/reports/live.png")
        self.assertIn("RUNTIME_EVIDENCE", self.run_checker().stderr)

        target = self.root / "custom/public/assets/img/customer.png"
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(b"\x89PNG\r\n\x1a\n\x00private")
        subprocess.run(["git", "-C", str(self.root), "add", "-f", str(target.relative_to(self.root))], check=True)
        self.assertIn("UNREVIEWED_BINARY", self.run_checker().stderr)

    def test_symlink_target_is_scanned_without_dereferencing(self) -> None:
        target = "/" + "Users/alice/private"
        link = self.root / "docs/link"
        link.parent.mkdir(parents=True, exist_ok=True)
        os.symlink(target, link)
        subprocess.run(["git", "-C", str(self.root), "add", "docs/link"], check=True)
        result = self.run_checker()
        self.assertIn("PERSONAL_PATH\tdocs/link", result.stderr)


if __name__ == "__main__":
    unittest.main()
