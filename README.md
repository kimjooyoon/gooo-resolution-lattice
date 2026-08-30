# Gooo Resolution Lattice

Gooo Resolution Lattice is a small, deterministic meta-program for lowering
claims without laundering uncertainty. It preserves an unresolved high-
resolution claim while it searches for a true lower-resolution claim:

```text
EXACT -> INVARIANT -> EXISTENCE
```

Every claim starts as `OPEN`. Supporting evidence appends `DISCHARGED` and
contradicting evidence appends `REFUTED`; no claim is deleted. Missing evidence
is an explicit six-coordinate `UNKNOWN` record containing `stage`, `step`,
`reason`, `unknown_class`, `next_operation`, and `blocked_by`.

The top-level state uses strict precedence:

```text
REFUTED > UNKNOWN > CLOSED
```

`FIXED_POINT`, an arbitrary parent resolution, malformed input, and requested
authority escalation are fail-closed. A lower-resolution closed claim does not
erase an unresolved exact claim.

The runtime is read-only with respect to its input. It writes only reports and
receipts beneath the caller-provided output directory. The `.gooo` source,
fixed denominator, generated semantic IR, generated receipts, and evaluator
are all bound in the CI report. Each descent edge carries a generated receipt
and a minimal causal frontier.

## Repository layout

- `internal/lattice`: the Go 1.27 resolution engine and its unit tests.
- `examples/resolution-lattice/main.gooo`: the twelve Gooo meta activities.
- `contracts/resolution-lattice-denominator-v1.json`: the fixed 12-cell
  denominator, balanced across proof choices and indicator classes.
- `cmd/gooo-lattice`: the `resolve` and `compile` commands used by CI.
- `scripts/evaluate.sh`: fail-closed binding and report generation.
- `.github/workflows/conformance.yml`: the only place validation is run.

The closed example intentionally includes normal, UNKNOWN, REFUTED, malformed,
FIXED_POINT, privilege-escalation, and missing-improvement-pair cases.

## Verification boundary

The implementation PR is verified by GitHub Actions with Go 1.27. Local
build/test/format/vet commands are intentionally not part of the workflow used
to author this repository; the CI artifact records `local_tests_run: 0`.
