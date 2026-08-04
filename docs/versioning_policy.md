## Versioning Policy

gopolars follows [Semantic Versioning](https://semver.org/). This document states the policy
`README.md` refers to, and — because the repository carries two independent `v0.N` numbering
schemes — spells out which one is a release.

### Release versions

Release versions are git tags: `v0.1.0`, `v0.2.0`, `v0.3.0`, … They are what `go get` resolves.

While the module is below `v1.0.0`:

- **A minor release may contain breaking changes to the public API.** SemVer permits this before
  `1.0.0`, and gopolars uses it: the public surface is still converging on the observable behavior
  of Python Polars.
- **Every breaking change is written down** in a migration note, `docs/v0_N_migration.md`, marked
  with `**BREAKING**`. `scripts/check_breaking_evidence.sh` enforces that such a marker is
  accompanied by a migration note and by this policy document.
- **Patch releases never change the public API** — they carry fixes, performance work, and
  documentation.

The public API is the `pkg/polars` package. Everything else under `pkg/` is exported for
composition inside the module but is treated as internal: it may change in any release without a
migration note.

### Conformance waves are not releases

Separately from release tags, the repository numbers **conformance waves** — the level of parity
reached against Python Polars. These appear as `v0.6`…`v1.0` in:

- `test/conformance/` (e.g. `v07_top30_conformance_test.go`, `v09_wave_b_medium_conformance_test.go`)
- `docs/parity/`, `docs/performance/` (e.g. `v0_6_budgets.json`)
- `docs/v0_6_migration.md`, `docs/release_checklist_v0_6.md`

**A wave number is not a release version.** Wave `v0.6` describes a parity milestone, not the
`v0.6.0` tag — at the time of writing the latest release tag is `v0.3.0`, well behind the wave
numbering. When adding a migration note, name it after the *release* it ships in, not the wave.

### What a breaking change requires

1. A `**BREAKING**` section in `docs/v0_N_migration.md` for the target release, saying what
   changed, why, and how to check calling code.
2. An OpenSpec change whose specs capture the new behavior, so the contract — not just the
   changelog — reflects it.
3. Tests pinning the new behavior, so the migration note's claims are executable rather than
   aspirational.

### Deprecation

There is currently no formal deprecation window or LTS commitment for `v0.x`. Rather than imply a
guarantee that has never been agreed, this policy states the actual practice: breaking changes are
documented and land in a minor release. If a deprecation process is adopted, it belongs here.
