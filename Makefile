.PHONY: test cover cover-report cover-enforce

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
