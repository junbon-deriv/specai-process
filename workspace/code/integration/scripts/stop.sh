#!/usr/bin/env bash
#
# stop.sh - Stop all services for Double Rise/Fall Pricing Service
#
# Usage:
#   ./scripts/stop.sh              # Stop services (keep volumes)
#   ./scripts/stop.sh --volumes    # Stop and remove volumes
#   ./scripts/stop.sh --force      # Force stop (immediate)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Default options
VOLUMES=false
FORCE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --volumes|-v)
            VOLUMES=true
            shift
            ;;
        --force|-f)
            FORCE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --volumes, -v  Remove volumes (persistent data)"
            echo "  --force, -f    Force stop containers immediately"
            echo "  --help, -h     Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "🛑 Stopping Double Rise/Fall Pricing Service integration..."
echo ""

COMPOSE_CMD="docker compose"

# Build down command
DOWN_ARGS=""

if [[ "$VOLUMES" == "true" ]]; then
    DOWN_ARGS="$DOWN_ARGS -v"
    echo "⚠️  Volumes will be removed (data will be lost)"
fi

if [[ "$FORCE" == "true" ]]; then
    echo "⚡ Forcing immediate stop..."
    $COMPOSE_CMD kill
fi

# Stop services
$COMPOSE_CMD down $DOWN_ARGS

echo ""
echo "✅ Services stopped successfully!"

if [[ "$VOLUMES" == "true" ]]; then
    echo "   Volumes have been removed."
fi
