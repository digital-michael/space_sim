# write_file MCP Tool

## Purpose

The VS Code Copilot agent has a built-in `create_file` tool that writes file
content in reverse line order and strips newlines between lines. Any multi-line
file it creates is syntactically invalid. This MCP server replaces it with a
`write_file` tool that writes correctly.

## Files

| File | Role |
|------|------|
| `tools/write_file_server.py` | The MCP server. A single-file, zero-dependency Python 3 script. |
| `.vscode/mcp.json` | Tells VS Code how to launch the server. |

## How to Enable

VS Code loads MCP servers automatically when a `.vscode/mcp.json` is present.
No manual installation or `pip install` is required.

1. Open the workspace in VS Code.
2. Open a Copilot chat (agent mode).
3. VS Code will offer to start the `write-file` server — click **Allow** (or
   set `"chat.mcp.enabled": true` in your settings if it does not appear).
4. The `write_file` tool is now available to the agent.

## How to Disable

Remove or rename `.vscode/mcp.json`. The tool disappears from the agent's
tool list on the next chat session. The server script is not affected.

Alternatively, comment out the `write-file` block inside `mcp.json`:

```json
{
  "servers": {
  }
}
```

## How to Extend

Add new tools inside `_handle()` in `write_file_server.py`:

1. Add a new entry to the `tools/list` response array (name + inputSchema).
2. Add a matching `elif name == "your_tool_name":` branch in the
   `tools/call` handler.

No changes to `mcp.json` are needed — the server is already registered.

## Protocol Notes

- **Transport**: stdio (newline-delimited JSON-RPC). VS Code launches the
  process on demand; no persistent daemon is needed.
- **Protocol version**: `2024-11-05` (MCP spec).
- **Dependencies**: none. The script uses only Python 3 stdlib (`json`, `os`,
  `sys`). Python 3.8+ is sufficient.

## Input

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | Absolute path of the file to write. |
| `content` | string | yes | Full text content. Existing files are overwritten. |

The tool returns an error if `path` is not absolute.
