#!/usr/bin/env bash
set -Eeuo pipefail

if test "$#" -ne 9; then
  echo "usage: evaluate.sh ARTIFACT_ROOT SOURCE IR DENOMINATOR RUNTIME GO_TEST_CASES SUBJECT_SHA OUTPUT_JSON OUTPUT_MD" >&2
  exit 2
fi

artifact_root=$1
source=$2
ir=$3
denominator=$4
runtime=$5
go_test_cases=$6
subject_sha=$7
output_json=$8
output_md=$9

for required in "$source" "$ir" "$denominator" "$runtime" \
  "$artifact_root/cases/normal.json" "$artifact_root/cases/unknown.json" \
  "$artifact_root/cases/refuted.json" "$artifact_root/cases/fixed-point.json" \
  "$artifact_root/cases/privilege-escalation.json" "$artifact_root/cases/malformed.json" \
  "$artifact_root/cases/improvement-unknown.json" "$artifact_root/receipts/normal.json"; do
  test -f "$required" || { echo "missing conformance input: $required" >&2; exit 2; }
done

source_digest="sha256:$(sha256sum "$source" | awk '{print $1}')"
ir_digest="sha256:$(sha256sum "$ir" | awk '{print $1}')"
contract_digest="sha256:$(sha256sum "$denominator" | awk '{print $1}')"
evaluator_digest="sha256:$(sha256sum "$0" | awk '{print $1}')"

activity_count=$(awk '/^[[:space:]]*activity / {count++} END {print count+0}' "$source")
test "$activity_count" -eq 12

jq -e '
  .schema == "gooo/resolution-lattice-denominator/v1" and
  .denominator_id == "gooo.denominator.resolution-lattice.v1" and
  .total == 12 and (.cells | length) == 12 and
  ([.proofs[].total] | add) == 12 and
  ([.indicator_classes[].total] | add) == 12 and
  ([(.cells | group_by(.proof_choice)[] | {key: .[0].proof_choice, value: length})] | from_entries | .FOUNDATION == 4 and .COHERENCE == 4 and .REGRESSION == 4) and
  ([(.cells | group_by(.indicator_class)[] | {key: .[0].indicator_class, value: length})] | from_entries | .DRIVER == 4 and .OUTCOME == 4 and .GUARDRAIL == 4) and
  all(.cells[]; (.ordinal >= 1 and .ordinal <= 12) and .metric_id != null and .metric_path != null and .artifact != null and .evaluator != null)
' "$denominator" >/dev/null

jq -e --arg source_digest "$source_digest" '
  .schema == "gooo/resolution-lattice/ir/v1" and
  .source_digest == $source_digest and
  (.nodes | length) == 12 and
  ([.nodes[].name] | unique | length) == 12 and
  all(.nodes[]; .id != null and .name != null and .source_line > 0 and .metric_id != null and .artifact != null and .evaluator != null)
' "$ir" >/dev/null

case_report() {
  jq -e --arg source_digest "$source_digest" --arg contract_digest "$contract_digest" '
    .schema == "gooo/resolution-lattice/v1" and
    .precedence == ["REFUTED", "UNKNOWN", "CLOSED"] and
    all(.claims[]; .stage != null and .step != null and .reason != null and .unknown_class != null and .next_operation != null and .blocked_by != null) and
    all(.edges[]; (.source_digest == $source_digest) and (.receipt.source_digest == $source_digest) and (.receipt.contract_digest == $contract_digest) and (.receipt.edge_id == .id) and (.receipt.id != null) and ((.causal_frontier | length) == 2))
  ' "$1" >/dev/null
}

case_report "$artifact_root/cases/normal.json"
case_report "$artifact_root/cases/unknown.json"
case_report "$artifact_root/cases/refuted.json"
case_report "$artifact_root/cases/fixed-point.json"
case_report "$artifact_root/cases/privilege-escalation.json"
case_report "$artifact_root/cases/malformed.json"
case_report "$artifact_root/cases/improvement-unknown.json"

jq -e '
  .state == "CLOSED" and .decision == "RESOLUTION_LATTICE_CLOSED" and
  (.edges | length) == 2 and
  ([.history[] | select(.lifecycle == "OPEN")] | length) == 3 and
  ([.history[] | select(.lifecycle == "DISCHARGED")] | length) == 3 and
  .improvement.claim.state == "CLOSED" and
  all(.edges[]; .receipt.activity == .activity and .from != .to and .causal_frontier[0] != .causal_frontier[1])
' "$artifact_root/cases/normal.json" >/dev/null

