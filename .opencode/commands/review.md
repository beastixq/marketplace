---
description: Review the current diff for bugs, regressions, boundary issues, and missing tests.
agent: plan
subtask: true
---

Review the current repository changes. Do not edit files.

If arguments are provided, focus on them: $ARGUMENTS

Inspect the smallest useful context first:

- `git status --short`
- `git diff --stat`
- relevant `git diff` hunks and nearby source files
- relevant docs from `AGENTS.md` only when they affect the review

Use a code-review stance:

- Lead with findings ordered by severity.
- Include concrete file and line references.
- Prioritize bugs, behavioral regressions, architecture boundary violations, data leaks across layers, transaction/consistency risks, error handling gaps, and missing tests.
- Keep summaries brief and secondary.
- If there are no findings, say that clearly and mention residual test risk.
