# HackForger API gap-fill — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 4 missing `/api/v1/hackforger/*` endpoints to close the parity gaps the skill audit found, plus remove 2 dead web routes and revise 2 E2E task files accordingly. Lands in PR #73 on top of the existing skill-rename commit.

**Architecture:** Each new API handler lives in its own file under `routers/api/v1/hackforger/`. Form structs are defined inline at the top of each handler file (matching `credits.go` / `hackathon.go` convention). Routes are appended to the existing `/hackforger` group in `routers/api/v1/api.go`. Business logic stays in `models/hackforger/` and `services/hackforger/`; the org join-request flow extracts a small service function so web and API call the same code.

**Tech Stack:** Go 1.22+, XORM, Forgejo `web.Bind` for form binding, integration tests via `testfixtures` + `MakeRequest`.

**Spec:** `docs/superpowers/specs/2026-04-20-hackforger-api-gap-fill-design.md`

**Branch:** `feat/hackforger-api-skill` (worktree at `~/.config/superpowers/worktrees/hackforger/rename-api-skill`). Base: `v0.1-dev/hackforger`. PR #73 already open.

---

## Spec coverage map

| Spec section | Tasks |
|--------------|-------|
| Endpoint 1 — Attachments | 1, 2, 12 (test) |
| Endpoint 2 — Reputation settings | 3, 4, 13 (test) |
| Endpoint 3 — Phase-types CRUD | 5, 6, 14 (test) |
| Endpoint 4 — Org join-request (+ service extraction) | 7, 8, 9, 15 (test) |
| Route registration patch | 10 |
| Build verification + commit 2 | 11 |
| Tests commit (3) | 16 |
| Skill reference updates (commit 4) | 17–22 |
| Dead-route cleanup + E2E revision (commit 5) | 23–27 |

---

## Pre-flight

- [ ] **Step 0a: Confirm worktree + branch**

```bash
cd ~/.config/superpowers/worktrees/hackforger/rename-api-skill
git status      # expect: clean working tree on feat/hackforger-api-skill
git log --oneline -3   # expect spec commit (f5187f42e3) on top of skill-rename commit
```

- [ ] **Step 0b: Copy app.ini for local dev (per CLAUDE.md)**

```bash
mkdir -p custom/conf
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini
```

- [ ] **Step 0c: Verify base build passes before any changes**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -5
```
Expected: `go build` completes, `gitea` binary written, no errors.

---

# Commit 2 — 4 API handlers + form structs + route registration

## Task 1: Attachment handler form + skeleton

**Files:**
- Create: `routers/api/v1/hackforger/attachment.go`

- [ ] **Step 1.1: Create the handler file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"fmt"
	"net/http"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/upload"
	"forgejo.org/services/attachment"
	"forgejo.org/services/context"
)

// AttachmentResponse represents the JSON returned after an upload.
// swagger:response HackforgerAttachment
type AttachmentResponse struct {
	UUID string `json:"uuid"`
}

// UploadAttachmentAPI uploads a platform-level (RepoID=-1) attachment used
// by hackathon descriptions, grant project pages, and submissions.
func UploadAttachmentAPI(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/attachments hackforgerAttachment hackforgerUploadAttachment
	// ---
	// summary: Upload a platform-level attachment (HackForger rich text)
	// consumes:
	//   - multipart/form-data
	// parameters:
	//   - name: file
	//     in: formData
	//     type: file
	//     required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/HackforgerAttachment"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	if !setting.Attachment.Enabled {
		ctx.Error(http.StatusNotFound, "AttachmentDisabled", "attachment is not enabled")
		return
	}

	file, header, err := ctx.Req.FormFile("file")
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FormFile", fmt.Sprintf("FormFile: %v", err))
		return
	}
	defer file.Close()

	attach, err := attachment.UploadAttachment(ctx, file, setting.Attachment.AllowedTypes, header.Size, &repo_model.Attachment{
		Name:       header.Filename,
		UploaderID: ctx.Doer.ID,
		RepoID:     -1, // HackForger platform-level attachment (no specific repo)
	})
	if err != nil {
		if upload.IsErrFileTypeForbidden(err) {
			ctx.Error(http.StatusBadRequest, "FileTypeForbidden", "file type not allowed")
			return
		}
		log.Error("UploadAttachmentAPI: %v", err)
		ctx.Error(http.StatusInternalServerError, "UploadFailed", "upload failed")
		return
	}

	ctx.JSON(http.StatusOK, AttachmentResponse{UUID: attach.UUID})
}
```

- [ ] **Step 1.2: Verify it compiles standalone**

```bash
go build ./routers/api/v1/hackforger/...
```
Expected: no output (success).

## Task 2: Reputation settings handler

**Files:**
- Create: `routers/api/v1/hackforger/admin_settings.go`

- [ ] **Step 2.1: Create the file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"encoding/json"
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
)

// UpdateReputationSettingsForm is the body for PUT /hackforger/admin/reputation/settings.
// Empty fields are skipped (partial update); non-empty fields must be valid JSON.
type UpdateReputationSettingsForm struct {
	Weights string `json:"weights"`
	Tiers   string `json:"tiers"`
}

// ReputationSettingsResponse is the GET/PUT response.
// swagger:response HackforgerReputationSettings
type ReputationSettingsResponse struct {
	Weights string `json:"weights"`
	Tiers   string `json:"tiers"`
}

// GetReputationSettingsAPI returns the current platform reputation algorithm config.
func GetReputationSettingsAPI(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/admin/reputation/settings hackforgerAdmin hackforgerGetReputationSettings
	// ---
	// summary: Get platform reputation algorithm settings
	// produces:
	//   - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/HackforgerReputationSettings"
	weights, _ := hackforger_model.GetSetting(ctx, "reputation.weights")
	tiers, _ := hackforger_model.GetSetting(ctx, "reputation.tiers")
	ctx.JSON(http.StatusOK, ReputationSettingsResponse{Weights: weights, Tiers: tiers})
}

