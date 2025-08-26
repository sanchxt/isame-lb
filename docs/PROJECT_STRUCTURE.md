# Project Structure

This document provides a detailed explanation of the Isame Load Balancer project structure, including the purpose and functionality of each file and directory.

## Root Directory

### Configuration Files

**`go.mod`** - Go module file

- **Purpose**: Defines the Go module and its dependencies
- **Current State**: Module name `github.com/sanchxt/isame-lb` with Go version 1.24.6
- **Dependencies**: Currently no external dependencies (using only standard library)

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

- **`main.go`** (main.go:1-41)

  - **Purpose**: Entry point for the load balancer server
  - **Functionality**:
    - Sets up HTTP server on port 8080
    - Implements health check endpoint (`/health`)
    - Implements service info endpoint (`/`)
    - Configures server timeouts (15s read/write, 60s idle)
  - **Current State**: Simple HTTP server returning JSON responses

- **`main_test.go`** (main_test.go:1-63)
  - **Purpose**: Unit tests for main server handlers
  - **Coverage**: Tests both health and root handlers
  - **Test Types**: HTTP response validation, content-type checking

**`cmd/isame-ctl/`** - Command-line interface tool

- **`main.go`** (main.go:1-33)
  - **Purpose**: CLI tool for managing the load balancer
  - **Current Commands**: `version`, `help`
  - **State**: Basic stub implementation for Phase 0
  - **Future**: Will be expanded in Phase 3 for configuration management

### `internal/` - Private Application Code

**`internal/config/`** - Configuration management

- **`config.go`** (config.go:1-24)

  - **Purpose**: Configuration structure and validation
  - **Current Structure**: Basic `Config` struct with `Version` and `Service` fields
  - **Functions**:
    - `NewDefaultConfig()`: Creates default configuration
    - `Validate()`: Applies defaults for empty fields
  - **Future**: Will be extended for backend server configuration, load balancing settings

- **`config_test.go`** (config_test.go:1-61)
  - **Purpose**: Unit tests for configuration functionality
  - **Test Coverage**: Default config creation, validation logic
  - **Test Strategy**: Table-driven tests with multiple scenarios

**`internal/server/`** - HTTP server implementation

- **`server.go`** (server.go:1-60)

  - **Purpose**: Abstracted server implementation with clean architecture
  - **Structure**: `Server` struct encapsulating config and HTTP server
  - **Methods**:
    - `New()`: Creates server instance
    - `Start()`: Starts HTTP server with routing
    - `Shutdown()`: Graceful server shutdown
    - Handler methods for health and root endpoints
  - **Advantages**: More testable and modular than the main.go implementation

- **`server_test.go`** (server_test.go:1-75)
  - **Purpose**: Comprehensive server testing
  - **Test Coverage**: Server creation, handler functionality, HTTP responses
  - **Testing Approach**: Uses `httptest` for HTTP handler testing

**`internal/balancer/`** - Load balancing algorithms (Future Phase 1)

- **Purpose**: Will contain load balancing algorithms (round-robin, weighted, least-connections)
- **Current State**: Empty directory, placeholder for Phase 1 development

**`internal/health/`** - Health checking functionality (Future Phase 1)

- **Purpose**: Backend server health checking and monitoring
- **Current State**: Empty directory, placeholder for Phase 1 development

**`internal/metrics/`** - Observability and metrics (Future Phase 1)

- **Purpose**: Prometheus metrics collection and monitoring
- **Current State**: Empty directory, placeholder for Phase 1 development

### `pkg/` - Public API/Interfaces (Future)

- **Purpose**: Public interfaces that external packages could use
- **Current State**: Empty directory, will be populated in later phases

## Configuration and Deployment

### `configs/` - Configuration Examples

- **Purpose**: Will contain example YAML configuration files
- **Current State**: Empty directory, placeholder for Phase 1

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

- **`ci.yml`** (ci.yml:1-65)
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

## Architecture Patterns

### Current Design Principles

1. **Clean Architecture**: Separation of concerns with `internal/` packages
2. **Testability**: Comprehensive unit tests with dependency injection
3. **SOLID Principles**: Single responsibility, dependency inversion evident in server design
4. **Standard Go Conventions**: Following Go project layout standards

### Future Architecture (Planned)

1. **Modular Load Balancing**: Pluggable algorithms in `internal/balancer/`
2. **Health Monitoring**: Separate health checking system
3. **Observability**: Comprehensive metrics and tracing
4. **Configuration-Driven**: YAML-based configuration system

## Development Workflow

### Current Commands

```bash
make build        # Build both binaries
make test         # Run comprehensive tests
make lint         # Full linting and static analysis
make run          # Build and run the server
make dev-setup    # Setup development environment
```

### Testing Strategy

- **Unit Tests**: Each package has comprehensive test coverage
- **HTTP Testing**: Uses `httptest` for handler testing
- **Race Detection**: All tests run with race condition detection
- **Table-Driven Tests**: Used for configuration validation and multiple scenarios

This structure provides a solid foundation for the phased development approach, with clear separation of concerns and room for future expansion while maintaining clean architecture principles.
