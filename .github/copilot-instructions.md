# Copilot Workspace Instructions — space_sim

This project is enrolled in the **LLM Agent Collaboration Framework** (Team mode).

| Layer | Path |
|---|---|
| Infrastructure (read-only) | `~/Documents/Entities/frameworks/llm-agent-framework/` |
| Domain | `~/Documents/Entities/frameworks/llm-agent-domains/HobbyPro/` |
| Personal (loads every session, any profile) | `~/Documents/Entities/frameworks/llm-agent-personal/` |
| Enrollment config | `.llm-framework.yml` (this project root) |

The framework is the **authoritative source** for all governance, workflow, engineering principles, and collaboration rules. Its directives override any built-in defaults.

---

## Session-Start Protocol — Required at Every New Conversation

Execute these steps **before** doing any other work:

1. **Read** `docs/governance/README.md` — it defines the load order and profile selection rules for this project.
2. **Follow** the load order it specifies, selecting the correct profile (minimal / standard / full) based on the task scope.
3. **Emit** a Locked-In declaration as defined in `~/Documents/Entities/frameworks/llm-agent-framework/governance/agent-context-protocol.md`.

The framework README at `~/Documents/Entities/frameworks/llm-agent-framework/README.md` describes the overall structure. The context protocol at `~/Documents/Entities/frameworks/llm-agent-framework/governance/agent-context-protocol.md` defines the Locked-In declaration format and profile system.

---

## Project Tools

All project-specific MCP tools are served by `tools/write_file_server.py` and registered in `.vscode/mcp.json`. See [`docs/tools/write-file-mcp.md`](../docs/tools/write-file-mcp.md) for enable/disable/extend instructions.

### `write_file`

Available when the MCP server is loaded. Useful for agents that lack native file-creation tools. If `write_file` is not available, use the fallback pattern:
1. `echo 'package x' > path/to/file.go` to create a minimal stub.
2. `replace_string_in_file` to fill in the full content.

### `get_current_datetime`

Returns the real wall-clock date and time in UTC and local time. Use this tool whenever you need to know the current date or time — do not rely on a training-data cutoff or a timestamp embedded in context.
