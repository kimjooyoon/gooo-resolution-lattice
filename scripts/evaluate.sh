#!/usr/bin/env bash
set -Eeuo pipefail

if test "$#" -ne 9; then
  echo "usage: evaluate.sh ARTIFACT_ROOT SOURCE IR DENOMINATOR RUNTIME GO_TEST_TOTAL SUBJECT_SHA OUTPUT_JSON OUTPUT_MD" >&2
  exit 2
fi

artifact_root=$1
source=$2
ir=$3
denominator=$4
runtime=$5
go_test_total=$6
subject_sha=$7
output_json=$8
output_md=$9
case_ids="normal unknown dependency-blocked decision-unknown refuted contradiction fixed-point privilege-escalation malformed improvement-unknown"

for required in "$source" "$ir" "$denominator" "$runtime"; do
  test -f "$required" || { echo "missing conformance input: $required" >&2; exit 2; }
done
for case_id in $case_ids; do
  test -f "$artifact_root/cases/$case_id.json" || { echo "missing case report: $case_id" >&2; exit 2; }
  test -f "$artifact_root/receipts/$case_id.json" || { echo "missing case receipts: $case_id" >&2; exit 2; }
done

source_digest="sha256:$(sha256sum "$source" | awk '{print $1}')"
ir_digest="sha256:$(sha256sum "$ir" | awk '{print $1}')"
contract_digest="sha256:$(sha256sum "$denominator" | awk '{print $1}')"
evaluator_digest="sha256:$(sha256sum "$0" | awk '{print $1}')"
source_path=$(printf '%s\n' "$source" | sed 's#^\./##')
evaluator_path=$(printf '%s\n' "$0" | sed 's#^\./##')

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
  .schema == "gooo/resolution-lattice/ir/v2" and
  .source_digest == $source_digest and
  .resolution_ladder == ["PROJECT", "ARTIFACT", "ACTIVITY", "PREDICATE", "FIELD"] and
  (.nodes | length) == 12 and
  ([.nodes[].name] | unique | length) == 12 and
  all(.nodes[]; .id != null and .name != null and .source_line > 0 and .metric_id != null and .artifact != null and .evaluator != null)
' "$ir" >/dev/null

case_report() {
  jq -e --arg source_digest "$source_digest" --arg contract_digest "$contract_digest" '
    .schema == "gooo/resolution-lattice/v2" and
    .resolution_ladder == ["PROJECT", "ARTIFACT", "ACTIVITY", "PREDICATE", "FIELD"] and
    .precedence == ["REFUTED", "UNKNOWN", "CLOSED"] and
    all(.claims[]; .stage != null and .step != null and .reason != null and .unknown_class != null and .next_operation != null and .blocked_by != null) and
    all(.edges[];
      (.source_digest == $source_digest) and
      (.receipt.source_digest == $source_digest) and
      (.receipt.contract_digest == $contract_digest) and
      (.receipt.edge_id == .id) and
      (.receipt.id != null) and
      (.receipt.stage != null) and (.receipt.step != null) and (.receipt.reason != null) and
      (.receipt.unknown_class != null) and (.receipt.next_operation != null) and
      (.receipt.blocked_by != null) and ((.causal_frontier | length) == 2) and
      (.from != .to)
    )
  ' "$1" >/dev/null
}

for case_id in $case_ids; do
  case_report "$artifact_root/cases/$case_id.json"
  jq -e --arg case_id "$case_id" --slurpfile report "$artifact_root/cases/$case_id.json" '
    .schema == "gooo/resolution-lattice/receipts/v2" and
    ((.case_id == $case_id) or ($case_id == "malformed" and .case_id == "")) and
    .receipts == ($report[0].edges | map(.receipt))
  ' "$artifact_root/receipts/$case_id.json" >/dev/null
done

jq -e '
  .state == "CLOSED" and .decision == "RESOLUTION_LATTICE_CLOSED" and
  .resolved_claim.resolution == "PROJECT" and (.edges | length) == 0 and
  .improvement.claim.state == "CLOSED" and (.improvement.metrics | length) == 2 and
  all(.improvement.metrics[]; .state == "CLOSED" and .exact_pair == true and .before != null and .after != null and .delta != null)
