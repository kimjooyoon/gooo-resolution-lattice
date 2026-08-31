package lattice

import (
	"strings"
	"testing"
)

func TestResolveNormalAcceptsExplicitFixedPoint(t *testing.T) {
	report := ResolveJSON([]byte(normalInput("normal")), testMeta())
	if report.State != StateClosed || report.Decision != DecisionClosed {
		t.Fatalf("normal input did not close: %#v", report)
	}
	if report.ResolvedClaim == nil || report.ResolvedClaim.Resolution != "PROJECT" {
		t.Fatalf("normal input did not remain at the highest supported resolution: %#v", report.ResolvedClaim)
	}
	if len(report.Edges) != 0 {
		t.Fatalf("normal input descended despite project evidence: %#v", report.Edges)
	}
	if report.Improvement.Claim.State != StateClosed || len(report.Improvement.Metrics) != 2 {
		t.Fatalf("exact improvement metrics were not closed: %#v", report.Improvement)
	}
	for _, metric := range report.Improvement.Metrics {
		if !metric.ExactPair || metric.Before == nil || metric.After == nil || metric.Delta == nil {
			t.Fatalf("metric lost exact pair evidence: %#v", metric)
		}
	}
}

func TestResolveDirectUnknownLowersAndPreservesFirstCause(t *testing.T) {
	report := ResolveJSON([]byte(directUnknownInput()), testMeta())
	if report.State != StateUnknown || report.Decision != DecisionUnknown {
		t.Fatalf("direct unknown was not preserved: %#v", report)
	}
	if report.ResolvedClaim == nil || report.ResolvedClaim.Resolution != "ARTIFACT" {
		t.Fatalf("direct unknown did not lower to the first supported level: %#v", report.ResolvedClaim)
	}
	if report.FirstDirectCause == nil || report.FirstDirectCause.Resolution != "PROJECT" || report.FirstDirectCause.UnknownClass != UnknownDirect {
		t.Fatalf("first direct cause was not preserved: %#v", report.FirstDirectCause)
	}
	if len(report.Edges) != 1 || report.Edges[0].From != "PROJECT" || report.Edges[0].To != "ARTIFACT" {
		t.Fatalf("unexpected descent path: %#v", report.Edges)
	}
	receipt := report.Edges[0].Receipt
	if receipt.Stage != "PROJECT" || receipt.Step != "OBSERVE_PROJECT_EVIDENCE" || receipt.Reason != "PROJECT_EVIDENCE_MISSING" || receipt.UnknownClass != UnknownDirect || receipt.NextOperation != "PROVIDE_PROJECT_EVIDENCE" || len(receipt.BlockedBy) != 1 {
		t.Fatalf("descent receipt lost the six-coordinate cause: %#v", receipt)
	}
	if len(report.Edges[0].CausalFrontier) != 2 || len(report.MinimalDependencyBlockedFrontier) != 0 {
		t.Fatalf("descent frontier was not minimal: %#v", report)
	}
}

func TestResolveDependencyUnknownUsesAllFiveLevels(t *testing.T) {
	report := ResolveJSON([]byte(dependencyInput()), testMeta())
	if report.State != StateUnknown || report.Decision != DecisionUnknown {
		t.Fatalf("dependency unknown was not preserved: %#v", report)
	}
	if len(report.Edges) != 4 {
		t.Fatalf("expected one descent per unresolved ladder edge: %#v", report.Edges)
	}
	for index, edge := range report.Edges {
		if edge.From != ResolutionLevels[index] || edge.To != ResolutionLevels[index+1] {
			t.Fatalf("edge %d skipped a ladder level: %#v", index, edge)
		}
		if edge.Receipt.UnknownClass == "" || edge.Receipt.Stage == "" || edge.Receipt.Step == "" || edge.Receipt.Reason == "" || edge.Receipt.NextOperation == "" || len(edge.Receipt.BlockedBy) == 0 {
			t.Fatalf("edge %d receipt lost an unknown coordinate: %#v", index, edge.Receipt)
		}
	}
	if report.FirstDirectCause == nil || report.FirstDirectCause.Resolution != "PROJECT" {
		t.Fatalf("direct cause was replaced by a dependency blocker: %#v", report.FirstDirectCause)
	}
	if strings.Join(report.MinimalDependencyBlockedFrontier, ",") != "artifact-source" {
		t.Fatalf("dependency frontier was not minimal and deterministic: %#v", report.MinimalDependencyBlockedFrontier)
	}
}

func TestResolveUnknownDecisionFailsClosedWithoutFixedPoint(t *testing.T) {
	report := ResolveJSON([]byte(unknownDecisionInput()), testMeta())
	if report.State != StateUnknown || report.Decision != DecisionFailClosed || report.FeedbackCode != FeedbackDecisionUnknown {
		t.Fatalf("unknown upper decision was accepted: %#v", report)
	}
	if report.Claims[0].UnknownClass != UnknownDecision || report.Claims[0].Reason != FeedbackDecisionUnknown {
		t.Fatalf("unknown upper decision lost its feedback cause: %#v", report.Claims)
	}
}

