package lattice

import "encoding/json"

const (
	Schema          = "gooo/resolution-lattice/v1"
	InputSchema     = "gooo/resolution-lattice/input/v1"
	IRSchema        = "gooo/resolution-lattice/ir/v1"
	ToolDigest      = "sha256:3a7f4b5c0d7b9e3f1a7a7a5df3c1cbd91e68bf7a2b79d720d4b0c8e2b37f7e62"
	StateClosed     = "CLOSED"
	StateUnknown    = "UNKNOWN"
	StateRefuted    = "REFUTED"
	LifecycleOpen   = "OPEN"
	LifecycleClose  = "DISCHARGED"
	LifecycleRefute = "REFUTED"
)

var ResolutionLevels = []string{"EXACT", "INVARIANT", "EXISTENCE"}

type Input struct {
	Schema          string              `json:"schema"`
	CaseID          string              `json:"case_id"`
	Subject         string              `json:"subject"`
	StartResolution string              `json:"start_resolution"`
	Evidence        map[string]Evidence `json:"evidence"`
	Authority       AuthorityInput      `json:"authority"`
	Improvement     *ImprovementInput   `json:"improvement"`
}

type Evidence struct {
	Status   string `json:"status"`
	Observed string `json:"observed"`
	Digest   string `json:"digest"`
}

type AuthorityInput struct {
	ObservationMode              string `json:"observation_mode"`
	RequestedRepositoryWrites    int    `json:"requested_repository_writes"`
	RequestedPrivilegeEscalation bool   `json:"requested_privilege_escalation"`
}

type ImprovementInput struct {
	InputDigest     string      `json:"input_digest"`
	ToolDigest      string      `json:"tool_digest"`
	ContractDigest  string      `json:"contract_digest"`
	ExactBefore     *ExactValue `json:"exact_before"`
	ExactAfter      *ExactValue `json:"exact_after"`
	UtilityEvidence bool        `json:"utility_evidence"`
}

type ExactValue struct {
	Value  string `json:"value"`
	Digest string `json:"digest"`
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
	Schema            string `json:"schema"`
	ID                string `json:"id"`
	EdgeID            string `json:"edge_id"`
	CaseID            string `json:"case_id"`
	Activity          string `json:"activity"`
	From              string `json:"from"`
	To                string `json:"to"`
	InputDigest       string `json:"input_digest"`
	ToolDigest        string `json:"tool_digest"`
	ContractDigest    string `json:"contract_digest"`
	SourceDigest      string `json:"source_digest"`
	OutputClaimDigest string `json:"output_claim_digest"`
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

type ImprovementResult struct {
	Claim              Claim `json:"claim"`
	PairPresent        bool  `json:"pair_present"`
	SameInputDigest    bool  `json:"same_input_digest"`
	SameToolDigest     bool  `json:"same_tool_digest"`
	SameContractDigest bool  `json:"same_contract_digest"`
}

type Binding struct {
	Activity   string `json:"activity"`
	SourcePath string `json:"source_path"`
	IRNode     string `json:"ir_node"`
	MetricID   string `json:"metric_id"`
	Artifact   string `json:"artifact"`
	Evaluator  string `json:"evaluator"`
}

type Meta struct {
	SourcePath     string
	SourceDigest   string
	ContractDigest string
	ToolDigest     string
	Bindings       map[string]Binding
}

type Report struct {
	Schema             string            `json:"schema"`
	Decision           string            `json:"decision"`
	CaseID             string            `json:"case_id"`
	Subject            string            `json:"subject"`
	InputDigest        string            `json:"input_digest"`
	ToolDigest         string            `json:"tool_digest"`
	ContractDigest     string            `json:"contract_digest"`
	State              string            `json:"state"`
	Precedence         []string          `json:"precedence"`
	Claims             []Claim           `json:"claims"`
	ResolvedClaim      *Claim            `json:"resolved_claim"`
	History            []HistoryEvent    `json:"history"`
	Edges              []DescentEdge     `json:"edges"`
	CausalFrontier     []string          `json:"causal_frontier"`
	Improvement        ImprovementResult `json:"improvement"`
	Authority          AuthorityReport   `json:"authority"`
	GeneratedArtifacts []string          `json:"generated_artifacts"`
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
}

type SemanticIR struct {
	Schema       string       `json:"schema"`
	SourcePath   string       `json:"source_path"`
	SourceDigest string       `json:"source_digest"`
	Nodes        []ActivityIR `json:"nodes"`
}

type ActivityIR struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SourceLine int    `json:"source_line"`
	MetricID   string `json:"metric_id"`
	Artifact   string `json:"artifact"`
	Evaluator  string `json:"evaluator"`
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
