#!/usr/bin/env python3
"""Fail when the public HackForger tree contains private or business content."""

from __future__ import annotations

import argparse
import hashlib
import ipaddress
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

from boundary_guard_policy import is_guarded_path


POLICY_REL = "scripts/ci/private-content-markers.txt"

FORBIDDEN_PATH_PREFIXES = (
    ".claude/projects/",
    ".claude/worktrees/",
    ".worktrees/",
    "custom/public/assets/landing/",
    "docs/landing-page/",
    "docs/ops/",
)
FORBIDDEN_PATH_EXACT = {
    ".claude/scheduled_tasks.lock",
    "CLAUDE.local.md",
    "scripts/cleanup-invalid-hackathons.sql",
}
FORBIDDEN_BINARY_SUFFIXES = {
    ".7z",
    ".avi",
    ".bin",
    ".bmp",
    ".bz2",
    ".class",
    ".db",
    ".dmg",
    ".doc",
    ".docx",
    ".eot",
    ".gif",
    ".gz",
    ".ico",
    ".jar",
    ".jpeg",
    ".jpg",
    ".mov",
    ".mp4",
    ".o",
    ".otf",
    ".pdf",
    ".png",
    ".rar",
    ".sqlite",
    ".svg",
    ".tar",
    ".tgz",
    ".ttf",
    ".wav",
    ".webm",
    ".webp",
    ".woff",
    ".woff2",
    ".xls",
    ".xlsx",
    ".zip",
}
PROJECT_OWNED_PREFIXES = (
    ".agents/",
    ".claude/",
    ".github/",
    "custom/",
    "deploy/",
    "docs/",
    "options/hackforger-help/",
    "scripts/",
    "skills/",
    "templates/hackforger/",
)
REVIEWED_PUBLIC_BINARY_HASHES = {
    # Neutral HackForger brand assets. Any byte change requires boundary review.
    "custom/public/assets/fonts/manrope-medium.woff2": "19874318747181a650eda439c37955220b849d9c4797c9e0718ee67d4bf929bc",
    "custom/public/assets/fonts/manrope-regular.woff2": "849290ef12a2eeb9af5c11924120d11aa4ae8b435ed3347d7fc8bc240c293ca3",
    "custom/public/assets/fonts/manrope-semibold.woff2": "f7ac6258da20ab7541939b59851155753d1d24f1b30cbcb949077a3faa3d1593",
    "custom/public/assets/fonts/poppins-bold.woff2": "9338e65fc077355c7a87ae0d64cc101e23b9bf8ad78ae65f0f319c857311b526",
    "custom/public/assets/fonts/poppins-medium.woff2": "cd36de204aca2d5fa263a731f7c20009b5e3d754ba1f1e03c33e93a48f3e7446",
    "custom/public/assets/fonts/poppins-regular.woff2": "7d93459d86585bfcdbb7e0376056226adb25821ee54b96236fe2123e9560929f",
    "custom/public/assets/fonts/poppins-semibold.woff2": "f4e80d9dfd374d02989b87a27b5ed4cb78fbb177c27f1478e9a8b0afb7513149",
    "custom/public/assets/img/favicon.png": "3a147442d1f4cd1fcb8c6038f994b6efd87be6530e1e92277de28139efc28562",
    "custom/public/assets/img/favicon.svg": "9e90f2d49e9d0e03a2ec39f75919ab94774e3d550d8c888223683960261996e6",
    "custom/public/assets/img/logo-dark.svg": "9c338a73f1b374aee25ecf4bfcf1b80f5e179ae3e9319bd3a1818885803d91e9",
    "custom/public/assets/img/logo-light.svg": "02c8c0e652a76262dc2492b600dd1c69451a5e006aac0d1680d3317ba5499405",
    "custom/public/assets/img/logo.png": "71a2a2d7c4cb6ae8742b452680e2ddb04d63476c3afecd086bffe36234a52cc0",
    "custom/public/assets/img/logo.svg": "02c8c0e652a76262dc2492b600dd1c69451a5e006aac0d1680d3317ba5499405",
}
IPV4_RE = re.compile(
    r"(?<![0-9])(?<![0-9]\.)(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?![0-9])(?!\.[0-9])"
)
IPV6_BRACKET_RE = re.compile(
    r"\[([0-9A-Fa-f:.]+(?:%[A-Za-z0-9_.-]+)?)\]"
)
IPV6_BARE_RE = re.compile(
    r"(?<![A-Za-z0-9_.:-])([0-9A-Fa-f:.]+(?:%[A-Za-z0-9_.-]+)?)(?![A-Za-z0-9_.:-])"
)
PERSONAL_HOME_RE = re.compile(
    r"/Users/[A-Za-z0-9._-]+(?![A-Za-z0-9._-])", re.IGNORECASE
)
INTERNAL_HOST_RE = re.compile(
    r"(?ix)(?:"
    r"\b[a-z0-9-]+\.inside\.[a-z0-9.-]+\b|"
    r"\b[a-z0-9-]+(?:\.[a-z0-9-]+)*\.(?:internal|lan)\b|"
    r"(?:https?://|ssh://|git@)[^\s/'\"`]+\.(?:internal|lan)\b|"
    r"\b(?:host(?:name)?|server|url)\s*[:=]\s*[a-z0-9.-]+\.(?:internal|lan)\b"
    r")"
)
PRIVATE_KEY_RE = re.compile(r"-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY-----")
TOKEN_RE = re.compile(r"\b(?:ghp_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|AKIA[0-9A-Z]{16})\b")
SLACK_WEBHOOK_RE = re.compile(
    r"https://hooks\.slack\.com/services/[A-Za-z0-9_-]+/[A-Za-z0-9_-]+/[A-Za-z0-9_-]+"
)
CREDENTIAL_RE = re.compile(
    r"(?im)\b(?:password|passwd|token|secret|api[_-]?key|sudo[_-]?pass|admin[_-]?pass)\b"
    r"[ \t]*[:=][ \t]*[\"']?([^\s\"'`;#]+)"
)
SAFE_CREDENTIAL_PREFIXES = ("$", "<", "__")
SAFE_CREDENTIAL_VALUES = {"...", "example", "redacted", "null", "none", "~", "true", "false"}
INTERNAL_HOST_KEY_SUFFIXES = (".error." + "internal", ".desc." + "internal")
INTERNAL_HOST_EXEMPT_HASHES = {
    # Whole-file hashes pin reviewed upstream validation and migration fixtures.
    "cmd/hook.go": "5e8eb64b7c9152a8034bbcf3b3b275e41e824aa869b0d1d951d59a00d2538248",
    "docs/plans/2026-04-13-fix-error-notification-patterns-design.md": "a8e5a84cfaaf5a994e3cf8b1b870169bb9f96f17f79f7f879f8e987b3e609602",
    "modules/lfs/content_store.go": "7d10477091c994718a74a45956e1fd5ca0d3f4f6a0fae6a6a9e2b69271eb5beb",
    "modules/queue/base_levelqueue.go": "bcf3c9978cbfebb239b21bf833f0fd1f5ce5b0d31e251c3a62e76cc2649d535b",
    "modules/queue/base_levelqueue_unique.go": "33c39c5f63dcb3c09d25b7e2dc1c67e5f46b643e04d354fdb5d0f008bc021b9d",
    "modules/validation/validurl_test.go": "7fab3acd97e7aaa106dce1d50db14a118aefb0ea820d3b30a776284e6d6b86e8",
    "modules/validation/validurllist_test.go": "21a8ca92f218a1c0fd77a7a44beaae23f24b61a78322251c77a4704182369308",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fafter=Y3Vyc29yOnYyOpLPAAABgogj3SDOT44Tug%253D%253D&direction=asc&per_page=45&sort=created&state=all": "b3c26edc670c72fbe22f0cd3147f2451c760a685b2bbb9b15686ef973208007b",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fdirection=asc&per_page=45&sort=created&state=all": "9c238830ad460cf30ed021675a75b9421999664fd641d5fdd6eb95eb4fd6d258",
    "web_src/js/components/DashboardRepoList.vue": "f7a6035f6022721297457c350d78b6273375d330926bf3739ac529242d82be97",
}
CREDENTIAL_SCAN_SUFFIXES = {
    ".bash",
    ".cfg",
    ".conf",
    ".env",
    ".ini",
    ".json",
    ".md",
    ".plist",
    ".sh",
    ".toml",
    ".txt",
    ".yaml",
    ".yml",
    ".zsh",
}
CREDENTIAL_SCAN_NAMES = {".gitattributes", ".gitignore", ".gitmodules", "dockerfile"}
CREDENTIAL_EXEMPT_HASHES = {
    # Whole-file hashes pin reviewed upstream examples and test fixtures.
    ".forgejo/workflows/build-release-integration.yml": "480187bc0740cebf44ed5e5218a169d9a645c5780ef337732c0caa0cf3e2dac6",
    "docker/root/etc/nsswitch.conf": "3a4a17b59122890fda91ddd11b6d73df51bdfccf0ed9ccdca6c49358fd040433",
    "models/fixtures/ModerationFeatures/user.yml": "5a7678f1c8081255923c137c25f16013a4f3065e7090cdd2bfc0f6a369268e89",
    "models/fixtures/TestActivateUserEmail/user.yml": "aacfc20e87d7525b2e94ab1d0ed139de4ef632d1ad4765a9b5c60805f19c7129",
    "models/fixtures/TestTwoFactorWithPasswordChange/user.yml": "eb0a7685c9533fdd146baba3aaa6fa22e33b12dab9fe122d3dbb64c9252a373b",
    "models/fixtures/access_token.yml": "c122506c2462a47db53428b4af4017256e6a5a33860b08a313508b0e33aedb29",
    "models/fixtures/action_runner_token.yml": "530221f53071545eac3e2d14c76f0de42775dc10896f8a5ed662112901d61b53",
    "models/fixtures/two_factor.yml": "1f30ed002d3ee6e7bcec3a874a870ec19775f0b57a8d346a2fac7aa32c1914a4",
    "models/fixtures/user.yml": "e1e82433b058f11fc5af9d0a4bb1d20b60f82019833ffa983ed21e60a3824ef1",
    "models/gitea_migrations/fixtures/Test_MigrateTwoFactorToKeying/two_factor.yml": "aa1209f02c3b379ab7f716356600ca58e8b176bda847f0f224b1bc72f9bb1edb",
    "models/user/fixtures/user.yml": "7a521c5d62203c964b0159438bce24cc582625387f5f8ed9ccad7ffed455492a",
    "tests/integration/fixtures/TestAdminDeleteUser/user.yml": "d8c223614945d30fe8ef96299c060a99d9f4fad2d13cf7419c9c2a833c36fa46",
    "tests/integration/fixtures/TestAdminModerationViewReports/user.yml": "eb3bae131b11eaa0514e959f039e8d802df20e4fdfa398cb7009e19f943d0775",
    "tests/integration/fixtures/TestUserPasswordResetOAuth2/user.yml": "e88ef20e922eb18fec9285e542770d87cb6c5d8a7d1199ba0257bb0ee6f37cea",
}
IP_EXEMPT_HASHES = {
    # Whole-file hashes pin reviewed upstream examples/tests containing IPs.
    "custom/conf/app.example.ini": "8c59a85175271beddef3060301efe950cddfa3b32ca946da21444864d72b1b05",
    "models/auth/oauth2.go": "a7675a0bb79e003ffd3b4861b1714f02c7f02a4ff6f5ababda53157a6500d4f0",
    "models/auth/oauth2_test.go": "c21ae2f3b723a3f13f9090db6bb9fd760367ca7c7bb38fad0e85325b38cd5401",
    "models/fixtures/action_run.yml": "fcc3022a944c92b5f2c2a60ce23e944b7a5644d1fbaf63e384b41d7a7551ca14",
    "modules/forgefed/actor_person_test.go": "ceb33f4f44e95a0812108fa46ad984016f773e4936053ac75491724f14bf255a",
    "modules/hostmatcher/hostmatcher.go": "5e2467a091be8577c56566d9082bf73273814abefc5e374d9ccfb57aac02afe5",
    "modules/hostmatcher/hostmatcher_test.go": "ac757c77fc65f4684687605670a0312fc6a7ca500f2977c43d64749699e361e4",
    "modules/hostmatcher/http.go": "33ee740cb714b3e01974c5e953ab15097e860995ad3cbf19c2dba718d4db40ba",
    "modules/markup/html_test.go": "475ed03709d88b645c7387d7300f598c8fc7d5c6c376ae19eee042ddcabf7ad3",
    "modules/packages/chef/metadata_test.go": "cd86e3898734633357946d65312dbdfbea455c44868a016ea0d20745c702a93f",
    "modules/packages/nuget/metadata_test.go": "d37296dd02d592e26896ff844a028e69b4dcb91bdd2cd9cf2b947efdac02696d",
    "modules/validation/helpers_test.go": "65a25558ba440106d9444cca1fd74d13b65f17f4f27c96e7edaccd0291ed69c3",
    "options/license/xinetd": "7b3cf46961d23cf845ac7c52075693b6ca31389e1534c655333b4b6fe0459bd4",
    "options/license/Community-Spec-1.0": "3291598ecec867f166a20e7538e173b05fbe0af9d739a6cabfb27121d93fb96e",
    "routers/web/auth/oauth.go": "1b953ee4f1d350ffc5bf8a1e656df8470317088f47573dabe4cafcb73164b14f",
    "services/migrations/migrate_test.go": "db0eeb80a68dd4d3e5d8d6d8a2c3620429eff11b8978257902f27beef9717902",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fafter=Y3Vyc29yOnYyOpLPAAABaaHKpHDOGUReFQ%253D%253D&direction=asc&per_page=45&sort=created&state=all": "b1bc91158aac4ac5071eb155b6f098661acf1e3cca5c8f51d0b739659a32e31f",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fafter=Y3Vyc29yOnYyOpLPAAABgogj3SDOT44Tug%253D%253D&direction=asc&per_page=45&sort=created&state=all": "b3c26edc670c72fbe22f0cd3147f2451c760a685b2bbb9b15686ef973208007b",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fdirection=asc&per_page=45&sort=created&state=all": "9c238830ad460cf30ed021675a75b9421999664fd641d5fdd6eb95eb4fd6d258",
    "tests/integration/api_packages_nuget_test.go": "2e6f8c498ce110c9693539745cca4678aa69bcddc19d6af7ae76d1e0ed571792",
    "tests/integration/api_repo_test.go": "dae6d13a03b8c7433fbcded5b3471d7e3cd4da6858c21def6a1caf95082dfc28",
}
IPV6_EXEMPT_HASHES = {
    # Whole-file hashes pin upstream network parsing/matching fixtures and one
    # public migration snapshot. Editing any fixture re-enables IPv6 review.
    "modules/forgefed/actor_person_test.go": "ceb33f4f44e95a0812108fa46ad984016f773e4936053ac75491724f14bf255a",
    "modules/git/url/url_test.go": "1033b34ff0861a31a82ae1676cf9ed02c59ff758e358b309593b67fc16643143",
    "modules/hostmatcher/hostmatcher.go": "5e2467a091be8577c56566d9082bf73273814abefc5e374d9ccfb57aac02afe5",
    "modules/hostmatcher/hostmatcher_test.go": "ac757c77fc65f4684687605670a0312fc6a7ca500f2977c43d64749699e361e4",
    "modules/setting/database_test.go": "24188094fec9b2a5fc8f5195e79d66f6f66e44d0b4decea44f0a9343defdd92c",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fafter=Y3Vyc29yOnYyOpLPAAABgogj3SDOT44Tug%253D%253D&direction=asc&per_page=45&sort=created&state=all": "b3c26edc670c72fbe22f0cd3147f2451c760a685b2bbb9b15686ef973208007b",
}
PERSONAL_PATH_EXEMPT_HASHES = {
    # Whole-file hashes pin reviewed upstream API routes, templates, fixtures,
    # and examples whose lowercase `/users/...` text is not a local home path.
    "contrib/launchd/io.gitea.web.plist": "724667b726d366ae48c8c626f1870bf143249df66bb1fa7f9f8ab4d0e43f2fbc",
    "docs/product/user-journeys/hacker-social.md": "16c7bb7518269a8bbb68b528bac2f99c18680b5e4de0af9e9f0efbbc09c29db2",
    "docs/product/user-journeys/platform-bot.md": "2381402fae472190b6dd9e2af56654724cedfb08a10ef797bdf58db079430921",
    "docs/product/user-journeys/zh-CN/hacker-social.md": "a3128ceea3a9f6acd5ed1cd320b07f3bda1a1e43f9f19ecd7e42abf4bde77392",
    "docs/product/user-journeys/zh-CN/platform-bot.md": "2a870a84ecb7cc80bc2885be5684099dab98820f3e4a0afcb944fd8348a2f7f4",
    "docs/product/user-stories/agent.md": "6e842750b080a4928f0e62bea52ff7fb34772e629edc6e0de9e228d1110ad8fb",
    "docs/product/user-stories/zh-CN/agent.md": "209f820e022541f6950a6eef76b26f474d94c0fae862404b8e26fb46ae3278d0",
    "modules/forgefed/actor_test.go": "15495f9c5653464b3255ff9eb9828396a8f53a4dd6e7d2bb9738320229be6e2f",
    "routers/api/v1/api.go": "47304c9b5b0e597e9b32716992096bdc4769978a10dcad2bd6cab4e9c971620a",
    "routers/api/v1/user/user.go": "f4075c1a4dadd4e6957cc87435fd8db15a62c9f1fdba4d8c1953b2f2ed4c3a6d",
    "routers/web/admin/users_test.go": "59a7a20296f1ef68b9061107507d0ba63c8e99d46a849c2ee939d7d8b9dda31f",
    "routers/web/home.go": "944fef027d5639be379b952d4782f1dcdc03af23d65bd2d9de4b407c46b328d9",
    "routers/web/web.go": "a24878ac59795172e9a509912d062c57fbcaddc7458a7a37fa029f2be9fa0b06",
    "services/migrations/testdata/github/full_download/GET_%2Frepos%2Fforgejo%2Ftest_repo": "6cb69cea3a7d40a41fe885dbf819d4482f5cba87561a66713fdd4721f1a7dd53",
    "services/migrations/testdata/github/full_download/GET_%2Frepos%2Fforgejo%2Ftest_repo%2Fissues%2F2%2Fcomments%3Fdirection=asc&per_page=100&sort=created": "7e19872c3b0085ca6f5e379a96124bfe3fd2c06c3c16a28859e29ebca7456076",
    "services/migrations/testdata/github/full_download/GET_%2Frepos%2Fforgejo%2Ftest_repo%2Fissues%3Fdirection=asc&per_page=2&sort=created&state=all": "b3ce59319b08701e2c72d6e7728dbff79059fd861c814cc05a0b79b28389db5c",
    "services/migrations/testdata/github/full_download/GET_%2Frepos%2Fforgejo%2Ftest_repo%2Fmilestones%3Fpage=1&per_page=100&state=all": "9805e08c9d7a656b84b0f0c1d78ae626828eb47c07b5b6cd0a9a6d2dc836ea5d",
    "services/migrations/testdata/github/full_download/GET_%2Frepos%2Fforgejo%2Ftest_repo%2Fpulls%2F3%2Freviews%2F3096999684%2Fcomments%3Fper_page=100": "abff155de932540d3cb721aa7376f76dd8f7899eb9b48458f34c2e21ca5712bf",
    "services/migrations/testdata/github/full_download/GET_%2Frepos%2Fforgejo%2Ftest_repo%2Fpulls%2F3%2Freviews%2F3097007243%2Fcomments%3Fper_page=100": "6ffd4d017fbf39e11d8a2616cce20a9c7d52c2d2df0621f1c3cc79dc2125f043",
    "services/migrations/testdata/github/full_download/GET_%2Frepos%2Fforgejo%2Ftest_repo%2Fpulls%2F3%2Freviews%3Fper_page=100": "c2b2301bf30206396f34871644d26580007e9c7f405440ce4449e2a9d80696e4",
    "services/migrations/testdata/github/full_download/GET_%2Frepos%2Fforgejo%2Ftest_repo%2Fpulls%3Fdirection=asc&page=1&per_page=2&sort=created&state=all": "205c716af938afca1de1e32ce331e7830f0641caceef88798f5379bb352e2a79",
    "services/migrations/testdata/github/full_download/GET_%2Frepos%2Fforgejo%2Ftest_repo%2Freleases%3Fpage=1&per_page=100": "f2fc17f3d7645a1f9469dcf3be3ec22cb26e252ce096b293613757f802c44917",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic": "195b902edbc1dd13a3a43abedcb9e4f58e753707fb3a7ff1b1a70dcd8b95d5f8",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fafter=Y3Vyc29yOnYyOpLPAAABaaHKpHDOGUReFQ%253D%253D&direction=asc&per_page=45&sort=created&state=all": "b1bc91158aac4ac5071eb155b6f098661acf1e3cca5c8f51d0b739659a32e31f",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fafter=Y3Vyc29yOnYyOpLPAAABgogj3SDOT44Tug%253D%253D&direction=asc&per_page=45&sort=created&state=all": "b3c26edc670c72fbe22f0cd3147f2451c760a685b2bbb9b15686ef973208007b",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fafter=Y3Vyc29yOnYyOpLPAAABmqckTzDO2bC8uw%253D%253D&direction=asc&per_page=45&sort=created&state=all": "0fcaf56a4e9329c2d570b5aa17cc9418a0e43b1df604857774176780eee92f90",
    "services/migrations/testdata/github/pagination/GET_%2Frepos%2Fnigoroll%2Flibvmod-dynamic%2Fissues%3Fdirection=asc&per_page=45&sort=created&state=all": "9c238830ad460cf30ed021675a75b9421999664fd641d5fdd6eb95eb4fd6d258",
    "skills/hackforger-api/SKILL.md": "878400f8a26ea12e61bce7ec299d106a5e31609e8792fa23e956b28981c3912a",
    "templates/admin/user/list.tmpl": "4bb9ad9fa86b8ea034004ef268c8b3e81243c77aba9babe2a287aae98d8f3bf3",
    "templates/swagger/v1_json.tmpl": "3992e58cbf39f4637ea0db4b4b930d5efc68c48598afa7da7a50f7108c9464a7",
    "tests/integration/admin_user_test.go": "1ed34a87a8f582bc674ca40e4e39e07fe9de4887b94d6d331fdf234d67724a31",
    "tests/integration/api_admin_org_test.go": "51eeb89f85623971e72bfd9397515c2f39685dc669c0f2534879db23b20dcfff",
    "tests/integration/api_admin_test.go": "da793b5245bfe5b1d9911b19787d12261e3ba6888e85e04d963e5fe208dc247e",
    "tests/integration/api_gpg_keys_test.go": "7e72274659496e87572622786f1e4337527fbabedf39e029e816fe4c09668ddc",
    "tests/integration/api_packages_conan_test.go": "f87c4d52d5488c66b67e843ce57345ab1eb983d3cc2b0e8f13ba1248936b0be6",
    "tests/integration/api_quota_management_test.go": "e418a9e510def7f85007487a0d09c0769871c15e230d556e5920216f60e08176",
    "tests/integration/api_token_test.go": "edf2d177785022e00924c551c5132104b1a96a3c2d5fb5169479537eaed34e4c",
    "tests/integration/api_user_org_perm_test.go": "ba6b709da8b5ec85a7ad62c10654c773655e01595a59918fa7dc02d8da5de0de",
    "tests/integration/api_user_search_test.go": "8911f681144cc566a77eb919e644271559af3adca389f632a3b0eeee5e14e8d4",
    "tests/integration/hackforger_reputation_test.go": "7877f0db63a3cbbc052389beb4389cbd288bad90dd2fba9f7de46df473270080",
    "tests/integration/oauth_test.go": "d21faa6ad7407232bc77c670ac4a46e03ff1615d64f5c9692c0c27ca8e2c57a7",
    "tests/integration/setting_test.go": "8ecd19332837d85ef045eb6830077af5efc6b99a8ebb6f93940bfe8ab6f0a80a",
    "tests/integration/user_profile_attributes_test.go": "22f1a98ae6688a7458a1c43d0bb96fd9e391f07ca33cbff5a8077b74d6910ead",
    "tests/integration/user_redirect_test.go": "b0b5b38d8d4d1434e14bf9d1f11bfbd2c89dfe7f99a92ad8df72ed3a7ed180bd",
    "tests/integration/user_test.go": "b9359a1d7e0baf6b9b753307033221b55c903ef4372f9be43ca2e9602f9fea6c",
}
SECRET_MATERIAL_EXEMPT_HASHES = {
    # Whole-file hashes pin reviewed upstream cryptography fixtures. Editing a
    # fixture re-enables the secret-material finding until it is re-reviewed.
    "models/asymkey/ssh_key_test.go": "fb57df7d1346dc22d3146bb2203c8013c5d3699b6d8af95ce6fa983eec9a739b",
    "modules/util/keypair_test.go": "c988e17edac4cfa35af179d0e80f6e52c6473874f3c049da788dd41d97967f96",
    "modules/util/util_test.go": "d3dbe4906a957659d6208b9c1ff4100110de90e9c26395bd9855a814a253e420",
    "tests/integration/api_httpsig_test.go": "ffc7d6b41fd33fd903cd3a87fc61065cb105097d93c74c9bf05166869403d105",
    "tests/integration/api_packages_chef_test.go": "d87bdb9d403e1ef982612707dad7243ee3078514ffdc6f8afd29615657e7a8af",
    "tests/integration/ssh-signing-key": "01cee98d27d9e08425deb204f3aeae3deb4dc4a4c5da5b51e971314ac3eef30c",
}


