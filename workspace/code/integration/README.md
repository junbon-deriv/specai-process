# Double Rise/Fall Pricing Service - Integration Environment

This directory contains the Docker Compose setup for local development and integration testing of the `service-pricer-doublerisefall` service.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Integration Environment                       │
│                                                                  │
│  ┌──────────────────────┐      ┌──────────────────────────────┐│
│  │   service-feed       │      │ service-pricer-doublerisefall ││
│  │   (mock)             │◄────▶│                               ││
│  │                      │ gRPC │   gRPC: 50051                 ││
│  │   Port: 50051        │      │   HTTP: 8081 (/health,/ready) ││
│  └──────────────────────┘      └──────────────────────────────┘│
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- **Docker**: 20.10+ ([Install Docker](https://docs.docker.com/get-docker/))
- **Docker Compose**: V2 (included with Docker Desktop)
- **curl**: For health checks (usually pre-installed)
- **grpcurl** (optional): For gRPC API testing

### Installing grpcurl (Optional)

```bash
# macOS
brew install grpcurl

# Go install
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

## Quick Start

```bash
# 1. Navigate to integration directory
cd workspace/code/integration

# 2. Start all services
./scripts/start.sh

# 3. Check health status
./scripts/health-check.sh

# 4. View logs
./scripts/logs.sh

# 5. Run integration tests
./scripts/test.sh

# 6. Stop services
./scripts/stop.sh
```

## Service Configuration

### Service: service-pricer-doublerisefall

| Port | Protocol | Purpose |
|------|----------|---------|
| 50051 | gRPC | Main service API |
| 8081 | HTTP | Health check endpoints |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FEED_SERVICE_ADDR` | `service-feed:50051` | Address of service-feed |
| `CONFIG_PATH` | `/app/config/symbols.yml` | Symbol configuration path |
| `GRPC_PORT` | `50051` | gRPC server port |
| `HEALTH_PORT` | `8081` | HTTP health port |
| `LOG_LEVEL` | `info` | Logging level |

### Dependency: service-feed (Mock)

In this integration setup, `service-feed` is mocked with a placeholder container. For full integration testing, replace with the actual service-feed image.

## Available Scripts

| Script | Description |
|--------|-------------|
| `./scripts/start.sh` | Start all services |
| `./scripts/stop.sh` | Stop all services |
| `./scripts/logs.sh` | View service logs |
| `./scripts/health-check.sh` | Check service health |
| `./scripts/reset.sh` | Reset and restart all services |
| `./scripts/test.sh` | Run integration tests |

### Script Options

#### start.sh
```bash
./scripts/start.sh              # Start in background
./scripts/start.sh --build      # Rebuild images first
./scripts/start.sh --attach     # Start with logs attached
```

#### stop.sh
```bash
./scripts/stop.sh               # Stop services
./scripts/stop.sh --volumes     # Stop and remove volumes
./scripts/stop.sh --force       # Force immediate stop
```

#### logs.sh
```bash
./scripts/logs.sh                           # All logs
./scripts/logs.sh --service service-pricer-doublerisefall  # Specific service
./scripts/logs.sh --tail 50                 # Last 50 lines
./scripts/logs.sh --no-follow               # No live follow
```

#### health-check.sh
```bash
./scripts/health-check.sh           # Check once
./scripts/health-check.sh --wait    # Wait until healthy
```

## Health Endpoints

| Endpoint | URL | Expected Response |
|----------|-----|-------------------|
| Health | `http://localhost:8081/health` | `{"status": "healthy"}` |
| Ready | `http://localhost:8081/ready` | `{"status": "ready"}` |

### Testing Health Endpoints

```bash
# Health check
curl http://localhost:8081/health

# Readiness check
curl http://localhost:8081/ready
```

## gRPC API Testing

### Using grpcurl

```bash
# List available services
grpcurl -plaintext localhost:50051 list

# List methods for a service
grpcurl -plaintext localhost:50051 list doublerisefall.v1.DoubleRiseFallService

# Call GetAsk method
grpcurl -plaintext -d '{
  "option_parameters": {
    "symbol": "R_100",
    "contract_type": "RISE",
    "currency": "USD",
    "first_duration": "1m",
    "second_duration": "2m",
    "stake": "10.00"
  }
}' localhost:50051 doublerisefall.v1.DoubleRiseFallService/GetAsk
```

## Troubleshooting

### Services Won't Start

```bash
# Check Docker is running
docker info

# Check for port conflicts
lsof -i :50051
lsof -i :8081

# View detailed logs
./scripts/logs.sh --no-follow
```

### Health Check Fails

```bash
# Check container status
docker compose ps

# Check specific service logs
./scripts/logs.sh --service service-pricer-doublerisefall

# Restart services
./scripts/reset.sh
```

### gRPC Connection Refused

```bash
# Verify port is open
nc -zv localhost 50051

# Check container networking
docker compose exec service-pricer-doublerisefall netstat -tlnp
```

### Build Fails

```bash
# Rebuild with no cache
docker compose build --no-cache

# Check Dockerfile
cat ../service-pricer-doublerisefall/Dockerfile
```

## Development Workflow

### Making Code Changes

1. Make changes to service code in `../service-pricer-doublerisefall/`
2. Rebuild and restart:
   ```bash
   ./scripts/start.sh --build
   ```
3. Run tests:
   ```bash
   ./scripts/test.sh
   ```

### Debugging

1. Start with logs attached:
   ```bash
   ./scripts/start.sh --attach
   ```
2. Or connect to running container:
   ```bash
   docker compose exec service-pricer-doublerisefall sh
   ```

### Full Reset

```bash
# Complete reset (removes all data)
./scripts/reset.sh --rebuild
```

## Production Notes

### Replacing Mock service-feed

For production integration testing, update `docker-compose.yml`:

```yaml
service-feed:
  image: regentmarkets/service-feed:latest
  environment:
    - DATABASE_URL=postgresql://...
  ports:
    - "50052:50051"
```

### Kubernetes Deployment

The service is designed for Kubernetes deployment with:
- gRPC health checking via `grpc.health.v1.Health/Check`
- HTTP health endpoints for readiness/liveness probes
- Stateless design for horizontal scaling

Example Kubernetes probes:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8081
  initialDelaySeconds: 15
  periodSeconds: 20

readinessProbe:
  httpGet:
    path: /ready
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
```

## File Structure

```
integration/
├── docker-compose.yml      # Main orchestration file
├── .env.example            # Environment template
├── .gitignore              # Git ignore rules
├── README.md               # This documentation
├── scripts/
│   ├── start.sh            # Start services
│   ├── stop.sh             # Stop services
│   ├── logs.sh             # View logs
│   ├── health-check.sh     # Health verification
│   ├── reset.sh            # Reset environment
│   └── test.sh             # Run tests
└── tests/
    └── health-checks.sh    # Automated health tests
```

## Support

For issues with:
- **Service code**: See `../service-pricer-doublerisefall/README.md`
- **Architecture**: See `../../output/architecture/architecture.md`
- **API specification**: See `../../output/api/service-pricer-doublerisefall_internal.md`
