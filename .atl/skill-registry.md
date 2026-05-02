# Skill Registry — mi-proyecto

Generated: 2026-05-01

## Project Conventions

| Source | Notes |
|--------|-------|
| `.opencodeignore` | Ignores node_modules, .next, .bun, dist, build, *.log, .env* |
| `AGENTS.md` | Uses gentle-ai persona: Senior Architect, clean/hexagonal architecture focus, Spanish (Rioplatense) / English, engram persistence |

No project-level convention files found (no CLAUDE.md, .cursorrules, GEMINI.md).

## Available Skills

### User-Level Skills (from `~/.config/opencode/skives/`)

| Skill | Triggers | Compact Rules Summary |
|-------|----------|----------------------|
| **branch-pr** | Creating a PR, opening a PR, preparing changes for review | PR must link an approved issue. Must have exactly one `type:*` label. Automated checks must pass. Uses `gh pr create`. |
| **go-testing** | Writing Go tests, using teatest, adding test coverage | Table-driven tests standard. Use `testing/quick` for property-based tests. Golden files in `testdata/`. Integration: `net/http/httptest`. Bubbletea: `teatest`. |
| **issue-creation** | Creating GitHub issue, bug report, feature request | Blank issues disabled — must use template. Every issue gets `status:needs-review`. Maintainer adds `status:approved` before PR. Questions go to Discussions. |
| **judgment-day** | "judgment day", "dual review", "doble review", "que lo juzguen" | Launches 2 blind judge sub-agents. Synthesizes findings, applies fixes, re-judges up to 2 iterations or escalate. |
| **skill-creator** | Create new skill, add agent instructions, document patterns | Creates `SKILL.md` with YAML frontmatter. Must include name, description, triggers, allowed-tools, and step-by-step workflow. |
| **skill-registry** | "update skills", "skill registry", "actualizar skills" | Scans user skills + project conventions. Writes `.atl/skill-registry.md`. Saves to engram if available. |

## SDD Skills (managed by workflow, not manual)

sdd-init, sdd-explore, sdd-propose, sdd-spec, sdd-design, sdd-tasks, sdd-apply, sdd-verify, sdd-archive, sdd-onboard — part of the Spec-Driven Development workflow.

## Resolution

When delegating a sub-agent, resolve the relevant skill by matching trigger context against the task. Inject compact rules into the sub-agent's launch prompt.
