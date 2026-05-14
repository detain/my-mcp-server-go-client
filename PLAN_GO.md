---
status: not-started
phase: 1
updated: 2026-05-13
---

# Implementation Plan: Go MCP Proxy Server - Client

## Goal

Replace the PHP-based Client MCP proxy server with a Go implementation that compiles to a single static native binary, supporting Streamable HTTP and STDIO transports, dynamic OpenAPI tool generation, and OAuth 2.1 protected resource metadata.

**Working Directory:** `/home/sites/my-mcp-server-go-client`
**Reference Implementation:** `/home/sites/my-mcp-server-go-client/my-mcp-server-php-client` (PHP version to base off)
**Commit Author:** Joe Huss `<detain@interserver.net>`

## Context & Decisions

| Decision | Rationale | Source |
|----------|-----------|--------|
| Use Go as primary language | Official MCP SDK with OAuth 2.1 support (`oauthex`), single static binary, excellent cross-compilation | `ref:ses_1dc4c5519ffeimTaIt5tGuyPv8` |
| Use Gin HTTP framework | Fastest Go HTTP framework, mature ecosystem, zero allocations | `ref:ses_1dc4c5519ffeimTaIt5tGuyPv8` |
| Use `kin-openapi` for OpenAPI parsing | Well-maintained, supports OpenAPI 3.x | `ref:ses_1dc4c5519ffeimTaIt5tGuyPv8` |
| Support both Streamable HTTP and STDIO | Match existing PHP implementation features | PHP client implementation |

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Go MCP Proxy Binary                          │
│                                                                  │
│  ┌─────────────┐    ┌─────────────────┐    ┌──────────────────┐ │
│  │  Transport  │───▶│  MCP Server     │───▶│  OpenAPI Tool    │ │
│  │  Layer      │    │  (go-sdk)       │    │  Generator       │ │
│  └─────────────┘    └─────────────────┘    └──────────────────┘ │
│         │                   │                      │            │
│         │                   │                      ▼            │
│         │                   │            ┌──────────────────┐  │
│         │                   │            │  HTTP Client     │  │
│         │                   │            │  (upstream API)  │  │
│         │                   │            └──────────────────┘  │
│         │                   │                                    │
│  ┌─────────────┐           │                                    │
│  │ OAuth 2.1   │◀──────────┘                                    │
│  │ Endpoints   │                                                │
│  └─────────────┘                                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Project Setup [PENDING]

### 1.1 Initialize Go Module
```bash
cd /home/sites/my-mcp-server-go-client
go mod init github.com/myadmin/go-mcp-proxy-client
```

### 1.2 Add Dependencies
```go
// go.mod
module github.com/myadmin/go-mcp-proxy-client

go 1.23

require (
    github.com/modelcontextprotocol/go-sdk v0.3.0
    github.com/gin-gonic/gin v1.10.0
    github.com/getkin/kin-openapi v1.0.1
    github.com/joho/godotenv v1.5.1
    github.com/stretchr/testify v1.10.0 // testing
)
```

