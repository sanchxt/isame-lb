# Isame Load Balancer

A HTTP load balancer written in Go, featuring YAML-based configuration, health checking, metrics collection, and graceful shutdown handling.

## Quick Start

### Prerequisites

- Go 1.22 or later
- Make (optional, but recommended)

### Build and Run

```bash
# Build the project
make build

# run with example configuration
./bin/isame-lb -config=configs/example.yaml

# run with development configuration
./bin/isame-lb -config=configs/dev.yaml

# run with defaults (no upstreams - for testing)
./bin/isame-lb
```

### Configuration

Create a YAML configuration file:

```yaml
version: "1.0.0"
service: "my-load-balancer"

server:
  port: 8080

upstreams:
  - name: "web-servers"
    algorithm: "round_robin"
    backends:
      - url: "http://localhost:3000"
        weight: 1
      - url: "http://localhost:3001"
        weight: 1

health:
  enabled: true
  interval: "30s"
  timeout: "5s"
  path: "/health"

metrics:
  enabled: true
  port: 9090
  path: "/metrics"
```

### Development

```bash
# setup development environment
make dev-setup

# run tests
make test

# run linter
make lint

# format code
make fmt
```

## API Endpoints

### Load Balancer (Port 8080)

- `GET /health` - Load balancer health check
- `GET /status` - Detailed status with backend health information
- `/*` - All other requests are proxied to backend servers

### Metrics Server (Port 9090)

- `GET /metrics` - Prometheus metrics endpoint
- `GET /health` - Metrics server health check

### Examples

```bash
# check load balancer health
curl http://localhost:8080/health

# get detailed status
curl http://localhost:8080/status

# view Prometheus metrics
curl http://localhost:9090/metrics

# send request through load balancer
curl http://localhost:8080/api/users
```

## Project Structure

```txt
├── cmd/
│   ├── isame-lb/        # Main load balancer server
│   └── isame-ctl/       # CLI tool (future phases)
├── internal/
│   ├── config/          # YAML configuration management
│   ├── server/          # HTTP server orchestration
│   ├── balancer/        # Load balancing algorithms
│   ├── health/          # Active health checking
│   ├── metrics/         # Prometheus metrics collection
│   └── proxy/           # HTTP reverse proxy
├── configs/             # Example YAML configurations
│   ├── example.yaml     # Full-featured example
│   └── dev.yaml         # Development configuration
└── docs/                # Documentation
```

## Architecture

The load balancer follows clean architecture principles:

- **Modular Design**: Each component is in its own package with clear interfaces
- **Configuration-Driven**: YAML-based configuration with validation
- **Health-Aware**: Automatic backend health monitoring and failover
- **Observable**: Comprehensive Prometheus metrics
- **Production-Ready**: Graceful shutdown, proper error handling, logging

## Documentation

- [Implementation Plan](docs/IMPLEMENTATION_PLAN.md) - Development phases and roadmap
- [Project Structure](docs/PROJECT_STRUCTURE.md) - Detailed explanation of project structure and files

## Testing

```bash
# run all tests
make test

# run tests with coverage
go test -cover ./...

# run tests for a specific package
go test ./internal/config
```

## Contributing

This project emphasizes clean, testable code. See the development phases for planned features.

### Code Style

- Use `gofmt` for formatting
- Follow Go naming conventions
- Write tests for all new functionality
- Use golangci-lint for static analysis
