#!/usr/bin/env bash
#
# logs.sh - View logs from all services
#
# Usage:
#   ./scripts/logs.sh                     # Follow all logs
#   ./scripts/logs.sh --service NAME      # Follow specific service logs
#   ./scripts/logs.sh --tail 100          # Show last 100 lines
#   ./scripts/logs.sh --no-follow         # Show logs without following

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Default options
SERVICE=""
TAIL="100"
FOLLOW=true

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --service|-s)
            SERVICE="$2"
            shift 2
            ;;
        --tail|-t)
            TAIL="$2"
            shift 2
            ;;
        --no-follow|-n)
            FOLLOW=false
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --service, -s NAME   Show logs for specific service"
            echo "  --tail, -t N         Show last N lines (default: 100)"
            echo "  --no-follow, -n      Don't follow logs"
            echo "  --help, -h           Show this help message"
            echo ""
            echo "Available services:"
            echo "  - service-feed"
            echo "  - service-pricer-doublerisefall"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

COMPOSE_CMD="docker compose"

# Build log command
LOG_ARGS="--tail $TAIL"

if [[ "$FOLLOW" == "true" ]]; then
    LOG_ARGS="$LOG_ARGS -f"
fi

echo "📋 Viewing logs..."
echo "   Press Ctrl+C to stop"
echo ""

if [[ -n "$SERVICE" ]]; then
    $COMPOSE_CMD logs $LOG_ARGS "$SERVICE"
else
    $COMPOSE_CMD logs $LOG_ARGS
fi
