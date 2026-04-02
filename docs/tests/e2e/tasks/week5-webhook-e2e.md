# E2E Test: Webhook Dispatch for HackForger Events

> **Web 端测试是必选项，不可使用单元测试或 API 测试替代 Web 界面操作。关键验证节点必须使用 agent-browser 截屏记录作为测试证据。**

## Prerequisites
- HackForger running on http://localhost:3000
- A repo owned by hackforger user (e.g., hackforger/test-repo)
- Login: hackforger / admin1234

## Test Cases

### TC-W1: HackForger events visible in webhook form
1. Login as hackforger
2. Navigate to repo Settings → Webhooks → Add Webhook → Forgejo
3. Scroll to Events section
4. **Screenshot**: `docs/tests/e2e/screenshots/week5/webhook-events-form.png`
5. **Verify**: "HackForger Events" fieldset is visible with checkboxes for hackathon_created, bounty_created, grant_round_created, etc.

### TC-W2: Create webhook with HackForger events
1. Fill webhook URL: `https://webhook.site/<unique-id>` or `http://localhost:1234/webhook`
2. Select "Hackathon Created" and "Bounty Created" events
3. Click "Add Webhook"
4. **Screenshot**: `docs/tests/e2e/screenshots/week5/webhook-created.png`
5. **Verify**: Webhook appears in list with active status

### TC-W3: Trigger event and verify delivery
1. Create a hackathon via Web UI (or use API for data setup)
2. Navigate back to webhook settings → click on the webhook → Recent Deliveries
3. **Screenshot**: `docs/tests/e2e/screenshots/week5/webhook-delivery.png`
4. **Verify**: A delivery record exists with `hackathon_created` event
5. Click the delivery to see payload details
6. **Screenshot**: `docs/tests/e2e/screenshots/week5/webhook-payload.png`
7. **Verify**: Payload JSON contains `action`, `entity_type`, `entity_id`, `entity_name`, `sender` fields

### TC-W4: Event filtering works
1. Trigger a bounty creation event
2. Check webhook deliveries
3. **Verify**: Only subscribed events (hackathon_created, bounty_created) trigger deliveries
4. **Screenshot**: `docs/tests/e2e/screenshots/week5/webhook-filter.png`

## Report Template
Save report to: `docs/tests/e2e/week5-webhook-e2e-report.md`