@dataclass(frozen=True, order=True)
class Finding:
    rule: str
    path: str


@dataclass(frozen=True)
class MarkerFingerprint:
    length: int
    rolling_hash: int
    sha256: str


@dataclass(frozen=True)
class IndexEntry:
    mode: str
    oid: str


def git_index_entries(root: Path) -> tuple[dict[str, IndexEntry], set[str]]:
    proc = subprocess.run(
        ["git", "-C", str(root), "ls-files", "--stage", "-z"],
        check=True,
        stdout=subprocess.PIPE,
    )
    entries: dict[str, IndexEntry] = {}
    unmerged: set[str] = set()
    for raw in proc.stdout.split(b"\0"):
        if not raw:
            continue
        metadata, path_bytes = raw.split(b"\t", 1)
        mode, oid, stage = metadata.decode("ascii").split()
        path = path_bytes.decode("utf-8", "surrogateescape")
        if stage != "0":
            unmerged.add(path)
            continue
        entries[path] = IndexEntry(mode, oid)
    return entries, unmerged


def git_untracked_files(root: Path) -> set[str]:
    proc = subprocess.run(
        ["git", "-C", str(root), "ls-files", "-z", "--others", "--exclude-standard"],
        check=True,
        stdout=subprocess.PIPE,
    )
    return {
        item.decode("utf-8", "surrogateescape")
        for item in proc.stdout.split(b"\0")
        if item
    }


