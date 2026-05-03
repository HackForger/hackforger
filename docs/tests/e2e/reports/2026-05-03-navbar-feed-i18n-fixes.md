---
pr: 148, 150
commits:
  - ccee93c9a4
  - 0646e18621
tested_against: https://hackforger.inside.h2os.cloud
tested_at: 2026-05-03T22:00:00+08:00
e2e_owner: claude
admin_signoff:
  by: allen.woods
  at: 2026-05-03T22:35:00+08:00
  notes: |
    Stage 2 walkthrough confirmed all three observable bugs:
      - PR #148: logged in as non-admin (linyilun), navbar dropdown does NOT
        show 创建黑客松 / 新建资助 entries — '看不到了'.
      - PR #150 (feed): /?repo-search-tab=hackathons now shows 'haiquan 加入了组织
        opc-2026-shuzhi-w1 4 小时前' for the previously-empty card.
      - PR #150 (i18n): /grants/new — '积分预算' label renders correctly (after
        the second rebuild, since the first build's bindata regen had a
        stale-hash race; resolved by removing modules/options/bindata.go.hash
        and rebuilding).
---

# Navbar admin guard (#148) + feed render & i18n (#150)

Two follow-up patches that shipped together with the org-repo registration work.

## PR #148 — `ccee93c9a4` — Custom navbar override missed admin guard

Original issue #118 added `{{if .SignedUser.IsAdmin}}` around 创建黑客松 / 新建资助 in `templates/base/head_navbar.tmpl`, but `custom/templates/base/head_navbar.tmpl` (which Forgejo prefers at runtime) never got the same guard. Non-admin users (linyilun) saw clickable entries that bounced to login.

**Fix:** mirrored the upstream guard into the custom override.

**Verified:** admin walkthrough on `https://hackforger.inside.h2os.cloud/` as linyilun — the create entries are no longer visible in the navbar dropdown.

## PR #150 — `0646e18621` — Feed render + i18n holes

Two visible bugs:

1. `community_feeds.tmpl` had no branch for OpType=60 (`ActionOrgJoinRequest`), so cards with that op_type rendered as `[avatar] username 时间 [pulse]` with empty action description. Triggered after every hackathon registration via `AddOrgUser`.
2. `grants/new.tmpl:42` referenced `hackforger.grant.budget_credits` but only `grant.round.budget_credits` was defined → label rendered as the literal key. Plus a hardcoded `placeholder="0 = personal"` with no i18n.

**Fix:**
- `community_feeds.tmpl`: add OpType 60 branch using new `feed.org_join_request` locale key (en-US + zh-CN).
- `helper.go::HackforgerActionIcon`: add `octicon-organization` mapping for OpType 60.
- `grants/new.tmpl`: route the placeholder through `ctx.Locale.TrString` using new `grant.org_id_personal_hint` key.
- Add `grant.budget_credits` key in both locales.

**Verified:**
- Dashboard `/?repo-search-tab=hackathons`: the previously-empty card shows `haiquan 加入了组织 opc-2026-shuzhi-w1 4 小时前` with the organization icon.
- `/grants/new` (admin): "积分预算" label renders correctly (no key string visible).

## Build note (worth remembering)

The bindata generator (`build/generate-bindata.go`) hashes filename + mtime + size of every file under `options/`. After editing locale `.ini` files, the first `make backend` produced a binary where the new keys weren't embedded — `strings ./gitea | grep grant.budget_credits` came back empty (though the bindata is zstd-compressed so this isn't conclusive).

The reliable fix when locale changes don't seem to land:

```bash
rm -f modules/options/bindata.go.hash
touch options/locale/locale_*.ini
TAGS="bindata sqlite sqlite_unlock_notify" make backend
bash scripts/restart-gitea.sh
```

Worth investigating why the hash check missed our edits (mtime + size both changed). Possibly a race between Edit tool's atomic-replace and the hash-time check. Out of scope for this PR.
