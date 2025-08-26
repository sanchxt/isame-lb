# Isame Load Balancer

A high-performance HTTP load balancer written in Go, following a phased development approach.

## Current Status

🚧 **Phase 0 - Bootstrap** (Current)

- ✅ Project structure established
- ✅ Basic HTTP server with health endpoint
- ✅ Development tooling (Makefile, linting, testing)

## Quick Start

### Prerequisites

- Go 1.22 or later
- Make (optional, but recommended)

### Build and Run

```bash
# Build the project
make build

# Run the load balancer
make run

# Or run directly after building
./bin/isame-lb
```

The load balancer will start on port 8080 by default.

### Development

```bash
# Setup development environment
make dev-setup

# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

## API Endpoints

- `GET /health` - Health check endpoint (returns 200 OK)
- `GET /` - Basic service information

### Example

```bash
# Check health
curl http://localhost:8080/health

# Get service info
curl http://localhost:8080/
```

## Project Structure

```txt
├── cmd/
│   ├── isame-lb/        # Main load balancer server
│   └── isame-ctl/       # CLI tool (future phases)
├── internal/
│   ├── config/          # Configuration management
│   ├── server/          # HTTP server implementation
│   ├── balancer/        # Load balancing algorithms (Phase 1)
│   ├── health/          # Health checking (Phase 1)
│   └── metrics/         # Observability (Phase 1)
├── pkg/                 # Public interfaces
├── configs/             # Example configurations
├── deploy/docker/       # Docker deployment files
└── docs/                # Documentation
```

## Documentation

- [Implementation Plan](docs/IMPLEMENTATION_PLAN.md) - Development phases and roadmap
- [Project Structure](docs/PROJECT_STRUCTURE.md) - Detailed explanation of project structure and files

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/config
```

## Contributing

This project follows SOLID principles and emphasizes clean, testable code. See the development phases for planned features.

### Code Style

- Use `gofmt` for formatting
- Follow Go naming conventions
- Write tests for all new functionality
- Use golangci-lint for static analysis
