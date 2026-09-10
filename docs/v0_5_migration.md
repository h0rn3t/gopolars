## v0.5.0 Migration Notes

Target release: **v0.5.0** — the next release after the `v0.4.1` tag. (Note: `v0.6`/`v0.7`/`v0.9`
elsewhere in `docs/` and `test/conformance/` are *conformance wave* numbers, not releases — see
[`versioning_policy.md`](versioning_policy.md).)

### **BREAKING**: the minimum Go version is now 1.27

`go.mod` moved from `go 1.26.1` to `go 1.27`. A module that builds gopolars with Go 1.26 or older
will now fail to compile — this is the only compatibility break in the release.

**Why.** The portable vector kernels added in this release (`pkg/simd/vec_simd.go`) are written
against the Go 1.27 standard-library `simd` package. Those types are vector-length-agnostic, so a
single body of Go covers NEON on `arm64`, AVX/AVX2/AVX512 on `amd64`, and a pure-Go emulation
elsewhere — which is what finally gives `arm64` accelerated reductions without shipping
hand-written assembly for it.

**How to check your code.**

```bash
go version          # must report go1.27 or newer
go build ./...      # against gopolars v0.5.0
```

If you are pinned to Go 1.26, stay on `v0.4.1` until you can upgrade the toolchain:

```bash
go get github.com/h0rn3t/gopolars@v0.4.1
```

**What is unchanged: the public API.** The exported surface of `pkg/polars` is byte-for-byte
identical to `v0.4.1` — no signature, type, or name changed. Ordinary use needs no source changes
beyond the toolchain upgrade. The `v0.4.0` migration notes still cover the one behavioral breaking
change in the `v0.4`/`v0.5` line (`DataFrame.Clone` shares column buffers).

Note that the vector kernels are **opt-in at build time**: the default `go build` is unaffected and
keeps the existing runtime-dispatched AVX2 path on `amd64`. Only `GOEXPERIMENT=simd` compiles them.

### Highlights

Non-breaking, but user-visible:

- **Portable vector kernels under `GOEXPERIMENT=simd`.** Measured on Apple M4 Pro (arm64/NEON,
  2 float64 lanes — wider vectors on x86 should do better), 1M float64:
  `MinMaxWhereFloat64` 4178.3 µs → 435.6 µs (−90%), `SumWhereFloat64` 547.5 µs → 267.8 µs (−51%),
  `MaxFloat64` 277.8 µs → 133.8 µs (−52%), `MinFloat64` 272.0 µs → 151.0 µs (−44%).
  On `amd64` the vector path sits *below* the AVX2 gate: the hand-written assembly keeps priority
  and the portable kernels serve as the pre-AVX2 fallback. See [`pkg/simd/doc.go`](../pkg/simd/doc.go).
- **Rolling regression under Go 1.27 fixed.** `rollSumState.add`/`remove`/`total` moved from pointer
  to value receivers. A pointer receiver made the caller's accumulator address-taken, pinning all
  four fields to stack slots and turning every windowed step into a load-modify-store chain the next
  step had to wait on. On Go 1.27 that chain cost 5–7× what it cost on 1.26.1 for the same
  arithmetic, taking `rolling_sum`/`rolling_mean` from 4.9 ms to 12 ms at 1M rows / window 100.
  By value the fields stay in registers on both toolchains. The arithmetic — Kahan compensation,
  null and NaN counting — is unchanged.
- **Internal deduplication, −203 lines of code** with no behavior change: the per-dtype comparison
  ladders in `pkg/expr`, `pkg/expr/evalbatch`, `pkg/frame` and `pkg/exec` collapsed onto generic
  `cmpOrdered`/`compareOrdered` helpers; the six copies of the string/list/datetime namespace map
  loop in `pkg/polars/series_namespace.go` onto one `mapValues[T]`; `evalCast`'s five cast arms onto
  `castRows[T]`. `Series.Std` now delegates to `Var`, and `Series.Max`/`Min` no longer make a second
  full pass over the column in the common case.

### Migration guidance

- Upgrade your toolchain to **Go 1.27+**. That is the whole migration.
- No API signatures changed; recompilation is enough.
- `cmp.Compare` was deliberately *not* adopted in the comparison helpers: it orders NaN below every
  other value, whereas gopolars treats any NaN pair as equal for ordering. Sort results are
  unchanged.
