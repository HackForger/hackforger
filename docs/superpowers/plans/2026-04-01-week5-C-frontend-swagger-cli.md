# Week 5 Sub-Plan C: Frontend + Swagger + CLI

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the ⌘K search modal, complete Swagger annotations, generate hackforger-cli, and create Claude Code skill.

**Architecture:** Modal is vanilla JS (not Vue SPA) injected via Go template. CLI generated from Swagger spec then hand-tuned with cobra. Skill wraps CLI for AI-native usage.

**Tech Stack:** Go templates, vanilla JS, CSS (hf-* design system), cobra CLI, openapi-generator, Swagger.

**Spec:** `docs/superpowers/specs/2026-04-01-week5-phase5-design.md` (Sections 5.5-5.6, 7)

**Depends on:** Sub-Plan B (search + assistant routes implemented, all API endpoints stable)

---

## Chunk 1: ⌘K Search Modal (C2)

### Task 1: Add search icon to navbar

**Files:**
- Modify: `templates/base/head_navbar.tmpl`

- [ ] **Step 1.1: Read current navbar template**

Read `templates/base/head_navbar.tmpl` to understand structure.

- [ ] **Step 1.2: Add search button before notification bell**

Insert before the notification `<a>` element (around line 115):

```html
<button class="item not-mobile tw-mx-0 hf-search-trigger" id="hf-search-trigger"
  data-tooltip-content="{{ctx.Locale.Tr "hackforger.search.placeholder"}}"
  aria-label="{{ctx.Locale.Tr "hackforger.search.placeholder"}}">
  {{svg "octicon-search"}}
</button>
```

This button appears for both signed-in and anonymous users (search is public).

- [ ] **Step 1.3: Include search modal template**

At the bottom of `head_navbar.tmpl` (before closing `</nav>`), add:

```html
{{template "hackforger/search_modal" .}}
```

- [ ] **Step 1.4: Commit**

```bash
git add templates/base/head_navbar.tmpl
git commit -m "feat(ui): add search icon to navbar for ⌘K modal trigger"
```

### Task 2: Create search modal template

**Files:**
- Create: `templates/hackforger/search_modal.tmpl`

- [ ] **Step 2.1: Write modal HTML**

```html
<div id="hf-search-modal" class="hf-search-modal" style="display:none">
  <div class="hf-search-overlay"></div>
  <div class="hf-search-dialog">
    <div class="hf-search-input-wrap">
      {{svg "octicon-search" 16 "hf-search-icon"}}
      <input id="hf-search-input" type="text"
        placeholder="{{ctx.Locale.Tr "hackforger.search.placeholder"}}"
        autocomplete="off" autofocus>
      <kbd class="hf-search-kbd">⌘K</kbd>
    </div>

    <div id="hf-assistant-panel" class="hf-search-assistant" style="display:none">
      <div id="hf-assistant-message"></div>
      <span id="hf-assistant-disclaimer" class="hf-search-disclaimer"></span>
    </div>

    <div class="hf-search-tabs">
      <button class="hf-search-tab active" data-scope="all">
        {{ctx.Locale.Tr "hackforger.search.scope.all"}}
      </button>
      <button class="hf-search-tab" data-scope="hackathons">
        {{ctx.Locale.Tr "hackforger.search.scope.hackathons"}}
      </button>
      <button class="hf-search-tab" data-scope="bounties">
        {{ctx.Locale.Tr "hackforger.search.scope.bounties"}}
      </button>
      <button class="hf-search-tab" data-scope="grants">
        {{ctx.Locale.Tr "hackforger.search.scope.grants"}}
      </button>
    </div>

    <div id="hf-search-results" class="hf-search-results"
      data-empty-text="{{ctx.Locale.Tr "hackforger.search.no_results"}}">
      <div class="hf-search-empty">
        {{ctx.Locale.Tr "hackforger.search.no_results"}}
      </div>
    </div>
  </div>
</div>
```

- [ ] **Step 2.2: Commit**

