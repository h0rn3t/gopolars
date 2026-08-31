.PHONY: test cover cover-report cover-enforce benchstat bench-simd

# Run the full test suite.
test:
	go test ./...

# Run pkg/... tests with coverage and apply the per-package threshold gate.
# Follows the default MODE baked into scripts/coverage.sh.
cover:
	./scripts/coverage.sh

# Coverage summary that never fails the build (useful while iterating).
cover-report:
	MODE=report ./scripts/coverage.sh

# Coverage gate that fails when any package is below its threshold (CI mode).
cover-enforce:
	MODE=enforce ./scripts/coverage.sh

# ---------------------------------------------------------------------------
# Benchmarking
# ---------------------------------------------------------------------------

# benchstat is run via `go run pkg@version`, which resolves in its own module
# context and so does NOT add golang.org/x/perf to go.mod. That matters here:
# gopolars is a library, and a `tool` directive would push x/perf plus upgraded
# x/net, x/sys, x/text and x/tools minimums into every consumer's module graph
# for the sake of a measurement tool. Override BENCHSTAT to use a local binary.
BENCHSTAT ?= go run golang.org/x/perf/cmd/benchstat@latest

BENCH ?= .
BENCH_PKG ?= ./bench/micro
# benchstat needs >= 6 samples to report a confidence interval at level 0.95;
# below that it prints "± ∞" and no p-value.
BENCH_COUNT ?= 6
# bench/micro sweeps up to 10M rows; the 1s default per benchmark makes a full
# A/B sweep take minutes. Raise this when a result looks noisy.
BENCH_TIME ?= 300ms
BENCH_OUT ?= /tmp/gopolars-bench

# Print the pinned benchstat's usage — also warms the module cache.
benchstat:
	$(BENCHSTAT) -h

# A/B the default build against GOEXPERIMENT=simd, the two views CI builds.
# Override the selection like: make bench-simd BENCH='SumWhere|MinMax'
bench-simd:
	@mkdir -p $(BENCH_OUT)
	go test $(BENCH_PKG) -run '^$$' -bench '$(BENCH)' -count $(BENCH_COUNT) -benchtime $(BENCH_TIME) \
		> $(BENCH_OUT)/default.txt
	GOEXPERIMENT=simd go test $(BENCH_PKG) -run '^$$' -bench '$(BENCH)' -count $(BENCH_COUNT) -benchtime $(BENCH_TIME) \
		> $(BENCH_OUT)/simd.txt
	$(BENCHSTAT) default=$(BENCH_OUT)/default.txt simd=$(BENCH_OUT)/simd.txt
