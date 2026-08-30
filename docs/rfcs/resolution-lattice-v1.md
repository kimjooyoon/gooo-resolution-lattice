# Resolution Lattice v1

## Contract

The lattice has one legal starting resolution, `EXACT`, and two deterministic
lower-resolution edges:

```text
EXACT --DescendExactToInvariant--> INVARIANT
INVARIANT --DescendInvariantToExistence--> EXISTENCE
```

At every resolution the evaluator emits an `OPEN` history event before
examining evidence. `SUPPORTS` appends `DISCHARGED`; `CONTRADICTS` appends
`REFUTED`; `MISSING` leaves the claim `UNKNOWN` and retains its six
coordinates. The top-level state is computed with `REFUTED > UNKNOWN > CLOSED`.

An edge is valid only when its receipt binds the case input digest, tool
digest, contract digest, Gooo source digest, activity, semantic IR node, and
output claim digest. Its minimal causal frontier is exactly the source and
destination `OPEN` event IDs. The frontier is not padded with unrelated
ancestors.

## Improvement claims

An improvement claim is `CLOSED` only when the input, tool, and contract
digests are present and an exact before/after pair is present. Utility evidence
alone is insufficient. Missing pair evidence is `UNKNOWN` with
`CAUSALITY_UNPROVEN` and the required six coordinates.

## Fail-closed boundary

Malformed JSON, malformed input contracts, arbitrary parent resolutions,
`FIXED_POINT`, non-read-only authority, and requested repository writes produce
`FAIL_CLOSED` / `REFUTED`. The command never writes beneath the input path; all
reports and receipt bundles are created only below the caller's output path.

## CI evidence

The fixed denominator has 12 cells: four `FOUNDATION`, four `COHERENCE`, and
four `REGRESSION`, with four each of `DRIVER`, `OUTCOME`, and `GUARDRAIL`.
The evaluator binds each cell one-to-one to a Gooo activity, source digest,
generated semantic IR node, generated artifact path, and evaluator path. It
also records physical file/line inventory, runtime resource measurements,
test-cache counts, artifact size, and repository writes as integers.
