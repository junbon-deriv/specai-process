#!/usr/bin/env bash
#
# health-checks.sh - Automated health check verification tests
#
# This script verifies all health endpoints are working correctly.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

PASSED=0
FAILED=0

test_endpoint() {
    local name=$1
    local url=$2
    local expected_status=${3:-200}
    
    echo -n "  $name: "
    
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    
    if [[ "$status" == "$expected_status" ]]; then
        echo -e "${GREEN}OK${NC} (HTTP $status)"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC} (expected $expected_status, got $status)"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

echo "🏥 Health Check Tests"
echo "====================="
echo ""

# Test HTTP health endpoints
echo "HTTP Health Endpoints:"
test_endpoint "/health" "http://localhost:8081/health" 200 || true
test_endpoint "/ready" "http://localhost:8081/ready" 200 || true

echo ""
echo "====================="
echo -e "Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}"

exit $FAILED