def git_object_format(root: Path) -> str:
    object_format = subprocess.run(
        ["git", "-C", str(root), "rev-parse", "--show-object-format"],
        check=True,
        text=True,
        stdout=subprocess.PIPE,
    ).stdout.strip()
    if object_format not in {"sha1", "sha256"}:
        raise ValueError(f"unsupported Git object format: {object_format}")
    return object_format


def read_git_blob(root: Path, oid: str) -> bytes:
    if not re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", oid):
        raise ValueError("invalid Git blob object id")
    return subprocess.run(
        ["git", "-C", str(root), "cat-file", "blob", oid],
        check=True,
        stdout=subprocess.PIPE,
    ).stdout


def load_markers(path: Path) -> tuple[MarkerFingerprint, ...]:
    markers: list[MarkerFingerprint] = []
    for raw in path.read_text(encoding="utf-8").splitlines():
        value = raw.strip()
        if not value or value.startswith("#"):
            continue
        parts = value.split()
        if len(parts) != 3:
            raise ValueError(f"malformed marker fingerprint in {path}")
        length_text, rolling_text, digest = parts
        if not length_text.isdigit() or int(length_text) < 1:
            raise ValueError(f"invalid marker length in {path}")
        if not re.fullmatch(r"[0-9a-f]{16}", rolling_text):
            raise ValueError(f"invalid rolling hash in {path}")
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError(f"invalid SHA-256 in {path}")
        markers.append(MarkerFingerprint(int(length_text), int(rolling_text, 16), digest))
    if not markers:
        raise ValueError(f"marker policy is empty: {path}")
    return tuple(markers)


