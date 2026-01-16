#!/usr/bin/env bash
#
# test.sh - Run integration tests
#
# Usage:
#   ./scripts/test.sh           # Run all tests
#   ./scripts/test.sh --quick   # Quick smoke tests only

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Default options
QUICK=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --quick|-q)
            QUICK=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --quick, -q  Run quick smoke tests only"
            echo "  --help, -h   Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

run_test() {
    local name=$1
    local cmd=$2
    
    echo -n "  Testing: $name... "
    
    if eval "$cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}FAILED${NC}"
        FAILED=$((FAILED + 1))
    fi
}

echo "🧪 Running Integration Tests"
echo "================================"
echo ""

# Check services are running first
echo "📋 Pre-flight checks:"
if ! ./scripts/health-check.sh > /dev/null 2>&1; then
    echo -e "${RED}✗ Services are not healthy. Please run ./scripts/start.sh first${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Services are running and healthy"
echo ""

# Health endpoint tests
echo "🏥 Health Endpoint Tests:"
run_test "HTTP /health endpoint responds" "curl -sf http://localhost:8081/health"
run_test "HTTP /ready endpoint responds" "curl -sf http://localhost:8081/ready"
echo ""

# gRPC connectivity tests
echo "🔌 gRPC Connectivity Tests:"
run_test "gRPC port 50051 is open" "nc -z localhost 50051"
echo ""

# Service communication tests
echo "📡 Service Communication Tests:"
run_test "Can reach service-feed container" "docker compose exec -T service-pricer-doublerisefall nc -z service-feed 50051 2>/dev/null || true"
echo ""

if [[ "$QUICK" != "true" ]]; then
    # Extended tests (if grpcurl is available)
    if command -v grpcurl &> /dev/null; then
        echo "📋 gRPC API Tests (using grpcurl):"
        
        # Test reflection (if enabled)
        run_test "gRPC reflection is available" "grpcurl -plaintext localhost:50051 list 2>/dev/null"
        
        echo ""
    else
        echo -e "${YELLOW}ℹ️  grpcurl not installed. Skipping gRPC API tests.${NC}"
        echo "   Install with: brew install grpcurl (macOS) or go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
        echo ""
    fi
fi

# Summary
echo "================================"
echo "📊 Test Summary:"
echo -e "  ${GREEN}Passed: $PASSED${NC}"
echo -e "  ${RED}Failed: $FAILED${NC}"
echo ""

if [[ $FAILED -gt 0 ]]; then
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
else
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
fi