```bash
git add templates/hackforger/search_modal.tmpl
git commit -m "feat(ui): add search modal template with input, assistant panel, scope tabs, results"
```

### Task 3: Write search modal CSS

**Files:**
- Create: `web_src/css/features/hackforger/search-modal.css`
- Modify: CSS import entry point (check `web_src/css/index.css` or similar)

- [ ] **Step 3.1: Read the existing CSS import structure**

Check how HackForger CSS is currently imported (from PR #12 UI modernization).

- [ ] **Step 3.2: Write modal styles**

Follow the `hf-*` design system from PR #12 (River Gorge x Ink Splash palette):

```css
/* search-modal.css — ⌘K command palette */
.hf-search-modal {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 15vh;
}

.hf-search-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
}

.hf-search-dialog {
  position: relative;
  width: 100%;
  max-width: 640px;
  background: var(--color-body);
  border: 1px solid var(--color-secondary);
  border-radius: 12px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.2);
  overflow: hidden;
}

.hf-search-input-wrap {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-secondary);
  gap: 8px;
}

.hf-search-input-wrap input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 16px;
  background: transparent;
  color: var(--color-text);
}

.hf-search-kbd {
  padding: 2px 6px;
  font-size: 12px;
  border: 1px solid var(--color-secondary);
  border-radius: 4px;
  color: var(--color-text-light);
}

.hf-search-assistant {
  padding: 12px 16px;
  background: var(--color-box-body);
  border-bottom: 1px solid var(--color-secondary);
}

.hf-search-disclaimer {
  display: block;
  text-align: right;
  font-size: 12px;
  color: var(--color-text-light);
  margin-top: 4px;
}

.hf-search-tabs {
  display: flex;
  gap: 0;
  border-bottom: 1px solid var(--color-secondary);
}

.hf-search-tab {
  flex: 1;
  padding: 8px;
  text-align: center;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  color: var(--color-text-light);
}

.hf-search-tab.active {
  color: var(--color-primary);
  border-bottom: 2px solid var(--color-primary);
}

.hf-search-results {
  max-height: 400px;
  overflow-y: auto;
}

.hf-search-result-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  gap: 10px;
  cursor: pointer;
  border-bottom: 1px solid var(--color-secondary);
}

.hf-search-result-item:hover {
  background: var(--color-hover);
}

.hf-search-result-title {
  flex: 1;
  font-size: 14px;
}

.hf-search-result-badge {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
}

.hf-search-empty {
  padding: 24px;
  text-align: center;
  color: var(--color-text-light);
}
```

- [ ] **Step 3.3: Import the CSS**

Add import in the appropriate CSS entry point (follow existing pattern from PR #12).

- [ ] **Step 3.4: Commit**

```bash
git add web_src/css/features/hackforger/search-modal.css web_src/css/
git commit -m "feat(ui): search modal CSS — hf-* design system, command palette style"
```

### Task 4: Write search modal JS

**Files:**
- Create: `web_src/js/features/hackforger/search-modal.js`
- Modify: `web_src/js/features/hackforger/init.js`

- [ ] **Step 4.1: Write modal interaction logic**

```js
// search-modal.js — ⌘K search modal interaction
export function initSearchModal() {
  const modal = document.getElementById('hf-search-modal');
  const input = document.getElementById('hf-search-input');
  const trigger = document.getElementById('hf-search-trigger');
  const overlay = modal?.querySelector('.hf-search-overlay');
  const resultsContainer = document.getElementById('hf-search-results');
  const assistantPanel = document.getElementById('hf-assistant-panel');
  const assistantMessage = document.getElementById('hf-assistant-message');
  const assistantDisclaimer = document.getElementById('hf-assistant-disclaimer');
  const tabs = modal?.querySelectorAll('.hf-search-tab');

  if (!modal || !input) return;

  let debounceTimer = null;
  let currentScope = 'all';

  function openModal() {
    modal.style.display = 'flex';
    input.focus();
    input.value = '';
    resultsContainer.innerHTML = '<div class="hf-search-empty">' +
      resultsContainer.dataset.emptyText + '</div>';
    assistantPanel.style.display = 'none';
  }

  function closeModal() {
    modal.style.display = 'none';
    input.value = '';
  }

  function renderResults(results) {
    if (!results || results.length === 0) {
      resultsContainer.innerHTML = '<div class="hf-search-empty">' +
        resultsContainer.dataset.emptyText + '</div>';
      return;
    }

    const typeIcons = {
      hackathon: 'octicon-rocket',
      bounty: 'octicon-gift',
      grant: 'octicon-heart',
    };

    resultsContainer.innerHTML = results.map((r) => `
      <a href="${r.url || '#'}" class="hf-search-result-item">
        <svg class="octicon"><use href="#${typeIcons[r.type] || 'octicon-search'}"></use></svg>
        <span class="hf-search-result-title">${escapeHtml(r.title)}</span>
        <span class="hf-search-result-badge hf-badge">${escapeHtml(r.status)}</span>
      </a>
    `).join('');
  }

  async function doSearch(query) {
    if (!query.trim()) {
      resultsContainer.innerHTML = '<div class="hf-search-empty">' +
        resultsContainer.dataset.emptyText + '</div>';
      assistantPanel.style.display = 'none';
      return;
    }

    // Parallel fetch: search + assistant
    const [searchResp, assistantResp] = await Promise.all([
      fetch(`${window.config?.appSubUrl || ''}/hackforger/search?q=${encodeURIComponent(query)}&scope=${currentScope}`),
      fetch(`${window.config?.appSubUrl || ''}/hackforger/assistant/chat`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({query}),
      }).catch(() => null), // assistant failure is non-fatal
    ]);

    if (searchResp.ok) {
      const data = await searchResp.json();
      renderResults(data.results);
    }

    if (assistantResp?.ok) {
      const assistant = await assistantResp.json();
      assistantMessage.textContent = assistant.message;
      assistantDisclaimer.textContent = assistant.disclaimer;
      assistantPanel.style.display = 'block';
    }
  }

  // Trigger
  trigger?.addEventListener('click', openModal);
  overlay?.addEventListener('click', closeModal);

  // ⌘K / Ctrl+K
  document.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault();
      if (modal.style.display === 'none' || !modal.style.display) {
        openModal();
      } else {
        closeModal();
      }
    }
    if (e.key === 'Escape' && modal.style.display !== 'none') {
      closeModal();
    }
  });

  // Debounced search
  input.addEventListener('input', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => doSearch(input.value), 300);
  });

  // Scope tabs
  tabs?.forEach((tab) => {
    tab.addEventListener('click', () => {
      tabs.forEach((t) => t.classList.remove('active'));
      tab.classList.add('active');
      currentScope = tab.dataset.scope;
      if (input.value.trim()) {
        doSearch(input.value);
      }
    });
  });
}

function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}
```

- [ ] **Step 4.2: Register in init.js**

In `web_src/js/features/hackforger/init.js`, add:

```js
import {initSearchModal} from './search-modal.js';

// In the onDomReady callback:
initSearchModal();
```

- [ ] **Step 4.3: Build frontend**

Run: `make frontend`
Expected: webpack compiles successfully.

- [ ] **Step 4.4: Commit**

```bash
git add web_src/js/features/hackforger/search-modal.js web_src/js/features/hackforger/init.js
git commit -m "feat(ui): ⌘K search modal JS — debounced search, scope tabs, assistant panel"
```

### Task 5: Add search i18n keys

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 5.1: Add keys to en-US**

> **Note:** The `assistant.*` keys were already added in Plan B Task 7. Only **append** the `search.*` keys to the existing `[hackforger]` section. Do NOT overwrite the section.

Under `[hackforger]` section:
```ini
search.placeholder = Search or Ask something
search.scope.all = All
search.scope.hackathons = Hackathons
search.scope.bounties = Bounties
search.scope.grants = Grants
search.no_results = No results found
```

- [ ] **Step 5.2: Add keys to zh-CN**

```ini
search.placeholder = 搜索或提问
search.scope.all = 全部
search.scope.hackathons = 黑客松
search.scope.bounties = 悬赏
search.scope.grants = 资助
search.no_results = 未找到结果
```

- [ ] **Step 5.3: Commit**

```bash
git add options/locale/
git commit -m "i18n: add search + assistant keys for en-US and zh-CN"
```

### Task 6: Build and verify frontend

- [ ] **Step 6.1: Build both frontend and backend**

```bash
make frontend
TAGS="bindata sqlite sqlite_unlock_notify" make backend
```

- [ ] **Step 6.2: Verify no template parse errors**

Start server and check for panics:
```bash
./gitea web 2>&1 | head -20
```

- [ ] **Step 6.3: Commit if any fixes needed**

---

## Chunk 2: Swagger + CLI + Skill (C1, C3, C4)

### Task 7: Swagger annotation completion (C1)

**Files:**
- Modify: `routers/api/v1/hackforger/search.go` (add swagger annotation)
- Modify: `routers/api/v1/hackforger/assistant.go` (add swagger annotation)
- Modify: all other `routers/api/v1/hackforger/*.go` files (audit and fix)

- [ ] **Step 7.1: Audit existing swagger annotations**

Run: `grep -c "swagger:operation" routers/api/v1/hackforger/*.go`
List files with missing or incomplete annotations.

- [ ] **Step 7.2: Add swagger annotations to search.go**

```go
// swagger:operation GET /hackforger/search hackforger hackforgerSearch
// ---
// summary: Search HackForger entities
// produces:
// - application/json
// parameters:
// - name: q
//   in: query
//   description: Search keyword
//   type: string
//   required: true
// - name: scope
//   in: query
//   description: Search scope (all, hackathons, bounties, grants)
//   type: string
//   default: all
// - name: page
//   in: query
//   type: integer
// - name: limit
//   in: query
//   type: integer
// responses:
//   "200":
//     "$ref": "#/responses/HackforgerSearchResults"
```

- [ ] **Step 7.3: Add swagger annotations to assistant.go**

Similar pattern for `POST /hackforger/assistant/chat`.

- [ ] **Step 7.4: Fix any missing annotations in other files**

- [ ] **Step 7.5: Generate and verify swagger**

```bash
make swagger
make swagger-check
```

Expected: both commands succeed with no errors.

- [ ] **Step 7.6: Commit**

```bash
git add routers/api/v1/hackforger/ templates/swagger/
git commit -m "docs(swagger): complete HackForger API annotations — search, assistant, all endpoints"
```

### Task 8: Generate CLI client from Swagger (C3)

**Files:**
- Create: `cmd/hackforger-cli/main.go`
- Create: `cmd/hackforger-cli/root.go`
- Create: `cmd/hackforger-cli/hackathon.go`
- Create: `cmd/hackforger-cli/bounty.go`
- Create: `cmd/hackforger-cli/grant.go`
- Create: `cmd/hackforger-cli/credits.go`
- Create: `cmd/hackforger-cli/reputation.go`
- Create: `cmd/hackforger-cli/search.go`
- Create: `cmd/hackforger-cli/feed.go`
- Create: `cmd/hackforger-cli/assistant.go`
- Modify: `Makefile` (add hackforger-cli target)

- [ ] **Step 8.1: Generate Go client from swagger.json**

```bash
# Install openapi-generator if needed
# brew install openapi-generator

openapi-generator generate \
  -i ./public/swagger.json \
  -g go \
  -o ./internal/hackforger-client \
  --additional-properties=packageName=hackforgerclient
```

Review generated code. We only need the HackForger endpoints.

- [ ] **Step 8.2: Create CLI entry point**

`cmd/hackforger-cli/main.go`:
```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

`cmd/hackforger-cli/root.go`:
```go
package main

import (
	"github.com/spf13/cobra"
)

var (
	flagURL    string
	flagToken  string
	flagOutput string
)

var rootCmd = &cobra.Command{
	Use:   "hackforger-cli",
	Short: "CLI for HackForger platform",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "", "HackForger instance URL (env: HACKFORGER_URL)")
	rootCmd.PersistentFlags().StringVar(&flagToken, "token", "", "API token (env: HACKFORGER_TOKEN)")
	rootCmd.PersistentFlags().StringVar(&flagOutput, "output", "table", "Output format: json, table, yaml")
}
```

- [ ] **Step 8.3: Add resource commands (one per file)**

Each file adds a cobra command with subcommands. Example for search:

`cmd/hackforger-cli/search.go`:
```go
package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search HackForger entities",
}

var searchQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Search by keyword",
	RunE: func(cmd *cobra.Command, args []string) error {
		q, _ := cmd.Flags().GetString("q")
		scope, _ := cmd.Flags().GetString("scope")
		// Call API: GET /api/v1/hackforger/search?q=...&scope=...
		// Format and print results
		fmt.Printf("Searching for %q in scope %q...\n", q, scope)
		return nil
	},
}

func init() {
	searchQueryCmd.Flags().String("q", "", "Search keyword")
	searchQueryCmd.Flags().String("scope", "all", "Scope: all, hackathons, bounties, grants")
	searchCmd.AddCommand(searchQueryCmd)
	rootCmd.AddCommand(searchCmd)
}
```

Repeat for hackathon, bounty, grant, credits, reputation, feed, assistant commands.

- [ ] **Step 8.4: Add Makefile target**

```makefile
.PHONY: hackforger-cli
hackforger-cli:
	go build -o hackforger-cli ./cmd/hackforger-cli/
```

- [ ] **Step 8.5: Build and test**

```bash
make hackforger-cli
./hackforger-cli --help
./hackforger-cli search query --q "test" --url http://localhost:3000 --token $FORGEJO_TOKEN
```

- [ ] **Step 8.6: Commit**

```bash
git add cmd/hackforger-cli/ Makefile
git commit -m "feat: hackforger-cli — standalone Go CLI for all HackForger API endpoints"
```

### Task 9: Claude Code skill (C4)

**Files:**
- Create: `.claude/skills/hackforger-api.md`
- Modify: `.mcp.json`

- [ ] **Step 9.1: Create skill file**

`.claude/skills/hackforger-api.md`:
```markdown
---
name: hackforger-api
description: Interact with HackForger platform via hackforger-cli. Use when needing to query or manage hackathons, bounties, grants, credits, or search the platform.
---

