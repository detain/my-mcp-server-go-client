# AI Client Integration Guide

A comprehensive guide for integrating the MyAdmin MCP (Model Context Protocol) client with 19+ AI-powered code editors and coding assistants.

## Overview

The MyAdmin MCP client (`mcp-proxy-client`) supports two transport modes for communicating with AI assistants:

### STDIO Mode (Local Binary)

In STDIO mode, the AI client spawns the MCP binary as a local subprocess. Communication happens through standard input/output streams. This is ideal for:
- Local development
- Privacy-sensitive configurations
- When the AI client and MCP server run on the same machine

### Streamable HTTP Mode (Remote Server)

In Streamable HTTP mode, the client connects to a running MCP server over HTTP. This is ideal for:
- Remote server deployments
- Shared infrastructure setups
- Cross-machine configurations

## Prerequisites

Before starting, ensure you have:

1. **Built the binary:**
   ```bash
   make build
   ```

   This creates `bin/mcp-proxy-client` (plus platform-specific binaries)

2. **Obtained your API key** from your MyAdmin panel

3. **Noted your server details:**
   - `OPENAPI_SPEC_URL`: URL to your OpenAPI specification (e.g., `https://my.interserver.net/spec/openapi.yaml`)
   - `API_BASE_URL`: Base URL for your API (e.g., `https://my.interserver.net/apiv2`)

## Quick Start

