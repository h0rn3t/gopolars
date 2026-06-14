---
name: expand-test-coverage
description: Workflow command scaffold for expand-test-coverage in gopolars.
allowed_tools: ["Bash", "Read", "Write", "Grep", "Glob"]
---

# /expand-test-coverage

Use this workflow when working on **expand-test-coverage** in `gopolars`.

## Goal

Add new or expand existing test coverage across multiple packages.

## Common Files

- `pkg/**/**_test.go`
- `scripts/coverage.sh`
- `Makefile`
- `.github/workflows/ci.yml`

## Suggested Sequence

1. Understand the current state and failure mode before editing.
2. Make the smallest coherent change that satisfies the workflow goal.
3. Run the most relevant verification for touched files.
4. Summarize what changed and what still needs review.

## Typical Commit Signals

- Add new *_test.go files or expand existing test files across multiple pkg/ subdirectories.
- Update scripts or Makefile to support new tests or coverage reporting.
- Update CI workflow files to ensure new tests are run.

## Notes

- Treat this as a scaffold, not a hard-coded script.
- Update the command if the workflow evolves materially.