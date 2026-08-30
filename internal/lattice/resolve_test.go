package lattice

import (
	"strings"
	"testing"
)

func TestResolveNormalDescendsAndDischarges(t *testing.T) {
	report := ResolveJSON([]byte(normalInput("normal")), testMeta())
	if report.State != StateClosed || report.Decision != "RESOLUTION_LATTICE_CLOSED" {
		t.Fatalf("normal input did not close: %#v", report)
	}
	if len(report.Edges) != 2 || len(report.CausalFrontier) != 2 {
		t.Fatalf("descent edges/frontier were not minimal: %#v", report)
	}
	if report.Improvement.Claim.State != StateClosed {
		t.Fatalf("exact improvement pair was not closed: %#v", report.Improvement)
	}
	if countLifecycle(report, LifecycleClose) != 3 {
		t.Fatalf("expected three discharged claims: %#v", report.History)
	}
	for _, edge := range report.Edges {
		if edge.Receipt.ID == "" || edge.Receipt.EdgeID != edge.ID || len(edge.CausalFrontier) != 2 {
			t.Fatalf("edge lost its receipt or frontier: %#v", edge)
		}
	}
}

func TestResolveUnknownPreservesSixCoordinatesAndLowersResolution(t *testing.T) {
	report := ResolveJSON([]byte(unknownInput()), testMeta())
	if report.State != StateUnknown || report.Decision != "RESOLUTION_LATTICE_UNKNOWN" {
		t.Fatalf("unknown input was not preserved: %#v", report)
	}
	if report.ResolvedClaim == nil || report.ResolvedClaim.Resolution != "EXISTENCE" {
		t.Fatalf("lower-resolution claim was not found: %#v", report.ResolvedClaim)
	}
	unknown := report.Claims[0]
	if unknown.State != StateUnknown || unknown.Stage == "" || unknown.Step == "" || unknown.Reason == "" || unknown.UnknownClass == "" || unknown.NextOperation == "" || len(unknown.BlockedBy) == 0 {
		t.Fatalf("UNKNOWN lost one of six coordinates: %#v", unknown)
	}
	if report.Claims[0].Reason != "EXACT_EVIDENCE_MISSING" || report.Claims[1].State != StateClosed {
		t.Fatalf("the unknown exact claim was laundered: %#v", report.Claims)
	}
	if report.Improvement.Claim.State != StateUnknown {
		t.Fatalf("missing exact before/after pair was inferred: %#v", report.Improvement)
	}
}

func TestResolveRefutedTakesPrecedence(t *testing.T) {
	report := ResolveJSON([]byte(refutedInput()), testMeta())
	if report.State != StateRefuted || report.Decision != "FAIL_CLOSED" {
		t.Fatalf("refuted input did not fail closed: %#v", report)
	}
	if len(report.Edges) != 0 || report.Claims[0].State != StateRefuted {
		t.Fatalf("refutation was not terminal: %#v", report)
	}
	if countLifecycle(report, LifecycleRefute) != 1 {
		t.Fatalf("refuted history was not preserved: %#v", report.History)
	}
}

func TestResolveRejectsMalformedFixedPointAndAuthorityEscalation(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		reason string
	}{
		{name: "malformed", input: "{", reason: "MALFORMED_INPUT"},
		{name: "fixed-point", input: strings.Replace(normalInput("fixed-point"), "\"EXACT\"", "\"FIXED_POINT\"", 1), reason: "FIXED_POINT_NOT_SUCCESS"},
		{name: "authority", input: strings.Replace(normalInput("authority"), "\"requested_repository_writes\":0", "\"requested_repository_writes\":1", 1), reason: "AUTHORITY_ESCALATION_REFUTED"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			report := ResolveJSON([]byte(testCase.input), testMeta())
			if report.State != StateRefuted || report.Decision != "FAIL_CLOSED" || report.Claims[0].Reason != testCase.reason {
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
	bindings := map[string]Binding{
		"DescendExactToInvariant":     {Activity: "DescendExactToInvariant", SourcePath: "main.gooo", IRNode: "ir-exact-invariant", Evaluator: "scripts/evaluate.sh"},
		"DescendInvariantToExistence": {Activity: "DescendInvariantToExistence", SourcePath: "main.gooo", IRNode: "ir-invariant-existence", Evaluator: "scripts/evaluate.sh"},
	}
	return Meta{SourcePath: "main.gooo", SourceDigest: "sha256:source", ContractDigest: "sha256:contract", ToolDigest: ToolDigest, Bindings: bindings}
}

func countLifecycle(report Report, lifecycle string) int {
	count := 0
	for _, event := range report.History {
		if event.Lifecycle == lifecycle {
			count++
		}
	}
	return count
}

func normalInput(caseID string) string {
	return `{"schema":"gooo/resolution-lattice/input/v1","case_id":"` + caseID + `","subject":"claim://test/1","start_resolution":"EXACT","evidence":{"EXACT":{"status":"SUPPORTS","observed":"exact","digest":"exact"},"INVARIANT":{"status":"SUPPORTS","observed":"invariant","digest":"invariant"},"EXISTENCE":{"status":"SUPPORTS","observed":"existence","digest":"existence"}},"authority":{"observation_mode":"READ_ONLY","requested_repository_writes":0,"requested_privilege_escalation":false},"improvement":{"input_digest":"input","tool_digest":"tool","contract_digest":"contract","exact_before":{"value":"before","digest":"before"},"exact_after":{"value":"after","digest":"after"},"utility_evidence":true}}`
}

func unknownInput() string {
	return `{"schema":"gooo/resolution-lattice/input/v1","case_id":"unknown","subject":"claim://test/2","start_resolution":"EXACT","evidence":{"EXACT":{"status":"MISSING","observed":"","digest":""},"INVARIANT":{"status":"SUPPORTS","observed":"invariant","digest":"invariant"},"EXISTENCE":{"status":"SUPPORTS","observed":"existence","digest":"existence"}},"authority":{"observation_mode":"READ_ONLY","requested_repository_writes":0,"requested_privilege_escalation":false},"improvement":{"input_digest":"input","tool_digest":"tool","contract_digest":"contract","exact_before":{"value":"before","digest":"before"},"exact_after":null,"utility_evidence":true}}`
}

func refutedInput() string {
	return `{"schema":"gooo/resolution-lattice/input/v1","case_id":"refuted","subject":"claim://test/3","start_resolution":"EXACT","evidence":{"EXACT":{"status":"CONTRADICTS","observed":"contradiction","digest":"contradiction"},"INVARIANT":{"status":"SUPPORTS","observed":"invariant","digest":"invariant"},"EXISTENCE":{"status":"SUPPORTS","observed":"existence","digest":"existence"}},"authority":{"observation_mode":"READ_ONLY","requested_repository_writes":0,"requested_privilege_escalation":false}}`
}