### 1.3 Create Project Structure
```
/home/sites/my-mcp-server-go-client/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── server/
│   │   ├── mcp.go               # MCP server setup
│   │   ├── handlers.go          # Tool handlers
│   │   └── transport.go         # HTTP/stdio transport
│   ├── openapi/
│   │   ├── parser.go            # OpenAPI spec fetching/parsing
│   │   ├── generator.go         # Tool definition generation
│   │   └── cache.go             # Spec caching
│   ├── proxy/
│   │   ├── client.go            # Upstream API client
│   │   └── auth.go              # Auth header extraction
│   └── oauth/
│       └── metadata.go          # OAuth 2.1 protected resource
├── my-mcp-server-php-client/    # PHP reference (already present)
├── .env.example
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 1.4 Set up Logging
Use `log/slog` from Go standard library for structured logging.

### 1.5 Create `.env.example`
```bash
OPENAPI_SPEC_URL=https://my.interserver.net/spec/openapi.yaml
API_BASE_URL=https://my.interserver.net/apiv2
SESSION_DIR=/tmp/mcp_client_sessions
CACHE_DIR=/tmp/mcp_client_cache
SERVER_NAME=myadmin-client-mcp
SERVER_VERSION=1.0.0
BEARER_TOKEN=
API_KEY=
SESSION_ID=
OAUTH_AUTHORIZATION_SERVER=
```

### 1.6 Verify Build Works
```bash
go build -o bin/mcp-proxy-client ./cmd/server
./bin/mcp-proxy-client --version
```

**After step 1.6:** Commit with message "feat: initial project structure and dependencies" and create PR, merge, delete branch, pull master.

---

## Phase 2: Core MCP Server [PENDING]

### 2.1 Implement MCP Server Initialization
Reference: `/home/sites/my-mcp-server-go-client/my-mcp-server-php-client/src/McpServerFactory.php`

Create `internal/server/mcp.go`:
- Initialize MCP server with `mcp.NewServer()`
- Configure server info (name, version)
- Set up pagination limit (1000)

### 2.2 Implement Streamable HTTP Transport Handler
```go
handler := mcp.NewStreamableHTTPHandler("/mcp", server, nil)
```

### 2.3 Implement Transport Detection
Create `internal/server/transport.go`:
```go
func detectTransport() string {
    stat, _ := os.Stdin.Stat()
    if (stat.Mode() & os.ModeCharDevice) != 0 {
        return "http" // TTY connected - run HTTP server
    }
    return "stdio" // Pipe/redirect - run stdio transport
}
```

### 2.4 Implement STDIO Transport Support
```go
if detectTransport() == "stdio" {
    mcp.ServeStdio(server)
} else {
    // Run HTTP server
    gin.Run(":8080")
}
```

### 2.5 Add OAuth 2.1 Protected Resource Metadata Endpoint
Use `oauthex` package to serve `/.well-known/oauth-protected-resource`

### 2.6 Test Basic Server
Test that server starts and responds to MCP handshake correctly.

### 2.7 Write Unit Tests
- `internal/server/transport_test.go` - Test transport detection
- `internal/server/mcp_test.go` - Test server initialization

**After step 2.7:** Commit with message "feat: core MCP server with transport detection" and create PR, merge, delete branch, pull master.

---

## Phase 3: OpenAPI Parser [PENDING]

### 3.1 Fetch OpenAPI Spec
Reference: PHP implementation's `OpenApiParser.php`

Create `internal/openapi/parser.go`:
```go
resp, err := http.Get(specURL)
spec, err := openapi3.NewLoader().LoadFromData(data)
```

### 3.2 Parse Operations and Map to MCP Tools
Create `internal/openapi/generator.go`:
- Extract `operationId` → tool name
- Extract `summary`/`description` → tool description
- Parse `parameters` (path, query, header) → input schema
- Parse `requestBody` → body schema
- Map HTTP methods (GET, POST, PUT, DELETE, PATCH)

### 3.3 Build JSON Schema from OpenAPI Parameter Definitions
Convert OpenAPI parameter definitions to JSON Schema for MCP tool input schemas.

### 3.4 Implement Caching
Create `internal/openapi/cache.go`:
- Cache parsed spec to filesystem (1 hour TTL)
- Invalidate on cache clear or TTL expiry
- Handle spec reload on cache invalidation

### 3.5 Write Unit Tests
- `internal/openapi/parser_test.go`
- `internal/openapi/generator_test.go`
- `internal/openapi/cache_test.go`

**After step 3.5:** Commit with message "feat: OpenAPI parser with tool generation and caching" and create PR, merge, delete branch, pull master.

---

## Phase 4: Dynamic Tool Registration [PENDING]

### 4.1 Register Tools from OpenAPI
Create `internal/server/handlers.go`:
```go
for path, pathItem := range spec.Paths {
    for method, operation := range pathItem.GetOperationsMap() {
        tool := generateTool(operation, path, method)
        server.AddTool(tool)
    }
}
```

### 4.2 Implement Tool Handler that Proxies to Upstream API
```go
func toolHandler(params map[string]any) (any, error) {
    // Build URL with path parameters substituted
    // Add query parameters
    // Set auth headers from incoming request
    // POST body if applicable
    // Call upstream API
    // Return response
}
```

### 4.3 Handle Auth Header Extraction
- Check `Authorization: Bearer <token>`
- Check `X-API-KEY: <key>`
- Check `sessionid: <id>`

### 4.4 Add Required Headers
- `X-API-APP: 1` - Short-circuits rate limiting for MCP callers
- `X-Request-Id` - For tracing

### 4.5 Write Unit Tests
- `internal/server/handlers_test.go` - Test tool handlers
- `internal/proxy/auth_test.go` - Test auth extraction

**After step 4.5:** Commit with message "feat: dynamic tool registration and proxy handlers" and create PR, merge, delete branch, pull master.

---

## Phase 5: Configuration & Environment [PENDING]

### 5.1 Load Environment Variables
Create `internal/config/config.go`:
- Load from `.env` file using godotenv
- Support environment variable overrides
- Validate required variables

### 5.2 Configuration Structure
```go
type Config struct {
    OpenAPISpecURL           string
    APIBaseURL               string
    SessionDir               string
    CacheDir                 string
    ServerName               string
    ServerVersion            string
    BearerToken              string
    APIKey                   string
    SessionID                string
    OAuthAuthorizationServer string
}
```

### 5.3 Environment Variables Table
| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAPI_SPEC_URL` | URL to fetch OpenAPI spec | Required |
| `API_BASE_URL` | Base URL of upstream API | Required |
| `SESSION_DIR` | Session storage directory | `/tmp/mcp_client_sessions` |
| `CACHE_DIR` | Tool cache directory | `/tmp/mcp_client_cache` |
| `SERVER_NAME` | MCP server name | `myadmin-client-mcp` |
| `SERVER_VERSION` | MCP server version | `1.0.0` |
| `BEARER_TOKEN` | Bearer token for stdio mode | - |
| `API_KEY` | API key for stdio mode | - |
| `SESSION_ID` | Session ID for stdio mode | - |
| `OAUTH_AUTHORIZATION_SERVER` | OAuth authorization server URL | Derived from `API_BASE_URL` |

