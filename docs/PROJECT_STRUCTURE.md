# Project Structure

This document provides a detailed explanation of the Isame Load Balancer project structure, including the purpose and functionality of each file and directory.

## Root Directory

### Configuration Files

**`go.mod`** - Go module file

- **Purpose**: Defines the Go module and its dependencies
- **Current State**: Module name `github.com/sanchxt/isame-lb` with Go version 1.24.6
- **Dependencies**:
  - `gopkg.in/yaml.v3 v3.0.1` - YAML configuration parsing and validation
  - `github.com/prometheus/client_golang v1.20.5` - Prometheus metrics collection and HTTP handler

**`Makefile`** - Build automation

- **Purpose**: Provides standardized build, test, and development commands
- **Key Targets**:
  - `build`: Compiles both `isame-lb` and `isame-ctl` binaries
  - `test`: Runs all tests with race condition detection
  - `lint`: Comprehensive linting with go vet, formatting checks, and staticcheck
  - `run`: Build and run the load balancer
  - `dev-setup`: Install development tools and dependencies
- **Build Output**: Creates binaries in `./bin/` directory

**`.golangci.yml`** - Linter configuration

- **Purpose**: Configures golangci-lint with comprehensive rules
- **Enabled Linters**: errcheck, gosimple, govet, ineffassign, staticcheck, typecheck, unused, gocyclo, gofmt, goimports, goconst, dupl, misspell, unparam, gosec, prealloc
- **Special Rules**: Excludes certain checks from test files, allows magic numbers in tests

**`.gitignore`** - Git ignore patterns

- **Purpose**: Excludes build artifacts and temporary files
- **Patterns**: `bin/` directory, `@*/` directories, `C*.md` files

**`LICENSE`** - Project license file

- **Purpose**: Defines the legal terms for using the project

## Source Code Structure

### `cmd/` - Application Entry Points

**`cmd/isame-lb/`** - Main load balancer server

- **`main.go`**

  - **Purpose**: Entry point for the load balancer server with configuration support
  - **Functionality**:
    - Command-line flag parsing for configuration file (`-config`)
    - YAML configuration loading with fallback to defaults
    - Integrated server initialization with all Phase 1 components
    - Proper error handling and logging
  - **Features**:
    - Supports both configuration file and default operation modes
    - Graceful shutdown handling
    - Production-ready error reporting
  - **Usage**: `./bin/isame-lb -config=configs/example.yaml`

- **`main_test.go`**
  - **Purpose**: Unit tests for main server handlers
  - **Coverage**: Tests both health and root handlers
  - **Test Types**: HTTP response validation, content-type checking

**`cmd/isame-ctl/`** - Command-line interface tool

- **`main.go`**
  - **Purpose**: CLI tool for managing the load balancer
  - **Current Commands**: `version`, `help`
  - **State**: Basic stub implementation for Phase 0
  - **Future**: Will be expanded in Phase 3 for configuration management

### `internal/` - Private Application Code

**`internal/config/`** - Configuration management

- **`config.go`**

  - **Purpose**: Comprehensive YAML configuration management with validation
  - **Structure**: Complete `Config` struct with all Phase 1 components:
    - `Version`, `Service`: Basic service metadata
    - `Server`: HTTP server configuration (port, timeouts, headers)
    - `Upstreams`: Backend server groups with load balancing algorithms
    - `Health`: Active health checking configuration
    - `Metrics`: Prometheus metrics server configuration
  - **Functions**:
    - `LoadConfig(path)`: Loads and validates YAML configuration from file
    - `LoadConfigWithDefaults(path)`: Loads config or returns defaults if file missing
    - `NewDefaultConfig()`: Creates default configuration for development
    - `Validate()`: Comprehensive validation with smart defaults
  - **Validation Features**: Port validation, timeout parsing, algorithm validation, backend URL validation
  - **Error Handling**: Detailed error messages with context for troubleshooting

