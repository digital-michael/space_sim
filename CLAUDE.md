# Claude Code — space_sim

This project is enrolled in the **LLM Agent Collaboration Framework** (Team mode).

| Layer | Path |
|---|---|
| Infrastructure (read-only) | `~/Projects/active/llm-agent-framework/` |
| Domain | `~/Projects/active/llm-agent-domains/photon-datum/` |
| Enrollment config | `.llm-framework.yml` (this project root) |

The framework is the **authoritative source** for all governance, workflow, engineering principles, and collaboration rules. Its directives override any built-in defaults.

---

## Session-Start Protocol — Required at Every New Conversation

Execute these steps **before** doing any other work:

1. **Read** `docs/governance/README.md` — it defines the load order and profile selection rules for this project.
2. **Follow** the load order it specifies, selecting the correct profile (minimal / standard / full) based on the task scope.
3. **Emit** a Locked-In declaration as defined in `~/Projects/active/llm-agent-framework/governance/agent-context-protocol.md`.

The framework README at `~/Projects/active/llm-agent-framework/README.md` describes the overall structure. The context protocol at `~/Projects/active/llm-agent-framework/governance/agent-context-protocol.md` defines the Locked-In declaration format and profile system.

---

## Project Tools

MCP tools are configured in `.vscode/mcp.json` and served by `tools/write_file_server.py`. Claude Code has native file-editing tools — use the built-in Read/Edit/Write tools for file operations. The MCP server is available for other integrations when needed.
