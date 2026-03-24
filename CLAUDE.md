# Project Instructions

## Project Structure

- `cmd/` — Application entry points
- `internal/` — Private application packages
- `docs/` — Documentation
- `plan/` — Executed implementation plans

## Reference Projects

These sibling projects are the basis for this MCP server. Research them before planning.

- `../openclaw/` — The main OpenClaw application (TypeScript monorepo). Contains the core domain logic, extensions, skills, and source code that this MCP server will expose.
- `../openclaw-mcp/` — An existing OpenClaw MCP implementation (TypeScript/Node). Reference for MCP protocol usage, tool definitions, and plugin structure.

## Coding Conventions

- Go
- Follow existing patterns in the codebase — check how similar features are implemented before writing new code.
- Never hardcode secrets or credentials. Secrets go in `.env`, non-secret config in `config.toml`.
- When modifying any logic, proactively search the codebase for similar patterns that should receive the same change.

## Testing

- When code is added or modified, write or update test cases covering the changes.
- Run the relevant tests to verify they pass before committing.
- Tests: run with `go test ./...`.
- Test both happy paths and error/edge cases (invalid input, missing data, unauthorized access).
- Aim for maximum test coverage: every public function, every branch, every error path.

## Documentation

When code changes affect behavior, APIs, architecture, or configuration, update the relevant documentation in `docs/` and any affected `README.md` files to stay in sync.

## Git

- Do not push to remote unless explicitly asked.
- Write clear, descriptive commit messages — lead with what changed and why, not how.
- Do not commit `.env`, credentials, or large binary files.
- Never merge directly to `main` with `git merge`. Always push the feature branch and create a pull request via `gh pr create`.
- Before merging any PR, check CI status with `gh pr checks <number>`. If any checks fail, fix the errors and push before merging.
- Do not commit every small change individually. Batch related small fixes into a single meaningful commit. Only commit when a logical unit of work is complete.

## Plans

For substantial code changes — new features, re-architecting, multi-file refactors, new integrations, etc. — always enter plan mode first and write a detailed plan before any implementation. Get user approval on the plan before proceeding.

Before writing a new plan, review existing plans in `plan/` to ensure consistency.

## Development Workflow

Follow this process exactly for every plan. Do NOT skip steps or batch them.

**Step A — Plan**
1. Read existing plans in `plan/` to check for reusable patterns and established conventions.
2. Write a detailed plan with phases, files, and architecture decisions.
3. Include a **testing plan** for each phase.
4. **Self-review the plan** before presenting to the user.
5. Get user approval before any implementation.

**Step B — Save the plan (before ANY code)**
1. Save the plan to `plan/` with a numbered prefix (e.g., `plan/003-feature-name.md`). Check existing files to determine the next number.
2. Commit the plan file immediately. No implementation code may be written before the plan is saved and committed.

**Step C — For EACH phase (repeat per phase, do not batch):**
1. **Implement** — write the code for this phase only.
2. **Test** — write or update unit test cases. Run all relevant tests (`go test ./...`). Fix any failures before proceeding.
3. **Review** — re-read all modified files. Verify consistency, no dead code, no duplicate logic.
4. **Document** — update `docs/` and affected `README.md` files for this phase's changes.
5. **Commit** — only if all tests pass and review is clean.

**Step D — Overall review**
1. After all phases are complete, review the full implementation against the plan for consistency.
2. Verify unit tests meet the testing plan.
3. If any issues are found, fix them following Step C.

**Step E — Update plan**
1. Update the plan file with post-execution reports.
2. Update `TODO.md` if applicable.

**Step F — Final commit and PR**
1. Commit the updated plan file.
2. Run ALL tests (`go test ./...`) one final time. Do not proceed if any test fails.
3. Push the branch and create a PR via `gh pr create`.
