---
name: add-or-update-benchmarks
description: Workflow command scaffold for add-or-update-benchmarks in gopolars.
allowed_tools: ["Bash", "Read", "Write", "Grep", "Glob"]
---

# /add-or-update-benchmarks

Use this workflow when working on **add-or-update-benchmarks** in `gopolars`.

## Goal

Add new benchmarks or update existing benchmark results and documentation.

## Common Files

- `bench/**`
- `docs/performance/**`
- `pkg/**.go`
- `run-bench.sh`
- `.github/workflows/*.yml`

## Suggested Sequence

1. Understand the current state and failure mode before editing.
2. Make the smallest coherent change that satisfies the workflow goal.
3. Run the most relevant verification for touched files.
4. Summarize what changed and what still needs review.

## Typical Commit Signals

- Edit or add files under bench/ to include new benchmark data, tests, or results.
- Update documentation or summary files (e.g., .md, .json, .csv) in bench/ or docs/performance/.
- Optionally update related Go implementation files in pkg/ to support new benchmarks.
- Update or run scripts (e.g., run-bench.sh) to generate or validate results.
- Update CI workflow files if benchmark automation is needed.

## Notes

- Treat this as a scaffold, not a hard-coded script.
- Update the command if the workflow evolves materially.