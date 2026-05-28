---
description: Run focused verification for the current change and report the result.
agent: plan
subtask: true
---

Verify the current repository changes. Do not edit files.

If arguments are provided, use them to scope verification: $ARGUMENTS

Workflow:

1. Inspect `git status --short` and the relevant diff.
2. Read `docs/testing.md` if needed to choose the right command.
3. Run the smallest useful verification commands for the touched layer.
4. Include `git diff --check` when files changed.
5. Report commands run, pass/fail status, and the smallest next fix if something fails.

Use these defaults unless the diff clearly calls for a narrower or broader check:

- service changes: `go test ./internal/service`
- handler changes: `go test ./internal/handler`
- web changes: `go test ./internal/web`
- repository SQL changes: `go test ./internal/repository` when `DATABASE_URL` is set
- broad or wiring changes: `go test ./...`
