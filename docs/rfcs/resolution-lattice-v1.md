# Resolution Lattice v2 execution contract

## Fixed resolution ladder

The only legal starting resolution is PROJECT. Lowering is deterministic and
never skips a level:

PROJECT --DescendProjectToArtifact--> ARTIFACT
ARTIFACT --DescendArtifactToActivity--> ACTIVITY
ACTIVITY --DescendActivityToPredicate--> PREDICATE
PREDICATE --DescendPredicateToField--> FIELD

At each visited level the evaluator emits an OPEN history event before
examining evidence. SUPPORTS appends DISCHARGED; CONTRADICTS appends REFUTED;
MISSING or UNKNOWN retains a six-coordinate UNKNOWN claim. A lower-resolution
closed claim is an observation at that level, not a replacement for an
unresolved higher-resolution claim.

## Unknown and refutation rules

The supported UNKNOWN classes are DIRECT_MISSING, DEPENDENCY_BLOCKED,
DECISION_UNKNOWN, and CAUSALITY_UNPROVEN. The first DIRECT_MISSING claim is
preserved as first_direct_cause. Dependency blockers are normalized,
deduplicated, and kept as the minimal dependency-blocked frontier; unrelated
ancestors are not added.

The top-level precedence is REFUTED > UNKNOWN > CLOSED. A known contradiction
therefore wins over any UNKNOWN. Only an explicit FIXED_POINT decision is
accepted. Any other upper decision remains FAIL_CLOSED with the feedback code
FEEDBACK_COVERAGE_DECISION_UNKNOWN. Malformed input, invalid evidence, an
arbitrary parent resolution, and authority escalation are fail-closed.

Each descent edge generates a receipt bound to the case input, tool,
denominator, Gooo source, semantic IR node, and output claim digest. The
receipt also repeats stage, step, reason, unknown_class, next_operation, and
blocked_by from the cause being lowered. The edge frontier contains exactly
the source and destination OPEN events.

## Improvement claims

An improvement claim is UNKNOWN unless both exact before and after values exist
under the same fixture, input, tool, and contract digests. An exact pair may
claim only these two raw metrics:

- unidentified_cause_frontier_count
- minimum_cause_reach_stage_count

No aggregate score is computed, summed, or used to infer utility. Missing or
non-identical pair identity remains UNKNOWN with CAUSALITY_UNPROVEN.

## Fixed denominator and CI boundary

The existing denominator remains fixed at 12 cells: four FOUNDATION, four
COHERENCE, and four REGRESSION cells, with four each of DRIVER, OUTCOME, and
GUARDRAIL. It is an enumeration of independently bound conformance cells,
not a score.

GitHub Actions is the execution authority and installs Go 1.27. The CI
artifact records build/test/conformance wall time and peak RSS,
test total/executed/reused/skipped/not-observed, Go and Gooo physical
file/line inventory, file and descendant-directory counts, repository writes,
and local_tests_run=0. Reports and receipts are written only below the
caller-owned output directory.