' "$artifact_root/cases/normal.json" >/dev/null

jq -e '
  .state == "UNKNOWN" and .decision == "RESOLUTION_LATTICE_UNKNOWN" and
  .first_direct_cause.resolution == "PROJECT" and .first_direct_cause.unknown_class == "DIRECT_MISSING" and
  .resolved_claim.resolution == "ARTIFACT" and (.edges | length) == 1 and
  any(.claims[]; .resolution == "PROJECT" and .state == "UNKNOWN" and .stage == "PROJECT" and .step == "OBSERVE_PROJECT_EVIDENCE" and .reason == "PROJECT_EVIDENCE_MISSING" and .unknown_class == "DIRECT_MISSING" and .next_operation == "PROVIDE_PROJECT_EVIDENCE" and (.blocked_by | length) == 1)
' "$artifact_root/cases/unknown.json" >/dev/null

jq -e '
  .state == "UNKNOWN" and .decision == "RESOLUTION_LATTICE_UNKNOWN" and
  .first_direct_cause.resolution == "PROJECT" and (.edges | length) == 4 and
  .edges[0].from == "PROJECT" and .edges[0].to == "ARTIFACT" and
  .edges[1].from == "ARTIFACT" and .edges[1].to == "ACTIVITY" and
  .edges[2].from == "ACTIVITY" and .edges[2].to == "PREDICATE" and
  .edges[3].from == "PREDICATE" and .edges[3].to == "FIELD" and
  .resolved_claim.resolution == "FIELD" and
  .minimal_dependency_blocked_frontier == ["artifact-source"] and
  all(.edges[1:3][]; .receipt.unknown_class == "DEPENDENCY_BLOCKED")
' "$artifact_root/cases/dependency-blocked.json" >/dev/null

jq -e '
  .state == "UNKNOWN" and .decision == "FAIL_CLOSED" and
  .feedback_code == "FEEDBACK_COVERAGE_DECISION_UNKNOWN" and
  any(.claims[]; .unknown_class == "DECISION_UNKNOWN" and .reason == "FEEDBACK_COVERAGE_DECISION_UNKNOWN")
' "$artifact_root/cases/decision-unknown.json" >/dev/null

jq -e '
  .state == "REFUTED" and .decision == "FAIL_CLOSED" and
  any(.claims[]; .resolution == "PROJECT" and .state == "REFUTED") and
  any(.history[]; .lifecycle == "REFUTED") and (.edges | length) == 0
' "$artifact_root/cases/refuted.json" >/dev/null

jq -e '
  .state == "REFUTED" and .decision == "FAIL_CLOSED" and
  any(.claims[]; .resolution == "PROJECT" and .state == "UNKNOWN") and
  any(.claims[]; .resolution == "ARTIFACT" and .state == "REFUTED") and
  any(.claims[]; .unknown_class == "DECISION_UNKNOWN")
' "$artifact_root/cases/contradiction.json" >/dev/null

jq -e '.state == "REFUTED" and .decision == "FAIL_CLOSED" and .claims[0].reason == "FIXED_POINT_NOT_SUCCESS"' "$artifact_root/cases/fixed-point.json" >/dev/null
jq -e '.state == "REFUTED" and .decision == "FAIL_CLOSED" and .claims[0].reason == "AUTHORITY_ESCALATION_REFUTED"' "$artifact_root/cases/privilege-escalation.json" >/dev/null
jq -e '.state == "REFUTED" and .decision == "FAIL_CLOSED" and .claims[0].reason == "MALFORMED_INPUT"' "$artifact_root/cases/malformed.json" >/dev/null

jq -e '
  .state == "UNKNOWN" and .decision == "RESOLUTION_LATTICE_UNKNOWN" and
  .improvement.claim.state == "UNKNOWN" and .improvement.pair_present == false and
  .improvement.claim.reason == "EXACT_BEFORE_AFTER_PAIR_MISSING" and
  .improvement.claim.unknown_class == "CAUSALITY_UNPROVEN" and
  .improvement.claim.stage == "IMPROVEMENT" and
  .improvement.claim.step == "REQUIRE_EXACT_BEFORE_AFTER_PAIR" and
  .improvement.claim.next_operation == "PROVIDE_EXACT_BEFORE_AFTER_PAIR" and
  (.improvement.metrics | length) == 2 and
  all(.improvement.metrics[]; .state == "UNKNOWN" and .exact_pair == false)
