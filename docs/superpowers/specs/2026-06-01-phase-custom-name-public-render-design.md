# Fix: hackathon stage custom name not shown on public page (issue #182)

## Problem

When an admin edits a hackathon stage (phase) name in the management page, the
new name is saved to `phase.custom_name` and shown correctly in the management
UI, but the **public hackathon detail page** keeps showing the phase type's
default i18n name. Reported on https://www.synnovator.com, reproduces every time.

Reproduced locally (port 3000, `hackforger` DB) on hackathon
`opc-2026-crossborder-w2`: setting `custom_name` on a phase, the public page
still rendered the default name (`Registration`); the custom name appeared 0
times in the page HTML.

## Root cause

Two renderers display phase names and only one honors `custom_name`:

- Management UI (`web_src/js/components/hackforger/PhaseTimeline.vue:59`) —
  `if (phase.custom_name) return phase.custom_name;` then falls back to the
  type's default name. **Correct.**
- Public page (`templates/hackforger/hackathon/view.tmpl:30`) —
  `{{if .PhaseType}}{{ctx.Locale.Tr .PhaseType.DisplayNameI18n}}{{end}}` —
  renders only the type's default i18n name, **never reads `.CustomName`.** Bug.

The data layer is fine: `custom_name` is stored (`models/hackforger/phase.go`)
and loaded into the template via `GetPhases` → `Phases`. Only the public
template's display logic is wrong.

## Scope

Single line. Verified the only public template rendering hackathon phase names
is `view.tmpl:30`. The admin `phase_types.tmpl` edits the type *catalog* (not
instances) and is correct as-is; the grants timeline uses fixed status keys
(different feature). No other site affected.

## Change

`templates/hackforger/hackathon/view.tmpl:30`

```go-html-template
{{/* before */}}
<div class="hf-step-name">{{if .PhaseType}}{{ctx.Locale.Tr .PhaseType.DisplayNameI18n}}{{end}}</div>

{{/* after: custom name wins, fall back to default i18n name (mirrors PhaseTimeline.vue) */}}
<div class="hf-step-name">{{if .CustomName}}{{.CustomName}}{{else if .PhaseType}}{{ctx.Locale.Tr .PhaseType.DisplayNameI18n}}{{end}}</div>
```

### Design points

- **Precedence mirrors the Vue reference**: non-empty `custom_name` wins, else
  the type's default i18n name. The two renderers finally agree.
- **Empty-string handling**: `custom_name` defaults to `''`; Go's
  `{{if .CustomName}}` treats `''` as false, so unedited phases keep their
  default name.
- **XSS**: `{{.CustomName}}` is user input but Go `html/template` auto-escapes
  it. Safe.
- **No backend/model/migration/i18n changes** — pure display fix.

## Build & verify

- Template-only change → `make backend` (bindata re-embed) + restart. No
  `make frontend`.
- E2E: log in as admin → management page → rename a stage → confirm the public
  page shows the new name; confirm an unedited stage still shows its default.

## Out of scope

- Refactoring the two renderers into one shared source of truth (future).
- Grant round timeline (unaffected).