### 5.4 Write Unit Tests
- `internal/config/config_test.go`

**After step 5.4:** Commit with message "feat: configuration management with env file support" and create PR, merge, delete branch, pull master.

---

## Phase 6: OAuth 2.1 Protected Resource Metadata [PENDING]

### 6.1 Implement Protected Resource Metadata Endpoint
Create `internal/oauth/metadata.go`:
```go
func protectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(oauthex.ProtectedResourceMetadata{
        Resource:               serverURL,
        AuthorizationServers:   []string{authServerURL},
        ScopesSupported:        []string{"read", "write"},
        BearerMethodsSupported: []string{"header"},
    })
}
```

### 6.2 Add WWW-Authenticate Header
Return `WWW-Authenticate: Bearer realm="mcp"` on 401 responses.

### 6.3 Validate Bearer Tokens
If Bearer tokens are provided, validate them appropriately.

### 6.4 Write Unit Tests
- `internal/oauth/metadata_test.go`

**After step 6.4:** Commit with message "feat: OAuth 2.1 protected resource metadata endpoint" and create PR, merge, delete branch, pull master.

---

## Phase 7: Build & Distribution [PENDING]

### 7.1 Create Makefile
```makefile
build: go build -o bin/mcp-proxy-client ./cmd/server

build-linux-amd64:
    GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/mcp-proxy-client-linux-amd64 ./cmd/server

build-linux-arm64:
    GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/mcp-proxy-client-linux-arm64 ./cmd/server

build-darwin-amd64:
    GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/mcp-proxy-client-darwin-amd64 ./cmd/server

build-darwin-arm64:
    GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/mcp-proxy-client-darwin-arm64 ./cmd/server

build-windows:
    GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/mcp-proxy-client-windows-amd64.exe ./cmd/server

test:
    go test -v -race ./...

clean:
    rm -rf bin/
```

### 7.2 Build for Multiple Targets
- Linux AMD64
- Linux ARM64
- macOS AMD64
- macOS ARM64 (Apple Silicon)
- Windows AMD64

### 7.3 Create Distribution Archives
Create `.tar.gz` for Linux/macOS, `.zip` for Windows.

**After step 7.3:** Commit with message "feat: build system and multi-platform distribution" and create PR, merge, delete branch, pull master.

---

## Phase 8: GitHub Actions Setup [PENDING]

### 8.1 Create CI Workflow

**File:** `.github/workflows/ci.yml`
```yaml
name: CI

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          
      - name: Download dependencies
        run: go mod download
        
      - name: Run tests with coverage
        run: go test -v -race -coverprofile=coverage.out ./...
        
      - name: Upload coverage
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out

  build:
    runs-on: ubuntu-latest
    needs: test
    strategy:
      matrix:
        goos: [linux, darwin, windows]
        goarch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          
      - name: Build
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
        run: |
          ext=""
          if [ "$GOOS" = "windows" ]; then ext=".exe"; fi
          go build -ldflags="-s -w" -o bin/mcp-proxy-client-${{ matrix.goos }}-${{ matrix.goarch }}$ext ./cmd/server
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: mcp-proxy-client-${{ matrix.goos }}-${{ matrix.goarch }}
          path: bin/*
```

