#!/usr/bin/env bash
set -euo pipefail

if grep -R "\*\*BREAKING\*\*" openspec/changes openspec/specs docs >/dev/null 2>&1; then
  test -f docs/v0_4_migration.md || test -f docs/v0_5_migration.md || test -f docs/v0_6_migration.md || test -f docs/v0_7_migration.md || test -f docs/v0_8_migration.md
  test -f docs/versioning_policy.md
fi
