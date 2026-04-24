# Install hackforger-api skill

This skill follows the [Agent Skills standard](https://agentskills.io) — the same format read by Claude Code, OpenAI Codex CLI, Cursor, Gemini CLI, GitHub Copilot, and 30+ other clients. Each client decides its own install path (e.g. `~/.claude/skills/`, `~/.codex/skills/`, `~/.cursor/skills/`), so the cleanest install is to **ask your agent to install itself**.

## One-line install (recommended)

Open your agent (Claude Code / Codex / Cursor / etc.) and paste:

```
请帮我安装这个 skill：https://github.com/HackForger/hackforger/tree/v0.1-dev/hackforger/skills/hackforger-api

把 SKILL.md 和 references/ 目录放进你的 skill 安装目录（按你这个 client 的约定决定路径），
完成后告诉我安装到了哪个路径。
```

Or in English:

```
Please install this skill for yourself:
https://github.com/HackForger/hackforger/tree/v0.1-dev/hackforger/skills/hackforger-api

Fetch SKILL.md and the references/ directory and place them in your client's
skill install path (you know the convention for your platform). Tell me where
you installed it when done.
```

The agent fetches the GitHub URL, reads SKILL.md + references/ contents, and places them under whichever path that specific client uses. No platform-specific installer to maintain.

## After install

Set these env vars in your shell:

```bash
export FORGEJO_TOKEN=<personal-access-token>
export FORGEJO_URL=https://hackforger.inside.h2os.cloud   # or your instance URL
```

Generate a token at `<FORGEJO_URL>/-/user/settings/applications`. Scope must
include the resources you intend to touch.

## Manual install (if your agent can't fetch URLs)

1. Clone or download this repo
2. Copy the `skills/hackforger-api/` directory to your client's skill path:
   - **Claude Code** (project): `<repo>/.claude/skills/hackforger-api/`
   - **Claude Code** (user): `~/.claude/skills/hackforger-api/`
   - **OpenAI Codex CLI**: `~/.codex/skills/hackforger-api/`
   - **Cursor**: `~/.cursor/skills/hackforger-api/`
   - **Other clients**: see your client's documentation for its skill directory
3. Restart your client (some clients cache the skill list at startup)
4. Set env vars per the section above

## What ships in this skill

- `SKILL.md` — entry point, agent reads this on slash invocation
- `references/` — module-by-module API reference docs the agent loads on demand
  - `README.md`, `attachments.md`, `bounties.md`, `credits.md`,
    `feed-search-reputation.md`, `gaps.md`, `grants.md`, `hackathons.md`,
    `orgs.md`, `user-stories.md`
