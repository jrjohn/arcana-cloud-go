#!/bin/bash

# Start Arcana Cloud Go in monolithic mode using Docker Compose

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

echo "Starting Arcana Cloud Go (Monolithic Mode)..."
echo "================================================"

# Check if .env exists
if [ ! -f .env ]; then
    echo "Creating .env from .env.example..."
    cp .env.example .env
    echo "Please update .env with your configuration, especially JWT_SECRET"
fi

# Build and start
docker-compose -f deployment/docker/docker-compose.yaml up --build -d

echo ""
echo "Services started successfully!"
echo ""
echo "Access points:"
echo "  - REST API:    http://localhost:8080"
echo "  - gRPC:        localhost:9090"
echo "  - Health:      http://localhost:8080/health"
echo ""
echo "View logs: docker-compose -f deployment/docker/docker-compose.yaml logs -f"
echo "Stop:      docker-compose -f deployment/docker/docker-compose.yaml down"
