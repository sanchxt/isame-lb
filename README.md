# Isame Load Balancer

A production-ready HTTP/HTTPS load balancer written in Go, featuring SSL/TLS termination, multiple load balancing algorithms, circuit breaker pattern, retry logic with exponential backoff, per-client rate limiting, health checking, and comprehensive metrics collection.

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

# run with TLS/HTTPS enabled (development)
./bin/isame-lb -config=configs/dev-tls.yaml

# run with defaults (no upstreams - for testing)
./bin/isame-lb
```

### Configuration

Create a YAML configuration file:

```yaml
version: "2.0.0"
service: "my-load-balancer"

server:
  port: 8080

upstreams:
  - name: "web-servers"
    algorithm: "weighted_round_robin"  # round_robin, weighted_round_robin, least_connections
    backends:
      - url: "http://localhost:3000"
        weight: 3  # Higher weight = more traffic
      - url: "http://localhost:3001"
        weight: 2
      - url: "http://localhost:3002"
        weight: 1
    rate_limit:  # Optional per-upstream rate limiting
      enabled: true
      requests_per_ip: 100
      window_size: "1m"

health:
  enabled: true
  interval: "30s"
  timeout: "5s"
  path: "/health"
  unhealthy_threshold: 3
  healthy_threshold: 2

metrics:
  enabled: true
  port: 9090
  path: "/metrics"

# Circuit breaker for failing backends
circuit_breaker:
  enabled: true
  failure_threshold: 5
  timeout: "60s"

# Automatic retry with exponential backoff
retry:
  enabled: true
  max_attempts: 3
  initial_backoff: "100ms"
  max_backoff: "2s"

# TLS/SSL termination (optional)
tls:
  enabled: true
  cert_file: "certs/dev/server.crt"
  key_file: "certs/dev/server.key"
  min_version: "1.2"  # "1.2" or "1.3"
```

### TLS/SSL Setup

For HTTPS support:

```bash
# Generate development certificates
cd certs/dev && ./generate.sh

# Start with TLS enabled
./bin/isame-lb -config=configs/dev-tls.yaml

# Test HTTPS endpoint (with self-signed cert)
curl -k https://localhost:8443/health
```

For production, use certificates from a trusted CA like Let's Encrypt. See [certs/README.md](certs/README.md) for details.

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

### Load Balancer

**HTTP (Port 8080 - default):**
- `GET /health` - Load balancer health check
- `GET /status` - Detailed status with backend health information
- `/*` - All other requests are proxied to backend servers

**HTTPS (Port 8443 - when TLS enabled):**
- Same endpoints as HTTP, accessed via HTTPS with TLS termination

### Metrics Server (Port 9090)

- `GET /metrics` - Prometheus metrics endpoint
- `GET /health` - Metrics server health check

### Examples

```bash
# check load balancer health (HTTP)
curl http://localhost:8080/health

# check load balancer health (HTTPS with self-signed cert)
curl -k https://localhost:8443/health

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
│   ├── server/          # HTTP/HTTPS server orchestration
│   ├── tls/             # TLS certificate management
│   ├── balancer/        # Load balancing algorithms (RR, WRR, LC)
│   ├── circuitbreaker/  # Circuit breaker pattern
│   ├── retry/           # Retry logic with exponential backoff
│   ├── ratelimit/       # Per-client sliding window rate limiter
│   ├── health/          # Active health checking
│   ├── metrics/         # Prometheus metrics collection
│   └── proxy/           # HTTP reverse proxy with feature integration
├── certs/               # TLS certificates
│   ├── dev/             # Development self-signed certificates
│   └── README.md        # Certificate management guide
├── configs/             # Example YAML configurations
│   ├── example.yaml     # Phase 3 feature showcase (with TLS)
│   ├── dev-tls.yaml     # Development with HTTPS enabled
│   └── dev.yaml         # Development configuration
└── docs/                # Documentation
```

## Features

### Security
- **TLS/SSL Termination**: HTTPS support with certificate management
- **Configurable TLS Versions**: TLS 1.2 and 1.3 support
- **Custom Cipher Suites**: Optional cipher suite configuration
- **Dual HTTP/HTTPS**: Run both protocols simultaneously

### Load Balancing Algorithms
- **Round-Robin**: Simple, equal distribution across backends
- **Weighted Round-Robin**: Smooth distribution respecting backend weights (nginx-style)
- **Least Connections**: Routes to backend with fewest active connections

### Reliability & Resilience
- **Circuit Breaker**: Automatically stops sending traffic to failing backends
- **Retry Logic**: Exponential backoff with jitter for transient failures
- **Health Checking**: Active monitoring with configurable thresholds

### Traffic Management
- **Rate Limiting**: Per-client sliding window with configurable limits per upstream
- **Request Routing**: Intelligent backend selection based on algorithm and health

### Observability
- **Prometheus Metrics**: Request counts, latencies, health status, connections
- **Detailed Status**: Real-time backend health and connection information
- **Comprehensive Logging**: All operations logged with context

## Architecture

The load balancer follows clean architecture principles:

- **Modular Design**: Each component is in its own package with clear interfaces
- **Configuration-Driven**: YAML-based configuration with validation
- **Health-Aware**: Automatic backend health monitoring and failover
- **Resilient**: Circuit breaker and retry logic for fault tolerance
- **Rate-Limited**: Per-client traffic control with sliding window
- **Observable**: Comprehensive Prometheus metrics
- **Production-Ready**: Graceful shutdown, proper error handling, logging
- **Thread-Safe**: All concurrent operations properly synchronized
- **Well-Tested**: 82.8% average test coverage with race detection

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
