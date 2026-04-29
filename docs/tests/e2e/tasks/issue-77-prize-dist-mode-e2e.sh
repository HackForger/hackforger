#!/bin/bash
# E2E smoke: verify issue #77 fix on local instance using admin basic auth
set -euo pipefail
BASE="http://localhost:3000/api/v1"
AUTH="-u hackforger:admin1234"
JSON="Content-Type: application/json"

echo "=== Step 1: Verify migration ran on track 121 ==="
TRACK121_MODE=$(curl -s $AUTH "$BASE/hackforger/hackathons/93/tracks" | jq -r '.[] | select(.ID==121) | .PrizeDistMode')
echo "track 121 mode: $TRACK121_MODE"
[ "$TRACK121_MODE" = "winner_takes_all" ] || { echo "FAIL: migration didn't run"; exit 1; }

echo
echo "=== Step 2: Find an org owned by hackforger admin ==="
ORG_ID=$(curl -s $AUTH "$BASE/users/hackforger/orgs" | jq -r '.[0].id // empty')
echo "org_id: $ORG_ID"
[ -n "$ORG_ID" ] || { echo "FAIL: no org found"; exit 1; }

echo
echo "=== Step 3: Create test hackathon via API ==="
SLUG="issue-77-smoke-$(date +%s)"
HACK_RESP=$(curl -s -X POST $AUTH -H "$JSON" "$BASE/hackforger/hackathons" \
  -d "{\"name\":\"Issue 77 Smoke\",\"slug\":\"$SLUG\",\"org_id\":$ORG_ID}")
HACK_ID=$(echo "$HACK_RESP" | jq -r '.id // .ID // empty')
echo "created hackathon: $HACK_ID (slug=$SLUG)"
[ -n "$HACK_ID" ] || { echo "FAIL: create hackathon"; echo "$HACK_RESP"; exit 1; }

echo
echo "=== Step 4: Create track WITHOUT prize_dist_mode (THE BUG CONDITION) ==="
TRACK_RESP=$(curl -s -X POST $AUTH -H "$JSON" "$BASE/hackforger/hackathons/$HACK_ID/tracks" \
  -d '{"name":"Smoke Track","prize_credits":100}')
TRACK_ID=$(echo "$TRACK_RESP" | jq -r '.ID // empty')
TRACK_MODE=$(echo "$TRACK_RESP" | jq -r '.PrizeDistMode // empty')
echo "track id=$TRACK_ID mode=$TRACK_MODE"
[ "$TRACK_MODE" = "winner_takes_all" ] || { echo "FAIL: API did not default mode"; echo "$TRACK_RESP"; exit 1; }

echo
echo "=== Step 5: Create track with INVALID mode (expect 400) ==="
INVALID_STATUS=$(curl -s $AUTH -o /tmp/invalid.json -w "%{http_code}" -X POST -H "$JSON" \
  "$BASE/hackforger/hackathons/$HACK_ID/tracks" \
  -d '{"name":"Bad Track","prize_credits":1,"prize_dist_mode":"definitely_not_a_mode"}')
echo "invalid mode status: $INVALID_STATUS"
[ "$INVALID_STATUS" = "400" ] || { echo "FAIL: expected 400, got $INVALID_STATUS"; cat /tmp/invalid.json; exit 1; }

echo
echo "=== Step 6: Create track tiered with malformed JSON (expect 400) ==="
MALFORMED_STATUS=$(curl -s $AUTH -o /tmp/malformed.json -w "%{http_code}" -X POST -H "$JSON" \
  "$BASE/hackforger/hackathons/$HACK_ID/tracks" \
  -d '{"name":"Bad JSON","prize_credits":1,"prize_dist_mode":"tiered","prize_dist_ratios":"not even json"}')
echo "malformed JSON status: $MALFORMED_STATUS"
[ "$MALFORMED_STATUS" = "400" ] || { echo "FAIL: expected 400, got $MALFORMED_STATUS"; cat /tmp/malformed.json; exit 1; }

echo
echo "=== Step 7: Create track tiered with bad ratios sum (expect 400) ==="
BADRATIO_STATUS=$(curl -s $AUTH -o /tmp/badratio.json -w "%{http_code}" -X POST -H "$JSON" \
  "$BASE/hackforger/hackathons/$HACK_ID/tracks" \
  -d '{"name":"Bad Ratio","prize_credits":1,"prize_dist_mode":"tiered","prize_dist_ratios":"[{\"rank\":1,\"pct\":50},{\"rank\":2,\"pct\":40}]"}')
echo "bad ratios status: $BADRATIO_STATUS"
[ "$BADRATIO_STATUS" = "400" ] || { echo "FAIL: expected 400, got $BADRATIO_STATUS"; cat /tmp/badratio.json; exit 1; }

echo
echo "=== Step 8: Cleanup ==="
curl -s $AUTH -X DELETE "$BASE/hackforger/hackathons/$HACK_ID" > /dev/null && echo "deleted hackathon $HACK_ID"

echo
echo "ALL SMOKE TESTS PASSED ✓"
