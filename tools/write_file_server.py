#!/usr/bin/env python3
# =============================================================================
# WHY THIS FILE EXISTS
#
# The VS Code Copilot agent's built-in `create_file` tool has a confirmed bug:
# it writes file content in reverse line order and strips newlines between
# lines, producing syntactically invalid output for any multi-line file.
#
# This MCP server exposes a `write_file` tool that simply calls
# open(path, "w").write(content), which works correctly. VS Code loads it via
# .vscode/mcp.json and makes it available to the Copilot agent as a first-class
# tool, replacing `create_file` for all new-file creation tasks in this project.
#
# Nothing below this block is affected by these comments. They are standard
# Python line comments and are ignored by the interpreter at runtime.
# =============================================================================

import json
import os
import sys


def _respond(mid, result):
    print(json.dumps({"jsonrpc": "2.0", "id": mid, "result": result}), flush=True)


def _error(mid, code, message):
    print(json.dumps({"jsonrpc": "2.0", "id": mid,
                       "error": {"code": code, "message": message}}), flush=True)


def _handle(msg):
    method = msg.get("method", "")
    mid = msg.get("id")  # None for notifications — no response sent

    if method == "initialize":
        _respond(mid, {
            "protocolVersion": "2024-11-05",
            "capabilities": {"tools": {}},
            "serverInfo": {"name": "write-file", "version": "1.0.0"},
        })

    elif method == "notifications/initialized":
        pass  # notification; no response

    elif method == "tools/list":
        _respond(mid, {"tools": [{
            "name": "write_file",
            "description": (
                "Write text content to a file at an absolute path, creating "
                "parent directories as needed. Use this instead of create_file."
            ),
            "inputSchema": {
                "type": "object",
                "properties": {
                    "path": {
                        "type": "string",
                        "description": "Absolute path of the file to write.",
                    },
                    "content": {
                        "type": "string",
                        "description": "Full text content to write to the file.",
                    },
                },
                "required": ["path", "content"],
            },
        }]})

    elif method == "tools/call":
        name = msg.get("params", {}).get("name", "")
        args = msg.get("params", {}).get("arguments", {})

        if name != "write_file":
            _error(mid, -32601, f"Unknown tool: {name}")
            return

        path = args.get("path", "").strip()
        content = args.get("content", "")

        if not os.path.isabs(path):
            _respond(mid, {
                "content": [{"type": "text",
                             "text": f"Error: path must be absolute, got: {path!r}"}],
                "isError": True,
            })
            return

        try:
            parent = os.path.dirname(path)
            if parent:
                os.makedirs(parent, exist_ok=True)
            with open(path, "w", encoding="utf-8") as f:
                f.write(content)
            _respond(mid, {
                "content": [{"type": "text", "text": f"Written: {path}"}],
            })
        except OSError as exc:
            _respond(mid, {
                "content": [{"type": "text", "text": f"Error: {exc}"}],
                "isError": True,
            })

    elif mid is not None:
        # Unknown method with an id — reply with method-not-found.
        _error(mid, -32601, f"Method not found: {method}")


def main():
    for raw in sys.stdin:
        raw = raw.strip()
        if not raw:
            continue
        try:
            msg = json.loads(raw)
        except json.JSONDecodeError:
            continue
        _handle(msg)


if __name__ == "__main__":
    main()
