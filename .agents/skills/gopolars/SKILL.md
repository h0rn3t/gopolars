```markdown
# gopolars Development Patterns

> Auto-generated skill from repository analysis

## Overview
This skill teaches the core development patterns, coding conventions, and key workflows used in the `gopolars` Go codebase. It covers file organization, naming conventions, import/export styles, and how to contribute benchmarks and tests effectively. The guide is designed to help both new and experienced contributors maintain consistency and quality across the project.

## Coding Conventions

### File Naming
- Use **snake_case** for all file names.
  - Example:  
    ```
    data_frame.go
    string_utils.go
    ```

### Import Style
- Use **relative imports** within the project.
  - Example:
    ```go
    import (
        "github.com/gopolars/pkg/series"
        "./utils"
    )
    ```

### Export Style
- Use **named exports** for functions, types, and variables.
  - Example:
    ```go
    // In series.go
    package series

    type Series struct {
        // fields
    }

    func NewSeries(data []int) *Series {
        // implementation
    }
    ```

## Workflows

### Add or Update Benchmarks
**Trigger:** When you want to add new performance benchmarks or update benchmark results.  
**Command:** `/add-benchmark`

1. Edit or add files under `bench/` to include new benchmark data, tests, or results.
2. Update documentation or summary files (e.g., `.md`, `.json`, `.csv`) in `bench/` or `docs/performance/`.
3. Optionally update related Go implementation files in `pkg/` to support new benchmarks.
4. Update or run scripts (such as `run-bench.sh`) to generate or validate results.
5. Update CI workflow files if benchmark automation is needed.

**Example:**
```bash
# Add a new benchmark result
vim bench/new_benchmark.csv

# Update summary documentation
vim docs/performance/benchmarks.md

# Run benchmark script
./run-bench.sh
```

### Expand Test Coverage
**Trigger:** When you want to improve or expand test coverage for the codebase.  
**Command:** `/expand-tests`

1. Add new `*_test.go` files or expand existing test files across multiple `pkg/` subdirectories.
2. Update scripts or the `Makefile` to support new tests or coverage reporting.
3. Update CI workflow files to ensure new tests are run.

**Example:**
```go
// In pkg/series/series_test.go
package series

import "testing"

func TestNewSeries(t *testing.T) {
    s := NewSeries([]int{1, 2, 3})
    if s.Len() != 3 {
        t.Errorf("Expected length 3, got %d", s.Len())
    }
}
```
```bash
# Run tests
go test ./pkg/...
```

## Testing Patterns

- Test files are named with the pattern `*_test.go`.
- The testing framework is not explicitly specified; standard Go `testing` is assumed.
- Place tests alongside implementation files within the same package directory.

**Example:**
```go
// In pkg/utils/math_test.go
package utils

import "testing"

func TestAdd(t *testing.T) {
    if Add(2, 3) != 5 {
        t.Error("Add(2, 3) should be 5")
    }
}
```

## Commands

| Command         | Purpose                                              |
|-----------------|------------------------------------------------------|
| /add-benchmark  | Add or update performance benchmarks and documentation|
| /expand-tests   | Add or expand test coverage across the codebase      |
```