func TestResolveKnownContradictionPrecedesUnknown(t *testing.T) {
	report := ResolveJSON([]byte(contradictionInput()), testMeta())
	if report.State != StateRefuted || report.Decision != DecisionFailClosed {
		t.Fatalf("known contradiction did not take precedence: %#v", report)
	}
	artifactRefuted := false
	for _, claim := range report.Claims {
		if claim.Resolution == "ARTIFACT" && claim.State == StateRefuted {
			artifactRefuted = true
		}
	}
	if !artifactRefuted {
		t.Fatalf("contradiction was not retained at its resolution: %#v", report.Claims)
	}
	if report.FeedbackCode != "" {
		t.Fatalf("unknown decision feedback survived a known refutation: %#v", report)
	}
}

func TestResolveRejectsMalformedFixedPointAndAuthorityEscalation(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		reason string
	}{
		{name: "malformed", input: "{", reason: "MALFORMED_INPUT"},
		{name: "fixed-point", input: strings.Replace(normalInput("fixed-point"), `"start_resolution":"PROJECT"`, `"start_resolution":"FIXED_POINT"`, 1), reason: "FIXED_POINT_NOT_SUCCESS"},
		{name: "authority", input: strings.Replace(normalInput("authority"), `"requested_repository_writes":0`, `"requested_repository_writes":1`, 1), reason: "AUTHORITY_ESCALATION_REFUTED"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			report := ResolveJSON([]byte(testCase.input), testMeta())
			if report.State != StateRefuted || report.Decision != DecisionFailClosed || report.Claims[0].Reason != testCase.reason {
				t.Fatalf("case was not fail-closed: %#v", report)
			}
		})
	}
}

func TestCanonicalDigestIgnoresJSONWhitespace(t *testing.T) {
	left := CanonicalDigest([]byte(`{"b":2,"a":1}`))
	right := CanonicalDigest([]byte("{\n  \"a\": 1,\n  \"b\": 2\n}"))
	if left != right {
		t.Fatalf("canonical digest changed with whitespace: %s %s", left, right)
	}
}

func testMeta() Meta {
	bindings := map[string]Binding{}
	for index, activity := range []string{"DescendProjectToArtifact", "DescendArtifactToActivity", "DescendActivityToPredicate", "DescendPredicateToField"} {
		bindings[activity] = Binding{Activity: activity, SourcePath: "main.gooo", IRNode: "ir-" + strings.ToLower(activity), Evaluator: "scripts/evaluate.sh", From: ResolutionLevels[index], To: ResolutionLevels[index+1]}
	}
	return Meta{SourcePath: "main.gooo", SourceDigest: "sha256:source", ContractDigest: "sha256:contract", ToolDigest: ToolDigest, Bindings: bindings}
}

func normalInput(caseID string) string {
	return `{"schema":"gooo/resolution-lattice/input/v2","case_id":"` + caseID + `","subject":"claim://test/1","start_resolution":"PROJECT","decision":"FIXED_POINT","evidence":{"PROJECT":{"status":"SUPPORTS","observed":"project","digest":"project"},"ARTIFACT":{"status":"SUPPORTS","observed":"artifact","digest":"artifact"},"ACTIVITY":{"status":"SUPPORTS","observed":"activity","digest":"activity"},"PREDICATE":{"status":"SUPPORTS","observed":"predicate","digest":"predicate"},"FIELD":{"status":"SUPPORTS","observed":"field","digest":"field"}},"authority":{"observation_mode":"READ_ONLY","requested_repository_writes":0,"requested_privilege_escalation":false},"improvement":{"fixture_digest":"fixture","input_digest":"input","tool_digest":"tool","contract_digest":"contract","exact_before":{"value":"before","digest":"before","unidentified_cause_frontier_count":4,"minimum_cause_reach_stage_count":5},"exact_after":{"value":"after","digest":"after","unidentified_cause_frontier_count":2,"minimum_cause_reach_stage_count":3},"utility_evidence":true}}`
}

func directUnknownInput() string {
	return strings.Replace(normalInput("direct"), `"status":"SUPPORTS","observed":"project","digest":"project"`, `"status":"MISSING","observed":"","digest":""`, 1)
}

func dependencyInput() string {
	input := normalInput("dependency")
	input = strings.Replace(input, `"decision":"FIXED_POINT",`, "", 1)
	input = strings.Replace(input, `"status":"SUPPORTS","observed":"project","digest":"project"`, `"status":"MISSING","observed":"","digest":""`, 1)
	for _, level := range []string{"ARTIFACT", "ACTIVITY", "PREDICATE"} {
		old := `"` + level + `":{"status":"SUPPORTS","observed":"` + strings.ToLower(level) + `","digest":"` + strings.ToLower(level) + `"}`
		new := `"` + level + `":{"status":"UNKNOWN","observed":"","digest":"","unknown_class":"DEPENDENCY_BLOCKED","blocked_by":["` + strings.ToLower(level) + `-source"]}`
		input = strings.Replace(input, old, new, 1)
	}
	return input
}

func unknownDecisionInput() string {
	return strings.Replace(normalInput("decision"), `"decision":"FIXED_POINT"`, `"decision":"UNKNOWN"`, 1)
}

func contradictionInput() string {
	input := strings.Replace(normalInput("contradiction"), `"decision":"FIXED_POINT",`, `"decision":"UNKNOWN",`, 1)
	input = strings.Replace(input, `"status":"SUPPORTS","observed":"project","digest":"project"`, `"status":"MISSING","observed":"","digest":""`, 1)
	return strings.Replace(input, `"status":"SUPPORTS","observed":"artifact","digest":"artifact"`, `"status":"CONTRADICTS","observed":"contradiction","digest":"contradiction"`, 1)
}
