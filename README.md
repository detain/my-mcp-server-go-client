# MCP Proxy Client

A standalone Go-based MCP (Model Context Protocol) proxy server that proxies requests to a MyAdmin API. This server fetches its tool definitions from a remote OpenAPI spec and handles MCP protocol communication via Streamable HTTP transport.

## Features

- **Streamable HTTP transport** - Standard MCP 2025 protocol support
- **Dynamic tool loading** - Fetches tool definitions from remote OpenAPI spec
- **File-based session persistence** - Sessions stored on filesystem
- **OAuth 2.1 protected resource metadata** - RFC 9700 compliant endpoint
- **Auth header forwarding** - Supports X-API-KEY, sessionid, and Bearer tokens
- **Tool caching** - Caches parsed OpenAPI tools for performance
- **STDIO transport mode** - For Claude Desktop and Cursor integration
- **Self-contained binary** - No PHP or web server required

## Requirements

- Go 1.23+
- No external runtime dependencies (statically linked)

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/myadmin/go-mcp-proxy-client.git
cd go-mcp-proxy-client

# Build the binary
go build -o bin/mcp-proxy-client ./cmd/server
```

The binary will be created at `bin/mcp-proxy-client` (or `bin/mcp-proxy-client.exe` on Windows).

### Binary Size

The compiled binary is approximately 5-15 MB depending on platform, making it easy to distribute and deploy.

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAPI_SPEC_URL` | Yes | - | URL to fetch OpenAPI spec from (JSON or YAML) |
| `API_BASE_URL` | Yes | - | Base URL of the API to proxy to |
| `SESSION_DIR` | No | `/tmp/mcp_client_sessions` | Directory for session storage |
| `CACHE_DIR` | No | `/tmp/mcp_client_cache` | Directory for cached tool definitions |
| `SERVER_NAME` | No | `myadmin-client-mcp` | Name advertised in MCP handshake |
| `SERVER_VERSION` | No | `1.0.0` | Version advertised in MCP handshake |
| `OAUTH_AUTHORIZATION_SERVER` | No | Derived from `API_BASE_URL` | OAuth authorization server URL |

### Authentication Variables (for STDIO mode)

| Variable | Required | Description |
|----------|----------|-------------|
| `API_KEY` | No | API key for authentication |
| `SESSION_ID` | No | Session ID for authentication |
| `BEARER_TOKEN` | No | Bearer token for authentication |

## Running the Server

### STDIO Mode (Default for Claude Desktop / Cursor)

When run without a TTY (e.g., from Claude Desktop), the server automatically uses STDIO mode:

```bash
# Set required environment variables
export OPENAPI_SPEC_URL=https://my.interserver.net/spec/openapi.yaml
export API_BASE_URL=https://my.interserver.net/apiv2
export API_KEY=your_api_key

# Run the binary
./bin/mcp-proxy-client
```

### HTTP Mode (Interactive/Development)

When run with a TTY connected, the server starts an HTTP server on port 8080:

```bash
# Set required environment variables
export OPENAPI_SPEC_URL=https://my.interserver.net/spec/openapi.yaml
export API_BASE_URL=https://my.interserver.net/apiv2
export API_KEY=your_api_key

# Run the binary - will start HTTP server on :8080
./bin/mcp-proxy-client
```

### Command Line Options

```
-config string
    Path to .env configuration file
-version
    Show version and exit
```

## Web Server Configuration

For production HTTP deployments, you can run the binary behind a reverse proxy.

### Apache vhost

```apache
<VirtualHost *:443>
    ServerName mcp-proxy.example.com
    ProxyPreserveHost On

    # Proxy MCP traffic to the Go binary
    ProxyPass /mcp http://localhost:8080/mcp
    ProxyPassReverse /mcp http://localhost:8080/mcp

    # Proxy OAuth metadata endpoint
    ProxyPass /.well-known/oauth-protected-resource http://localhost:8080/.well-known/oauth-protected-resource
    ProxyPassReverse /.well-known/oauth-protected-resource http://localhost:8080/.well-known/oauth-protected-resource

    # Ensure proper headers reach the backend
    RequestHeader set X-Forwarded-Proto "https"
    RequestHeader set X-Forwarded-For "%{REMOTE_ADDR}s"
</VirtualHost>
```

