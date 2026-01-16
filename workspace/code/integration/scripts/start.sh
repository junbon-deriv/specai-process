#!/usr/bin/env bash
#
# start.sh - Start all services for Double Rise/Fall Pricing Service
#
# Usage:
#   ./scripts/start.sh           # Start all services
#   ./scripts/start.sh --build   # Rebuild and start
#   ./scripts/start.sh --detach  # Start in background (default)
#   ./scripts/start.sh --attach  # Start with logs attached

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Default options
BUILD=false
DETACH=true

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --build|-b)
            BUILD=true
            shift
            ;;
        --attach|-a)
            DETACH=false
            shift
            ;;
        --detach|-d)
            DETACH=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --build, -b    Rebuild images before starting"
            echo "  --attach, -a   Start with logs attached (foreground)"
            echo "  --detach, -d   Start in background (default)"
            echo "  --help, -h     Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check if .env exists, if not copy from example
if [[ ! -f .env ]]; then
    echo "📋 No .env file found. Copying from .env.example..."
    cp .env.example .env
    echo "   Please review .env and customize if needed."
fi

echo "🚀 Starting Double Rise/Fall Pricing Service integration..."
echo ""

# Build command
COMPOSE_CMD="docker compose"

if [[ "$BUILD" == "true" ]]; then
    echo "🔨 Building images..."
    $COMPOSE_CMD build
    echo ""
fi

# Start services
if [[ "$DETACH" == "true" ]]; then
    echo "📦 Starting services in background..."
    $COMPOSE_CMD up -d
    echo ""
    echo "✅ Services started successfully!"
    echo ""
    echo "📊 Service Status:"
    $COMPOSE_CMD ps
    echo ""
    echo "💡 Useful commands:"
    echo "   ./scripts/logs.sh          - View logs"
    echo "   ./scripts/health-check.sh  - Check service health"
    echo "   ./scripts/stop.sh          - Stop services"
else
    echo "📦 Starting services with logs attached..."
    echo "   Press Ctrl+C to stop"
    echo ""
    $COMPOSE_CMD up
fi
