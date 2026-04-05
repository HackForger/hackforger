---
name: E2E must be web-only with screenshots
description: E2E tests must use agent-browser web UI operations, never unit/API tests as substitute; key checkpoints require screenshots as evidence
type: feedback
---

E2E tests MUST use web interface operations via agent-browser. Unit tests and API tests cannot substitute for web UI testing in E2E.

**Why:** The user requires proof that the full user-facing flow works end-to-end through the actual browser interface, not just that the backend API returns correct results.

**How to apply:**
- Every E2E prompt must explicitly state: "Web 端测试是必选项，不可使用单元测试或 API 测试替代 Web 界面操作"
- Key verification points must include `agent-browser screenshot` commands
- Screenshots saved to `docs/tests/e2e/screenshots/` and referenced in E2E reports as evidence
- Use `agent-browser --session` for multi-user flows, `eval --stdin` for session-auth POST calls
