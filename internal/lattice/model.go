package lattice

import "encoding/json"

const (
	Schema                  = "gooo/resolution-lattice/v2"
	InputSchema             = "gooo/resolution-lattice/input/v2"
	LegacyInputSchema       = "gooo/resolution-lattice/input/v1"
	IRSchema                = "gooo/resolution-lattice/ir/v2"
	ToolDigest              = "sha256:7c4f0f2d9c1a5e8b3a6d4f1e0c2b9a8d6e5f4c3b2a1908172635443322110ffe"
	StateClosed             = "CLOSED"
	StateUnknown            = "UNKNOWN"
	StateRefuted            = "REFUTED"
	LifecycleOpen           = "OPEN"
	LifecycleClose          = "DISCHARGED"
	LifecycleRefute         = "REFUTED"
	UnknownDirect           = "DIRECT_MISSING"
	UnknownDependency       = "DEPENDENCY_BLOCKED"
	UnknownDecision         = "DECISION_UNKNOWN"
	UnknownCausality        = "CAUSALITY_UNPROVEN"
	FeedbackDecisionUnknown = "FEEDBACK_COVERAGE_DECISION_UNKNOWN"
	DecisionFailClosed      = "FAIL_CLOSED"
	DecisionClosed          = "RESOLUTION_LATTICE_CLOSED"
	DecisionUnknown         = "RESOLUTION_LATTICE_UNKNOWN"
)

var ResolutionLevels = []string{"PROJECT", "ARTIFACT", "ACTIVITY", "PREDICATE", "FIELD"}

var UnknownClasses = []string{UnknownDirect, UnknownDependency, UnknownDecision, UnknownCausality}

type Input struct {
	Schema          string              `json:"schema"`
	CaseID          string              `json:"case_id"`
	Subject         string              `json:"subject"`
	StartResolution string              `json:"start_resolution"`
	Decision        string              `json:"decision"`
	Evidence        map[string]Evidence `json:"evidence"`
	Authority       AuthorityInput      `json:"authority"`
	Improvement     *ImprovementInput   `json:"improvement"`
}

