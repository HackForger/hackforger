# i18n Error Pattern: Service Layer → Web Handler

## Problem

Go service functions return `error` — but `fmt.Errorf("english message")` doesn't go through Forgejo's i18n system. If the web handler does `ctx.Flash.Error(err.Error())`, the user sees raw English.

## Correct Pattern

**Service layer:** Return typed errors (not `fmt.Errorf`):

```go
type ErrNoTracks struct{ HackathonID int64 }
func (e ErrNoTracks) Error() string { return fmt.Sprintf("...") }  // for logs only
func IsErrNoTracks(err error) bool { _, ok := err.(ErrNoTracks); return ok }
```

**Web handler:** Check error type, use `ctx.Tr()` for user-facing message:

```go
if err != nil {
    if hackforger_service.IsErrNoTracks(err) {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.no_tracks"))
    } else {
        ctx.Flash.Error(err.Error())  // fallback for unexpected errors
    }
}
```

**API handler:** Can use the Go error message directly (API consumers expect English).

## Why Not `ctx.Tr()` in Service Layer?

Service functions take `context.Context`, not `*context.Context` (web context). They don't have access to `ctx.Tr()` / `ctx.Locale`. This is by design — services are called from both web handlers and API handlers, and they shouldn't know about HTTP/rendering.

## Current Status

| Error | Typed Error | Handler i18n |
|-------|-----------|-------------|
| No tracks to publish | ✅ `ErrNoTracks` | ✅ `hackathon.error.no_tracks` |
| No submissions to judge | ✅ `ErrNoSubmissions` | ✅ `hackathon.error.no_submissions` |
| Wrong phase (Draft/Open/Hacking/Judging) | ❌ `fmt.Errorf` | ⚠️ Falls back to English |
| Cannot cancel finished | ❌ `fmt.Errorf` | ⚠️ Falls back to English |
| Duplicate registration | ✅ `ErrDuplicateRegistration` | ✅ `hackathon.register.already_registered` |
| Not org member | ❌ handler-level check | ✅ `hackathon.error.not_org_member` |

The remaining `fmt.Errorf` cases in the service layer are defensive checks — the web handler already blocks these at the handler level with proper i18n. They only fire if there's a race condition or direct API call. Low priority to convert, but should be done for consistency.

## Rule

**Never show `err.Error()` to users without checking the error type first.** If you can't match the type, use a generic i18n key like `hackforger.hackathon.error.invalid_phase`.
