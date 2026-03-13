#!/usr/bin/env bash
set -euo pipefail

test -f docs/performance/v0_6_budgets.json
test -f docs/parity/v0_6_coverage.json

go test ./bench/micro -run TestNope -bench . -benchmem >/tmp/gopolars_micro_bench.txt
go test ./bench/macro -run TestNope -bench . -benchmem >/tmp/gopolars_macro_bench.txt

test -s /tmp/gopolars_micro_bench.txt
test -s /tmp/gopolars_macro_bench.txt