### Build the Binary

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Or build specific platforms
make build-linux-amd64
make build-darwin-arm64
make build-windows
```

Binaries will be in the `bin/` directory:
- `mcp-proxy-client` (Linux)
- `mcp-proxy-client-darwin-amd64` (macOS Intel)
- `mcp-proxy-client-darwin-arm64` (macOS Apple Silicon)
- `mcp-proxy-client-windows-amd64.exe` (Windows)

### Make Binary Executable

```bash
chmod +x bin/mcp-proxy-client
```

---

## Claude Desktop

Claude Desktop by Anthropic is a dedicated AI assistant application that supports MCP integrations.

### STDIO Mode

1. **Locate your Claude Desktop configuration file:**

   **macOS/Linux:**
   ```
   ~/Library/Application Support/Claude/claude_desktop_config.json
   ```

   **Windows:**
   ```
   %APPDATA%\Claude\claude_desktop_config.json
   ```

2. **Add the MCP server configuration:**

   ```json
   {
     "mcpServers": {
       "myadmin-client": {
         "command": "/absolute/path/to/bin/mcp-proxy-client",
         "env": {
           "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
           "API_BASE_URL": "https://my.interserver.net/apiv2",
           "API_KEY": "your_api_key_here"
         }
       }
     }
   }
   ```

3. **Restart Claude Desktop**

### Streamable HTTP Mode

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

### Troubleshooting

- **"Binary not executable"**: Run `chmod +x /absolute/path/to/bin/mcp-proxy-client`
- **"Connection refused"**: Ensure the Streamable HTTP server is running
- **Auth failures**: Verify your API key is correct

---

## Cursor

Cursor is an AI-powered code editor built on VS Code.

### STDIO Mode

1. **Open Cursor Settings** → **MCP Server** (or edit `~/.cursor/mcp_settings.json`)

2. **Add the server configuration:**
   ```json
   {
     "mcpServers": {
       "myadmin-client": {
         "command": "/path/to/bin/mcp-proxy-client",
         "env": {
           "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
           "API_BASE_URL": "https://my.interserver.net/apiv2",
           "API_KEY": "your_api_key_here"
         }
       }
     }
   }
   ```

3. **Restart Cursor**

### Streamable HTTP Mode

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

### Troubleshooting

- **MCP not showing in sidebar**: Restart Cursor completely
- **Tool call failures**: Check that the binary path is absolute (not relative)

---

## VS Code (with MCP Extension)

VS Code requires the "MCP" extension to use the protocol.

### STDIO Mode

1. **Install the MCP extension** for VS Code

2. **Open Settings (JSON)** and add:
   ```json
   {
     "mcp.servers": {
       "myadmin-client": {
         "command": "/path/to/bin/mcp-proxy-client",
         "env": {
           "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
           "API_BASE_URL": "https://my.interserver.net/apiv2",
           "API_KEY": "your_api_key_here"
         }
       }
     }
   }
   ```

3. **Reload VS Code**

### Streamable HTTP Mode

```json
{
  "mcp.servers": {
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

---

## Zed

Zed is a high-performance, GPU-accelerated code editor.

### STDIO Mode

1. **Open Zed Settings** → **MCP**

2. **Add the server configuration:**
   ```json
   {
     "mcpServers": {
       "myadmin-client": {
         "command": "/path/to/bin/mcp-proxy-client",
         "env": {
           "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
           "API_BASE_URL": "https://my.interserver.net/apiv2",
           "API_KEY": "your_api_key_here"
         }
       }
     }
   }
   ```

3. **Restart Zed**

### Streamable HTTP Mode

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

---

## JetBrains IDEs

JetBrains IDEs (IntelliJ IDEA, PyCharm, WebStorm, etc.) require the MCP plugin.

### STDIO Mode

1. **Install the MCP plugin** from JetBrains Marketplace

2. **Open Settings** → **MCP**

3. **Add a new server:**
   - **Name:** `myadmin-client`
   - **Type:** Command
   - **Command:** `/path/to/bin/mcp-proxy-client`
   - **Environment variables:**
     ```
     OPENAPI_SPEC_URL=https://my.interserver.net/spec/openapi.yaml
     API_BASE_URL=https://my.interserver.net/apiv2
     API_KEY=your_api_key_here
     ```

4. **Click OK and restart IDE**

### Streamable HTTP Mode

- **Type:** HTTP URL
- **URL:** `https://mcp.example.com/mcp`
- **Headers:** `X-API-KEY: your_api_key_here`

### Troubleshooting

- **Plugin not found**: Search "MCP" in JetBrains Marketplace
- **Environment variables not persisting**: Use the IDE's MCP settings UI, not shell environment

---

## Sourcegraph Cody

Cody is Sourcegraph's AI coding assistant.

### STDIO Mode

1. **Open Sourcegraph Cody settings**

2. **Configure custom MCP server:**
   ```json
   {
     "cody.mcpServers": {
       "myadmin-client": {
         "command": "/path/to/bin/mcp-proxy-client",
         "env": {
           "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
           "API_BASE_URL": "https://my.interserver.net/apiv2",
           "API_KEY": "your_api_key_here"
         }
       }
     }
   }
   ```

### Streamable HTTP Mode

```json
{
  "cody.mcpServers": {
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

---

## GitHub Copilot

GitHub Copilot uses a different architecture but can be configured through its extensibility points.

### Configuration

Copilot doesn't have direct MCP support. For Copilot integration, consider:

1. **Using Copilot in combination with Claude Desktop** - Run Claude Desktop separately for MCP tools
2. **Building a Copilot extension** that calls the MCP server

### Alternative Approach

Use the MCP client as a standalone tool and reference its output in Copilot conversations.

---

## Tabnine

Tabnine is an AI coding assistant that supports MCP through its extension API.

### STDIO Mode

1. **Install Tabnine extension** for your IDE

2. **Configure Tabnine to use MCP tools** via settings:
   ```json
   {
     "tabnine.mcp.enabled": true,
     "tabnine.mcp.servers": {
       "myadmin-client": {
         "command": "/path/to/bin/mcp-proxy-client",
         "env": {
           "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
           "API_BASE_URL": "https://my.interserver.net/apiv2",
           "API_KEY": "your_api_key_here"
         }
       }
     }
   }
   ```

### Streamable HTTP Mode

```json
{
  "tabnine.mcp.servers": {
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

---

## Codeium

Codeium offers AI code completion and chat.

### Configuration

Codeium's MCP support is available through its Windsurf editor or Codeium extension:

1. **Install Codeium extension** or use **Windsurf**

2. **Configure MCP settings** similar to other editors:
   ```json
   {
     "codeium.mcp.servers": {
       "myadmin-client": {
         "command": "/path/to/bin/mcp-proxy-client",
         "env": {
           "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
           "API_BASE_URL": "https://my.interserver.net/apiv2",
           "API_KEY": "your_api_key_here"
         }
       }
     }
   }
   ```

### Streamable HTTP Mode

```json
{
  "codeium.mcp.servers": {
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

---

## Replit

Replit is an online IDE with AI capabilities.

### MCP Integration

1. **Create a new Replit** or open an existing one

2. **Add a `.replit` file** or configure through Replit's MCP settings:
   ```toml
   [mcp]
   
   [[mcp.servers]]
   name = "myadmin-client"
   type = "stdio"
   command = "/path/to/bin/mcp-proxy-client"
   
   [mcp.servers.myadmin-client.env]
   OPENAPI_SPEC_URL = "https://my.interserver.net/spec/openapi.yaml"
   API_BASE_URL = "https://my.interserver.net/apiv2"
   API_KEY = "your_api_key_here"
   ```

3. **For Streamable HTTP:**
   ```toml
   [[mcp.servers]]
   name = "myadmin-client"
   type = "streamable-http"
   url = "https://mcp.example.com/mcp"
   
   [mcp.servers.myadmin-client.headers]
   X-API-KEY = "your_api_key_here"
   ```

---

## Supermaven

Supermaven is a fast AI code completion tool.

### Configuration

Supermaven MCP configuration is typically done through its settings:

```json
{
  "supermaven.mcp.enabled": true,
  "supermaven.mcp.servers": {
    "myadmin-client": {
      "command": "/path/to/bin/mcp-proxy-client",
      "env": {
        "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
        "API_BASE_URL": "https://my.interserver.net/apiv2",
        "API_KEY": "your_api_key_here"
      }
    }
  }
}
```

### Streamable HTTP Mode

```json
{
  "supermaven.mcp.servers": {
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

---

## Morph

Morph is an AI-powered development environment.

### STDIO Mode

1. **Open Morph settings** → **MCP Servers**

2. **Add new server:**
   ```json
   {
     "name": "myadmin-client",
     "type": "stdio",
     "command": "/path/to/bin/mcp-proxy-client",
     "env": {
       "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
       "API_BASE_URL": "https://my.interserver.net/apiv2",
       "API_KEY": "your_api_key_here"
     }
   }
   ```

### Streamable HTTP Mode

```json
{
   "name": "myadmin-client",
   "type": "streamable-http",
   "url": "https://mcp.example.com/mcp",
   "headers": {
     "X-API-KEY": "your_api_key_here"
   }
}
```

---

## Aider

Aider is a command-line AI coding assistant.

### STDIO Mode

```bash
# Using environment variables
export OPENAPI_SPEC_URL="https://my.interserver.net/spec/openapi.yaml"
export API_BASE_URL="https://my.interserver.net/apiv2"
export API_KEY="your_api_key_here"

aider --mcp myadmin-client --mcp-command /path/to/bin/mcp-proxy-client
```

### Streamable HTTP Mode

```bash
aider --mcp myadmin-client --mcp-url https://mcp.example.com/mcp --mcp-header "X-API-KEY: your_api_key_here"
```

### Alternative: Use with `--mcp-uds` (Unix Domain Socket)

```bash
aider --mcp myadmin-client --mcp-uds /tmp/mcp.sock
```

---

## Goose

Goose is an open-source AI agent by Block.

### Configuration

Create `~/.config/goose/mcp_servers.json`:

```json
{
  "mcpServers": {
    "myadmin-client": {
      "command": "/path/to/bin/mcp-proxy-client",
      "env": {
        "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
        "API_BASE_URL": "https://my.interserver.net/apiv2",
        "API_KEY": "your_api_key_here"
      }
    }
  }
}
```

### Streamable HTTP Mode

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

---

## Roo Code (Continue)

Roo Code (formerly Continue) is a VS Code extension for AI pair programming.

### STDIO Mode

1. **Install Roo Code extension** in VS Code

2. **Edit `.continue/mcp_config.json`** in your project:
   ```json
   {
     "mcpServers": {
       "myadmin-client": {
         "command": "/path/to/bin/mcp-proxy-client",
         "env": {
           "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
           "API_BASE_URL": "https://my.interserver.net/apiv2",
           "API_KEY": "your_api_key_here"
         }
       }
     }
   }
   ```

3. **Or configure via VS Code settings:**
   ```json
   {
     "continue.mcpServers": {
       "myadmin-client": {
         "command": "/path/to/bin/mcp-proxy-client",
         "env": {
           "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
           "API_BASE_URL": "https://my.interserver.net/apiv2",
           "API_KEY": "your_api_key_here"
         }
       }
     }
   }
   ```

### Streamable HTTP Mode

```json
{
  "continue.mcpServers": {
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

---

## CodeRabbit

CodeRabbit is an AI code reviewer for pull requests.

### Configuration

CodeRabbit uses MCP for tool integrations:

1. **Configure via CodeRabbit settings** or `.coderabbit.yaml`:

   ```yaml
   mcp:
     servers:
       myadmin-client:
         type: stdio
         command: /path/to/bin/mcp-proxy-client
         env:
           OPENAPI_SPEC_URL: https://my.interserver.net/spec/openapi.yaml
           API_BASE_URL: https://my.interserver.net/apiv2
           API_KEY: your_api_key_here
   ```

### Streamable HTTP Mode

```yaml
mcp:
  servers:
    myadmin-client:
      type: streamable-http
      url: https://mcp.example.com/mcp
      headers:
        X-API-KEY: your_api_key_here
```

---

## Greptile

Greptile is an AI tool for codebase navigation and understanding.

### Configuration

Configure via Greptile settings:

```json
{
  "mcpServers": {
    "myadmin-client": {
      "command": "/path/to/bin/mcp-proxy-client",
      "env": {
        "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
        "API_BASE_URL": "https://my.interserver.net/apiv2",
        "API_KEY": "your_api_key_here"
      }
    }
  }
}
```

### Streamable HTTP Mode

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

---

## Bugasaki

Bugasaki is an AI debugging assistant.

### Configuration

```json
{
  "bugasaki.mcp": {
    "myadmin-client": {
      "command": "/path/to/bin/mcp-proxy-client",
      "env": {
        "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
        "API_BASE_URL": "https://my.interserver.net/apiv2",
        "API_KEY": "your_api_key_here"
      }
    }
  }
}
```

### Streamable HTTP Mode

```json
{
  "bugasaki.mcp": {
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

---

## Fathom

Fathom is an AI coding assistant focused on code explanation.

### Configuration

Configure via Fathom settings:

```json
{
  "mcp.servers": {
    "myadmin-client": {
      "command": "/path/to/bin/mcp-proxy-client",
      "env": {
        "OPENAPI_SPEC_URL": "https://my.interserver.net/spec/openapi.yaml",
        "API_BASE_URL": "https://my.interserver.net/apiv2",
        "API_KEY": "your_api_key_here"
      }
    }
  }
}
```

### Streamable HTTP Mode

```json
{
  "mcp.servers": {
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

---

## Common Issues

### Binary Not Executable

**Error:** `Permission denied` or `cannot execute binary`

**Solution:**
```bash
chmod +x /path/to/bin/mcp-proxy-client
```

### Environment Variables Not Loading (STDIO)

**Error:** Tools fail with authentication errors

**Solution:**
- Ensure environment variables are set in the JSON config, not shell
- Check for typos in variable names (case-sensitive)
- Verify no extra whitespace

### Connection Refused (HTTP Mode)

**Error:** `Connection refused` when calling MCP tools

**Solution:**
1. Verify the MCP server is running:
   ```bash
   curl https://mcp.example.com/mcp/health
   ```
2. Check firewall rules
3. Ensure correct URL in configuration

### Auth Failures

**Error:** `401 Unauthorized` or `Authentication failed`

**Solution:**
1. Verify API key is correct
2. Check API key hasn't expired
3. Ensure `X-API-KEY` header is correctly formatted
4. For STDIO mode, verify `API_KEY` env var is set

### Binary Path Issues

**Error:** `ENOENT: no such file or directory`

**Solution:**
- Use absolute paths, not relative paths
- On Windows, use forward slashes or escaped backslashes
- Verify the binary exists at the specified path

### macOS Security Warning

**Error:** "mcp-proxy-client cannot be opened because it is from an unidentified developer"

**Solution:**
1. Go to **System Preferences** → **Security & Privacy**
2. Click **Open Anyway** for mcp-proxy-client
3. Or run: `xattr -d com.apple.quarantine /path/to/bin/mcp-proxy-client`

### Streaming Timeout

**Error:** Request timeout in Streamable HTTP mode

**Solution:**
1. Check network latency
2. Verify server isn't overloaded
3. Consider using STDIO mode for lower latency

---

## Environment Variables Reference

| Variable | Description | Example |
|----------|-------------|---------|
| `OPENAPI_SPEC_URL` | URL to OpenAPI specification | `https://my.interserver.net/spec/openapi.yaml` |
| `API_BASE_URL` | Base URL for API requests | `https://my.interserver.net/apiv2` |
| `API_KEY` | Your API authentication key | `sk_live_xxxxxxxxxxxx` |
| `HTTP_TIMEOUT` | Request timeout in seconds (optional) | `30` |
| `LOG_LEVEL` | Logging level: debug, info, warn, error (optional) | `info` |

---

## Security Best Practices

1. **Never commit API keys to version control**
   - Use environment variables or secrets management
   - Add `bin/mcp-proxy-client` to `.gitignore` if it contains secrets

2. **Use HTTPS for all HTTP connections**
   - Never use plain HTTP in production
   - Verify SSL certificates

3. **Restrict file permissions**
   ```bash
   chmod 600 ~/.config/*/mcp*.json
   ```

4. **Rotate API keys regularly**
   - Generate new keys from your MyAdmin panel
   - Update configurations promptly

5. **Use the principle of least privilege**
   - Only grant necessary permissions to the API key
   - Use separate keys for different environments

---

## Getting Help

If you encounter issues not covered here:

1. Check the [main README](README.md) for troubleshooting
2. Review the [PLAN_GO.md](PLAN_GO.md) for implementation details
3. Open an issue on the project repository
