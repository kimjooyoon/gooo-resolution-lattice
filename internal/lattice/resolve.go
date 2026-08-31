package lattice

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type levelSpec struct {
	Name          string
	Stage         string
	Step          string
	MissingReason string
	NextOperation string
	BlockedBy     string
	EdgeActivity  string
}

var levelSpecs = []levelSpec{
	{Name: "PROJECT", Stage: "PROJECT", Step: "OBSERVE_PROJECT_EVIDENCE", MissingReason: "PROJECT_EVIDENCE_MISSING", NextOperation: "PROVIDE_PROJECT_EVIDENCE", BlockedBy: "project-evidence"},
	{Name: "ARTIFACT", Stage: "ARTIFACT", Step: "OBSERVE_ARTIFACT_EVIDENCE", MissingReason: "ARTIFACT_EVIDENCE_MISSING", NextOperation: "PROVIDE_ARTIFACT_EVIDENCE", BlockedBy: "artifact-evidence", EdgeActivity: "DescendProjectToArtifact"},
	{Name: "ACTIVITY", Stage: "ACTIVITY", Step: "OBSERVE_ACTIVITY_EVIDENCE", MissingReason: "ACTIVITY_EVIDENCE_MISSING", NextOperation: "PROVIDE_ACTIVITY_EVIDENCE", BlockedBy: "activity-evidence", EdgeActivity: "DescendArtifactToActivity"},
	{Name: "PREDICATE", Stage: "PREDICATE", Step: "OBSERVE_PREDICATE_EVIDENCE", MissingReason: "PREDICATE_EVIDENCE_MISSING", NextOperation: "PROVIDE_PREDICATE_EVIDENCE", BlockedBy: "predicate-evidence", EdgeActivity: "DescendActivityToPredicate"},
	{Name: "FIELD", Stage: "FIELD", Step: "OBSERVE_FIELD_EVIDENCE", MissingReason: "FIELD_EVIDENCE_MISSING", NextOperation: "PROVIDE_FIELD_EVIDENCE", BlockedBy: "field-evidence", EdgeActivity: "DescendPredicateToField"},
}