// UpdateReputationSettingsAPI partially updates weights and/or tiers.
func UpdateReputationSettingsAPI(ctx *context.APIContext) {
	// swagger:operation PUT /hackforger/admin/reputation/settings hackforgerAdmin hackforgerUpdateReputationSettings
	// ---
	// summary: Update platform reputation algorithm settings
	// consumes:
	//   - application/json
	// parameters:
	//   - name: body
	//     in: body
	//     schema:
	//       "$ref": "#/definitions/UpdateReputationSettingsForm"
	// responses:
	//   "200":
	//     "$ref": "#/responses/HackforgerReputationSettings"
	//   "422":
	//     "$ref": "#/responses/validationError"
	form := web.GetForm(ctx).(*UpdateReputationSettingsForm)

	if form.Weights != "" {
		if !json.Valid([]byte(form.Weights)) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidJSON", "weights is not valid JSON")
			return
		}
		if err := hackforger_model.SetSetting(ctx, "reputation.weights", form.Weights); err != nil {
			ctx.InternalServerError(err)
			return
		}
	}
	if form.Tiers != "" {
		if !json.Valid([]byte(form.Tiers)) {
			ctx.Error(http.StatusUnprocessableEntity, "InvalidJSON", "tiers is not valid JSON")
			return
		}
		if err := hackforger_model.SetSetting(ctx, "reputation.tiers", form.Tiers); err != nil {
			ctx.InternalServerError(err)
			return
		}
	}

	weights, _ := hackforger_model.GetSetting(ctx, "reputation.weights")
	tiers, _ := hackforger_model.GetSetting(ctx, "reputation.tiers")
	ctx.JSON(http.StatusOK, ReputationSettingsResponse{Weights: weights, Tiers: tiers})
}
```

- [ ] **Step 2.2: Verify model has GetSetting (used above)**

```bash
grep -n "func GetSetting" models/hackforger/setting.go
```
Expected: a function `func GetSetting(ctx context.Context, key string) (string, error)` (or similar; if signature differs, adjust the calls in step 2.1).

- [ ] **Step 2.3: Compile**

```bash
go build ./routers/api/v1/hackforger/...
```
Expected: success.

## Task 3: Phase-types CRUD handlers

**Files:**
- Create: `routers/api/v1/hackforger/admin_phase_types.go`

- [ ] **Step 3.1: Read the model so the handler matches its signatures**

```bash
sed -n '1,120p' models/hackforger/phase_type.go
```
Confirm field names of `hackforger_model.PhaseType` and signatures of `GetAllPhaseTypes`, `GetPhaseTypeByID`, `CreatePhaseType`, `UpdatePhaseType`, `DeletePhaseType`. Use these exact names in the handler.

- [ ] **Step 3.2: Create the handler file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
)

// CreatePhaseTypeForm is the JSON body for POST.
type CreatePhaseTypeForm struct {
	ActivityKind    string `json:"activity_kind" binding:"Required;In(hackathon,bounty,grant)"`
	Key             string `json:"key" binding:"Required"`
	DisplayNameI18n string `json:"display_name_i18n" binding:"Required"`
	IsUnique        bool   `json:"is_unique"`
	AllowedActions  string `json:"allowed_actions"`
	DefaultOrder    int    `json:"default_order"`
}

// UpdatePhaseTypeForm is the JSON body for PUT — same shape, full overwrite.
type UpdatePhaseTypeForm = CreatePhaseTypeForm

// ListPhaseTypesAPI returns the global phase type catalog (flat array).
func ListPhaseTypesAPI(ctx *context.APIContext) {
	// swagger:operation GET /hackforger/admin/phase-types hackforgerAdmin hackforgerListPhaseTypes
	// ---
	// summary: List all phase types (admin)
	// responses:
	//   "200":
	//     description: array of phase types
	pts, err := hackforger_model.GetAllPhaseTypes(ctx)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, pts)
}

// CreatePhaseTypeAPI inserts a new phase type.
func CreatePhaseTypeAPI(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/admin/phase-types hackforgerAdmin hackforgerCreatePhaseType
	// ---
	// summary: Create a phase type (admin)
	// parameters:
	//   - name: body
	//     in: body
	//     schema:
	//       "$ref": "#/definitions/CreatePhaseTypeForm"
	// responses:
	//   "201":
	//     description: created phase type
	form := web.GetForm(ctx).(*CreatePhaseTypeForm)
	allowed := form.AllowedActions
	if allowed == "" {
		allowed = "[]"
	}
	pt := &hackforger_model.PhaseType{
		ActivityKind:    form.ActivityKind,
		Key:             form.Key,
		DisplayNameI18n: form.DisplayNameI18n,
		IsUnique:        form.IsUnique,
		AllowedActions:  allowed,
		DefaultOrder:    form.DefaultOrder,
	}
	if err := hackforger_model.CreatePhaseType(ctx, pt); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusCreated, pt)
}

// UpdatePhaseTypeAPI replaces all fields of an existing phase type.
func UpdatePhaseTypeAPI(ctx *context.APIContext) {
	// swagger:operation PUT /hackforger/admin/phase-types/{id} hackforgerAdmin hackforgerUpdatePhaseType
	// ---
	// summary: Update a phase type (admin)
	// parameters:
	//   - name: id
	//     in: path
	//     type: integer
	//     required: true
	//   - name: body
	//     in: body
	//     schema:
	//       "$ref": "#/definitions/CreatePhaseTypeForm"
	// responses:
	//   "200":
	//     description: updated phase type
	//   "404":
	//     "$ref": "#/responses/notFound"
	id := ctx.ParamsInt64(":id")
	pt, err := hackforger_model.GetPhaseTypeByID(ctx, id)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	if pt == nil {
		ctx.NotFound()
		return
	}
	form := web.GetForm(ctx).(*UpdatePhaseTypeForm)
	pt.ActivityKind = form.ActivityKind
	pt.Key = form.Key
	pt.DisplayNameI18n = form.DisplayNameI18n
	pt.IsUnique = form.IsUnique
	pt.AllowedActions = form.AllowedActions
	if pt.AllowedActions == "" {
		pt.AllowedActions = "[]"
	}
	pt.DefaultOrder = form.DefaultOrder
	if err := hackforger_model.UpdatePhaseType(ctx, pt); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, pt)
}

// DeletePhaseTypeAPI removes a phase type.
func DeletePhaseTypeAPI(ctx *context.APIContext) {
	// swagger:operation DELETE /hackforger/admin/phase-types/{id} hackforgerAdmin hackforgerDeletePhaseType
	// ---
	// summary: Delete a phase type (admin)
	// parameters:
	//   - name: id
	//     in: path
	//     type: integer
	//     required: true
	// responses:
	//   "204":
	//     description: deleted
	//   "404":
	//     "$ref": "#/responses/notFound"
	id := ctx.ParamsInt64(":id")
	pt, err := hackforger_model.GetPhaseTypeByID(ctx, id)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	if pt == nil {
		ctx.NotFound()
		return
	}
	if err := hackforger_model.DeletePhaseType(ctx, id); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
```