def contains_marker(value: str, markers: tuple[MarkerFingerprint, ...]) -> bool:
    data = value.lower().encode("utf-8", "surrogateescape")
    mask = (1 << 64) - 1
    base = 257
    by_length: dict[int, dict[int, set[str]]] = {}
    for marker in markers:
        by_length.setdefault(marker.length, {}).setdefault(marker.rolling_hash, set()).add(marker.sha256)

    for length, candidates in by_length.items():
        if len(data) < length:
            continue
        factor = pow(base, length - 1, 1 << 64)
        rolling = 0
        for byte in data[:length]:
            rolling = ((rolling * base) + byte) & mask
        for start in range(0, len(data) - length + 1):
            if start:
                rolling = (rolling - (data[start - 1] * factor)) & mask
                rolling = ((rolling * base) + data[start + length - 1]) & mask
            digests = candidates.get(rolling)
            if digests and hashlib.sha256(data[start : start + length]).hexdigest() in digests:
                return True
    return False


def allowed_ip(value: str) -> bool:
    try:
        address = ipaddress.ip_address(value)
    except ValueError:
        return False
    allowed_networks = (
        ipaddress.ip_network("0.0.0.0/32"),
        ipaddress.ip_network("127.0.0.0/8"),
        ipaddress.ip_network("192.0.2.0/24"),
        ipaddress.ip_network("198.51.100.0/24"),
        ipaddress.ip_network("203.0.113.0/24"),
    )
    return any(address in network for network in allowed_networks)


