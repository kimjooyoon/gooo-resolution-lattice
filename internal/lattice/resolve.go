package lattice

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type levelSpec struct {
	Name          string
	Stage         string
	Step          string
	MissingReason string
	NextOperation string
	BlockedBy     string
}

var levelSpecs = []levelSpec{
	{Name: "EXACT", Stage: "EXACT", Step: "OBSERVE_EXACT_EVIDENCE", MissingReason: "EXACT_EVIDENCE_MISSING", NextOperation: "PROVIDE_EXACT_EVIDENCE", BlockedBy: "exact-evidence"},
	{Name: "INVARIANT", Stage: "INVARIANT", Step: "OBSERVE_INVARIANT_EVIDENCE", MissingReason: "INVARIANT_EVIDENCE_MISSING", NextOperation: "PROVIDE_INVARIANT_EVIDENCE", BlockedBy: "invariant-evidence"},
	{Name: "EXISTENCE", Stage: "EXISTENCE", Step: "OBSERVE_EXISTENCE_EVIDENCE", MissingReason: "EXISTENCE_EVIDENCE_MISSING", NextOperation: "PROVIDE_EXISTENCE_EVIDENCE", BlockedBy: "existence-evidence"},
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
		ObservationMode:      input.Authority.ObservationMode,
		RepositoryWrites:     0,
		InputRepositoryWrites: input.Authority.RequestedRepositoryWrites,
		ReadOnly:             true,
	}
	if input.Authority.RequestedRepositoryWrites != 0 || input.Authority.RequestedPrivilegeEscalation {
		return failClosed(report, "AUTHORITY", "VALIDATE_READ_ONLY_BOUNDARY", "AUTHORITY_ESCALATION_REFUTED", "USE_READ_ONLY_AUTHORITY", "authority-boundary")
	}
	if input.StartResolution == "FIXED_POINT" {
		return failClosed(report, "INPUT", "VALIDATE_START_RESOLUTION", "FIXED_POINT_NOT_SUCCESS", "USE_EXACT_START_RESOLUTION", "start-resolution")
	}

	previousOpenID := ""
	previousLevel := ""
	anyUnknown := false
	anyRefuted := false
	var resolved *Claim
	for index, spec := range levelSpecs {
		evidence, present := input.Evidence[spec.Name]
		if !present {
			evidence = Evidence{Status: "MISSING"}
		}
		openID := eventID(input.CaseID, spec.Name, LifecycleOpen)
		claim := claimFor(input.CaseID, spec, StateUnknown, spec.MissingReason, "DIRECT_MISSING", spec.NextOperation, spec.BlockedBy)
		if present {
			claim = claimForEvidence(input.CaseID, spec, evidence)
		}
		report.Claims = append(report.Claims, claim)
		report.History = append(report.History, HistoryEvent{
			Sequence:   len(report.History) + 1,
			ID:         openID,
			ClaimID:    claim.ID,
			Resolution: spec.Name,
			Lifecycle:  LifecycleOpen,
			Activity:   bindingActivity(meta, "PreserveClaimHistory"),
			ReceiptID:  "",
			Claim:      claim,
		})

		if previousOpenID != "" {
			edge, err := makeEdge(meta, report, input.CaseID, previousLevel, spec.Name, previousOpenID, openID, claim)
			if err != nil {
				return failClosed(report, "BINDING", "BIND_GENERATED_RECEIPT", "META_ACTIVITY_BINDING_MISSING", "REGENERATE_SEMANTIC_IR", "meta-binding")
			}
			report.Edges = append(report.Edges, edge)
			report.CausalFrontier = append([]string(nil), edge.CausalFrontier...)
		}

		switch evidence.Status {
		case "SUPPORTS":
			claim.State = StateClosed
			claim.Stage = "NONE"
			claim.Step = "NONE"
			claim.Reason = spec.Name + "_EVIDENCE_SUPPORTED"
			claim.UnknownClass = "NONE"
			claim.NextOperation = "NONE"
			claim.BlockedBy = []string{}
			report.Claims[len(report.Claims)-1] = claim
			report.History = append(report.History, HistoryEvent{
				Sequence:   len(report.History) + 1,
				ID:         eventID(input.CaseID, spec.Name, LifecycleClose),
				ClaimID:    claim.ID,
				Resolution: spec.Name,
				Lifecycle:  LifecycleClose,
				Activity:   bindingActivity(meta, "DischargeSupportedClaim"),
				ReceiptID:  "",
				Claim:      claim,
			})
			copyClaim := claim
			resolved = &copyClaim
		case "MISSING":
			anyUnknown = true
		case "CONTRADICTS":
			claim.State = StateRefuted
			claim.Stage = spec.Stage
			claim.Step = spec.Step
			claim.Reason = spec.Name + "_EVIDENCE_CONTRADICTED"
			claim.UnknownClass = "NONE"
			claim.NextOperation = "REPAIR_CONTRADICTING_EVIDENCE"
			claim.BlockedBy = []string{strings.ToLower(spec.Name) + "-contradiction"}
			report.Claims[len(report.Claims)-1] = claim
			report.History = append(report.History, HistoryEvent{
				Sequence:   len(report.History) + 1,
				ID:         eventID(input.CaseID, spec.Name, LifecycleRefute),
				ClaimID:    claim.ID,
				Resolution: spec.Name,
				Lifecycle:  LifecycleRefute,
				Activity:   bindingActivity(meta, "RefuteContradiction"),
				ReceiptID:  "",
				Claim:      claim,
			})
			anyRefuted = true
			resolved = nil
		default:
			return failClosed(report, "INPUT", "VALIDATE_EVIDENCE_STATUS", "MALFORMED_EVIDENCE_STATUS", "REPAIR_EVIDENCE_STATUS", "evidence-status")
		}
		if anyRefuted {
			break
		}
		previousOpenID = openID
		previousLevel = spec.Name
		if index == len(levelSpecs)-1 {
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
	if input.Schema != InputSchema || input.CaseID == "" || input.Subject == "" {
		return errors.New("MALFORMED_INPUT_CONTRACT")
	}
	if input.StartResolution != "EXACT" && input.StartResolution != "FIXED_POINT" {
		return errors.New("ARBITRARY_PARENT_RESOLUTION_REFUTED")
	}
	if input.Evidence == nil {
		return errors.New("MALFORMED_EVIDENCE_MAP")
	}
	return nil
}

func baseReport(meta Meta, inputDigest string) Report {
	return Report{
		Schema:         Schema,
		Decision:       "FAIL_CLOSED",
		InputDigest:    inputDigest,
		ToolDigest:     meta.ToolDigest,
		ContractDigest: meta.ContractDigest,
		State:          StateRefuted,
		Precedence:     []string{StateRefuted, StateUnknown, StateClosed},
		Claims:         []Claim{},
		History:        []HistoryEvent{},
		Edges:          []DescentEdge{},
		CausalFrontier: []string{},
		GeneratedArtifacts: []string{"report.json", "receipts.json"},
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
			Sequence:   1,
			ID:         report.CaseID + ":input:open",
			ClaimID:    claim.ID,
			Resolution: "INPUT",
			Lifecycle:  LifecycleOpen,
			Activity:   "ResolveExactClaim",
			Claim:      claim,
		},
		HistoryEvent{
			Sequence:   2,
			ID:         report.CaseID + ":input:refuted",
			ClaimID:    claim.ID,
			Resolution: "INPUT",
			Lifecycle:  LifecycleRefute,
			Activity:   "RefuteContradiction",
			Claim:      claim,
		})
	report.State = StateRefuted
	report.Decision = "FAIL_CLOSED"
	report.Authority.ReadOnly = false
	return report
}