- [ ] **Step 3.3: Compile**

```bash
go build ./routers/api/v1/hackforger/...
```
Expected: success. If `In(hackathon,bounty,grant)` validator binding tag isn't supported, drop it (keep `Required`) and let the model layer reject invalid values.

## Task 4: Extract org join-request to a service function

**Files:**
- Create: `services/hackforger/org.go`
- Modify: `routers/web/hackforger/org.go` — replace inline logic with service call

- [ ] **Step 4.1: Read the existing web handler**

```bash
sed -n '1,80p' routers/web/hackforger/org.go
```
Note the imports it uses (notify_service, ActionOrgJoinRequest, etc.) and the duplicate-detection guard.

- [ ] **Step 4.2: Create the service**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"context"
	"errors"

	organization_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	notify_service "forgejo.org/services/notify"
)

// ErrAlreadyOrgMember is returned when the doer is already a member or owner.
var ErrAlreadyOrgMember = errors.New("user is already a member of the org")

// IsErrAlreadyOrgMember reports whether err == ErrAlreadyOrgMember.
func IsErrAlreadyOrgMember(err error) bool { return errors.Is(err, ErrAlreadyOrgMember) }

// RequestOrgJoin notifies the org's owners that a user wants to join.
// Returns ErrAlreadyOrgMember if the doer already belongs to the org.
// Fire-and-forget: no row is persisted; this only publishes a feed event.
func RequestOrgJoin(ctx context.Context, org *organization_model.Organization, doer *user_model.User) error {
	isMember, err := organization_model.IsOrganizationMember(ctx, org.ID, doer.ID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyOrgMember
	}
	notify_service.HackforgerEntityCreated(ctx, doer, notify_service.HackforgerEntity{
		Kind: notify_service.ActionOrgJoinRequest,
		ID:   org.ID,
		Name: org.Name,
	})
	return nil
}
```

> If the existing web handler uses a different `notify_service` API shape, mirror it exactly here. Adjust the struct literal to whatever signature `notify_service.HackforgerEntityCreated` actually accepts in this codebase. Don't invent fields.

- [ ] **Step 4.3: Replace inline logic in the web handler**

Replace the body of `JoinOrgRequest` in `routers/web/hackforger/org.go` to call `hackforger_service.RequestOrgJoin(ctx, org, ctx.Doer)`. Map `ErrAlreadyOrgMember` to `ctx.Flash.Info(ctx.Tr("hackforger.org.already_member"))`. Other errors → `ctx.ServerError`.

- [ ] **Step 4.4: Compile**

```bash
go build ./services/hackforger/... ./routers/web/hackforger/...
```
Expected: success.

## Task 5: Org join-request API handler

**Files:**
- Create: `routers/api/v1/hackforger/org.go`

- [ ] **Step 5.1: Create the file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger

import (
	"net/http"

	hackforger_service "forgejo.org/services/hackforger"
	"forgejo.org/services/context"
)

// OrgJoinRequestResponse is the API response for the join-request endpoint.
// swagger:response HackforgerOrgJoinRequest
type OrgJoinRequestResponse struct {
	Status string `json:"status"`
}

// CreateOrgJoinRequestAPI publishes a "user wants to join this org" notification.
// Fire-and-forget: returns 202 because no resource is persisted.
func CreateOrgJoinRequestAPI(ctx *context.APIContext) {
	// swagger:operation POST /hackforger/orgs/{org}/join-request hackforgerOrg hackforgerCreateOrgJoinRequest
	// ---
	// summary: Request to join a HackForger organization
	// parameters:
	//   - name: org
	//     in: path
	//     type: string
	//     required: true
	// responses:
	//   "202":
	//     "$ref": "#/responses/HackforgerOrgJoinRequest"
	//   "422":
	//     "$ref": "#/responses/validationError"
	//   "404":
	//     "$ref": "#/responses/notFound"
	if err := hackforger_service.RequestOrgJoin(ctx, ctx.Org.Organization, ctx.Doer); err != nil {
		if hackforger_service.IsErrAlreadyOrgMember(err) {
			ctx.Error(http.StatusUnprocessableEntity, "AlreadyMember", "user is already a member of this organization")
			return
		}
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusAccepted, OrgJoinRequestResponse{Status: "notified"})
}
```

- [ ] **Step 5.2: Compile**

```bash
go build ./routers/api/v1/hackforger/...
```
Expected: success.