' "$artifact_root/cases/improvement-unknown.json" >/dev/null

jq -e --arg source_digest "$source_digest" --arg contract_digest "$contract_digest" '
  .schema == "gooo/resolution-lattice/runtime/v1" and
  .build_wall_ms >= 0 and .test_wall_ms >= 0 and .conformance_wall_ms >= 0 and
  .peak_rss_kib > 0 and
  .peak_rss_kib_by_phase.build > 0 and .peak_rss_kib_by_phase.test > 0 and .peak_rss_kib_by_phase.conformance > 0 and
  .repository_writes == 0 and .local_tests_run == 0 and
  .test_counts.total >= 1 and .test_counts.executed >= 1 and .test_counts.reused >= 0 and .test_counts.skipped >= 0 and .test_counts.not_observed >= 0 and
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

cell_metrics=$(jq -S --arg source_path "$source_path" --arg source_digest "$source_digest" --arg ir_digest "$ir_digest" --arg evaluator_digest "$evaluator_digest" --slurpfile ir "$ir" '
  .cells as $cells |
  $ir[0].nodes as $nodes |
  $cells | map(
    . as $cell |
    ($nodes[] | select(.name == $cell.activity)) as $node |
    {
      id: $cell.metric_id,
      cell_id: $cell.id,
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
      evaluator_digest: $evaluator_digest,
      denominator_cell: $cell.ordinal
    }
  )
' "$denominator")

case_states=$(jq -S -n \
  --slurpfile normal "$artifact_root/cases/normal.json" \
  --slurpfile unknown "$artifact_root/cases/unknown.json" \
  --slurpfile dependency "$artifact_root/cases/dependency-blocked.json" \
  --slurpfile decision "$artifact_root/cases/decision-unknown.json" \
  --slurpfile refuted "$artifact_root/cases/refuted.json" \
  --slurpfile contradiction "$artifact_root/cases/contradiction.json" \
  --slurpfile fixed "$artifact_root/cases/fixed-point.json" \
  --slurpfile privilege "$artifact_root/cases/privilege-escalation.json" \
  --slurpfile malformed "$artifact_root/cases/malformed.json" \
  --slurpfile improvement "$artifact_root/cases/improvement-unknown.json" '
  [
    {id:"normal", state:$normal[0].state, decision:$normal[0].decision},
    {id:"unknown", state:$unknown[0].state, decision:$unknown[0].decision},
    {id:"dependency-blocked", state:$dependency[0].state, decision:$dependency[0].decision},
    {id:"decision-unknown", state:$decision[0].state, decision:$decision[0].decision},
    {id:"refuted", state:$refuted[0].state, decision:$refuted[0].decision},
    {id:"contradiction", state:$contradiction[0].state, decision:$contradiction[0].decision},
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
  --arg source_path "$source_path" \
  --arg source_digest "$source_digest" \
  --arg ir_digest "$ir_digest" \
  --arg contract_digest "$contract_digest" \
  --arg evaluator_path "$evaluator_path" \
  --arg evaluator_digest "$evaluator_digest" \
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
  --argjson go_test_total "$go_test_total" '
  {
    schema: "gooo/resolution-lattice/ci-report/v2",
    decision: "RESOLUTION_LATTICE_CONFORMANCE_CLOSED",
    subject_sha: $subject_sha,
    precedence: ["REFUTED", "UNKNOWN", "CLOSED"],
    aggregation: "NO_SCORE_SUMMATION",
    bindings: {
      gooo_source: {path:$source_path, digest:$source_digest, activity_count:12, resolution_ladder:["PROJECT", "ARTIFACT", "ACTIVITY", "PREDICATE", "FIELD"]},
      semantic_ir: {path:"semantic-ir.json", digest:$ir_digest},
      denominator: {path:"contracts/resolution-lattice-denominator-v1.json", digest:$contract_digest, fixed_denominator:12},
      evaluator: {path:$evaluator_path, digest:$evaluator_digest}
    },
    summary: {
      fixed_denominator: 12,
      closed_cells: 12,
      unknown_cells: 0,
      refuted_cells: 0,
      proof_choices: {FOUNDATION:{closed:4,total:4},COHERENCE:{closed:4,total:4},REGRESSION:{closed:4,total:4}},
      indicator_classes: {DRIVER:{closed:4,total:4},OUTCOME:{closed:4,total:4},GUARDRAIL:{closed:4,total:4}}
    },
    cases: $cases,
    case_summary: {
      total: ($cases | length),
      closed: ([$cases[] | select(.state == "CLOSED")] | length),
      unknown: ([$cases[] | select(.state == "UNKNOWN")] | length),
      refuted: ([$cases[] | select(.state == "REFUTED")] | length)
    },
    unknown_classes_covered: ["DIRECT_MISSING", "DEPENDENCY_BLOCKED", "DECISION_UNKNOWN", "CAUSALITY_UNPROVEN"],
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
      total: $runtime.test_counts.total,
      executed: $runtime.test_counts.executed,
      reused: $runtime.test_counts.reused,
      skipped: $runtime.test_counts.skipped,
      not_observed: $runtime.test_counts.not_observed,
      conformance_cases: ($cases | length),
      local_tests_run: 0,
      cli_reported_go_test_total: $go_test_total
    },
    artifacts: {
      count_scope: "generated evidence payload before human summaries",
      files: $artifact_files,
      bytes: $artifact_bytes,
      repository_writes: $runtime.repository_writes
    },
    improvement_rule: {
      exact_before_after_pair_required: true,
      exact_identity_fields: ["fixture_digest", "input_digest", "tool_digest", "contract_digest"],
      claimed_metrics: ["unidentified_cause_frontier_count", "minimum_cause_reach_stage_count"],
      missing_pair_state: "UNKNOWN",
      utility_inference_from_good_numbers: false,
      score_summation: false
    }
  }
' > "$output_json"

jq -r '
  "# Gooo Resolution Lattice CI artifact",
  "",
  "- decision: \(.decision)",
  "- subject SHA: \(.subject_sha)",
  "- fixed denominator cells: \(.summary.closed_cells)/\(.summary.fixed_denominator) (no score summation)",
  "- cases: \(.case_summary.total) (CLOSED \(.case_summary.closed), UNKNOWN \(.case_summary.unknown), REFUTED \(.case_summary.refuted))",
  "- UNKNOWN classes covered: \(.unknown_classes_covered | join(", "))",
  "- resolution ladder: \(.bindings.gooo_source.resolution_ladder | join(" -> "))",
  "- descendant directories: \(.inventory.descendant_dirs)",
  "- regular files excluding root README: \(.inventory.regular_files)",
  "- Go physical files/lines: \(.inventory.go_files)/\(.inventory.go_physical_lines)",
  "- Gooo physical files/lines: \(.inventory.gooo_files)/\(.inventory.gooo_physical_lines)",
  "- build/test/conformance wall ms: \(.runtime.build_wall_ms)/\(.runtime.test_wall_ms)/\(.runtime.conformance_wall_ms)",
  "- peak RSS KiB: \(.runtime.peak_rss_kib)",
  "- tests total/executed/reused/skipped/not-observed: \(.tests.total)/\(.tests.executed)/\(.tests.reused)/\(.tests.skipped)/\(.tests.not_observed)",
  "- generated evidence files/bytes: \(.artifacts.files)/\(.artifacts.bytes)",
  "- repository writes: \(.artifacts.repository_writes)",
  "- local tests: \(.tests.local_tests_run)",
  "",
  "Binding",
  "",
  "- Gooo source: \(.bindings.gooo_source.path) / \(.bindings.gooo_source.digest)",
  "- semantic IR: \(.bindings.semantic_ir.path) / \(.bindings.semantic_ir.digest)",
  "- fixed denominator: \(.bindings.denominator.path) / \(.bindings.denominator.digest)",
  "- evaluator: \(.bindings.evaluator.path) / \(.bindings.evaluator.digest)",
  "",
  "Metric bindings",
  "",
  (.metrics[] | "- \(.id): CLOSED; cell=\(.denominator_cell); activity=\(.activity); source=\(.source_path); ir=\(.ir_node); artifact=\(.generated_artifact); evaluator=\(.evaluator)")
' "$output_json" > "$output_md"
