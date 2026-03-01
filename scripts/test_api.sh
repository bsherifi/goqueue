#!/bin/bash
# Test script for GoQueue API.
# Usage: Start the server first (go run . --workers 2), then run this script.

BASE="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
PASS=0
FAIL=0

check() {
    local name="$1"
    local expected="$2"
    local response="$3"

    if echo "$response" | grep -q "$expected"; then
        echo -e "  ${GREEN}PASS${NC} $name"
        PASS=$((PASS + 1))
    else
        echo -e "  ${RED}FAIL${NC} $name"
        echo "    Expected to contain: $expected"
        echo "    Got: $response"
        FAIL=$((FAIL + 1))
    fi
}

echo -e "${BLUE}--- Health ---${NC}"
RESP=$(curl -s "$BASE/api/health")
check "GET /api/health" '"status":"ok"' "$RESP"

echo -e "\n${BLUE}--- Submit jobs ---${NC}"
RESP=$(curl -s -X POST "$BASE/api/jobs" -d '{"type":"email","payload":{"to":"test@test.com"}}')
check "POST /api/jobs (email)" '"type":"email"' "$RESP"
JOB_ID=$(echo "$RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

RESP=$(curl -s -X POST "$BASE/api/jobs" -d '{"type":"sms","payload":{"to":"+1234"}}')
check "POST /api/jobs (sms)" '"type":"sms"' "$RESP"

RESP=$(curl -s -X POST "$BASE/api/jobs" -d '{"type":"report"}')
check "POST /api/jobs (no payload)" '"type":"report"' "$RESP"

RESP=$(curl -s -X POST "$BASE/api/jobs" -d '{}')
check "POST /api/jobs (missing type)" '"error"' "$RESP"

echo -e "\n${BLUE}--- Get job ---${NC}"
RESP=$(curl -s "$BASE/api/jobs/$JOB_ID")
check "GET /api/jobs/{id}" "$JOB_ID" "$RESP"

echo -e "\n${BLUE}--- List jobs ---${NC}"
RESP=$(curl -s "$BASE/api/jobs")
check "GET /api/jobs" '"data":\[' "$RESP"

echo -e "\n${BLUE}--- Stats ---${NC}"
sleep 4  # Wait for workers to process jobs
RESP=$(curl -s "$BASE/api/stats")
check "GET /api/stats" '"total":' "$RESP"

echo -e "\n${BLUE}--- Filter by status ---${NC}"
RESP=$(curl -s "$BASE/api/jobs?status=completed")
check "GET /api/jobs?status=completed" '"completed"' "$RESP"

echo -e "\n${BLUE}--- Submit and cancel ---${NC}"
# Submit several jobs to ensure at least one stays pending
for i in 1 2 3 4 5; do
    RESP=$(curl -s -X POST "$BASE/api/jobs" -d "{\"type\":\"cancel_test_$i\",\"payload\":{}}")
done
CANCEL_ID=$(echo "$RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
RESP=$(curl -s -X DELETE "$BASE/api/jobs/$CANCEL_ID")
check "DELETE /api/jobs/{id}" '"cancelled"\|"error"' "$RESP"

echo -e "\n${BLUE}--- Retry ---${NC}"
# Wait for jobs to finish, then find a failed one or test error case
sleep 5
RESP=$(curl -s "$BASE/api/jobs?status=failed")
FAILED_ID=$(echo "$RESP" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$FAILED_ID" ]; then
    RESP=$(curl -s -X POST "$BASE/api/jobs/$FAILED_ID/retry")
    check "POST /api/jobs/{id}/retry" '"pending"\|"running"' "$RESP"
else
    echo -e "  ${BLUE}SKIP${NC} No failed jobs to retry"
fi

echo -e "\n${BLUE}--- 404 / errors ---${NC}"
RESP=$(curl -s "$BASE/api/jobs/nonexistent")
check "GET /api/jobs/nonexistent (404)" '"error"' "$RESP"

RESP=$(curl -s -X POST "$BASE/api/jobs/nonexistent/retry")
check "POST retry nonexistent (error)" '"error"' "$RESP"

echo ""
echo -e "Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}"
