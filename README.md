# Gooo Resolution Lattice

Gooo Resolution Lattice is a deterministic executable meta-program for
lowering claims without laundering uncertainty. Its fixed resolution ladder is:

PROJECT -> ARTIFACT -> ACTIVITY -> PREDICATE -> FIELD

Every claim starts as OPEN. Supporting evidence appends DISCHARGED and
contradicting evidence appends REFUTED; no claim is deleted. Semantic UNKNOWN
is never treated as FIXED_POINT: the unresolved claim remains at its original
level while the evaluator searches for a lower-resolution claim. Each descent
receipt retains stage, step, reason, unknown_class, next_operation, and
blocked_by.

The top-level state uses strict precedence:

REFUTED > UNKNOWN > CLOSED

Only an explicit FIXED_POINT decision is accepted. An unrecognized upper
decision is FAIL_CLOSED with FEEDBACK_COVERAGE_DECISION_UNKNOWN. Malformed
input, contradictions, an arbitrary parent resolution, and requested authority
escalation are also fail-closed. A lower-resolution closed claim does not erase
an unresolved higher-resolution claim. Known contradictions always outrank
UNKNOWN.

The first direct cause and the minimal dependency-blocked frontier are kept
separately. The fixed conformance corpus covers DIRECT_MISSING,
DEPENDENCY_BLOCKED, DECISION_UNKNOWN, and CAUSALITY_UNPROVEN.

The runtime is read-only with respect to its input. It writes only reports and
receipts beneath the caller-provided output directory. The .gooo source, fixed
denominator, generated semantic IR, generated receipts, and evaluator are all
bound in the CI report.

Improvement is never inferred from a score. Under identical fixture, input,
tool, and contract digests, the only comparable metrics are
unidentified_cause_frontier_count and minimum_cause_reach_stage_count. Without
an exact before/after pair, improvement remains UNKNOWN.

## Repository layout

- internal/lattice: the Go 1.27 resolution engine and its unit tests.
- examples/resolution-lattice/main.gooo: the twelve Gooo meta activities.
- contracts/resolution-lattice-denominator-v1.json: the fixed 12-cell
  denominator, balanced across proof choices and indicator classes.
- cmd/gooo-lattice: the resolve and compile commands used by CI.
- scripts/evaluate.sh: fail-closed binding and report generation.
- .github/workflows/conformance.yml: the only place validation is run.

The closed example includes normal, every UNKNOWN class,
contradiction/refutation, malformed input, invalid FIXED_POINT,
authority-escalation, and missing-improvement-pair cases.

## Verification boundary

The implementation PR is verified by GitHub Actions with Go 1.27. Local tests
are not run; the CI artifact records local_tests_run: 0 and reports build,
test, and conformance wall time and peak RSS, test
total/executed/reused/skipped/not-observed, Go/Gooo files and physical lines,
files/directories, and repository writes.