## Usage

Use `hackforger-cli` to interact with the HackForger instance at `$HACKFORGER_URL`.

### Authentication
- URL: `$HACKFORGER_URL` (default: https://hackforger.inside.h2os.cloud)
- Token: `$FORGEJO_TOKEN`

### Common Commands

```bash
# Search
hackforger-cli search query --q "keyword" --scope all

# Hackathons
hackforger-cli hackathon list
hackforger-cli hackathon get <id>

# Bounties
hackforger-cli bounty list --status open
hackforger-cli bounty get --repo owner/repo --id <id>

# Grants
hackforger-cli grant round-list
hackforger-cli grant round-get <id>

# Credits
hackforger-cli credits balance

# Feed
hackforger-cli feed list --type global

# AI Assistant
hackforger-cli assistant chat --query "What bounties are available?"
```

### Output Formats
- `--output table` (default) — human-readable table
- `--output json` — raw JSON
- `--output yaml` — YAML format
```

- [ ] **Step 9.2: Update .mcp.json**

Remove non-functional `forgejo-mcp` entry:

```json
{
  "mcpServers": {
    "context7": {
      "command": "npx",
      "args": ["-y", "@anthropic/context7-mcp"]
    }
  }
}
```

- [ ] **Step 9.3: Commit**

```bash
git add .claude/skills/hackforger-api.md .mcp.json
git commit -m "feat: Claude Code skill for hackforger-cli + clean up .mcp.json"
```