func ResolveJSON(raw []byte, meta Meta) Report {
	report := baseReport(meta, CanonicalDigest(raw))
	var input Input
	if err := json.Unmarshal(raw, &input); err != nil {
		return failClosed(report, "INPUT", "PARSE_INPUT", "MALFORMED_INPUT", "REPAIR_INPUT", "input-json")
	}
	report.CaseID = input.CaseID
	report.Subject = input.Subject
	if err := validateInput(input); err != nil {
		return failClosed(report, "INPUT", "VALIDATE_INPUT", err.Error(), "REPAIR_INPUT", "input-contract")
	}
	report.Authority = AuthorityReport{
		ObservationMode:       input.Authority.ObservationMode,
		RepositoryWrites:      0,
		InputRepositoryWrites: input.Authority.RequestedRepositoryWrites,
		ReadOnly:              true,
	}
	if input.Authority.RequestedRepositoryWrites != 0 || input.Authority.RequestedPrivilegeEscalation {
		return failClosed(report, "AUTHORITY", "VALIDATE_READ_ONLY_BOUNDARY", "AUTHORITY_ESCALATION_REFUTED", "USE_READ_ONLY_AUTHORITY", "authority-boundary")
	}
	if input.StartResolution == "FIXED_POINT" {
		return failClosed(report, "INPUT", "VALIDATE_START_RESOLUTION", "FIXED_POINT_NOT_SUCCESS", "USE_PROJECT_START_RESOLUTION", "start-resolution")
	}

	anyUnknown := false
	anyRefuted := false
	decisionUnknown := input.Decision != "" && input.Decision != "FIXED_POINT"
	if decisionUnknown {
		claim := Claim{
			ID:            input.CaseID + ":decision",
			Resolution:    "PROJECT",
			State:         StateUnknown,
			Stage:         "DECISION",
			Step:          "VALIDATE_UPPER_DECISION",
			Reason:        FeedbackDecisionUnknown,
			UnknownClass:  UnknownDecision,
			NextOperation: "PROVIDE_EXPLICIT_FIXED_POINT",
			BlockedBy:     []string{"upper-decision"},
		}
		report.Claims = append(report.Claims, claim)
		report.History = append(report.History, HistoryEvent{
			Sequence:   len(report.History) + 1,
			ID:         eventID(input.CaseID, "DECISION", LifecycleOpen),
			ClaimID:    claim.ID,
			Resolution: claim.Resolution,
			Lifecycle:  LifecycleOpen,
			Activity:   bindingActivity(meta, "ResolveProjectClaim"),
			Claim:      claim,
		})
		anyUnknown = true
		report.DecisionInput = input.Decision
	}

	var previous *Claim
	var resolved *Claim
levelLoop:
	for _, spec := range levelSpecs {
		evidence, present := input.Evidence[spec.Name]
		if !present {
			evidence = Evidence{Status: "MISSING"}
		}
		status := evidence.Status
		if !present {
			status = "MISSING"
		}
		if present && status == "" {
			return failClosed(report, "INPUT", "VALIDATE_EVIDENCE_STATUS", "MALFORMED_EVIDENCE_STATUS", "REPAIR_EVIDENCE_STATUS", "evidence-status")
		}
		if (status == "MISSING" || status == "UNKNOWN") && evidence.UnknownClass != "" && !validUnknownClass(evidence.UnknownClass) {
			return failClosed(report, "INPUT", "VALIDATE_UNKNOWN_CLASS", "MALFORMED_UNKNOWN_CLASS", "REPAIR_UNKNOWN_CLASS", "unknown-class")
		}
		claim := claimForEvidence(input.CaseID, spec, evidence)
		report.Claims = append(report.Claims, claim)
		openID := eventID(input.CaseID, spec.Name, LifecycleOpen)
		report.History = append(report.History, HistoryEvent{
			Sequence:   len(report.History) + 1,
			ID:         openID,
			ClaimID:    claim.ID,
			Resolution: spec.Name,
			Lifecycle:  LifecycleOpen,
			Activity:   bindingActivity(meta, "PreserveClaimHistory"),
			Claim:      claim,
		})

		if previous != nil && previous.State == StateUnknown {
			edge, err := makeEdge(meta, report, input.CaseID, spec.EdgeActivity, *previous, claim, openID)
			if err != nil {
				return failClosed(report, "BINDING", "BIND_GENERATED_RECEIPT", "META_ACTIVITY_BINDING_MISSING", "REGENERATE_SEMANTIC_IR", "meta-binding")
			}
			report.Edges = append(report.Edges, edge)
			report.CausalFrontier = append([]string(nil), edge.CausalFrontier...)
			report.History[len(report.History)-2].ReceiptID = edge.Receipt.ID
			if claim.UnknownClass == UnknownDependency {
				report.MinimalDependencyBlockedFrontier = setMinimalDependencyFrontier(report.MinimalDependencyBlockedFrontier, claim.BlockedBy...)
			}
		}

		switch status {
		case "SUPPORTS":
			claim = dischargedClaim(input.CaseID, spec)
			report.Claims[len(report.Claims)-1] = claim
			report.History = append(report.History, HistoryEvent{
				Sequence:   len(report.History) + 1,
				ID:         eventID(input.CaseID, spec.Name, LifecycleClose),
				ClaimID:    claim.ID,
				Resolution: spec.Name,
				Lifecycle:  LifecycleClose,
				Activity:   bindingActivity(meta, "DischargeSupportedClaim"),
				Claim:      claim,
			})
			copyClaim := claim
			resolved = &copyClaim
			if previous == nil || previous.State != StateUnknown {
				previous = &copyClaim
				break levelLoop
			}
			previous = &copyClaim
			break levelLoop
		case "MISSING", "UNKNOWN":
			anyUnknown = true
			if claim.UnknownClass == UnknownDirect && report.FirstDirectCause == nil {
				copyClaim := claim
				report.FirstDirectCause = &copyClaim
			}
			if claim.UnknownClass == UnknownDependency {
				report.MinimalDependencyBlockedFrontier = setMinimalDependencyFrontier(report.MinimalDependencyBlockedFrontier, claim.BlockedBy...)
			}
			previous = &claim
		case "CONTRADICTS", "REFUTED":
			claim = refutedClaim(input.CaseID, spec)
			report.Claims[len(report.Claims)-1] = claim
			report.History = append(report.History, HistoryEvent{
				Sequence:   len(report.History) + 1,
				ID:         eventID(input.CaseID, spec.Name, LifecycleRefute),
				ClaimID:    claim.ID,
				Resolution: spec.Name,
				Lifecycle:  LifecycleRefute,
				Activity:   bindingActivity(meta, "RefuteContradiction"),
				Claim:      claim,
			})
			anyRefuted = true
			resolved = nil
			break
		default:
			return failClosed(report, "INPUT", "VALIDATE_EVIDENCE_STATUS", "MALFORMED_EVIDENCE_STATUS", "REPAIR_EVIDENCE_STATUS", "evidence-status")
		}
		if anyRefuted {
			break
		}
	}

	report.Improvement = evaluateImprovement(input, report.CaseID)
	if report.Improvement.Claim.State == StateUnknown {
		anyUnknown = true
	}
	if report.Improvement.Claim.State == StateRefuted {
		anyRefuted = true
	}
	report.ResolvedClaim = resolved
	report.State = precedence(anyRefuted, anyUnknown)
	report.Decision = decisionFor(report.State)
	if decisionUnknown && report.State != StateRefuted {
		report.Decision = DecisionFailClosed
		report.FeedbackCode = FeedbackDecisionUnknown
	}
	if report.State == StateUnknown && report.ResolvedClaim == nil && len(report.Claims) > 0 {
		last := report.Claims[len(report.Claims)-1]
		report.ResolvedClaim = &last
	}
	if report.State == StateRefuted {
		report.ResolvedClaim = nil
	}
	return report
}