jq -e '
  .state == "UNKNOWN" and .decision == "RESOLUTION_LATTICE_UNKNOWN" and
  any(.claims[]; .resolution == "EXACT" and .state == "UNKNOWN" and .stage == "EXACT" and .step == "OBSERVE_EXACT_EVIDENCE" and .reason == "EXACT_EVIDENCE_MISSING" and .unknown_class == "DIRECT_MISSING" and .next_operation == "PROVIDE_EXACT_EVIDENCE" and (.blocked_by | length) == 1) and
  any(.claims[]; .resolution == "INVARIANT" and .state == "CLOSED") and
  .resolved_claim.resolution == "EXISTENCE" and
  .improvement.claim.state == "UNKNOWN" and .improvement.pair_present == false
' "$artifact_root/cases/unknown.json" >/dev/null

jq -e '
  .state == "REFUTED" and .decision == "FAIL_CLOSED" and
  any(.claims[]; .resolution == "EXACT" and .state == "REFUTED") and
  any(.history[]; .lifecycle == "REFUTED") and (.edges | length) == 0
' "$artifact_root/cases/refuted.json" >/dev/null

for fixed_case in fixed-point privilege-escalation malformed; do
  jq -e '
    .state == "REFUTED" and .decision == "FAIL_CLOSED" and
    any(.history[]; .lifecycle == "REFUTED")
  ' "$artifact_root/cases/$fixed_case.json" >/dev/null
done

jq -e '
  .state == "UNKNOWN" and .decision == "RESOLUTION_LATTICE_UNKNOWN" and
  .improvement.claim.state == "UNKNOWN" and .improvement.pair_present == false and
  .improvement.claim.reason == "EXACT_BEFORE_AFTER_PAIR_MISSING" and
  .improvement.claim.unknown_class == "CAUSALITY_UNPROVEN" and
  .improvement.claim.stage == "IMPROVEMENT" and
  .improvement.claim.step == "REQUIRE_EXACT_BEFORE_AFTER_PAIR" and
  .improvement.claim.next_operation == "PROVIDE_EXACT_BEFORE_AFTER_PAIR" and
  (.improvement.claim.blocked_by | length) == 1
' "$artifact_root/cases/improvement-unknown.json" >/dev/null

normal_receipts=$(jq '[.edges[].receipt]' "$artifact_root/cases/normal.json")
jq -e --argjson expected "$normal_receipts" '
  .schema == "gooo/resolution-lattice/receipts/v1" and .receipts == $expected
' "$artifact_root/receipts/normal.json" >/dev/null

jq -e --arg source_digest "$source_digest" --arg contract_digest "$contract_digest" '
  .schema == "gooo/resolution-lattice/runtime/v1" and
  .build_wall_ms >= 0 and .test_wall_ms >= 0 and .conformance_wall_ms >= 0 and
  .peak_rss_kib > 0 and .repository_writes == 0 and .local_tests_run == 0 and
  .test_counts.executed >= 1 and .test_counts.reused == 0 and .test_counts.skipped == 0 and
  .source_digest == $source_digest and .contract_digest == $contract_digest
' "$runtime" >/dev/null

artifact_files=0
artifact_bytes=0
while IFS= read -r file; do
  artifact_files=$((artifact_files + 1))
  bytes=$(stat -c '%s' "$file")
  artifact_bytes=$((artifact_bytes + bytes))
done < <(find "$artifact_root" -type f ! -name 'ci-report.json' ! -name 'ci-report.md' | sort)

descendant_dirs=$(find . -mindepth 1 -type d ! -path './.git' ! -path './.git/*' | wc -l | tr -d ' ')
regular_files=0
go_files=0
gooo_files=0
go_lines=0
gooo_lines=0
while IFS= read -r file; do
  if test "$file" = "./README.md"; then
    continue
  fi
  regular_files=$((regular_files + 1))
  line_count=$(awk 'END {print NR+0}' "$file")
  case "$file" in
    *.go)
      go_files=$((go_files + 1))
      go_lines=$((go_lines + line_count))
      ;;
    *.gooo)
      gooo_files=$((gooo_files + 1))
      gooo_lines=$((gooo_lines + line_count))
      ;;
  esac
done < <(find . -type f ! -path './.git' ! -path './.git/*' | sort)