### 8.2 Create Release Workflow

**File:** `.github/workflows/release.yml`
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          
      - name: Run tests
        run: go test -v ./...
        
      - name: Build releases
        run: |
          mkdir -p releases
          for os in linux darwin windows; do
            for arch in amd64 arm64; do
              ext=""
              if [ "$os" = "windows" ]; then ext=".exe"; fi
              GOOS=$os GOARCH=$arch go build -ldflags="-s -w" \
                -o releases/mcp-proxy-client-$os-$arch$ext ./cmd/server
            done
          done
          
      - name: Create checksums
        cd releases
        sha256sum * > checksums.txt
        
      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: releases/*
          generate_release_notes: true
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### 8.3 Create Tag Update Workflow

**IMPORTANT:** When using `gh` CLI, you MUST unset GITHUB_TOKEN first:
```bash
unset GITHUB_TOKEN
gh auth status
```

**File:** `.github/workflows/update-tags.yml`
```yaml
name: Update Repository Metadata

on:
  release:
    types: [published]
  workflow_dispatch:

jobs:
  update-metadata:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          
      - name: Update GitHub Repository Description and Tags
        run: |
          unset GITHUB_TOKEN
          gh auth status || gh auth login
          
          # Get current release tag
          TAG=${GITHUB_REF#refs/tags/}
          
          # Update repository description (max 350 chars)
          gh repo edit --description "Go MCP Proxy Server for MyAdmin Client API - v$TAG"
          
          # Add 19 relevant tags to repository
          # Note: Repository topics are different from release tags
          gh repo edit \
            --add-topic mcp \
            --add-topic model-context-protocol \
            --add-topic go \
            --add-topic golang \
            --add-topic api-proxy \
            --add-topic openapi \
            --add-topic client-api \
            --add-topic automation \
            --add-topic cli \
            --add-topic server \
            --add-topic stdio \
            --add-topic streamable-http \
            --add-topic oauth2 \
            --add-topic rest-api \
            --add-topic Interserver \
            --add-topic hosting \
            --add-topic vps \
            --add-topic webhosting \
            --add-topic ssl \
            --add-topic billing
          
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**After step 8.3:** Commit with message "ci: GitHub Actions for CI/CD and releases" and create PR, merge, delete branch, pull master.

---

## Phase 9: Documentation [PENDING]

### 9.1 Create Comprehensive README.md
Include:
- Requirements (Go 1.23+)
- Installation instructions
- Environment variables table
- Web server configuration (Apache, Nginx, LiteSpeed)
- STDIO mode configuration
- Claude Desktop / Cursor integration
- Troubleshooting guide

### 9.2 Add Inline Code Documentation
- godoc comments on all exported functions
- Package documentation

### 9.3 Create Man Page
Create `man/mcp-proxy-client.1` man page.

**After step 9.3:** Commit with message "docs: comprehensive README and inline documentation" and create PR, merge, delete branch, pull master.

---

## Phase 10: AI Client Integration Instructions [PENDING]

### 10.1 Create Comprehensive Integration Guide

Create `INTEGRATION.md` documenting both STDIO and Streamable HTTP setup for each AI client.

#### Claude Desktop (STDIO)
```json
{
  "mcpServers": {
    "myadmin-client": {
      "command": "/path/to/mcp-proxy-client",
      "env": {
        "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
        "API_BASE_URL": "https://my.interserver.net/apiv2",
        "API_KEY": "your_api_key_here"
      }
    }
  }
}
```

#### Claude Desktop (Streamable HTTP)
```json
{
  "mcpServers": {
    "myadmin-client": {
      "type": "streamable-http",
      "url": "https://mcp.example.com/mcp",
      "headers": {
        "X-API-KEY": "your_api_key_here"
      }
    }
  }
}
```

#### Cursor (STDIO)
Settings → MCP Servers → Add server with same JSON structure.

#### Cursor (Streamable HTTP)
Same as Claude Desktop streamable-http configuration.

#### VS Code
Install "Model Context Protocol" extension, configure server.

#### Zed
Settings → Extensions → MCP → Add server.

#### JetBrains (IDEA, PyCharm, etc.)
Settings → Plugins → MCP → Configure.

#### Sourcegraph
Cody settings → MCP servers → Add server.

#### GitHub Copilot
Configure via Copilot settings.

#### Tabnine
Configure via Tabnine settings.

#### Codeium
Configure via Codeium settings.

#### Replit
Configure via Replit MCP integration.

#### Supermaven
Configure via Supermaven settings.

#### Morph
Configure via Morph MCP integration.

#### Aider
Configure via command line flags.

#### Goose
Configure via Goose settings.

#### Mastra
Configure via Mastra MCP integration.

#### Roo Code / Continue
Configure via VS Code extension.

#### Zed (already mentioned)
Settings → Extensions → MCP → Add server.

#### Other AI Development Tools
- CodeRabbit
- Greptile
- Bugasaki
- Fathom

### 10.2 For Each AI Client, Document:
1. STDIO transport setup (local binary)
2. Streamable HTTP transport setup (remote server)
3. Required environment variables / headers
4. Troubleshooting tips

**After step 10.2:** Commit with message "docs: AI client integration guide for all major editors" and create PR, merge, delete branch, pull master.

---

## Phase 11: Final Testing and Polish [PENDING]

### 11.1 Unit Tests
Ensure all tests pass:
```bash
go test -v -race ./...
```

### 11.2 Integration Tests
- Test against real MyAdmin client API
- Test Claude Desktop integration
- Test Cursor integration
- Test Streamable HTTP with remote client

### 11.3 Test MCP Inspector
```bash
npx @modelcontextprotocol/inspector /path/to/mcp-proxy-client
```

### 11.4 Performance Testing
- Benchmark tool invocation latency
- Verify binary size is 5-15 MB

### 11.5 Security Review
- Verify no credentials in logs
- Verify auth header handling
- Review error messages don't leak sensitive data

### 11.6 Create Release
- Tag v1.0.0
- Push tags
- Verify GitHub Actions run
- Verify release created with artifacts

**After step 11.6:** Commit with message "v1.0.0: initial release with full feature set" and create PR, merge, delete branch, pull master.

---

## Workflow Summary

After each phase's last step, perform:
```bash
# Commit changes
git add -A
git commit -m "feat: description of changes"

# Push to origin
git push origin HEAD

# Create PR (NOTE: Unset GITHUB_TOKEN before using gh cli)
unset GITHUB_TOKEN
gh pr create --title "feat: description" --body "Implements phase X features" --base master

# Merge PR
gh pr merge --squash --delete-branch

# Switch to master and pull
git checkout master
git pull origin master
```

---

## Verification Commands

1. **Build:**
   ```bash
   go build -o bin/mcp-proxy-client ./cmd/server
   ```

2. **Run tests:**
   ```bash
   go test -v -race ./...
   ```

3. **Test Claude Desktop integration:**
   ```bash
   # Set env vars and run
   OPENAPI_SPEC_URL=https://my.interserver.net/spec/openapi.yaml \
   API_BASE_URL=https://my.interserver.net/apiv2 \
   API_KEY=test \
   ./bin/mcp-proxy-client
   ```

4. **Verify binary size:**
   ```bash
   ls -lh bin/mcp-proxy-client
   # Should be 5-15 MB
   ```

5. **Protocol verification:**
   ```bash
   # Start server in HTTP mode
   OPENAPI_SPEC_URL=https://example.com/spec API_BASE_URL=https://example.com/api ./bin/mcp-proxy-client
   
   # Test with MCP inspector
   npx @modelcontextprotocol/inspector /path/to/mcp-proxy-client
   ```

---

## Notes

- 2026-05-13: Initial plan created `ref:ses_1dc4c5519ffeimTaIt5tGuyPv8`
- Reference PHP implementation at `/home/sites/my-mcp-server-go-client/my-mcp-server-php-client`
- Go's official SDK includes `oauthex` package for OAuth 2.1 protected resource metadata
- Cross-compilation is straightforward with `GOOS` and `GOARCH` environment variables
- Binary size of 5-15 MB is acceptable for desktop distribution
- **IMPORTANT:** Always `unset GITHUB_TOKEN` before using `gh` CLI commands to avoid conflicts
