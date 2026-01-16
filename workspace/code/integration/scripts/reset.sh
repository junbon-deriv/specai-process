#!/usr/bin/env bash
#
# reset.sh - Reset and restart all services
#
# Usage:
#   ./scripts/reset.sh           # Reset and restart services
#   ./scripts/reset.sh --rebuild # Rebuild images before restart

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Default options
REBUILD=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --rebuild|-r)
            REBUILD=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --rebuild, -r  Rebuild images before restart"
            echo "  --help, -h     Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "🔄 Resetting Double Rise/Fall Pricing Service integration..."
echo ""

# Stop all services and remove volumes
echo "🛑 Stopping services and removing volumes..."
./scripts/stop.sh --volumes

echo ""

# Start services
if [[ "$REBUILD" == "true" ]]; then
    echo "🔨 Rebuilding and starting services..."
    ./scripts/start.sh --build
else
    echo "🚀 Starting services..."
    ./scripts/start.sh
fi

echo ""

# Wait for health
echo "⏳ Waiting for services to be healthy..."
./scripts/health-check.sh --wait

echo ""
echo "✅ Reset complete!"