func validateInput(input Input) error {
	if (input.Schema != InputSchema && input.Schema != LegacyInputSchema) || input.CaseID == "" || input.Subject == "" {
		return errors.New("MALFORMED_INPUT_CONTRACT")
	}
	if input.StartResolution != "PROJECT" && input.StartResolution != "FIXED_POINT" {
		return errors.New("ARBITRARY_PARENT_RESOLUTION_REFUTED")
	}
	if input.Evidence == nil {
		return errors.New("MALFORMED_EVIDENCE_MAP")
	}
	return nil
}

func baseReport(meta Meta, inputDigest string) Report {
	return Report{
		Schema:                           Schema,
		Decision:                         DecisionFailClosed,
		InputDigest:                      inputDigest,
		ToolDigest:                       meta.ToolDigest,
		ContractDigest:                   meta.ContractDigest,
		ResolutionLadder:                 append([]string(nil), ResolutionLevels...),
		State:                            StateRefuted,
		Precedence:                       []string{StateRefuted, StateUnknown, StateClosed},
		Claims:                           []Claim{},
		History:                          []HistoryEvent{},
		Edges:                            []DescentEdge{},
		CausalFrontier:                   []string{},
		MinimalDependencyBlockedFrontier: []string{},
		GeneratedArtifacts:               []string{"report.json", "receipts.json"},
		Authority: AuthorityReport{
			ObservationMode: "UNKNOWN",
			ReadOnly:        true,
		},
	}
}

func failClosed(report Report, stage, step, reason, nextOperation, blockedBy string) Report {
	claim := Claim{
		ID:            report.CaseID + ":FAIL_CLOSED",
		Resolution:    "INPUT",
		State:         StateRefuted,
		Stage:         stage,
		Step:          step,
		Reason:        reason,
		UnknownClass:  "NONE",
		NextOperation: nextOperation,
		BlockedBy:     []string{blockedBy},
	}
	report.Claims = append(report.Claims, claim)
	report.History = append(report.History,
		HistoryEvent{
			Sequence:   len(report.History) + 1,
			ID:         report.CaseID + ":input:open",
			ClaimID:    claim.ID,
			Resolution: "INPUT",
			Lifecycle:  LifecycleOpen,
			Activity:   "ResolveProjectClaim",
			Claim:      claim,
		},
		HistoryEvent{
			Sequence:   len(report.History) + 1,
			ID:         report.CaseID + ":input:refuted",
			ClaimID:    claim.ID,
			Resolution: "INPUT",
			Lifecycle:  LifecycleRefute,
			Activity:   "RefuteContradiction",
			Claim:      claim,
		})
	report.State = StateRefuted
	report.Decision = DecisionFailClosed
	report.FeedbackCode = ""
	report.Authority.ReadOnly = true
	return report
}

func claimForEvidence(caseID string, spec levelSpec, evidence Evidence) Claim {
	switch evidence.Status {
	case "SUPPORTS":
		return Claim{ID: caseID + ":" + strings.ToLower(spec.Name), Resolution: spec.Name, State: StateClosed, Stage: "NONE", Step: "NONE", Reason: spec.Name + "_EVIDENCE_SUPPORTED", UnknownClass: "NONE", NextOperation: "NONE", BlockedBy: []string{}}
	case "CONTRADICTS", "REFUTED":
		return refutedClaim(caseID, spec)
	default:
		unknownClass := evidence.UnknownClass
		if unknownClass == "" {
			unknownClass = UnknownDirect
		}
		stage := evidence.Stage
		if stage == "" {
			stage = spec.Stage
		}
		step := evidence.Step
		if step == "" {
			step = spec.Step
		}
		reason := evidence.Reason
		if reason == "" {
			reason = spec.MissingReason
		}
		next := evidence.NextOperation
		if next == "" {
			next = spec.NextOperation
		}
		blockedBy := normalizeFrontier(evidence.BlockedBy)
		if len(blockedBy) == 0 {
			blockedBy = []string{spec.BlockedBy}
		}
		return Claim{
			ID:            caseID + ":" + strings.ToLower(spec.Name),
			Resolution:    spec.Name,
			State:         StateUnknown,
			Stage:         stage,
			Step:          step,
			Reason:        reason,
			UnknownClass:  unknownClass,
			NextOperation: next,
			BlockedBy:     blockedBy,
		}
	}
}

