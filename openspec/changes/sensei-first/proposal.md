# Proposal: Sensei-First — Agentic AI Tutor Refactor

## Intent

Transform GoDojo from a structured exercise platform into an agentic AI pair-programming experience. Gemma-4-31b-it has proven function-calling, code execution, and Google Search grounding capabilities. The current hardcoded exercise system limits personalization; refactoring to an agentic loop lets the sensei dynamically decide what to teach, generate exercises on-demand, and ground answers with live documentation. Student lands directly in chat — the sensei is the core.

## Scope

### In Scope
- Agentic sensei loop: send message → Gemma calls tools → execute locally → send results back → repeat until text response (max 5 rounds, 60s timeout)
- Two local tools: `create_exercise_file` (writes .go skeletons to workspace) and `read_roadmap_section` (reads roadmap.md)
- Built-in Gemma tools declared: GoogleSearch (live grounding) and CodeExecution (declared, wired in Phase 2)
- Simplified TUI: 3 states (SenseiChat, ToolRunning spinner, SessionSelector), chat as default landing
- WorkspaceManager: safe file ops under `~/.godojo/workspace/` with path-escape prevention
- Redefined ports: SenseiProvider, SessionStore, WorkspaceMgr, RoadmapReader
- Mark obsolete adapters/ports/domain types as deprecated (not deleted)

### Out of Scope
- Code execution/evaluation via Gemma's CodeExecution tool (Phase 2)
- Deletion of deprecated code
- Exercise browsing, hint system, test runner UIs

## Capabilities

### New Capabilities
- `sensei-agent-loop`: Orchestrates tool-calling conversation — parses functionCall, dispatches local execution, sends functionResponse, returns final text
- `workspace-manager`: Creates/lists exercise files under `~/.godojo/workspace/` with path validation
- `tool-registry`: Declares available tools to Gemma via `functionDeclarations` array, maps tool names to local execution

### Modified Capabilities
- `chat-provider`: Extended `SendMessage` to accept tools array and return `ContentPart` (text or functionCall); new `SendFunctionResponse` method
- `tui-model`: States reduced 7→3, chat becomes default landing, ToolRunning handles spinner during agent loop
- `ports`: New SenseiProvider, WorkspaceManager interfaces; deprecate ExerciseRepository, HintProvider, TestRunner

## Approach

Extend `gemini/chat_provider.go` to pass `functionDeclarations` in the REST request and parse `functionCall` from response parts. Build `SenseiService` as a loop that feeds function responses back into conversation history until Gemma returns text-only. WorkspaceManager enforces `~/.godojo/workspace/` prefix on all paths. TUI collapses to chat-first flow: ToolRunning state shows spinner while the agent loop executes between user messages. RoadmapService stays as-is.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/gemini/` | Modified | Tools param, functionCall parsing, functionResponse sending |
| `internal/core/services/sensei_service.go` | New | Agent loop orchestrator |
| `internal/core/ports/` | Modified | New ports; deprecate exercise/hint/test ports |
| `internal/adapters/workspace/` | New | Safe file operations |
| `internal/tui/` | Modified | 3-state model, chat landing, tool spinner |
| `cmd/godojo/main.go` | Modified | Wire new services, new system prompt |
| `internal/core/domain/` | Modified | Deprecate exercise.go, hint.go, test_result.go |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Gemma hallucinates invalid Go | Medium | Syntax validation before write; user edits post-generation |
| Infinite function-calling loop | Low | Max 5 rounds, 60s timeout per user message |
| Workspace path escape | Low | Prefix validation on all file paths |
| API latency degrades UX | Medium | Spinner + status messages during tool execution |

## Rollback Plan

Git revert. ChatProvider extensions are backward-compatible (tools optional). Deprecated code marked, not deleted — old and new TUI states coexist during migration.

## Dependencies

- Gemma-4-31b-it function calling (confirmed via Go SDK example)
- Existing ChatStore / session persistence (reuse unchanged)
- `roadmap-golang-guia-estudio.md` in project root

## Success Criteria

- [ ] TUI opens → lands in Sensei Chat
- [ ] Sensei reads roadmap sections via `read_roadmap_section`
- [ ] Sensei generates .go files via `create_exercise_file`, visible in `~/.godojo/workspace/`
- [ ] Sessions persist across restarts (ChatStore tests pass)
- [ ] All reusable-code tests pass (ChatStore, RoadmapService)
- [ ] New code ≥80% coverage
- [ ] No regressions in existing test suite