def ipv6_candidates(text: str) -> set[str]:
    candidates = {match.group(1) for match in IPV6_BRACKET_RE.finditer(text)}
    candidates.update(match.group(1) for match in IPV6_BARE_RE.finditer(text))
    return {candidate for candidate in candidates if candidate.count(":") >= 2}


def allowed_ipv6(value: str) -> bool:
    address_text = value.split("%", 1)[0]
    try:
        address = ipaddress.ip_address(address_text)
    except ValueError:
        return True
    if not isinstance(address, ipaddress.IPv6Address):
        return True
    allowed_networks = (
        ipaddress.ip_network("::/128"),
        ipaddress.ip_network("::1/128"),
        ipaddress.ip_network("2001:db8::/32"),
    )
    return any(address in network for network in allowed_networks)


def path_findings(path: str, markers: tuple[MarkerFingerprint, ...]) -> list[Finding]:
    findings: list[Finding] = []
    lower = path.lower()
    parts = Path(lower).parts
    if lower in {value.lower() for value in FORBIDDEN_PATH_EXACT} or lower.startswith(
        tuple(value.lower() for value in FORBIDDEN_PATH_PREFIXES)
    ):
        findings.append(Finding("BOUNDARY_PATH", path))
    if ".local." in lower or any(part.endswith(".local") for part in parts):
        findings.append(Finding("LOCAL_FILE", path))
    if (
        ".internal." in lower
        or any(part.endswith(".internal") for part in parts)
        or lower.startswith("private-content/")
        or "/private-content/" in lower
    ):
        findings.append(Finding("PRIVATE_FILE", path))
    if contains_marker(path, markers):
        findings.append(Finding("BUSINESS_MARKER", path))
    name = Path(path).name.lower()
    if name.startswith(".env") and not name.endswith((".example", ".template")):
        findings.append(Finding("ENV_FILE", path))
    if lower.startswith("docs/tests/") and Path(lower).suffix in FORBIDDEN_BINARY_SUFFIXES:
        findings.append(Finding("RUNTIME_EVIDENCE", path))
    if "__pycache__" in parts or Path(lower).suffix in {".pyc", ".pyo"}:
        findings.append(Finding("GENERATED_ARTIFACT", path))
    return findings


SENSITIVE_PATH_RULES = {"BUSINESS_MARKER", "PRIVATE_FILE"}