func dischargedClaim(caseID string, spec levelSpec) Claim {
	return Claim{ID: caseID + ":" + strings.ToLower(spec.Name), Resolution: spec.Name, State: StateClosed, Stage: "NONE", Step: "NONE", Reason: spec.Name + "_EVIDENCE_SUPPORTED", UnknownClass: "NONE", NextOperation: "NONE", BlockedBy: []string{}}
}

func refutedClaim(caseID string, spec levelSpec) Claim {
	return Claim{ID: caseID + ":" + strings.ToLower(spec.Name), Resolution: spec.Name, State: StateRefuted, Stage: spec.Stage, Step: spec.Step, Reason: spec.Name + "_EVIDENCE_CONTRADICTED", UnknownClass: "NONE", NextOperation: "REPAIR_CONTRADICTING_EVIDENCE", BlockedBy: []string{strings.ToLower(spec.Name) + "-contradiction"}}
}

func makeEdge(meta Meta, report Report, caseID, activity string, fromClaim, toClaim Claim, toOpenID string) (DescentEdge, error) {
	binding, ok := meta.Bindings[activity]
	if !ok {
		return DescentEdge{}, errors.New("missing edge activity")
	}
	edgeID := "edge/" + caseID + "/" + strings.ToLower(fromClaim.Resolution) + "-to-" + strings.ToLower(toClaim.Resolution)
	receiptID := "receipt/" + caseID + "/" + strings.ToLower(fromClaim.Resolution) + "-to-" + strings.ToLower(toClaim.Resolution)
	receipt := GeneratedReceipt{
		Schema:            "gooo/resolution-lattice/receipt/v2",
		ID:                receiptID,
		EdgeID:            edgeID,
		CaseID:            caseID,
		Activity:          activity,
		From:              fromClaim.Resolution,
		To:                toClaim.Resolution,
		Stage:             fromClaim.Stage,
		Step:              fromClaim.Step,
		Reason:            fromClaim.Reason,
		UnknownClass:      fromClaim.UnknownClass,
		NextOperation:     fromClaim.NextOperation,
		BlockedBy:         append([]string(nil), fromClaim.BlockedBy...),
		InputDigest:       report.InputDigest,
		ToolDigest:        report.ToolDigest,
		ContractDigest:    report.ContractDigest,
		SourceDigest:      meta.SourceDigest,
		OutputClaimDigest: DigestJSON(toClaim),
	}
	fromOpenID := eventID(caseID, fromClaim.Resolution, LifecycleOpen)
	return DescentEdge{
		ID:                edgeID,
		From:              fromClaim.Resolution,
		To:                toClaim.Resolution,
		Activity:          activity,
		SourcePath:        binding.SourcePath,
		SourceDigest:      meta.SourceDigest,
		IRNode:            binding.IRNode,
		GeneratedArtifact: "receipts.json",
		Evaluator:         binding.Evaluator,
		Receipt:           receipt,
		CausalFrontier:    []string{fromOpenID, toOpenID},
	}, nil
}

