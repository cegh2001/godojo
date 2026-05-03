# Skill Registry — godojo

Generated: 2026-05-03

## Project Conventions

| Source | Notes |
|--------|-------|
| `.opencodeignore` | Ignores node_modules, .next, .bun, dist, build, *.log, .env* |
| `README.md` | Hexagonal architecture, Bubbletea TUI, Gemini AI, Spanish Rioplatense |
| `backend/go.mod` | Module `godojo`, Go 1.26.2, Bubbletea v1.3.10, Lipgloss v1.1.0 |

No project-level convention files found (no CLAUDE.md, .cursorrules, GEMINI.md).

## Available Skills

### User-Level Skills (from `~/.config/opencode/skills/`)

| Skill | Triggers | Compact Rules Summary |
|-------|----------|----------------------|
| **branch-pr** | Creating a PR, opening a PR, preparing changes for review | PR must link an approved issue. Must have exactly one `type:*` label. Automated checks must pass. Uses `gh pr create`. |
| **go-testing** | Writing Go tests, using teatest, adding test coverage | Table-driven tests standard. Golden files in `testdata/`. Integration: real services + stubs. Bubbletea: `teatest`. Context timeout testing. Graceful degradation tests. |
| **issue-creation** | Creating GitHub issue, bug report, feature request | Blank issues disabled — must use template. Every issue gets `status:needs-review`. Maintainer adds `status:approved` before PR. Questions go to Discussions. |
| **judgment-day** | "judgment day", "dual review", "doble review", "que lo juzguen" | Launches 2 blind judge sub-agents. Synthesizes findings, applies fixes, re-judges up to 2 iterations or escalate. |
| **skill-creator** | Create new skill, add agent instructions, document patterns | Creates `SKILL.md` with YAML frontmatter. Must include name, description, triggers, allowed-tools, and step-by-step workflow. |

### Project-Level Skills

None found.

## SDD Skills (managed by workflow, not manual)

sdd-init, sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-apply, sdd-verify, sdd-archive, sdd-onboard — part of the Spec-Driven Development workflow.

## Resolution

When delegating a sub-agent, resolve the relevant skill by matching trigger context against the task. Inject compact rules into the sub-agent's launch prompt.