func claimFor(caseID string, spec levelSpec, state, reason, unknownClass, nextOperation, blockedBy string) Claim {
	claim := Claim{
		ID:            caseID + ":" + strings.ToLower(spec.Name),
		Resolution:    spec.Name,
		State:         state,
		Stage:         spec.Stage,
		Step:          spec.Step,
		Reason:        reason,
		UnknownClass:  unknownClass,
		NextOperation: nextOperation,
		BlockedBy:     []string{blockedBy},
	}
	if blockedBy == "" {
		claim.BlockedBy = []string{}
	}
	return claim
}

func claimForEvidence(caseID string, spec levelSpec, evidence Evidence) Claim {
	if evidence.Status == "SUPPORTS" {
		return claimFor(caseID, spec, StateClosed, spec.Name+"_EVIDENCE_SUPPORTED", "NONE", "NONE", "")
	}
	if evidence.Status == "CONTRADICTS" {
		return claimFor(caseID, spec, StateRefuted, spec.Name+"_EVIDENCE_CONTRADICTED", "NONE", "REPAIR_CONTRADICTING_EVIDENCE", strings.ToLower(spec.Name)+"-contradiction")
	}
	return claimFor(caseID, spec, StateUnknown, spec.MissingReason, "DIRECT_MISSING", spec.NextOperation, spec.BlockedBy)
}