func evaluateImprovement(input Input, caseID string) ImprovementResult {
	unknown := Claim{
		ID:            caseID + ":improvement",
		Resolution:    "FIELD",
		State:         StateUnknown,
		Stage:         "IMPROVEMENT",
		Step:          "REQUIRE_EXACT_BEFORE_AFTER_PAIR",
		Reason:        "EXACT_BEFORE_AFTER_PAIR_MISSING",
		UnknownClass:  UnknownCausality,
		NextOperation: "PROVIDE_EXACT_BEFORE_AFTER_PAIR",
		BlockedBy:     []string{"exact-before-after-pair"},
	}
	metrics := []ImprovementMetric{
		{Name: "unidentified_cause_frontier_count", State: StateUnknown, ExactPair: false},
		{Name: "minimum_cause_reach_stage_count", State: StateUnknown, ExactPair: false},
	}
	if input.Improvement == nil || input.Improvement.ExactBefore == nil || input.Improvement.ExactAfter == nil {
		return ImprovementResult{Claim: unknown, Metrics: metrics}
	}
	comparison := input.Improvement
	pair := comparison.ExactBefore.Digest != "" && comparison.ExactAfter.Digest != "" && comparison.ExactBefore.Value != "" && comparison.ExactAfter.Value != ""
	sameFixture := sameIdentity(comparison.FixtureDigest, comparison.ExactBefore.FixtureDigest, comparison.ExactAfter.FixtureDigest)
	sameInput := sameIdentity(comparison.InputDigest, comparison.ExactBefore.InputDigest, comparison.ExactAfter.InputDigest)
	sameTool := sameIdentity(comparison.ToolDigest, comparison.ExactBefore.ToolDigest, comparison.ExactAfter.ToolDigest)
	sameContract := sameIdentity(comparison.ContractDigest, comparison.ExactBefore.ContractDigest, comparison.ExactAfter.ContractDigest)
	result := ImprovementResult{Claim: unknown, PairPresent: pair, SameFixtureDigest: sameFixture, SameInputDigest: sameInput, SameToolDigest: sameTool, SameContractDigest: sameContract, Metrics: metrics}
	if !pair || !sameFixture || !sameInput || !sameTool || !sameContract {
		result.Claim.Reason = "EXACT_BEFORE_AFTER_DIGEST_MISMATCH"
		result.Claim.NextOperation = "PROVIDE_MATCHING_EXACT_BEFORE_AFTER_PAIR"
		result.Claim.BlockedBy = []string{"fixture-and-digests"}
		return result
	}
	beforeFrontier := comparison.ExactBefore.UnidentifiedCauseFrontierCount
	afterFrontier := comparison.ExactAfter.UnidentifiedCauseFrontierCount
	beforeStages := comparison.ExactBefore.MinimumCauseReachStageCount
	afterStages := comparison.ExactAfter.MinimumCauseReachStageCount
	if beforeStages == nil {
		beforeStages = comparison.ExactBefore.MinimumCauseReachStages
	}
	if afterStages == nil {
		afterStages = comparison.ExactAfter.MinimumCauseReachStages
	}
	if beforeFrontier == nil || afterFrontier == nil || beforeStages == nil || afterStages == nil {
		result.Claim.Reason = "EXACT_BEFORE_AFTER_METRICS_MISSING"
		result.Claim.NextOperation = "PROVIDE_EXACT_CAUSE_FRONTIER_METRICS"
		result.Claim.BlockedBy = []string{"exact-cause-frontier-metrics"}
		return result
	}
	claim := unknown
	claim.State = StateClosed
	claim.Stage = "NONE"
	claim.Step = "NONE"
	claim.Reason = "EXACT_BEFORE_AFTER_PAIR_OBSERVED"
	claim.UnknownClass = "NONE"
	claim.NextOperation = "NONE"
	claim.BlockedBy = []string{}
	frontierDelta := *afterFrontier - *beforeFrontier
	stageDelta := *afterStages - *beforeStages
	result.Claim = claim
	result.Metrics = []ImprovementMetric{
		{Name: "unidentified_cause_frontier_count", State: StateClosed, Before: beforeFrontier, After: afterFrontier, Delta: &frontierDelta, ExactPair: true},
		{Name: "minimum_cause_reach_stage_count", State: StateClosed, Before: beforeStages, After: afterStages, Delta: &stageDelta, ExactPair: true},
	}
	return result
}

func sameIdentity(topLevel, before, after string) bool {
	if before != "" || after != "" {
		return before != "" && before == after
	}
	return topLevel != ""
}

func precedence(refuted, unknown bool) string {
	if refuted {
		return StateRefuted
	}
	if unknown {
		return StateUnknown
	}
	return StateClosed
}

func decisionFor(state string) string {
	switch state {
	case StateClosed:
		return DecisionClosed
	case StateUnknown:
		return DecisionUnknown
	default:
		return DecisionFailClosed
	}
}

func eventID(caseID, resolution, lifecycle string) string {
	return fmt.Sprintf("event/%s/%s/%s", caseID, strings.ToLower(resolution), strings.ToLower(lifecycle))
}

func bindingActivity(meta Meta, activity string) string {
	if _, ok := meta.Bindings[activity]; ok {
		return activity
	}
	return activity
}

func normalizeFrontier(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func validUnknownClass(value string) bool {
	for _, class := range UnknownClasses {
		if value == class {
			return true
		}
	}
	return false
}

func setMinimalDependencyFrontier(existing []string, values ...string) []string {
	if len(existing) != 0 {
		return existing
	}
	return normalizeFrontier(values)
}

func DigestJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return Digest([]byte(fmt.Sprintf("%v", value)))
	}
	return Digest(data)
}
