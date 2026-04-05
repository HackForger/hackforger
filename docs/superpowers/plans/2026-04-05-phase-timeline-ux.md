# Phase Timeline UX Enhancements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix timezone bug, simplify date picker, add dropdown disabled states, phase rename support, and drag-and-drop reorder to PhaseTimeline.vue

**Architecture:** All 4 enhancements modify PhaseTimeline.vue. #2 and #4 also need backend changes (model, router, migration, route).

**Tech Stack:** Vue 3 (Options API), Go, XORM, vuedraggable/SortableJS

---

### Task 1: Fix Timezone Bug + Simplify Date Picker (Frontend Only)

**Files:**
- Modify: `web_src/js/components/hackforger/PhaseTimeline.vue`

**Step 1: Replace datetime helpers**

In `PhaseTimeline.vue`, replace `toDatetimeLocal` and `fromDatetimeLocal` methods:

```js
toDateLocal(unix) {
  if (!unix) return '';
  const d = new Date(unix * 1000);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
},
fromDateLocal(str, isEnd) {
  if (!str) return 0;
  const dt = isEnd ? `${str}T23:59:00` : `${str}T00:00:00`;
  return Math.floor(new Date(dt).getTime() / 1000);
},
```

**Step 2: Update template inputs**

Change all `<input type="datetime-local">` to `<input type="date">`.

For the phase list:
```html
<input type="date"
       :value="toDateLocal(phase.start_time)"
       :disabled="phaseState(phase) !== 'future'"
       @change="updatePhase(phase, 'start', $event)">
```
```html
<input type="date"
       :value="toDateLocal(phase.end_time)"
       :disabled="phaseState(phase) === 'locked'"
       @change="updatePhase(phase, 'end', $event)">
```

For the add-phase form:
```html
<input type="date" v-model="newPhase.startTime" class="tw-text-sm">
<input type="date" v-model="newPhase.endTime" class="tw-text-sm">
```

**Step 3: Update addPhase method**

Replace `fromDatetimeLocal` calls with `fromDateLocal`:
```js
const startTime = this.fromDateLocal(this.newPhase.startTime, false);
const endTime = this.fromDateLocal(this.newPhase.endTime, true);
```

**Step 4: Update updatePhase method**

```js
async updatePhase(phase, field, event) {
  const val = this.fromDateLocal(event.target.value, field === 'end');
  if (!val) return;
  const startTime = field === 'start' ? val : phase.start_time;
  const endTime = field === 'end' ? val : phase.end_time;
```

**Step 5: Build frontend and verify**

Run: `make frontend`

**Step 6: Commit**

```bash
git add web_src/js/components/hackforger/PhaseTimeline.vue
git commit -m "fix(phase): use date picker with local timezone instead of datetime-local UTC"
```

---

### Task 2: Dropdown — Show All Types with Disabled State

**Files:**
- Modify: `web_src/js/components/hackforger/PhaseTimeline.vue`
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

**Step 1: Add i18n keys**

In `locale_en-US.ini` under `[hackforger]`:
```ini
hackforger.phase.already_added = (already added)
```

In `locale_zh-CN.ini`:
```ini
hackforger.phase.already_added = （已添加）
```

**Step 2: Pass i18n string from server**

In `routers/web/hackforger/hackathon.go` ManageHackathon, add to the `ptJSON` struct and list building:
```go
type ptJSON struct {
    ID              int64  `json:"id"`
    Key             string `json:"key"`
    DisplayName     string `json:"display_name"`
    IsUnique        bool   `json:"is_unique"`
    DefaultOrder    int    `json:"default_order"`
    AlreadyAddedTip string `json:"already_added_tip"`
}
```
And in the loop: `AlreadyAddedTip: ctx.Locale.TrString("hackforger.phase.already_added")`

Do the same in `routers/web/hackforger/phase.go` `phaseTypeJSON` struct and `ManagePhases` handler.

**Step 3: Replace availableTypes with allTypesWithState**

In `PhaseTimeline.vue`, replace the `availableTypes` computed:

```js
allTypesWithState() {
  return this.phaseTypes.map((pt) => {
    const alreadyAdded = pt.is_unique && this.phases.some((p) => p.phase_type_id === pt.id);
    return {
      ...pt,
      disabled: alreadyAdded,
      label: alreadyAdded
        ? `${pt.display_name} ${pt.already_added_tip || '(already added)'}`
        : pt.display_name,
    };
  });
},
```

**Step 4: Update template dropdown**

```html
<select v-model="newPhase.phaseTypeId" class="tw-text-sm">
  <option value="" disabled>Select phase type...</option>
  <option v-for="pt in allTypesWithState" :key="pt.id"
          :value="pt.id" :disabled="pt.disabled">
    {{ pt.label }}
  </option>
</select>
```

**Step 5: Build and verify**

Run: `make frontend`

**Step 6: Commit**

```bash
git add web_src/js/components/hackforger/PhaseTimeline.vue options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini routers/web/hackforger/hackathon.go routers/web/hackforger/phase.go
git commit -m "feat(phase): show disabled unique types in dropdown with explanation"
```

