# ya-practicum-go-and-llm

A Go-based RAG (Retrieval-Augmented Generation) application using OpenAI and Qdrant.

## Configuration

The application can be configured using environment variables or command-line flags. Flags take precedence over environment variables.

### Environment Variables

```bash
# Server
export SERVER_PORT=8080

# OpenAI
export OPENAI_API_KEY=your-api-key-here
export OPENAI_MODEL=gpt-4o-mini
export OPENAI_EMBED_MODEL=text-embedding-3-large

# Qdrant
export QDRANT_HOST=localhost
export QDRANT_PORT=6334
export QDRANT_COLLECTION=docs

# RAG Settings
export CHUNK_SIZE=1000
export CHUNK_OVERLAP=200
export SEARCH_LIMIT=3
```

### Command-Line Flags

```bash
./server \
  -server-port=8080 \
  -openai-key=your-api-key-here \
  -openai-model=gpt-4o-mini \
  -openai-embed-model=text-embedding-3-large \
  -qdrant-host=localhost \
  -qdrant-port=6334 \
  -qdrant-collection=docs \
  -chunk-size=1000 \
  -chunk-overlap=200 \
  -search-limit=3
```

### Available Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-server-port` | `SERVER_PORT` | `8080` | HTTP server port |
| `-openai-key` | `OPENAI_API_KEY` | (required) | OpenAI API key |
| `-openai-model` | `OPENAI_MODEL` | `gpt-4o-mini` | OpenAI model for chat completions |
| `-openai-embed-model` | `OPENAI_EMBED_MODEL` | `text-embedding-3-large` | OpenAI model for embeddings |
| `-qdrant-host` | `QDRANT_HOST` | `localhost` | Qdrant server host |
| `-qdrant-port` | `QDRANT_PORT` | `6334` | Qdrant gRPC port (default: 6334) |
| `-qdrant-collection` | `QDRANT_COLLECTION` | `docs` | Qdrant collection name |
| `-chunk-size` | `CHUNK_SIZE` | `1000` | Text chunk size for splitting documents |
| `-chunk-overlap` | `CHUNK_OVERLAP` | `200` | Overlap between text chunks |
| `-search-limit` | `SEARCH_LIMIT` | `3` | Number of search results to return |

### Example Usage

```bash
# Using environment variables
export OPENAI_API_KEY=sk-...
go run cmd/server/main.go

# Using flags
go run cmd/server/main.go -openai-key=sk-... -server-port=3000

# Mix of both (flags override env vars)
export OPENAI_API_KEY=sk-...
go run cmd/server/main.go -server-port=3000
```

## Taskfile Commands

This project uses [Task](https://taskfile.dev/) for task automation. Install Task first:

```bash
# macOS
brew install go-task/tap/go-task

# Linux
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

# Or using Go
go install github.com/go-task/task/v3/cmd/task@latest
```

### Available Tasks

#### Development
- `task dev` - Start development environment (Qdrant + app locally)
- `task dev-docker` - Start everything in Docker containers
- `task run` - Run the application (requires Qdrant)

#### Building
- `task build` - Build the Go application
- `task docker-build` - Build the Docker image

#### Testing & Quality
- `task test` - Run all tests with coverage
- `task test-short` - Run tests in short mode
- `task test-coverage` - Show test coverage report
- `task lint` - Run golangci-lint
- `task lint-fix` - Run golangci-lint with auto-fix
- `task fmt` - Format Go code
- `task vet` - Run go vet
- `task check` - Run lint, test, and build
- `task all` - Run format, vet, lint, test, and build

#### Docker Compose
- `task docker-up` - Start Qdrant only
- `task docker-up-all` - Start all services (Qdrant + App)
- `task docker-down` - Stop all services
- `task docker-restart` - Restart services
- `task docker-logs` - Show logs from all services
- `task docker-logs-app` - Show app logs only
- `task docker-ps` - Show running services

#### Utilities
- `task clean` - Clean build artifacts
- `task install-deps` - Install Go dependencies
- `task install-tools` - Install development tools (golangci-lint)

### Quick Start

```bash
# 1. Install dependencies
task install-deps

# 2. Start Qdrant
task docker-up

# 3. Set your OpenAI API key
export OPENAI_API_KEY=sk-...

# 4. Run the application
task run

# Or run everything in Docker
task dev-docker
```