def path_metadata_findings(
    path: str, markers: tuple[MarkerFingerprint, ...]
) -> tuple[set[Finding], str]:
    """Scan a Git path as metadata and return findings plus a safe display name."""
    direct = path_findings(path, markers)
    global_findings = global_content_findings("path-name.md", path, markers)
    redact = bool(global_findings) or any(
        finding.rule in SENSITIVE_PATH_RULES for finding in direct
    )
    display = "path-redacted" if redact else path
    findings = {Finding(finding.rule, display) for finding in direct}
    findings.update(
        Finding(f"PATH_{finding.rule}", "path-redacted")
        for finding in global_findings
    )
    return findings, display


def credential_is_safe(value: str) -> bool:
    normalized = value.strip().lower()
    return normalized.startswith(SAFE_CREDENTIAL_PREFIXES) or normalized in SAFE_CREDENTIAL_VALUES


def is_lfs_pointer(data: bytes) -> bool:
    if len(data) > 1024:
        return False
    lines = data.decode("ascii", "ignore").splitlines()
    return (
        len(lines) >= 3
        and lines[0] == "version https://git-lfs.github.com/spec/v1"
        and re.fullmatch(r"oid sha256:[0-9a-f]{64}", lines[1]) is not None
        and re.fullmatch(r"size [0-9]+", lines[2]) is not None
    )


def should_scan_credentials(path: str, text_hash: str) -> bool:
    lower = path.lower()
    if lower.startswith("options/locale/"):
        return False
    expected_fixture_hash = CREDENTIAL_EXEMPT_HASHES.get(path)
    if expected_fixture_hash == text_hash:
        return False
    name = Path(lower).name
    return (
        name in CREDENTIAL_SCAN_NAMES
        or name.startswith(".env")
        or Path(lower).suffix in CREDENTIAL_SCAN_SUFFIXES
    )


def locale_internal_key_context(path: str, text: str, start: int, end: int) -> bool:
    if path in {"path-name.md", "ref-name.md", "commit-metadata.md"}:
        return False
    value = text[start:end]
    if not re.fullmatch(
        r"(?i)(?:[a-z0-9_.-]+\.)?(?:error|desc)\.internal", value
    ):
        return False
    line_start = text.rfind("\n", 0, start) + 1
    line_end = text.find("\n", end)
    if line_end < 0:
        line_end = len(text)
    line = text[line_start:line_end]
    relative_start = start - line_start
    relative_end = end - line_start
    if path.startswith("options/locale/") and re.fullmatch(
        rf"\s*{re.escape(value)}\s*=.*", line
    ):
        return True
    before = line[:relative_start]
    after = line[relative_end:]
    quoted = bool(before) and bool(after) and before[-1] in {'"', "'"} and after[0] == before[-1]
    return quoted and re.search(
        r"(?:\bTr(?:String)?\s*\(\s*|\bLocale\.Tr\s+)[\"']$", before
    ) is not None


def contains_internal_host(path: str, text: str) -> bool:
    for match in INTERNAL_HOST_RE.finditer(text):
        if locale_internal_key_context(path, text, match.start(), match.end()):
            continue
        return True
    return False


def global_content_findings(
    path: str, text: str, markers: tuple[MarkerFingerprint, ...]
) -> list[Finding]:
    findings: list[Finding] = []
    text_hash = hashlib.sha256(text.encode("utf-8", "surrogateescape")).hexdigest()
    if contains_marker(text, markers):
        findings.append(Finding("BUSINESS_MARKER", path))
    if PRIVATE_KEY_RE.search(text) or TOKEN_RE.search(text) or SLACK_WEBHOOK_RE.search(text):
        expected_fixture_hash = SECRET_MATERIAL_EXEMPT_HASHES.get(path)
        if expected_fixture_hash != text_hash:
            findings.append(Finding("SECRET_MATERIAL", path))
    if PERSONAL_HOME_RE.search(text):
        expected_personal_fixture_hash = PERSONAL_PATH_EXEMPT_HASHES.get(path)
        if expected_personal_fixture_hash != text_hash:
            findings.append(Finding("PERSONAL_PATH", path))
    if contains_internal_host(path, text):
        expected_internal_fixture_hash = INTERNAL_HOST_EXEMPT_HASHES.get(path)
        if expected_internal_fixture_hash != text_hash:
            findings.append(Finding("INTERNAL_HOST", path))
    if should_scan_credentials(path, text_hash):
        for match in CREDENTIAL_RE.finditer(text):
            if not credential_is_safe(match.group(1)):
                findings.append(Finding("LITERAL_CREDENTIAL", path))
                break
    expected_ip_fixture_hash = IP_EXEMPT_HASHES.get(path)
    ip_exempt = Path(path).suffix.lower() == ".svg" or expected_ip_fixture_hash == text_hash
    if not ip_exempt:
        for match in IPV4_RE.finditer(text):
            if not allowed_ip(match.group(0)):
                findings.append(Finding("NON_DOCUMENTATION_IP", path))
                break
    expected_ipv6_fixture_hash = IPV6_EXEMPT_HASHES.get(path)
    if expected_ipv6_fixture_hash != text_hash:
        for candidate in ipv6_candidates(text):
            if not allowed_ipv6(candidate):
                findings.append(Finding("NON_DOCUMENTATION_IPV6", path))
                break
    return findings


def load_git_baseline(root: Path, ref: str) -> tuple[dict[str, IndexEntry], str]:
    if not re.fullmatch(
        r"(?:refs/(?:heads|remotes)/[A-Za-z0-9._/-]+|[0-9a-f]{40}|[0-9a-f]{64})",
        ref,
    ):
        raise ValueError("baseline ref must be a full heads/remotes ref or commit object id")
    commit = subprocess.run(
        ["git", "-C", str(root), "rev-parse", "--verify", f"{ref}^{{commit}}"],
        check=True,
        text=True,
        stdout=subprocess.PIPE,
    ).stdout.strip()
    object_format = subprocess.run(
        ["git", "-C", str(root), "rev-parse", "--show-object-format"],
        check=True,
        text=True,
        stdout=subprocess.PIPE,
    ).stdout.strip()
    if object_format not in {"sha1", "sha256"}:
        raise ValueError(f"unsupported Git object format: {object_format}")
    tree = subprocess.run(
        ["git", "-C", str(root), "ls-tree", "-r", "-z", commit],
        check=True,
        stdout=subprocess.PIPE,
    ).stdout
    blobs: dict[str, IndexEntry] = {}
    for raw in tree.split(b"\0"):
        if not raw:
            continue
        metadata, path_bytes = raw.split(b"\t", 1)
        mode, object_type, oid = metadata.decode("ascii").split()
        if object_type == "blob":
            blobs[path_bytes.decode("utf-8", "surrogateescape")] = IndexEntry(mode, oid)
    return blobs, object_format