### Nginx config

```nginx
server {
    listen 443 ssl;
    server_name mcp-proxy.example.com;

    location /mcp {
        proxy_pass http://localhost:8080/mcp;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        # Forward authorization headers
        proxy_pass_header Authorization;
    }

    location /.well-known/oauth-protected-resource {
        proxy_pass http://localhost:8080/.well-known/oauth-protected-resource;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## API Endpoints

### MCP Protocol Endpoint

```
POST /mcp          - Send MCP JSON-RPC messages
GET /mcp           - OAuth protected resource metadata (via Accept header)
DELETE /mcp        - Close MCP session
OPTIONS /mcp       - CORS preflight
```

### OAuth Protected Resource Metadata

```
GET /.well-known/oauth-protected-resource
```

Returns RFC 9700 compliant protected resource metadata.

## Authentication

The proxy forwards authentication credentials from incoming requests to the backend API:

- **Bearer Token**: `Authorization: Bearer <token>`
- **API Key**: `X-API-KEY: <key>`
- **Session ID**: `sessionid: <session_id>`

These are passed through to the backend API via the corresponding headers.

## Claude Desktop Integration

Add to your Claude Desktop configuration file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
**Linux:** `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "myadmin-client": {
      "command": "/path/to/bin/mcp-proxy-client",
      "env": {
        "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
        "API_BASE_URL": "https://my.interserver.net/apiv2",
        "API_KEY": "your_api_key"
      }
    }
  }
}
```

**Note:** Use the full path to the binary. Unlike the PHP version which required `php bin/mcp`, the Go binary is self-contained and runs directly.

## Cursor Integration

Add to Cursor settings (Settings → MCP Servers):

```json
{
  "mcpServers": {
    "myadmin-client": {
      "command": "/path/to/bin/mcp-proxy-client",
      "env": {
        "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
        "API_BASE_URL": "https://my.interserver.net/apiv2",
        "API_KEY": "your_api_key"
      }
    }
  }
}
```

## Tool Caching

Tool definitions from the OpenAPI spec are cached in `CACHE_DIR`. The cache is invalidated when the remote spec's `Last-Modified` header is newer than the cache file.

### Force Cache Refresh

```bash
# Delete cache files to force refresh
rm -f /tmp/mcp_client_cache/tools_*.json

# Or set a custom cache directory
export CACHE_DIR=/custom/cache/path
```

## Troubleshooting

### "Failed to fetch OpenAPI spec"

- Verify `OPENAPI_SPEC_URL` is accessible from the server
- Check network connectivity and firewall rules
- Ensure the spec URL returns valid JSON or YAML
- Try fetching manually: `curl -I <OPENAPI_SPEC_URL>`

### "Missing required environment variables"

- Copy `.env.example` to `.env`
- Set both `OPENAPI_SPEC_URL` and `API_BASE_URL`

### "Connection refused" in HTTP mode

- Ensure the binary has permission to bind to port 8080
- Check if another process is using port 8080: `lsof -i :8080`

### Session issues

- Ensure `SESSION_DIR` is writable
- Check disk space if sessions fail to create
- Sessions expire after 1 hour by default

### Binary won't start

- Verify the binary is executable: `chmod +x bin/mcp-proxy-client`
- Check Go version: `go version` (requires 1.23+)
- Run with debug logging: `RUST_LOG=debug ./bin/mcp-proxy-client`

## Internal Packages

### `internal/config`

Configuration management - loads from `.env` files and environment variables with validation.

### `internal/server`

MCP server setup, HTTP handlers, and transport layer (STDIO and Streamable HTTP).

### `internal/openapi`

OpenAPI specification parsing and MCP tool generation from OpenAPI specs.

### `internal/proxy`

Upstream API client with authentication header forwarding.

### `internal/oauth`

OAuth 2.1 protected resource metadata endpoint (RFC 9700).

## License

Proprietary - InterServer Inc.
