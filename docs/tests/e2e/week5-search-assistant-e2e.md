# E2E Test: ⌘K Search + AI Assistant Modal

> **Web 端测试是必选项，不可使用单元测试或 API 测试替代 Web 界面操作。关键验证节点必须使用 agent-browser 截屏记录作为测试证据。**

## Prerequisites
- HackForger running on http://localhost:3000
- Test data: at least 1 hackathon, 1 bounty, 1 grant round in database
- Test user: hackforger / admin1234

## Test Cases

### TC-S1: Open modal via keyboard shortcut
1. Login as hackforger via agent-browser
2. Navigate to any page (e.g., dashboard)
3. Press ⌘K (or Ctrl+K)
4. **Screenshot**: `docs/tests/e2e/screenshots/week5/search-modal-open.png`
5. **Verify**: Modal is visible, input is focused, scope tabs show "All/Hackathons/Bounties/Grants"

### TC-S2: Search with results
1. Type "Hackathon" in the search input
2. Wait 500ms for debounce
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/search-results.png`
4. **Verify**: Result items appear with type icons and status badges

### TC-S3: Scope filtering
1. Click "Bounties" tab
2. Type "bug" or "Fix" in input
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/search-scope-bounties.png`
4. **Verify**: Only bounty results appear (or "No results found" if no match)

### TC-S4: AI assistant panel
1. Type any query in the search input
2. Wait for results to load
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/search-assistant-panel.png`
4. **Verify**: Assistant panel is visible with placeholder message text (not empty), disclaimer text visible

### TC-S5: Navigate from result
1. Click on a search result
2. **Verify**: Page navigates to the entity detail page (hackathon detail, bounty, or grant)
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/search-navigate-result.png`

### TC-S6: Close modal
1. Press ⌘K to open, then Escape to close
2. **Verify**: Modal is hidden
3. Open again via ⌘K, click overlay area (outside dialog)
4. **Verify**: Modal is hidden

### TC-S7: Open via navbar icon
1. Click the search icon (🔍) in the navbar
2. **Verify**: Modal opens
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/search-navbar-trigger.png`

### TC-S8: Empty search
1. Open modal, type "xyznonexistent999"
2. **Screenshot**: `docs/tests/e2e/screenshots/week5/search-empty.png`
3. **Verify**: "No results found" message displayed

## Report Template
Save report to: `docs/tests/e2e/week5-search-e2e-report.md`
Include all screenshots as embedded images.