## Task 6: Register the 4 routes

**Files:**
- Modify: `routers/api/v1/api.go` — append to the `/hackforger` group (around line 1808, before the org/repo blocks that follow `/hackforger`)

- [ ] **Step 6.1: Locate the insertion point**

```bash
grep -n 'm.Get("/feed"' routers/api/v1/api.go
```
Expected: a line near 1812. Insert the new routes immediately AFTER the `/feed` and `/search` registrations (i.e., still inside the `/hackforger` group), but BEFORE the `m.Group("/grants"` line.

- [ ] **Step 6.2: Insert route block**

After the `m.Get("/search", hackforger_api.SearchAPI)` line and before `m.Group("/grants"`, insert:

```go
			// Platform-level attachment upload
			m.Post("/attachments", reqToken(), hackforger_api.UploadAttachmentAPI)

			// Org join-request (singular, mirrors web /-/org/{org}/join-request)
			m.Group("/orgs/{org}", func() {
				m.Post("/join-request", reqToken(), hackforger_api.CreateOrgJoinRequestAPI)
			}, orgAssignment(true))

			// Admin: reputation settings + phase type catalog
			m.Group("/admin", func() {
				m.Group("/reputation/settings", func() {
					m.Get("", hackforger_api.GetReputationSettingsAPI)
					m.Put("", bind(hackforger_api.UpdateReputationSettingsForm{}), hackforger_api.UpdateReputationSettingsAPI)
				}, reqSiteAdmin())
				m.Group("/phase-types", func() {
					m.Get("", hackforger_api.ListPhaseTypesAPI)
					m.Post("", bind(hackforger_api.CreatePhaseTypeForm{}), hackforger_api.CreatePhaseTypeAPI)
					m.Put("/{id}", bind(hackforger_api.UpdatePhaseTypeForm{}), hackforger_api.UpdatePhaseTypeAPI)
					m.Delete("/{id}", hackforger_api.DeletePhaseTypeAPI)
				}, reqSiteAdmin())
			})
```

- [ ] **Step 6.3: Verify orgAssignment signature**

```bash
grep -n "func orgAssignment" routers/api/v1/api.go
```
Expected: `func orgAssignment(args ...bool) func(...)`. If the signature requires different args than `(true)`, adjust per the function comment.

- [ ] **Step 6.4: Build full backend**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -10
```
Expected: success. Routes are now compiled in.

## Task 7: Manual smoke test of new routes

- [ ] **Step 7.1: Start the server in this worktree**

```bash
rm -f data/queues/common/LOCK 2>/dev/null
./gitea web 2>&1 &
sleep 5
```

- [ ] **Step 7.2: Smoke-check routes are registered**

```bash
curl -s http://localhost:3000/api/v1/hackforger/admin/phase-types -H "Authorization: token $FORGEJO_TOKEN" | head -c 200
curl -s http://localhost:3000/api/v1/hackforger/admin/reputation/settings -H "Authorization: token $FORGEJO_TOKEN" | head -c 200
```
Expected: JSON responses (either array, object, or `{"message":"You must be an administrator..."}` if `$FORGEJO_TOKEN` isn't an admin token — that's still proof the route is wired).

- [ ] **Step 7.3: Stop the server**

```bash
kill %1
```

## Task 8: Commit 2

- [ ] **Step 8.1: Stage and commit**

```bash
cd ~/.config/superpowers/worktrees/hackforger/rename-api-skill
git add \
  routers/api/v1/hackforger/attachment.go \
  routers/api/v1/hackforger/admin_settings.go \
  routers/api/v1/hackforger/admin_phase_types.go \
  routers/api/v1/hackforger/org.go \
  services/hackforger/org.go \
  routers/web/hackforger/org.go \
  routers/api/v1/api.go

git commit -m "$(cat <<'EOF'
feat(api): close 4 hackforger API parity gaps