def git_blob_oid(data: bytes, object_format: str) -> str:
    framed = f"blob {len(data)}\0".encode("ascii") + data
    return hashlib.new(object_format, framed).hexdigest()


def baseline_differs(
    baseline_root: Path | None,
    baseline_blobs: dict[str, IndexEntry] | None,
    baseline_object_format: str | None,
    rel: str,
    data: bytes,
) -> bool:
    if baseline_root is not None:
        baseline = baseline_root / rel
        if baseline.is_symlink() or not baseline.is_file():
            return True
        try:
            return baseline.read_bytes() != data
        except OSError:
            return True
    if baseline_blobs is None or baseline_object_format is None:
        raise ValueError("an opaque-file baseline is required")
    entry = baseline_blobs.get(rel)
    return entry is None or entry.oid != git_blob_oid(data, baseline_object_format)


def blob_findings(
    path: str,
    data: bytes,
    markers: tuple[MarkerFingerprint, ...],
    baseline_root: Path | None,
    baseline_blobs: dict[str, IndexEntry] | None,
    baseline_object_format: str | None,
) -> set[Finding]:
    findings: set[Finding] = set()
    lower = path.lower()
    content_hash = hashlib.sha256(data).hexdigest()
    if is_lfs_pointer(data):
        findings.add(Finding("LFS_POINTER", path))
    try:
        text = data.decode("utf-8")
        invalid_utf8 = False
    except UnicodeDecodeError:
        text = ""
        invalid_utf8 = True
    opaque = (
        b"\0" in data
        or invalid_utf8
        or Path(lower).suffix in FORBIDDEN_BINARY_SUFFIXES
    )
    owned_opaque = lower.startswith(PROJECT_OWNED_PREFIXES) and opaque
    changed_opaque = opaque and baseline_differs(
        baseline_root,
        baseline_blobs,
        baseline_object_format,
        path,
        data,
    )
    if owned_opaque or changed_opaque:
        expected = REVIEWED_PUBLIC_BINARY_HASHES.get(lower)
        if expected is None or content_hash != expected:
            findings.add(Finding("UNREVIEWED_BINARY", path))
    if not opaque:
        findings.update(global_content_findings(path, text, markers))
    return findings


def resolve_commit(root: Path, value: str) -> str:
    if not re.fullmatch(r"[0-9a-f]{40}|[0-9a-f]{64}", value):
        raise ValueError("history endpoints must be commit object ids")
    commit = subprocess.run(
        ["git", "-C", str(root), "rev-parse", "--verify", f"{value}^{{commit}}"],
        check=True,
        text=True,
        stdout=subprocess.PIPE,
    ).stdout.strip()
    return commit


def history_range(root: Path, base: str, head: str) -> list[str]:
    base_commit = resolve_commit(root, base)
    head_commit = resolve_commit(root, head)
    output = subprocess.run(
        ["git", "-C", str(root), "rev-list", "--reverse", f"{base_commit}..{head_commit}"],
        check=True,
        text=True,
        stdout=subprocess.PIPE,
    ).stdout
    return [line for line in output.splitlines() if line]


def changed_commit_paths(root: Path, commit: str) -> list[str]:
    parents = subprocess.run(
        ["git", "-C", str(root), "rev-list", "--parents", "-n", "1", commit],
        check=True,
        text=True,
        stdout=subprocess.PIPE,
    ).stdout.split()
    command = [
        "git",
        "-C",
        str(root),
        "diff-tree",
        "--no-commit-id",
        "--no-renames",
        "--name-only",
        "-r",
        "-z",
    ]
    if len(parents) > 1:
        command.extend((parents[1], commit))
    else:
        command.extend(("--root", commit))
    output = subprocess.run(command, check=True, stdout=subprocess.PIPE).stdout
    return [
        item.decode("utf-8", "surrogateescape")
        for item in output.split(b"\0")
        if item
    ]


def commit_tree_entry(root: Path, commit: str, path: str) -> IndexEntry | None:
    output = subprocess.run(
        ["git", "-C", str(root), "ls-tree", "-z", commit, "--", f":(literal){path}"],
        check=True,
        stdout=subprocess.PIPE,
    ).stdout
    records = [record for record in output.split(b"\0") if record]
    if not records:
        return None
    metadata, _path = records[0].split(b"\t", 1)
    _mode, object_type, oid = metadata.decode("ascii").split()
    if object_type not in {"blob", "commit"}:
        raise ValueError(f"unsupported Git object type in history: {object_type}")
    return IndexEntry(_mode, oid)


def commit_blob(root: Path, entry: IndexEntry | None) -> bytes | None:
    if entry is None or entry.mode == "160000":
        return None
    return read_git_blob(root, entry.oid)


def filesystem_git_entry(root: Path, path: str, object_format: str) -> IndexEntry | None:
    full = root / path
    try:
        if full.is_symlink():
            data = os.readlink(full).encode("utf-8", "surrogateescape")
            mode = "120000"
        elif full.is_file():
            data = full.read_bytes()
            mode = "100755" if full.stat().st_mode & 0o111 else "100644"
        else:
            return None
    except OSError:
        return None
    return IndexEntry(mode, git_blob_oid(data, object_format))


def guard_entry_matches_baseline(
    path: str,
    entry: IndexEntry | None,
    root_object_format: str,
    baseline_root: Path | None,
    baseline_blobs: dict[str, IndexEntry] | None,
) -> bool:
    if baseline_root is not None:
        expected = filesystem_git_entry(baseline_root, path, root_object_format)
    elif baseline_blobs is not None:
        expected = baseline_blobs.get(path)
    else:
        raise ValueError("an opaque-file baseline is required")
    return entry == expected


def raw_commit_metadata(root: Path, commit: str) -> str:
    raw = subprocess.run(
        ["git", "-C", str(root), "cat-file", "commit", commit],
        check=True,
        stdout=subprocess.PIPE,
    ).stdout
    return raw.decode("utf-8", "surrogateescape")