---

### Task 3: Phase Rename (custom_name)

**Files:**
- Create: `models/forgejo_migrations/v15_hackforger-phase-custom-name.go`
- Modify: `models/hackforger/phase.go`
- Modify: `routers/web/hackforger/phase.go`
- Modify: `web_src/js/components/hackforger/PhaseTimeline.vue`

**Step 1: Add migration**

Create `models/forgejo_migrations/v15_hackforger-phase-custom-name.go`:

```go
package forgejo_migrations

func init() {
    Register("v15_hackforger-phase-custom-name", "Add custom_name column to phase table", func(x *xorm.Engine) error {
        type Phase struct {
            CustomName string `xorm:"VARCHAR(100) NOT NULL DEFAULT ''"`
        }
        return x.Sync(new(Phase))
    })
}
```

**Step 2: Add field to Phase struct**

In `models/hackforger/phase.go`, add to Phase struct:
```go
CustomName string `xorm:"VARCHAR(100) NOT NULL DEFAULT ''" json:"custom_name"`
```

**Step 3: Update UpdatePhase model method**

Add `custom_name` to the Cols list:
```go
func UpdatePhase(ctx context.Context, p *Phase) error {
    _, err := db.GetEngine(ctx).ID(p.ID).Cols("start_time", "end_time", "sort_order", "custom_name", "updated_unix").Update(p)
    return err
}
```

**Step 4: Extend ManagePhasesUpdate request**

In `routers/web/hackforger/phase.go`, extend the `ManagePhasesUpdate` req struct:
```go
var req struct {
    StartTime  int64  `json:"start_time"`
    EndTime    int64  `json:"end_time"`
    CustomName string `json:"custom_name"`
}
```

After decoding, before `UpdatePhaseTime`, update the custom name if provided:
```go
if req.CustomName != "" || ctx.Req.ContentLength > 0 {
    phase, _ := hackforger_model.GetPhaseByID(ctx, phaseID)
    if phase != nil {
        phase.CustomName = req.CustomName
        phase.StartTime = req.StartTime
        phase.EndTime = req.EndTime
        _ = hackforger_model.UpdatePhase(ctx, phase)
    }
}
```

Actually, refactor: have `ManagePhasesUpdate` do full update (time + custom_name) instead of calling `UpdatePhaseTime` for just time. Keep the guards from UpdatePhaseTime (locked check, overlap check, active start locked) but apply them inline or extract to a validation function.

**Step 5: Update Vue component — display logic**

In `phaseTypeName` method:
```js
phaseTypeName(phase) {
  if (phase.custom_name) return phase.custom_name;
  const pt = this.phaseTypes.find((t) => t.id === phase.phase_type_id);
  return pt ? pt.display_name : `Phase #${phase.id}`;
},
```

**Step 6: Update Vue component — rename UI**

Add `editingPhaseId` to data:
```js
data() {
  return {
    ...
    editingPhaseId: null,
    editingName: '',
  };
},
```

Add methods:
```js
startRename(phase) {
  this.editingPhaseId = phase.id;
  this.editingName = phase.custom_name || this.phaseTypeName(phase);
},
async saveRename(phase) {
  this.editingPhaseId = null;
  if (this.editingName === this.phaseTypeName(phase) && !phase.custom_name) return;
  // Clear custom_name if user reverts to type name
  const pt = this.phaseTypes.find((t) => t.id === phase.phase_type_id);
  const customName = (pt && this.editingName === pt.display_name) ? '' : this.editingName;
  this.loading = true;
  try {
    const resp = await PUT(`/hackathon/${this.hackathonSlug}/manage/phases/${phase.id}`, {
      data: { start_time: phase.start_time, end_time: phase.end_time, custom_name: customName },
    });
    if (resp.ok) {
      const data = await resp.json();
      this.phases = data.phases || [];
    }
  } finally {
    this.loading = false;
  }
},
cancelRename() {
  this.editingPhaseId = null;
},
```

**Step 7: Update template — rename inline**

Replace the `<strong>` phase name element:
```html
<template v-if="editingPhaseId === phase.id">
  <input type="text" v-model="editingName" class="tw-text-sm tw-w-32"
         @blur="saveRename(phase)" @keyup.enter="saveRename(phase)" @keyup.escape="cancelRename"
         ref="renameInput" autofocus>
</template>
<template v-else>
  <strong class="tw-w-32 tw-flex tw-items-center tw-gap-1">
    {{ phaseTypeName(phase) }}
    <button v-if="phaseState(phase) !== 'locked'" class="tw-text-gray tw-cursor-pointer"
            @click="startRename(phase)" title="Rename">
      <svg class="svg octicon-pencil" width="14" height="14"><use xlink:href="#svg-node-octicon-pencil"></use></svg>
    </button>
  </strong>
</template>
```

**Step 8: Build backend and frontend**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
make frontend
```

**Step 9: Commit**