Adds /api/v1/hackforger/* endpoints for the 4 web-only operations the
skill audit identified:

- POST /hackforger/attachments — platform-level (RepoID=-1) upload
- GET/PUT /hackforger/admin/reputation/settings — algorithm config
- CRUD /hackforger/admin/phase-types — phase type catalog
- POST /hackforger/orgs/{org}/join-request — fire-and-forget notification

Extracts services/hackforger/RequestOrgJoin so web and API call the
same business logic. Tests follow in the next commit.

Spec: docs/superpowers/specs/2026-04-20-hackforger-api-gap-fill-design.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

# Commit 3 — Integration tests

## Task 9: Attachment integration test

**Files:**
- Create: `tests/integration/hackforger_attachment_test.go`

- [ ] **Step 9.1: Read the test helper to mirror its style**

```bash
sed -n '1,80p' tests/integration/hackforger_test_helper_test.go
```
Note the names of helpers (`hackforgerLoginAs`, `hackforgerPost`, etc.) and the `tests.PrepareTestEnv` invocation pattern.

- [ ] **Step 9.2: Write the test file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIHackforgerAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Upload happy path", func(t *testing.T) {
		token := getUserToken(t, "user2", "write:user")

		body := &bytes.Buffer{}
		mw := multipart.NewWriter(body)
		fw, err := mw.CreateFormFile("file", "design.png")
		assert.NoError(t, err)
		_, err = fw.Write([]byte("fake png bytes"))
		assert.NoError(t, err)
		mw.Close()

		req := NewRequestWithBody(t, "POST", "/api/v1/hackforger/attachments", body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		AddTokenAuth(req, token)

		resp := MakeRequest(t, req, http.StatusOK)
		var got struct{ UUID string `json:"uuid"` }
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.NotEmpty(t, got.UUID)
		assert.True(t, strings.Contains(got.UUID, "-"), "uuid should be uuid-shaped")
	})

	t.Run("401 without token", func(t *testing.T) {
		req := NewRequestWithBody(t, "POST", "/api/v1/hackforger/attachments", bytes.NewBufferString(""))
		MakeRequest(t, req, http.StatusUnauthorized)
	})
}
```

> If `getUserToken` / `NewRequestWithBody` / `AddTokenAuth` have different names in this codebase, replace with the actual helpers (you'll see them in step 9.1).

- [ ] **Step 9.3: Run just this test**

```bash
go test -tags "sqlite sqlite_unlock_notify" -run TestAPIHackforgerAttachment ./tests/integration/... -count=1 -v 2>&1 | tail -20
```
Expected: 2 sub-tests pass.

## Task 10: Reputation settings test (append to existing file)

**Files:**
- Modify: `tests/integration/hackforger_reputation_test.go` — append a new `TestAPIHackforgerReputationSettings` function

- [ ] **Step 10.1: Append to the file**

```go
func TestAPIHackforgerReputationSettings(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Admin PUT then GET roundtrip", func(t *testing.T) {
		adminToken := getUserToken(t, "user1", "write:admin")

		putBody := map[string]string{
			"weights": `{"hackathon":0.5,"bounty":0.3,"grant":0.2}`,
			"tiers":   `[{"name":"bronze","min":0},{"name":"silver","min":100}]`,
		}
		req := NewRequestWithJSON(t, "PUT", "/api/v1/hackforger/admin/reputation/settings", putBody)
		AddTokenAuth(req, adminToken)
		MakeRequest(t, req, http.StatusOK)

		req = NewRequest(t, "GET", "/api/v1/hackforger/admin/reputation/settings")
		AddTokenAuth(req, adminToken)
		resp := MakeRequest(t, req, http.StatusOK)
		var got struct{ Weights, Tiers string }
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.Contains(t, got.Weights, "hackathon")
		assert.Contains(t, got.Tiers, "bronze")
	})

	t.Run("403 for non-admin", func(t *testing.T) {
		userToken := getUserToken(t, "user2", "read:admin")
		req := NewRequestWithJSON(t, "PUT", "/api/v1/hackforger/admin/reputation/settings",
			map[string]string{"weights": `{"x":1}`})
		AddTokenAuth(req, userToken)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("422 on invalid JSON", func(t *testing.T) {
		adminToken := getUserToken(t, "user1", "write:admin")
		req := NewRequestWithJSON(t, "PUT", "/api/v1/hackforger/admin/reputation/settings",
			map[string]string{"weights": "not-json"})
		AddTokenAuth(req, adminToken)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
	})
}
```

- [ ] **Step 10.2: Run**

```bash
go test -tags "sqlite sqlite_unlock_notify" -run TestAPIHackforgerReputationSettings ./tests/integration/... -count=1 -v 2>&1 | tail -20
```
Expected: 3 sub-tests pass.

## Task 11: Phase-types CRUD test

**Files:**
- Create: `tests/integration/hackforger_phase_type_test.go`

- [ ] **Step 11.1: Write the test file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIHackforgerPhaseType(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	adminToken := getUserToken(t, "user1", "write:admin")

	var createdID int64

	t.Run("Create", func(t *testing.T) {
		body := map[string]any{
			"activity_kind":     "hackathon",
			"key":               "test_phase_" + fmt.Sprint(t.Name()),
			"display_name_i18n": "test.phase.display",
			"is_unique":         false,
			"allowed_actions":   `["test"]`,
			"default_order":     99,
		}
		req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/admin/phase-types", body)
		AddTokenAuth(req, adminToken)
		resp := MakeRequest(t, req, http.StatusCreated)
		var got struct{ ID int64 `json:"id"` }
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.NotZero(t, got.ID)
		createdID = got.ID
	})

	t.Run("List includes the new entry", func(t *testing.T) {
		req := NewRequest(t, "GET", "/api/v1/hackforger/admin/phase-types")
		AddTokenAuth(req, adminToken)
		resp := MakeRequest(t, req, http.StatusOK)
		var pts []struct{ ID int64 `json:"id"` }
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&pts))
		found := false
		for _, p := range pts {
			if p.ID == createdID {
				found = true
				break
			}
		}
		assert.True(t, found, "newly created phase type should appear in list")
	})

	t.Run("Update", func(t *testing.T) {
		body := map[string]any{
			"activity_kind":     "hackathon",
			"key":               "test_phase_updated",
			"display_name_i18n": "test.phase.updated",
			"is_unique":         true,
			"allowed_actions":   `["updated"]`,
			"default_order":     100,
		}
		req := NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/hackforger/admin/phase-types/%d", createdID), body)
		AddTokenAuth(req, adminToken)
		MakeRequest(t, req, http.StatusOK)
	})

	t.Run("Delete", func(t *testing.T) {
		req := NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/hackforger/admin/phase-types/%d", createdID))
		AddTokenAuth(req, adminToken)
		MakeRequest(t, req, http.StatusNoContent)
	})

	t.Run("403 for non-admin POST", func(t *testing.T) {
		userToken := getUserToken(t, "user2", "write:user")
		req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/admin/phase-types",
			map[string]any{"activity_kind": "hackathon", "key": "x", "display_name_i18n": "x"})
		AddTokenAuth(req, userToken)
		MakeRequest(t, req, http.StatusForbidden)
	})
}
```

- [ ] **Step 11.2: Run**

```bash
go test -tags "sqlite sqlite_unlock_notify" -run TestAPIHackforgerPhaseType ./tests/integration/... -count=1 -v 2>&1 | tail -25
```
Expected: 5 sub-tests pass.

## Task 12: Org join-request test

**Files:**
- Create: `tests/integration/hackforger_org_test.go`

- [ ] **Step 12.1: Find a fixture user that is NOT a member of an org**

```bash
grep -l "Username:" models/fixtures/user.yml 2>/dev/null | head -1
sed -n '1,40p' models/fixtures/org_user.yml 2>/dev/null
```
Expected: pick a `user_X` that isn't in `org_user.yml` for some `orgY`. For Forgejo's seed data, `user5` is typically not in `org3`. Use whichever combination is clean.

- [ ] **Step 12.2: Write the test file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"
)

func TestAPIHackforgerOrgJoinRequest(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Non-member request happy path", func(t *testing.T) {
		// user5 is not in org3 in the standard fixtures; adjust if your fixtures differ.
		token := getUserToken(t, "user5", "write:organization")
		req := NewRequest(t, "POST", "/api/v1/hackforger/orgs/org3/join-request")
		AddTokenAuth(req, token)
		MakeRequest(t, req, http.StatusAccepted)
	})

	t.Run("Existing member returns 422", func(t *testing.T) {
		// user2 IS a member of org3 in standard fixtures.
		token := getUserToken(t, "user2", "write:organization")
		req := NewRequest(t, "POST", "/api/v1/hackforger/orgs/org3/join-request")
		AddTokenAuth(req, token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
	})

	t.Run("401 without token", func(t *testing.T) {
		req := NewRequest(t, "POST", "/api/v1/hackforger/orgs/org3/join-request")
		MakeRequest(t, req, http.StatusUnauthorized)
	})
}
```

- [ ] **Step 12.2b: Verify fixture assumptions before running**

```bash
grep -n "user_id: 5" models/fixtures/org_user.yml | head -5
grep -n "org_id: 3" models/fixtures/org_user.yml | head -10
```
If user5 happens to be a member of org3 in current fixtures, swap to a clean user/org pair from the grep output.

- [ ] **Step 12.3: Run**

```bash
go test -tags "sqlite sqlite_unlock_notify" -run TestAPIHackforgerOrgJoinRequest ./tests/integration/... -count=1 -v 2>&1 | tail -15
```
Expected: 3 sub-tests pass.

## Task 13: Run all hackforger tests + commit 3

- [ ] **Step 13.1: Run all hackforger integration tests as a regression sweep**

```bash
go test -tags "sqlite sqlite_unlock_notify" -run "TestAPIHackforger" ./tests/integration/... -count=1 2>&1 | tail -15
```
Expected: PASS (existing + 4 new test functions).

- [ ] **Step 13.2: Commit**

```bash
git add tests/integration/hackforger_attachment_test.go \
        tests/integration/hackforger_reputation_test.go \
        tests/integration/hackforger_phase_type_test.go \
        tests/integration/hackforger_org_test.go

git commit -m "$(cat <<'EOF'
test(api): integration coverage for new hackforger gap-fill endpoints

Happy path + auth checks for attachments, reputation settings,
phase-types CRUD, and org join-request. Reuses existing
hackforger_test_helper helpers.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

# Commit 4 — Sync skill references

## Task 14: Update slash-command modules table

**Files:**
- Modify: `.claude/commands/hackforger-api.md`

- [ ] **Step 14.1: Add 2 new rows + update existing rows for new endpoints**

In the modules table, change the Hackathons and Feed/Search/Reputation rows to mention the new admin coverage, and add 2 new rows:

```diff
 | Module | Reference | Covers |
 |--------|-----------|--------|
-| Hackathons | `docs/skills/hackforger-api/hackathons.md` | create / publish / register / submit / score / finalize / leaderboard / phases / criteria |
+| Hackathons | `docs/skills/hackforger-api/hackathons.md` | create / publish / register / submit / score / finalize / leaderboard / phases / criteria / admin phase type catalog |
 | Bounties | `docs/skills/hackforger-api/bounties.md` | create / apply / accept / complete / pay / cancel / winners (competitive) |
 | Grants | `docs/skills/hackforger-api/grants.md` | round CRUD / open / close / projects / approve / award / distribute |
 | Credits | `docs/skills/hackforger-api/credits.md` | balance / transactions / redeem options / orders / fulfill / admin deposit-deduct |
-| Feed/Search/Reputation | `docs/skills/hackforger-api/feed-search-reputation.md` | global feed / unified search / reputation leaderboard / assistant chat |
+| Feed/Search/Reputation | `docs/skills/hackforger-api/feed-search-reputation.md` | global feed / unified search / reputation leaderboard / reputation settings (admin) / assistant chat |
+| Orgs | `docs/skills/hackforger-api/orgs.md` | join requests |
+| Attachments | `docs/skills/hackforger-api/attachments.md` | platform-level rich-text uploads |
```

## Task 15: Add phase-type catalog section to hackathons.md

**Files:**
- Modify: `docs/skills/hackforger-api/hackathons.md`

- [ ] **Step 15.1: Append a new section at the end of the file**

```markdown
## Admin: phase type catalog

The **phase type catalog** is the global library of phase definitions (Registration, Development, Judging, etc.) that organizers pick from when configuring a hackathon's timeline. Per-hackathon `/phases` (above) creates *instances* from these types.

Auth: `reqToken() + reqSiteAdmin()`.

| Verb | Path | Purpose |
|------|------|---------|
| GET | `/hackforger/admin/phase-types` | List all phase types (flat array) |
| POST | `/hackforger/admin/phase-types` | Create (`activity_kind`, `key`, `display_name_i18n`, `is_unique`, `allowed_actions`, `default_order`) |
| PUT | `/hackforger/admin/phase-types/{id}` | Full overwrite |
| DELETE | `/hackforger/admin/phase-types/{id}` | Remove |

`activity_kind` is one of `hackathon`, `bounty`, `grant`. `display_name_i18n` is an i18n **key** (e.g. `hackforger.phase.registration`), not display text — clients translate via locale.
```

## Task 16: Add reputation settings to feed-search-reputation.md

**Files:**
- Modify: `docs/skills/hackforger-api/feed-search-reputation.md`

- [ ] **Step 16.1: Append after the Reputation table**

In the Reputation section, add 2 rows to the existing table OR append a new sub-section:

```markdown
### Reputation algorithm settings (admin)

| Verb | Path | Purpose |
|------|------|---------|
| GET | `/hackforger/admin/reputation/settings` | Read raw `weights` and `tiers` JSON |
| PUT | `/hackforger/admin/reputation/settings` | Update one or both fields (partial update; non-empty fields must be valid JSON) |
```

## Task 17: Create attachments.md

**Files:**
- Create: `docs/skills/hackforger-api/attachments.md`

- [ ] **Step 17.1: Write the file**

```markdown
# Attachments API

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/hackforger/attachments` | Upload a platform-level (RepoID=-1) attachment for use in hackathon descriptions, grant project pages, and submission READMEs |

## Why this exists

Forgejo's standard attachment endpoints (`/repos/{owner}/{repo}/issues/{index}/assets`, `/releases/{id}/assets`) all force a repo / issue / release scope and persist `RepoID > 0`. HackForger needs attachments for content that isn't bound to any repo — hackathon descriptions, grant project pitches, etc. — so this endpoint produces an attachment with `RepoID = -1`.

## Usage

```bash
curl -X POST \
  -H "Authorization: token $FORGEJO_TOKEN" \
  -F "file=@design.png" \
  https://hackforger.inside.h2os.cloud/api/v1/hackforger/attachments
# → {"uuid":"a3f8...e2c1"}
```

Reference the returned UUID in markdown like:
```markdown
![architecture diagram](/attachments/a3f8...e2c1)
```

## Errors

| Status | Cause |
|--------|-------|
| 400 | File extension not in `setting.Attachment.AllowedTypes` |
| 404 | Site attachment uploads disabled |
| 401 | No token |
```

## Task 18: Create orgs.md

**Files:**
- Create: `docs/skills/hackforger-api/orgs.md`

- [ ] **Step 18.1: Write the file**

```markdown
# Orgs API (HackForger extensions)

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/hackforger/orgs/{org}/join-request` | Notify the org's owners that the authenticated user wants to join |

## Semantics

This is **fire-and-forget**. No `join_request` row is persisted; the operation only publishes a `ActionOrgJoinRequest` notification visible to the org's owners. Hence the `202 Accepted` response — there is no resource to GET later, only a side-effecting notification.

To **accept** or **reject** a join request, use Forgejo's standard team membership API:

| Forgejo standard | Use for |
|------------------|---------|
| `PUT /teams/{id}/members/{username}` | Add the user to a team (effectively accepts) |
| `DELETE /teams/{id}/members/{username}` | Remove (or reject by inaction) |

## Usage

```bash
curl -X POST \
  -H "Authorization: token $FORGEJO_TOKEN" \
  https://hackforger.inside.h2os.cloud/api/v1/hackforger/orgs/web3-innovation/join-request
# → 202 {"status":"notified"}
```

## Errors

| Status | Cause |
|--------|-------|
| 422 | The authenticated user is already a member of the org |
| 404 | Org does not exist |
| 401 | No token |
```

## Task 19: Update README + close 4 gap rows

**Files:**
- Modify: `docs/skills/hackforger-api/README.md`
- Modify: `docs/skills/hackforger-api/gaps.md`

- [ ] **Step 19.1: README — add 2 entries to the Files list**

After the `feed-search-reputation.md` line, before `user-stories.md`:

```markdown
- **[orgs.md](orgs.md)** — HackForger extensions to orgs: join-request notification.
- **[attachments.md](attachments.md)** — Platform-level (RepoID=-1) rich-text uploads.
```

- [ ] **Step 19.2: gaps.md — replace the 4 rows in the "Real gaps" table with a closure note**

Replace the entire `## Real gaps` section's table (4 rows) with:

```markdown
## Real gaps

> ✅ All 4 previously-listed gaps closed in PR #73 (commit feat(api): close 4 hackforger API parity gaps). The endpoints are now documented in:
>
> - `attachments.md` — `POST /hackforger/attachments`
> - `feed-search-reputation.md` — `GET/PUT /hackforger/admin/reputation/settings`
> - `hackathons.md` — phase type catalog CRUD
> - `orgs.md` — `POST /hackforger/orgs/{org}/join-request`
>
> Backlog is empty as of 2026-04-20.
```

(Leave the "Dead routes" section in place — commit 5 cleans it.)

## Task 20: Commit 4

- [ ] **Step 20.1: Stage and commit**

```bash
git add .claude/commands/hackforger-api.md \
        docs/skills/hackforger-api/README.md \
        docs/skills/hackforger-api/hackathons.md \
        docs/skills/hackforger-api/feed-search-reputation.md \
        docs/skills/hackforger-api/attachments.md \
        docs/skills/hackforger-api/orgs.md \
        docs/skills/hackforger-api/gaps.md

git commit -m "$(cat <<'EOF'
docs(skills): sync hackforger-api references with new endpoints

Adds attachments.md and orgs.md, extends hackathons.md with the
phase type catalog section, extends feed-search-reputation.md with
the admin settings endpoints, updates the slash-command modules
table, and closes the 4 entries in gaps.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

# Commit 5 — Dead-route cleanup + E2E phase-step revision

## Task 21: Confirm no template buttons reference dead routes

- [ ] **Step 21.1: Grep templates and frontend**

```bash
grep -rn "manage/start\|manage/judge[^s]" templates/ web_src/ 2>&1 | grep -v ".git" | head -10
```
Expected: empty (no UI references to the dead URLs). If any are found, commit 5 must also remove those template/JS lines.

## Task 22: Remove dead routes from web.go

**Files:**
- Modify: `routers/web/web.go`

- [ ] **Step 22.1: Delete the 2 lines**

Open the file at the `/hackathon/{slug}/manage` group (around line 542). Remove these two lines:

```go
m.Post("/start", hackforger_web.ManagePhasePost)
m.Post("/judge", hackforger_web.ManagePhasePost)
```

(Keep `m.Post("/publish", ...)` and `m.Post("/cancel", ...)` — those are real.)

- [ ] **Step 22.2: Build to confirm nothing else referenced them**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -5
```
Expected: success.

## Task 23: Revise E2E task 3.1

**Files:**
- Modify: `docs/tests/e2e/tasks/full-cycle/01-hackathon-lifecycle.md`

- [ ] **Step 23.1: Replace Step 3.1 body**

Replace lines describing the click-to-start operation with cron-driven advancement:

```markdown
### Step 3.1: 等待 Hacking 阶段自动开启 (Open -> Hacking)
- **角色**: hackforger session（仅做观察验证，不操作）
- **前置**: Step 1.1 创建 Hackathon 时设置短时长（例如 registration phase 时长 90s，开始时间 = 创建时刻 + 0s）
- **操作**: 不触发任何 UI 操作；等待最多 5 分钟让 `hackforger_hackathon_status` cron 完成阶段切换。可在管理页 `/hackathon/web3-innovation/manage` 周期刷新观察状态条
- **验证**:
  - 状态自动从 `Open(1)` 变为 `Hacking(2)`，无需任何用户点击
  - Feed 事件: `hackathon_phase_changed(50)` -- 全局 + 组织可见
- **截图**: `screenshots/full-cycle/p3-01-hacking-started.png` —— 截图时机：状态变更后的管理页
- **设计说明**: 阶段切换由 gocron `hackforger_hackathon_status` (every 5min) + gocron 在 phase start/end 时刻触发的 `onPhaseEvent` 完成；不存在"开始 Hacking"按钮
```

## Task 24: Revise E2E task 7.1

**Files:**
- Modify: `docs/tests/e2e/tasks/full-cycle/04-submission-judging.md`

- [ ] **Step 24.1: Replace Step 7.1 body**

```markdown
### Step 7.1: 等待 Judging 阶段自动开启 (Hacking -> Judging)
- **角色**: hackforger session（仅做观察验证）
- **前置**: development phase 在 hackathon 创建时已配置较短时长
- **操作**: 不触发任何 UI 操作；等待 cron 切换
- **验证**:
  - 状态自动从 `Hacking(2)` 变为 `Judging(3)`
  - 不再接受新提交（验证：尝试 POST /submissions 应返回错误）
  - Feed 事件: `hackathon_phase_changed(50)`
- **截图**: `screenshots/full-cycle/p7-01-judging-started.png`
- **设计说明**: 同 Step 3.1
```

## Task 25: Remove dead-routes section from gaps.md

**Files:**
- Modify: `docs/skills/hackforger-api/gaps.md`

- [ ] **Step 25.1: Delete the entire `## Dead routes (clean-up, not parity gaps)` section through the end of the file (or up to the next `##` heading if more sections exist)**

After this edit, gaps.md should end with the "How to consume this list" section AND a brand-new closing line:

```markdown
> All listed gaps and dead-route cleanup items are closed as of 2026-04-20 (PR #73).
```

## Task 26: Final regression sweep

- [ ] **Step 26.1: Build + run all hackforger tests one more time**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -3
go test -tags "sqlite sqlite_unlock_notify" -run "TestAPIHackforger\|TestHackforger" ./tests/integration/... -count=1 2>&1 | tail -10
```
Expected: build succeeds, all hackforger tests pass.

## Task 27: Commit 5

- [ ] **Step 27.1: Stage and commit**

```bash
git add routers/web/web.go \
        docs/tests/e2e/tasks/full-cycle/01-hackathon-lifecycle.md \
        docs/tests/e2e/tasks/full-cycle/04-submission-judging.md \
        docs/skills/hackforger-api/gaps.md

# Plus any template/frontend files that step 21.1 surfaced
git status   # eyeball before commit

git commit -m "$(cat <<'EOF'
chore: remove dead phase-transition routes + revise E2E phase steps

Removes the /hackathon/{slug}/manage/start and /manage/judge web
routes — ManagePhasePost only handles publish/cancel, so these
routes flashed "unknown_action" on every click. Phase advancement
is owned by the hackforger_hackathon_status cron (@every 5min) +
gocron phase-event scheduling.

Updates E2E full-cycle steps 3.1 and 7.1 to wait for cron rather
than describe non-functional buttons. Closes the dead-routes
backlog in docs/skills/hackforger-api/gaps.md.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

## Task 28: Push + update PR

- [ ] **Step 28.1: Push the 4 new commits**

```bash
git push origin feat/hackforger-api-skill
```

- [ ] **Step 28.2: Append a comment to PR #73 summarising the additional commits**

```bash
gh pr comment 73 --body "$(cat <<'EOF'
Expanded scope per @allen.woods: closed all 4 API parity gaps in this PR.

New commits on top of the original docs commit:
1. feat(api): close 4 hackforger API parity gaps
2. test(api): integration coverage for new hackforger gap-fill endpoints
3. docs(skills): sync hackforger-api references with new endpoints
4. chore: remove dead phase-transition routes + revise E2E phase steps

`docs/skills/hackforger-api/gaps.md` is now empty backlog as of 2026-04-20.
EOF
)"
```

---

## Self-review checklist (run before declaring done)

- [ ] All 4 endpoints have a passing happy-path integration test
- [ ] At least one auth-failure test per protected endpoint
- [ ] `gaps.md` says "Backlog is empty"
- [ ] No reference to `/manage/start` or `/manage/judge` remains in code or docs
- [ ] PR #73 has 5 commits visible (1 original + 4 new)
- [ ] `make backend` succeeds with no warnings on the final commit