def scan_history(
    root: Path,
    commits: list[str],
    markers: tuple[MarkerFingerprint, ...],
    baseline_root: Path | None,
    baseline_blobs: dict[str, IndexEntry] | None,
    baseline_object_format: str | None,
) -> set[Finding]:
    findings: set[Finding] = set()
    root_object_format = git_object_format(root)
    for commit in commits:
        short = commit[:12]
        metadata = raw_commit_metadata(root, commit)
        for finding in global_content_findings("commit-metadata.md", metadata, markers):
            findings.add(Finding(f"HISTORY_{finding.rule}", f"{short}:COMMIT_METADATA"))
        for path in changed_commit_paths(root, commit):
            entry = commit_tree_entry(root, commit, path)
            if is_guarded_path(path) and not guard_entry_matches_baseline(
                path,
                entry,
                root_object_format,
                baseline_root,
                baseline_blobs,
            ):
                findings.add(
                    Finding("HISTORY_GUARD_CHANGE", f"{short}:guard-path-redacted")
                )
            # Deleting a path reduces public exposure. If it was added earlier
            # in this outgoing range, that earlier blob/path is scanned at its
            # own commit; a deletion must not block cleanup of legacy content.
            if entry is None:
                continue
            metadata_findings, display = path_metadata_findings(path, markers)
            for finding in metadata_findings:
                findings.add(Finding(f"HISTORY_{finding.rule}", f"{short}:{display}"))
            data = commit_blob(root, entry)
            if data is None:
                continue
            for finding in blob_findings(
                path,
                data,
                markers,
                baseline_root,
                baseline_blobs,
                baseline_object_format,
            ):
                findings.add(Finding(f"HISTORY_{finding.rule}", f"{short}:{display}"))
    return findings


def scan(
    root: Path,
    policy: Path,
    baseline_root: Path | None = None,
    baseline_blobs: dict[str, IndexEntry] | None = None,
    baseline_object_format: str | None = None,
    history_commits: list[str] | None = None,
) -> list[Finding]:
    markers = load_markers(policy)
    findings: set[Finding] = set()
    object_format = git_object_format(root)
    index_entries, unmerged = git_index_entries(root)
    untracked = git_untracked_files(root)
    for rel in unmerged:
        metadata_findings, display = path_metadata_findings(rel, markers)
        findings.update(metadata_findings)
        findings.add(Finding("UNMERGED_INDEX", display))
    for rel in sorted(index_entries.keys() | untracked):
        metadata_findings, display = path_metadata_findings(rel, markers)
        findings.update(metadata_findings)
        full = root / rel
        worktree_data: bytes | None = None
        if full.is_symlink():
            worktree_data = os.readlink(full).encode("utf-8", "surrogateescape")
        elif full.is_file():
            try:
                worktree_data = full.read_bytes()
            except OSError:
                findings.add(Finding("UNREADABLE_FILE", display))
        elif full.exists():
            entry = index_entries.get(rel)
            if entry is None or entry.mode != "160000":
                findings.add(Finding("NON_REGULAR_WORKTREE", display))

        versions: list[bytes] = []
        entry = index_entries.get(rel)
        if entry is not None:
            if entry.mode in {"100644", "100755", "120000"}:
                if worktree_data is None or git_blob_oid(worktree_data, object_format) != entry.oid:
                    versions.append(read_git_blob(root, entry.oid))
            elif entry.mode != "160000":
                findings.add(Finding("NON_BLOB_INDEX", display))
        if worktree_data is not None:
            versions.append(worktree_data)

        seen_versions: set[str] = set()
        for data in versions:
            digest = hashlib.sha256(data).hexdigest()
            if digest in seen_versions:
                continue
            seen_versions.add(digest)
            findings.update(
                Finding(finding.rule, display)
                for finding in blob_findings(
                    rel,
                    data,
                    markers,
                    baseline_root,
                    baseline_blobs,
                    baseline_object_format,
                )
            )
    if history_commits:
        findings.update(
            scan_history(
                root,
                history_commits,
                markers,
                baseline_root,
                baseline_blobs,
                baseline_object_format,
            )
        )
    return sorted(findings)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--policy", type=Path)
    baseline = parser.add_mutually_exclusive_group(required=True)
    baseline.add_argument("--baseline-root", type=Path)
    baseline.add_argument("--baseline-ref")
    parser.add_argument("--history-base-ref")
    parser.add_argument("--history-head-ref")
    parser.add_argument("--history-commit-list", type=Path)
    parser.add_argument("--history-only", action="store_true")
    parser.add_argument("--ref-name", action="append", default=[])
    args = parser.parse_args()
    root = args.root.resolve()
    policy = (args.policy or root / POLICY_REL).resolve()
    baseline_root = args.baseline_root.resolve() if args.baseline_root else None
    baseline_blobs = None
    baseline_object_format = None
    if args.baseline_ref:
        try:
            baseline_blobs, baseline_object_format = load_git_baseline(root, args.baseline_ref)
        except (ValueError, subprocess.CalledProcessError) as error:
            print(
                f"public repository boundary: invalid baseline ({type(error).__name__})",
                file=sys.stderr,
            )
            return 2
    history_commits: list[str] = []
    try:
        if args.history_commit_list:
            if args.history_base_ref or args.history_head_ref:
                raise ValueError("history commit list cannot be combined with a history range")
            raw_commits = args.history_commit_list.read_text(encoding="ascii").splitlines()
            history_commits = list(dict.fromkeys(resolve_commit(root, value) for value in raw_commits if value))
        elif args.history_base_ref or args.history_head_ref:
            if not args.history_base_ref or not args.history_head_ref:
                raise ValueError("history base and head refs must be provided together")
            history_commits = history_range(root, args.history_base_ref, args.history_head_ref)
        if args.history_only:
            if not (args.history_commit_list or args.history_base_ref):
                raise ValueError("history-only scanning requires a commit list or range")
            findings = sorted(
                scan_history(
                    root,
                    history_commits,
                    load_markers(policy),
                    baseline_root,
                    baseline_blobs,
                    baseline_object_format,
                )
            )
        else:
            findings = scan(
                root,
                policy,
                baseline_root,
                baseline_blobs,
                baseline_object_format,
                history_commits,
            )
        metadata_findings = set(findings)
        markers = load_markers(policy)
        for ref_name in args.ref_name:
            if not ref_name or len(ref_name) > 1024 or re.search(r"[\x00-\x20\x7f]", ref_name):
                metadata_findings.add(Finding("UNSAFE_REF_NAME", "ref-name-redacted"))
                continue
            for finding in path_findings(ref_name, markers):
                metadata_findings.add(Finding(f"REF_{finding.rule}", "ref-name-redacted"))
            for finding in global_content_findings("ref-name.md", ref_name, markers):
                metadata_findings.add(Finding(f"REF_{finding.rule}", "ref-name-redacted"))
        findings = sorted(metadata_findings)
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(
            f"public repository boundary: scan error ({type(error).__name__})",
            file=sys.stderr,
        )
        return 2
    if findings:
        print("public repository boundary: FAIL", file=sys.stderr)
        for finding in findings:
            print(f"{finding.rule}\t{finding.path}", file=sys.stderr)
        print(f"total findings: {len(findings)}", file=sys.stderr)
        return 1
    print("public repository boundary: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