type Evidence struct {
	Status        string   `json:"status"`
	Observed      string   `json:"observed"`
	Digest        string   `json:"digest"`
	Stage         string   `json:"stage,omitempty"`
	Step          string   `json:"step,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	UnknownClass  string   `json:"unknown_class,omitempty"`
	NextOperation string   `json:"next_operation,omitempty"`
	BlockedBy     []string `json:"blocked_by,omitempty"`
}

type AuthorityInput struct {
	ObservationMode              string `json:"observation_mode"`
	RequestedRepositoryWrites    int    `json:"requested_repository_writes"`
	RequestedPrivilegeEscalation bool   `json:"requested_privilege_escalation"`
}

type ImprovementInput struct {
	FixtureDigest   string      `json:"fixture_digest"`
	InputDigest     string      `json:"input_digest"`
	ToolDigest      string      `json:"tool_digest"`
	ContractDigest  string      `json:"contract_digest"`
	ExactBefore     *ExactValue `json:"exact_before"`
	ExactAfter      *ExactValue `json:"exact_after"`
	UtilityEvidence bool        `json:"utility_evidence"`
}

type ExactValue struct {
	Value                          string `json:"value"`
	Digest                         string `json:"digest"`
	FixtureDigest                  string `json:"fixture_digest,omitempty"`
	InputDigest                    string `json:"input_digest,omitempty"`
	ToolDigest                     string `json:"tool_digest,omitempty"`
	ContractDigest                 string `json:"contract_digest,omitempty"`
	UnidentifiedCauseFrontierCount *int   `json:"unidentified_cause_frontier_count,omitempty"`
	MinimumCauseReachStageCount    *int   `json:"minimum_cause_reach_stage_count,omitempty"`
	MinimumCauseReachStages        *int   `json:"minimum_cause_reach_stages,omitempty"`
}

type Claim struct {
	ID            string   `json:"id"`
	Resolution    string   `json:"resolution"`
	State         string   `json:"state"`
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type HistoryEvent struct {
	Sequence   int    `json:"sequence"`
	ID         string `json:"id"`
	ClaimID    string `json:"claim_id"`
	Resolution string `json:"resolution"`
	Lifecycle  string `json:"lifecycle"`
	Activity   string `json:"activity"`
	ReceiptID  string `json:"receipt_id"`
	Claim      Claim  `json:"claim"`
}

type GeneratedReceipt struct {
	Schema            string   `json:"schema"`
	ID                string   `json:"id"`
	EdgeID            string   `json:"edge_id"`
	CaseID            string   `json:"case_id"`
	Activity          string   `json:"activity"`
	From              string   `json:"from"`
	To                string   `json:"to"`
	Stage             string   `json:"stage"`
	Step              string   `json:"step"`
	Reason            string   `json:"reason"`
	UnknownClass      string   `json:"unknown_class"`
	NextOperation     string   `json:"next_operation"`
	BlockedBy         []string `json:"blocked_by"`
	InputDigest       string   `json:"input_digest"`
	ToolDigest        string   `json:"tool_digest"`
	ContractDigest    string   `json:"contract_digest"`
	SourceDigest      string   `json:"source_digest"`
	OutputClaimDigest string   `json:"output_claim_digest"`
}

type DescentEdge struct {
	ID                string           `json:"id"`
	From              string           `json:"from"`
	To                string           `json:"to"`
	Activity          string           `json:"activity"`
	SourcePath        string           `json:"source_path"`
	SourceDigest      string           `json:"source_digest"`
	IRNode            string           `json:"ir_node"`
	GeneratedArtifact string           `json:"generated_artifact"`
	Evaluator         string           `json:"evaluator"`
	Receipt           GeneratedReceipt `json:"receipt"`
	CausalFrontier    []string         `json:"causal_frontier"`
}

type ImprovementMetric struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	Before    *int   `json:"before"`
	After     *int   `json:"after"`
	Delta     *int   `json:"delta"`
	ExactPair bool   `json:"exact_pair"`
}

type ImprovementResult struct {
	Claim              Claim               `json:"claim"`
	PairPresent        bool                `json:"pair_present"`
	SameFixtureDigest  bool                `json:"same_fixture_digest"`
	SameInputDigest    bool                `json:"same_input_digest"`
	SameToolDigest     bool                `json:"same_tool_digest"`
	SameContractDigest bool                `json:"same_contract_digest"`
	Metrics            []ImprovementMetric `json:"metrics"`
}

type Binding struct {
	Activity   string `json:"activity"`
	SourcePath string `json:"source_path"`
	IRNode     string `json:"ir_node"`
	MetricID   string `json:"metric_id"`
	Artifact   string `json:"artifact"`
	Evaluator  string `json:"evaluator"`
	Resolution string `json:"resolution,omitempty"`
	From       string `json:"from,omitempty"`
	To         string `json:"to,omitempty"`
}

type Meta struct {
	SourcePath     string
	SourceDigest   string
	ContractDigest string
	ToolDigest     string
	Bindings       map[string]Binding
}

type Report struct {
	Schema                           string            `json:"schema"`
	Decision                         string            `json:"decision"`
	FeedbackCode                     string            `json:"feedback_code,omitempty"`
	DecisionInput                    string            `json:"decision_input,omitempty"`
	CaseID                           string            `json:"case_id"`
	Subject                          string            `json:"subject"`
	InputDigest                      string            `json:"input_digest"`
	ToolDigest                       string            `json:"tool_digest"`
	ContractDigest                   string            `json:"contract_digest"`
	ResolutionLadder                 []string          `json:"resolution_ladder"`
	State                            string            `json:"state"`
	Precedence                       []string          `json:"precedence"`
	Claims                           []Claim           `json:"claims"`
	ResolvedClaim                    *Claim            `json:"resolved_claim"`
	FirstDirectCause                 *Claim            `json:"first_direct_cause"`
	MinimalDependencyBlockedFrontier []string          `json:"minimal_dependency_blocked_frontier"`
	History                          []HistoryEvent    `json:"history"`
	Edges                            []DescentEdge     `json:"edges"`
	CausalFrontier                   []string          `json:"causal_frontier"`
	Improvement                      ImprovementResult `json:"improvement"`
	Authority                        AuthorityReport   `json:"authority"`
	GeneratedArtifacts               []string          `json:"generated_artifacts"`
}

type AuthorityReport struct {
	ObservationMode       string `json:"observation_mode"`
	RepositoryWrites      int    `json:"repository_writes"`
	InputRepositoryWrites int    `json:"input_repository_writes"`
	ReadOnly              bool   `json:"read_only"`
}

type Denominator struct {
	Schema           string    `json:"schema"`
	DenominatorID    string    `json:"denominator_id"`
	CandidateID      string    `json:"candidate_id"`
	Total            int       `json:"total"`
	Proofs           []Balance `json:"proofs"`
	IndicatorClasses []Balance `json:"indicator_classes"`
	Cells            []Cell    `json:"cells"`
}

type Balance struct {
	Choice string `json:"choice"`
	Class  string `json:"class"`
	Total  int    `json:"total"`
}

type Cell struct {
	Ordinal        int    `json:"ordinal"`
	ID             string `json:"id"`
	Activity       string `json:"activity"`
	ProofChoice    string `json:"proof_choice"`
	IndicatorClass string `json:"indicator_class"`
	MetricID       string `json:"metric_id"`
	MetricPath     string `json:"metric_path"`
	Artifact       string `json:"artifact"`
	Evaluator      string `json:"evaluator"`
	Resolution     string `json:"resolution,omitempty"`
	From           string `json:"from,omitempty"`
	To             string `json:"to,omitempty"`
}

type SemanticIR struct {
	Schema           string       `json:"schema"`
	SourcePath       string       `json:"source_path"`
	SourceDigest     string       `json:"source_digest"`
	ResolutionLadder []string     `json:"resolution_ladder"`
	Nodes            []ActivityIR `json:"nodes"`
}

type ActivityIR struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SourceLine int    `json:"source_line"`
	MetricID   string `json:"metric_id"`
	Artifact   string `json:"artifact"`
	Evaluator  string `json:"evaluator"`
	Resolution string `json:"resolution,omitempty"`
	From       string `json:"from,omitempty"`
	To         string `json:"to,omitempty"`
}

func (d Denominator) CellByActivity() map[string]Cell {
	result := make(map[string]Cell, len(d.Cells))
	for _, cell := range d.Cells {
		result[cell.Activity] = cell
	}
	return result
}

func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
