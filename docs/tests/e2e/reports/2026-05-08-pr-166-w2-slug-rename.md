---
pr: 166
commits:
  - 67d9b2a04e
  - e5d0456f7a
tested_against: http://localhost:3000 (= https://hackforger.inside.h2os.cloud Mac instance)
tested_at: 2026-05-08T11:50:00+08:00
e2e_owner: claude
admin_signoff:
  by: allenwoods
  at: 2026-05-08T11:50:00+08:00
  notes: "verbal in-chat authorization: '请将 PR #166 合并到生产分支并 deploy，改动很小，请直接帮我合并并推送'. Scope: this single deploy. One-line change in custom/public/assets/landing/index.html — renames the s1-w2 slug from opc-2026-shuzhi-w2 to opc-2026-shuzhi-w2-july because operations rebuilt the activity under a new slug (PR body confirmed both /hackathon/opc-2026-shuzhi-w2-july endpoints return 200 on dev + prod before merge)."
---

# PR #166 — W2 slug rename (opc-2026-shuzhi-w2 → opc-2026-shuzhi-w2-july)

Single-line landing change: the W2 button now points at the rebuilt
hackathon activity under the new slug. Closes #165.

## Verification

```js
window.HACKFORGER_LANDING_CONFIG.stages['s1-w2']
// → {"slug":"opc-2026-shuzhi-w2-july","enabled":true}  ✓

// click → http://localhost:3000/hackathon/opc-2026-shuzhi-w2-july  ✓
```

The W1 button (still `opc-2026-shuzhi-w1`) and other stage configs are
untouched — diff is exactly one line.

Pre-merge curl checks (from PR body) confirmed both
`hackforger.inside.h2os.cloud/hackathon/opc-2026-shuzhi-w2-july` and
`www.synnovator.com/hackathon/opc-2026-shuzhi-w2-july` return 200, so the
target activity exists on both instances before the deploy.