```bash
git add models/forgejo_migrations/v15_hackforger-phase-custom-name.go models/hackforger/phase.go routers/web/hackforger/phase.go web_src/js/components/hackforger/PhaseTimeline.vue
git commit -m "feat(phase): add custom_name field for phase renaming via pencil icon"
```

---

### Task 4: Drag-and-Drop Reorder

**Files:**
- Modify: `models/hackforger/phase.go` (add BatchUpdatePhaseSortOrder)
- Modify: `routers/web/hackforger/phase.go` (add ManagePhasesReorder)
- Modify: `routers/web/web.go` (add route)
- Modify: `web_src/js/components/hackforger/PhaseTimeline.vue`

**Step 1: Add batch update model method**

In `models/hackforger/phase.go`:

```go
// BatchUpdatePhaseSortOrder updates sort_order for multiple phases.
func BatchUpdatePhaseSortOrder(ctx context.Context, orders []struct{ ID int64; SortOrder int }) error {
    sess := db.GetEngine(ctx)
    for _, o := range orders {
        if _, err := sess.ID(o.ID).Cols("sort_order").Update(&Phase{SortOrder: o.SortOrder}); err != nil {
            return err
        }
    }
    return nil
}
```

**Step 2: Add reorder handler**

In `routers/web/hackforger/phase.go`:

```go
// ManagePhasesReorder reorders phases by updating sort_order.
func ManagePhasesReorder(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil {
        return
    }
    var req struct {
        Orders []struct {
            ID        int64 `json:"id"`
            SortOrder int   `json:"sort_order"`
        } `json:"orders"`
    }
    if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
        return
    }
    if err := hackforger_model.BatchUpdatePhaseSortOrder(ctx, req.Orders); err != nil {
        ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
        return
    }
    respondWithPhases(ctx, h.ID)
}
```

Note: the `BatchUpdatePhaseSortOrder` param type needs to match. Use a named type or adjust the model function signature to accept `[]struct{ ID int64; SortOrder int }` or define a proper type.

**Step 3: Add route**

In `routers/web/web.go`, in the hackathon manage phases group:
```go
m.Post("/phases/reorder", hackforger_web.ManagePhasesReorder)
```

Place BEFORE the `m.Put("/phases/{phase_id}", ...)` route to avoid path collision.

**Step 4: Update Vue — import and use vuedraggable**

In `PhaseTimeline.vue`, add import:
```js
import draggable from 'vuedraggable';
```

Add to component:
```js
components: { draggable },
```

**Step 5: Update Vue — drag handler**

Add method:
```js
async onDragEnd() {
  const orders = this.phases.map((p, i) => ({ id: p.id, sort_order: i + 1 }));
  this.loading = true;
  try {
    const resp = await POST(`/hackathon/${this.hackathonSlug}/manage/phases/reorder`, {
      data: { orders },
    });
    if (resp.ok) {
      const data = await resp.json();
      this.phases = data.phases || [];
    }
  } finally {
    this.loading = false;
  }
},
```

**Step 6: Update template — wrap in draggable**

Replace the `v-for` div with draggable:
```html
<draggable v-model="phases" item-key="id" handle=".drag-handle"
           :disabled="loading" @end="onDragEnd"
           :move="(evt) => phaseState(phases[evt.draggedContext.index]) !== 'locked'">
  <template #item="{ element: phase }">
    <div class="tw-flex tw-items-center tw-gap-3 tw-py-2 tw-border-b"
         :class="{'tw-opacity-50': phaseState(phase) === 'locked'}">
      <span v-if="phaseState(phase) !== 'locked'" class="drag-handle tw-cursor-grab tw-text-gray">&#x2807;</span>
      <span v-else class="tw-w-4"></span>
      <!-- rest of phase row content -->
    </div>
  </template>
</draggable>
```

Note: `sortedPhases` computed should be replaced with direct mutation of `phases` array for vuedraggable to work (it needs v-model on the array). Sort initially in `data()` or a watcher.

**Step 7: Adjust sortedPhases**

Remove `sortedPhases` computed. Instead, sort `this.phases` in-place when phases are loaded:
```js
sortPhases() {
  this.phases.sort((a, b) => a.sort_order - b.sort_order || a.start_time - b.start_time);
},
```

Call `sortPhases()` after every server response that updates `this.phases`.

Update template: replace `sortedPhases` references with `phases`.

**Step 8: Build backend and frontend**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
make frontend
```

**Step 9: Commit**

```bash
git add models/hackforger/phase.go routers/web/hackforger/phase.go routers/web/web.go web_src/js/components/hackforger/PhaseTimeline.vue
git commit -m "feat(phase): drag-and-drop reorder with vuedraggable"
```

---

### Task 5: Final Build + Smoke Test

**Step 1:** `make frontend && TAGS="bindata sqlite sqlite_unlock_notify" make backend`

**Step 2:** Restart server, navigate to manage page

**Step 3:** Verify:
- Date pickers show dates (no time), correct timezone
- Dropdown shows all 4 types, disabled ones have "(已添加)" label
- Pencil icon allows renaming phases
- Drag handle reorders phases
- All CRUD still works (add, update time, delete)

**Step 4:** Commit any fixes