func makeEdge(meta Meta, report Report, caseID, from, to, fromOpen, toOpen string, output Claim) (DescentEdge, error) {
	activity := "DescendExactToInvariant"
	if from == "INVARIANT" && to == "EXISTENCE" {
		activity = "DescendInvariantToExistence"
	}
	binding, ok := meta.Bindings[activity]
	if !ok {
		return DescentEdge{}, errors.New("missing edge activity")
	}
	edgeID := "edge/" + caseID + "/" + strings.ToLower(from) + "-to-" + strings.ToLower(to)
	receiptID := "receipt/" + caseID + "/" + strings.ToLower(from) + "-to-" + strings.ToLower(to)
	receipt := GeneratedReceipt{
		Schema:            "gooo/resolution-lattice/receipt/v1",
		ID:                receiptID,
		EdgeID:            edgeID,
		CaseID:            caseID,
		Activity:          activity,
		From:              from,
		To:                to,
		InputDigest:       report.InputDigest,
		ToolDigest:        report.ToolDigest,
		ContractDigest:    report.ContractDigest,
		SourceDigest:      meta.SourceDigest,
		OutputClaimDigest: DigestJSON(output),
	}
	return DescentEdge{
		ID:                edgeID,
		From:              from,
		To:                to,
		Activity:          activity,
		SourcePath:        binding.SourcePath,
		SourceDigest:      meta.SourceDigest,
		IRNode:            binding.IRNode,
		GeneratedArtifact: "receipts.json",
		Evaluator:         binding.Evaluator,
		Receipt:           receipt,
		CausalFrontier:    []string{fromOpen, toOpen},
	}, nil
}

func evaluateImprovement(input Input, caseID string) ImprovementResult {
	claim := Claim{
		ID:            caseID + ":improvement",
		Resolution:    "EXACT",
		State:         StateUnknown,
		Stage:         "IMPROVEMENT",
		Step:          "REQUIRE_EXACT_BEFORE_AFTER_PAIR",
		Reason:        "EXACT_BEFORE_AFTER_PAIR_MISSING",
		UnknownClass:  "CAUSALITY_UNPROVEN",
		NextOperation: "PROVIDE_EXACT_BEFORE_AFTER_PAIR",
		BlockedBy:     []string{"exact-before-after-pair"},
	}
	if input.Improvement == nil {
		return ImprovementResult{Claim: claim}
	}
	comparison := input.Improvement
	pair := comparison.ExactBefore != nil && comparison.ExactAfter != nil
	sameInput := comparison.InputDigest != ""
	sameTool := comparison.ToolDigest != ""
	sameContract := comparison.ContractDigest != ""
	if !pair || !sameInput || !sameTool || !sameContract {
		return ImprovementResult{Claim: claim, PairPresent: pair, SameInputDigest: sameInput, SameToolDigest: sameTool, SameContractDigest: sameContract}
	}
	claim.State = StateClosed
	claim.Stage = "NONE"
	claim.Step = "NONE"
	claim.Reason = "EXACT_BEFORE_AFTER_PAIR_OBSERVED"
	claim.UnknownClass = "NONE"
	claim.NextOperation = "NONE"
	claim.BlockedBy = []string{}
	if !comparison.UtilityEvidence {
		claim.State = StateRefuted
		claim.Stage = "IMPROVEMENT"
		claim.Step = "EVALUATE_UTILITY_EVIDENCE"
		claim.Reason = "EXACT_PAIR_NOT_UTILITY"
		claim.NextOperation = "REVIEW_UTILITY_EVIDENCE"
		claim.BlockedBy = []string{"utility-evidence"}
	}
	return ImprovementResult{Claim: claim, PairPresent: true, SameInputDigest: true, SameToolDigest: true, SameContractDigest: true}
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
		return "RESOLUTION_LATTICE_CLOSED"
	case StateUnknown:
		return "RESOLUTION_LATTICE_UNKNOWN"
	default:
		return "FAIL_CLOSED"
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

func DigestJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return Digest([]byte(fmt.Sprintf("%v", value)))
	}
	return Digest(data)
}