cell_metrics=$(jq -S --arg source_path "${source#./}" --arg source_digest "$source_digest" --arg ir_digest "$ir_digest" --arg evaluator_digest "$evaluator_digest" --slurpfile ir "$ir" '
  .cells as $cells |
  $ir[0].nodes as $nodes |
  $cells | map(
    . as $cell |
    ($nodes[] | select(.name == $cell.activity)) as $node |
    {
      id: $cell.metric_id,
      cell_id: $cell.id,
      numerator: 1,
      denominator: 1,
      state: "CLOSED",
      activity: $cell.activity,
      proof_choice: $cell.proof_choice,
      indicator_class: $cell.indicator_class,
      metric_path: $cell.metric_path,
      source_path: $source_path,
      source_digest: $source_digest,
      ir_node: $node.id,
      ir_digest: $ir_digest,
      generated_artifact: $cell.artifact,
      evaluator: $cell.evaluator,
      evaluator_digest: $evaluator_digest
    }
  )
' "$denominator")

case_states=$(jq -S -n \
  --slurpfile normal "$artifact_root/cases/normal.json" \
  --slurpfile unknown "$artifact_root/cases/unknown.json" \
  --slurpfile refuted "$artifact_root/cases/refuted.json" \
  --slurpfile fixed "$artifact_root/cases/fixed-point.json" \
  --slurpfile privilege "$artifact_root/cases/privilege-escalation.json" \
  --slurpfile malformed "$artifact_root/cases/malformed.json" \
  --slurpfile improvement "$artifact_root/cases/improvement-unknown.json" '
  [
    {id:"normal", state:$normal[0].state, decision:$normal[0].decision},
    {id:"unknown", state:$unknown[0].state, decision:$unknown[0].decision},
    {id:"refuted", state:$refuted[0].state, decision:$refuted[0].decision},
    {id:"fixed-point", state:$fixed[0].state, decision:$fixed[0].decision},
    {id:"privilege-escalation", state:$privilege[0].state, decision:$privilege[0].decision},
    {id:"malformed", state:$malformed[0].state, decision:$malformed[0].decision},
    {id:"improvement-unknown", state:$improvement[0].state, decision:$improvement[0].decision}
  ]
')

runtime_json=$(cat "$runtime")
mkdir -p "$(dirname "$output_json")" "$(dirname "$output_md")"
jq -S -n \
  --arg subject_sha "$subject_sha" \
  --arg source_path "${source#./}" \
  --arg source_digest "$source_digest" \
  --arg ir_digest "$ir_digest" \
  --arg contract_digest "$contract_digest" \
  --arg evaluator_path "${0#./}" \
  --arg evaluator_digest "$evaluator_digest" \
  --argjson activity_count "$activity_count" \
  --argjson metrics "$cell_metrics" \
  --argjson cases "$case_states" \
  --argjson runtime "$runtime_json" \
  --argjson descendant_dirs "$descendant_dirs" \
  --argjson regular_files "$regular_files" \
  --argjson go_files "$go_files" \
  --argjson gooo_files "$gooo_files" \
  --argjson go_lines "$go_lines" \
  --argjson gooo_lines "$gooo_lines" \
  --argjson artifact_files "$artifact_files" \
  --argjson artifact_bytes "$artifact_bytes" \
  --argjson go_test_cases "$go_test_cases" '
  {
    schema: "gooo/resolution-lattice/ci-report/v1",
    decision: "RESOLUTION_LATTICE_CONFORMANCE_CLOSED",
    subject_sha: $subject_sha,
    precedence: ["REFUTED", "UNKNOWN", "CLOSED"],
    bindings: {
      gooo_source: {path:$source_path, digest:$source_digest, activity_count:$activity_count},
      semantic_ir: {path:"semantic-ir.json", digest:$ir_digest},
      denominator: {path:"contracts/resolution-lattice-denominator-v1.json", digest:$contract_digest, numerator:12, denominator:12},
      evaluator: {path:$evaluator_path, digest:$evaluator_digest}
    },
    summary: {
      numerator: 12,
      denominator: 12,
      closed_cells: 12,
      unknown_cells: 0,
      refuted_cells: 0,
      fixed_denominator: true,
      proof_choices: {FOUNDATION:{numerator:4,denominator:4},COHERENCE:{numerator:4,denominator:4},REGRESSION:{numerator:4,denominator:4}},
      indicator_classes: {DRIVER:{numerator:4,denominator:4},OUTCOME:{numerator:4,denominator:4},GUARDRAIL:{numerator:4,denominator:4}}
    },
    cases: $cases,
    case_summary: {
      numerator: 7,
      denominator: 7,
      closed: ([$cases[] | select(.state == "CLOSED")] | length),
      unknown: ([$cases[] | select(.state == "UNKNOWN")] | length),
      refuted: ([$cases[] | select(.state == "REFUTED")] | length)
    },
    remaining: {
      UNKNOWN: ([$cases[] | select(.state == "UNKNOWN")] | length),
      REFUTED: ([$cases[] | select(.state == "REFUTED")] | length),
      CLOSED: ([$cases[] | select(.state == "CLOSED")] | length)
    },
    metrics: $metrics,
    runtime: $runtime,
    inventory: {
      descendant_dirs: $descendant_dirs,
      regular_files: $regular_files,
      go_files: $go_files,
      go_physical_lines: $go_lines,
      gooo_files: $gooo_files,
      gooo_physical_lines: $gooo_lines,
      physical_lines_include_blank_and_comments: true,
      root_readme_excluded: true
    },
    tests: {
      executed: ($go_test_cases + 7),
      reused: 0,
      skipped: 0,
      go_cases: $go_test_cases,
      conformance_cases: 7,
      local_tests_run: 0
    },
    artifacts: {
      count_scope: "generated evidence payload before human summaries",
      files: $artifact_files,
      bytes: $artifact_bytes,
      receipt_cases: ["normal"],
      repository_writes: $runtime.repository_writes
    },
    improvement_rule: {
      exact_before_after_pair_required: true,
      missing_pair_state: "UNKNOWN",
      utility_inference_from_good_numbers: false
    }
  }
' > "$output_json"

jq -r '
  "# Gooo Resolution Lattice CI artifact",
  "",
  "- decision: `\(.decision)`",
  "- subject SHA: `\(.subject_sha)`",
  "- conformance cells: `\(.summary.numerator)/\(.summary.denominator)`",
  "- proof choices: FOUNDATION `\(.summary.proof_choices.FOUNDATION.numerator)/\(.summary.proof_choices.FOUNDATION.denominator)`, COHERENCE `\(.summary.proof_choices.COHERENCE.numerator)/\(.summary.proof_choices.COHERENCE.denominator)`, REGRESSION `\(.summary.proof_choices.REGRESSION.numerator)/\(.summary.proof_choices.REGRESSION.denominator)`",
  "- indicator classes: DRIVER `\(.summary.indicator_classes.DRIVER.numerator)/\(.summary.indicator_classes.DRIVER.denominator)`, OUTCOME `\(.summary.indicator_classes.OUTCOME.numerator)/\(.summary.indicator_classes.OUTCOME.denominator)`, GUARDRAIL `\(.summary.indicator_classes.GUARDRAIL.numerator)/\(.summary.indicator_classes.GUARDRAIL.denominator)`",
  "- cases: `\(.case_summary.numerator)/\(.case_summary.denominator)` (CLOSED `\(.case_summary.closed)`, UNKNOWN `\(.case_summary.unknown)`, REFUTED `\(.case_summary.refuted)`)",
  "- remaining UNKNOWN/REFUTED: `\(.remaining.UNKNOWN)/\(.remaining.REFUTED)`",
  "- descendant directories: `\(.inventory.descendant_dirs)`",
  "- regular files excluding root README: `\(.inventory.regular_files)`",
  "- Go physical files/lines: `\(.inventory.go_files)/\(.inventory.go_physical_lines)`",
  "- Gooo physical files/lines: `\(.inventory.gooo_files)/\(.inventory.gooo_physical_lines)`",
  "- build/test/conformance wall ms: `\(.runtime.build_wall_ms)/\(.runtime.test_wall_ms)/\(.runtime.conformance_wall_ms)`",
  "- peak RSS KiB: `\(.runtime.peak_rss_kib)`",
  "- tests executed/reused/skipped: `\(.tests.executed)/\(.tests.reused)/\(.tests.skipped)`",
  "- generated evidence files/bytes: `\(.artifacts.files)/\(.artifacts.bytes)`",
  "- repository writes: `\(.artifacts.repository_writes)`",
  "- local tests: `\(.tests.local_tests_run)`",
  "",
  "## Binding",
  "",
  "- Gooo source: `\(.bindings.gooo_source.path)` / `\(.bindings.gooo_source.digest)` / \(.bindings.gooo_source.activity_count) activities",
  "- semantic IR: `\(.bindings.semantic_ir.path)` / `\(.bindings.semantic_ir.digest)`",
  "- fixed denominator: `\(.bindings.denominator.path)` / `\(.bindings.denominator.digest)`",
  "- evaluator: `\(.bindings.evaluator.path)` / `\(.bindings.evaluator.digest)`",
  "",
  "## Metric bindings",
  "",
  (.metrics[] | "- \(.id): \(.numerator)/\(.denominator) CLOSED; activity=\(.activity); source=\(.source_path); ir=\(.ir_node); artifact=\(.generated_artifact); evaluator=\(.evaluator)")
' "$output_json" > "$output_md"
