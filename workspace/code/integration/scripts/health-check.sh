#!/usr/bin/env bash
#
# health-check.sh - Check health status of all services
#
# Usage:
#   ./scripts/health-check.sh           # Check all services
#   ./scripts/health-check.sh --wait    # Wait until all healthy

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Default options
WAIT=false
MAX_WAIT=120  # seconds
CHECK_INTERVAL=5

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --wait|-w)
            WAIT=true
            shift
            ;;
        --max-wait|-m)
            MAX_WAIT="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --wait, -w        Wait until all services are healthy"
            echo "  --max-wait, -m N  Maximum wait time in seconds (default: 120)"
            echo "  --help, -h        Show this help message"
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

COMPOSE_CMD="docker compose"

check_service_health() {
    local service=$1
    local container_status
    
    container_status=$($COMPOSE_CMD ps --format json "$service" 2>/dev/null | head -1 || echo "{}")
    
    if [[ -z "$container_status" ]] || [[ "$container_status" == "{}" ]]; then
        echo "not_running"
        return
    fi
    
    local state
    state=$(echo "$container_status" | grep -o '"State":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
    local health
    health=$(echo "$container_status" | grep -o '"Health":"[^"]*"' | cut -d'"' -f4 || echo "")
    
    if [[ "$state" != "running" ]]; then
        echo "not_running"
    elif [[ "$health" == "healthy" ]]; then
        echo "healthy"
    elif [[ "$health" == "unhealthy" ]]; then
        echo "unhealthy"
    elif [[ -z "$health" ]]; then
        echo "no_healthcheck"
    else
        echo "starting"
    fi
}

check_http_health() {
    local host=$1
    local port=$2
    local path=$3
    
    if curl -sf "http://${host}:${port}${path}" > /dev/null 2>&1; then
        echo "healthy"
    else
        echo "unhealthy"
    fi
}

print_status() {
    local service=$1
    local status=$2
    
    case $status in
        healthy)
            echo -e "  ${GREEN}✓${NC} $service: ${GREEN}healthy${NC}"
            ;;
        unhealthy)
            echo -e "  ${RED}✗${NC} $service: ${RED}unhealthy${NC}"
            ;;
        starting)
            echo -e "  ${YELLOW}○${NC} $service: ${YELLOW}starting${NC}"
            ;;
        not_running)
            echo -e "  ${RED}✗${NC} $service: ${RED}not running${NC}"
            ;;
        no_healthcheck)
            echo -e "  ${YELLOW}?${NC} $service: ${YELLOW}no healthcheck${NC}"
            ;;
        *)
            echo -e "  ${YELLOW}?${NC} $service: ${YELLOW}$status${NC}"
            ;;
    esac
}

all_healthy() {
    local feed_status
    local pricer_status
    
    feed_status=$(check_service_health "service-feed")
    pricer_status=$(check_service_health "service-pricer-doublerisefall")
    
    [[ "$feed_status" == "healthy" || "$feed_status" == "no_healthcheck" ]] && \
    [[ "$pricer_status" == "healthy" ]]
}

run_health_check() {
    echo "🏥 Health Check Results:"
    echo ""
    
    # Check Docker services
    local feed_status
    local pricer_status
    
    feed_status=$(check_service_health "service-feed")
    pricer_status=$(check_service_health "service-pricer-doublerisefall")
    
    print_status "service-feed (mock)" "$feed_status"
    print_status "service-pricer-doublerisefall" "$pricer_status"
    
    echo ""
    
    # Check HTTP health endpoints if service is running
    if [[ "$pricer_status" == "healthy" ]] || [[ "$pricer_status" == "no_healthcheck" ]]; then
        echo "🔗 HTTP Health Endpoints:"
        echo ""
        
        local http_health
        http_health=$(check_http_health "localhost" "8081" "/health")
        print_status "/health endpoint" "$http_health"
        
        local http_ready
        http_ready=$(check_http_health "localhost" "8081" "/ready")
        print_status "/ready endpoint" "$http_ready"
        
        echo ""
    fi
    
    # Return status
    if all_healthy; then
        echo -e "${GREEN}✓ All services are healthy!${NC}"
        return 0
    else
        echo -e "${RED}✗ Some services are not healthy${NC}"
        return 1
    fi
}

if [[ "$WAIT" == "true" ]]; then
    echo "⏳ Waiting for services to become healthy (max ${MAX_WAIT}s)..."
    echo ""
    
    elapsed=0
    while ! all_healthy; do
        if [[ $elapsed -ge $MAX_WAIT ]]; then
            echo -e "${RED}✗ Timeout waiting for services to become healthy${NC}"
            echo ""
            run_health_check
            exit 1
        fi
        
        echo "   Checking... (${elapsed}s elapsed)"
        sleep $CHECK_INTERVAL
        elapsed=$((elapsed + CHECK_INTERVAL))
    done
    
    echo ""
fi

run_health_check
