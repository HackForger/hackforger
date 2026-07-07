# Landing Page: Date-Driven Stage Buttons

## Background

The HackForger landing page (`custom/public/assets/landing/index.html`) showcases
multiple competition stages (S1/S2/S3/Finals). Each stage has Wave cards with
action buttons ("查看详情", "即刻报名", disabled "X日开始报名"). The button
state should reflect whether the Wave's time window is in the past, active, or
future.

## Design Decision: Client-Side Date for UX Only

**The button state is a UX convenience, NOT a security control.**

### Why client-side?

1. **Server already enforces deadlines independently.** The `Phase` model
   (`models/hackforger/phase.go`) has `StartTime` / `EndTime` Unix timestamps.
   `AllowsAction(ctx, "hackathon", id, "register")` calls `CurrentPhase()`,
   which queries `start_time <= now AND end_time > now` using **server time**
   (`time.Now().Unix()`). If no phase is active, the server rejects
   registration on the `/hackathon/<slug>` detail page.

2. **The landing page button only navigates.** `handleRegisterClick()` does
   `window.location.href = '/hackathon/' + slug`. It does not submit anything.
   The actual registration POST goes through `RegisterPost`, which calls
   `AllowsAction` — server-validated.

3. **Therefore**: Even if a user manipulates their system clock to make the
   button appear "registerable", the server will reject the actual registration
   if the phase has ended. The client-side date is purely cosmetic.

### Risk Assessment

| Scenario | Mitigated? |
|---|---|
| User changes clock → button shows "即刻报名" → clicks → server rejects | Yes, server validates phase |
| User changes clock → bypasses registration deadline | No — server independently enforces |
| Clock skew (few hours) causes wrong button state | Acceptable UX; server is source of truth |

### When to use server time instead

If a page element has **security implications** (e.g., displaying whether a
submission window is open before the user fills a form), the page must either:

- Fetch server time via an API, OR
- Validate on the server when the form is submitted

For landing page buttons, this is unnecessary.

## Implementation Pattern

### 1. Mark up each button with `data-wave-start` and `data-wave-end`

```html
<button
  data-stage="s3-w2"
  data-wave-start="2026-07-04"
  data-wave-end="2026-07-23"
  class="...">
  即刻报名
</button>
```

Dates are `YYYY-MM-DD` (local time, start-of-day).

### 2. Add JS to compute state on page load

```javascript
(function() {
  function parseDate(s) {
    var parts = s.split('-');
    return new Date(parseInt(parts[0], 10), parseInt(parts[1], 10) - 1, parseInt(parts[2], 10));
  }
  function startOfDay(d) { return new Date(d.getFullYear(), d.getMonth(), d.getDate()); }

  var buttons = Array.from(document.querySelectorAll('[data-wave-start][data-wave-end]'));
  var today = startOfDay(new Date());

  buttons.forEach(function(btn) {
    var start = parseDate(btn.getAttribute('data-wave-start'));
    var end = parseDate(btn.getAttribute('data-wave-end'));
    // End date is inclusive: window closes at end of that day
    var endInclusive = new Date(end.getTime() + 24 * 60 * 60 * 1000 - 1);

    if (today < start) {
      // Future: disable, show "X月X日开始报名"
      setDisabled(btn, start);
    } else if (today > endInclusive) {
      // Past: disable, show "报名已结束"
      setExpired(btn);
    }
    // else: within window — keep HTML default (active/enabled)
  });
})();
```

### 3. Add i18n for dynamic button texts

In the `DICT` object (the i18n dictionary):

```javascript
"X月X日开始报名": "Opens MM-DD",   // e.g. "7月24日开始报名": "Opens Jul 24"
"报名已结束": "Registration Closed",
```

### 4. Style states

| State | bg | text | cursor |
|---|---|---|---|
| Active (default in HTML) | `bg-primary` | `text-black` | default |
| Future / Past | `bg-[#9CA3AF]` | `text-[#4B5563]` | `cursor-not-allowed` |

Toggle via `classList.remove(...)` / `classList.add(...)`.

## Reusing for Future Events

When adding a new season (e.g., 2027 Lingang):

1. Add `data-wave-start` / `data-wave-end` to each new Wave button.
2. Ensure the corresponding hackathon activity has phases configured in the
   backend with correct `start_time` / `end_time`. **Without phases, the server
   does not enforce any deadline** (`AllowsAction` returns `true` when `count == 0`).
3. The JS logic is generic — no code changes needed for new dates.
4. Add any new button text variants to `DICT` for i18n.

## File Location

- Implementation: `custom/public/assets/landing/index.html` (inline `<script>`)
- Server enforcement: `models/hackforger/phase.go` → `IsLocked()`, `IsActive()`, `AllowsAction()`
- Registration handler: `routers/web/hackforger/hackathon.go` → `RegisterPost()`