- **`config_test.go`**
  - **Purpose**: Comprehensive configuration testing
  - **Test Coverage**: YAML loading, validation, defaults, error cases
  - **Test Strategy**: Table-driven tests covering all validation scenarios
  - **File Testing**: Tests both successful config loading and error conditions

**`internal/server/`** - HTTP server implementation

- **`server.go`**

  - **Purpose**: Load balancer server orchestration and lifecycle management
  - **Structure**: `LoadBalancerServer` struct orchestrating all Phase 1 components:
    - Configuration management
    - HTTP server with proper timeouts
    - Health checker integration
    - Metrics collection
    - HTTP reverse proxy handler
  - **Methods**:
    - `New()`: Creates integrated server with all components
    - `Start()`: Starts HTTP server, health checker, and metrics server
    - `Shutdown()`: Graceful shutdown with proper cleanup and timeouts
    - Handler methods for health (`/health`) and status (`/status`) endpoints
  - **Features**:
    - Signal-based graceful shutdown (SIGINT, SIGTERM)
    - Separate metrics server on configurable port
    - Component lifecycle management
    - Error handling and logging
  - **Architecture**: Clean separation with dependency injection

- **`server_test.go`**
  - **Purpose**: Server integration and lifecycle testing
  - **Test Coverage**: Server creation, endpoint handlers, graceful shutdown
  - **Integration Testing**: Tests component integration and HTTP routing

**`internal/balancer/`** - Load balancing algorithms

- **`balancer.go`**

  - **Purpose**: Load balancing interface and algorithm implementations
  - **Interface**: `LoadBalancer` with `SelectBackend()` method for pluggable algorithms
  - **Implementation**: `RoundRobin` struct with thread-safe atomic counter
  - **Features**:
    - Health-aware backend selection (skips unhealthy backends)
    - Thread-safe operation using `sync/atomic`
    - HTTP request-aware selection (future extensibility)
    - Comprehensive error handling for no backends/unhealthy backends
  - **Algorithm**: Round-robin with wraparound counter

- **`balancer_test.go`**
  - **Purpose**: Comprehensive testing for load balancing algorithms
  - **Test Coverage**: Algorithm creation, backend selection, health awareness, thread safety
  - **Test Strategy**: Table-driven tests with concurrent access validation

**`internal/health/`** - Health checking functionality

- **`checker.go`**

  - **Purpose**: Active health checking system for backend servers
  - **Structure**: `Checker` struct with configurable thresholds and intervals
  - **Features**:
    - Active HTTP health checks to configurable endpoints
    - Configurable healthy/unhealthy thresholds with consecutive failure counting
    - Goroutine-based concurrent health checking per backend
    - Thread-safe status tracking using `sync.RWMutex`
    - Graceful shutdown with proper cleanup
  - **Health Logic**: Tracks consecutive failures/successes against thresholds
  - **HTTP Client**: Configured with proper timeouts matching health config

- **`checker_test.go`**
  - **Purpose**: Unit tests for health checking functionality
  - **Test Coverage**: Checker creation, backend status tracking, HTTP health checks
  - **Mock Testing**: Uses `httptest.NewServer` for controlled health endpoint testing

**`internal/metrics/`** - Observability and metrics

- **`collector.go`**

  - **Purpose**: Prometheus metrics collection for observability
  - **Metrics Types**:
    - `requestsTotal`: Counter vector (method, status, upstream)
    - `requestDuration`: Histogram vector (method, upstream) with standard buckets
    - `upstreamHealthy`: Gauge vector per upstream backend
    - `connectionsActive`: Gauge for current active connections
  - **Features**:
    - Proper metric registration and initialization
    - HTTP handler for `/metrics` endpoint
    - Thread-safe metric updates
    - Namespace and subsystem organization ("isame_lb")
  - **Integration**: Seamless integration with HTTP proxy for automatic metric recording

