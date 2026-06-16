# Copilot Workspace Instructions — space_sim

This project uses the LLM Agent Collaboration Framework. The framework is the authoritative source for all governance, workflow, engineering principles, and collaboration rules.

**Session-start load order:** `docs/governance/README.md`

The Locked-In declaration format is defined in `llm-agent-framework/governance/agent-context-protocol.md`.

---

## Project Tools

All project-specific MCP tools are served by `tools/write_file_server.py` and registered in `.vscode/mcp.json`. See [`docs/tools/write-file-mcp.md`](../docs/tools/write-file-mcp.md) for enable/disable/extend instructions.

### `write_file`

Available when the MCP server is loaded. Useful for agents that lack native file-creation tools. If `write_file` is not available, use the fallback pattern:
1. `echo 'package x' > path/to/file.go` to create a minimal stub.
2. `replace_string_in_file` to fill in the full content.

### `get_current_datetime`

Returns the real wall-clock date and time in UTC and local time. Use this tool whenever you need to know the current date or time — do not rely on a training-data cutoff or a timestamp embedded in context.