- **`collector_test.go`**
  - **Purpose**: Unit tests for metrics collection functionality
  - **Test Coverage**: Collector creation, metric recording, HTTP metrics endpoint
  - **Validation**: Ensures proper Prometheus metric format and values

**`internal/proxy/`** - HTTP reverse proxy

- **`proxy.go`**

  - **Purpose**: HTTP reverse proxy with load balancing integration
  - **Structure**: `Handler` struct orchestrating load balancers, health checker, and metrics
  - **Features**:
    - HTTP reverse proxy using `httputil.ReverseProxy`
    - Proper proxy header management (X-Forwarded-For, X-Forwarded-Proto, X-Forwarded-Host)
    - Load balancer integration with health-aware backend selection
    - Automatic metrics recording (requests, duration, errors)
    - Error handling for no upstreams/unhealthy backends
  - **Header Management**: Client IP extraction with X-Forwarded-For and X-Real-IP support
  - **Service Integration**: Custom `X-Load-Balancer` header with service name

- **`proxy_test.go`**
  - **Purpose**: Comprehensive HTTP proxy testing
  - **Test Coverage**: Proxy creation, HTTP forwarding, round-robin behavior, header management
  - **Integration Testing**: Uses real HTTP test servers for end-to-end validation
  - **Header Testing**: Validates all proxy headers are correctly set and forwarded

### `pkg/` - Public API/Interfaces (Future)

- **Purpose**: Public interfaces that external packages could use
- **Current State**: Empty directory, will be populated in later phases

## Configuration and Deployment

### `configs/` - Configuration Examples

- **`example.yaml`**

  - **Purpose**: Comprehensive example configuration showing all features
  - **Configuration**: Multiple upstreams (web-servers, api-servers) with weighted backends
  - **Features**: Server timeouts, health checking, metrics configuration
  - **Use Case**: Production-like setup with realistic backend configurations

- **`dev.yaml`**
  - **Purpose**: Development configuration for local testing
  - **Configuration**: Single upstream with localhost backends
  - **Features**: Simplified setup for development and testing
  - **Use Case**: Local development with multiple backend instances on different ports

### `deploy/docker/` - Docker Deployment

- **Purpose**: Docker-related files for containerized deployment
- **Current State**: Empty directory, placeholder for future phases

## Documentation

### `docs/` - Project Documentation

- **Purpose**: Contains detailed project documentation
- **Files**:
  - `IMPLEMENTATION_PLAN.md`: Development phases and roadmap
  - `PROJECT_STRUCTURE.md`: This file - detailed project structure explanation

### `@docs/` - Technical Design Documents

- **Purpose**: Contains high-level technical design documents
- **Note**: Uses `@` prefix to distinguish from user documentation
- **Files**:
  - `IMPLEMENTATION_PLAN.md`: Phase-by-phase development plan
  - `TECHNICAL_DESIGN.md`: System architecture and design decisions
  - `TECH_STACK.md`: Technology choices and rationale

## Build Artifacts

### `bin/` - Compiled Binaries

- **Purpose**: Output directory for compiled binaries
- **Contents**:
  - `isame-lb`: Main load balancer server binary
  - `isame-ctl`: CLI management tool binary
- **Git Status**: Ignored by version control
- **Management**: Cleaned by `make clean` command

## Development Infrastructure

### `.github/workflows/` - GitHub Actions CI/CD

- **`ci.yml`**
  - **Purpose**: Continuous integration pipeline
  - **Go Versions**: Tests against Go 1.22 and 1.24
  - **Pipeline Steps**:
    1. Dependency caching and download
    2. Code formatting verification
    3. Go vet static analysis
    4. Project build
    5. Test execution (including race condition detection)
    6. Staticcheck static analysis
  - **Features**: Module caching for faster builds, multi-version testing

### `.claude/` - Claude Code Configuration

- **`settings.local.json`**: Local settings for Claude Code development environment

## Development Workflow

### Current Commands

```bash
make build
make test
make lint
make run
make dev-setup
```
