package lattice

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

func LoadDenominator(path string) (Denominator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Denominator{}, err
	}
	var denominator Denominator
	if err := json.Unmarshal(data, &denominator); err != nil {
		return Denominator{}, err
	}
	if denominator.Schema != "gooo/resolution-lattice-denominator/v1" {
		return Denominator{}, errors.New("unsupported denominator schema")
	}
	if denominator.Total != len(denominator.Cells) || denominator.Total != 12 {
		return Denominator{}, errors.New("denominator total is not fixed at 12")
	}
	if len(denominator.Proofs) != 3 || len(denominator.IndicatorClasses) != 3 {
		return Denominator{}, errors.New("denominator balance dimensions changed")
	}
	seen := make(map[string]bool, len(denominator.Cells))
	for _, cell := range denominator.Cells {
		if cell.Ordinal < 1 || cell.Ordinal > denominator.Total || cell.ID == "" || cell.Activity == "" || seen[cell.Activity] {
			return Denominator{}, errors.New("denominator contains an invalid or duplicate activity")
		}
		seen[cell.Activity] = true
	}
	return denominator, nil
}

func CompileSource(sourcePath string, source []byte, denominator Denominator) (SemanticIR, error) {
	activities := parseActivities(string(source))
	if len(activities) != denominator.Total {
		return SemanticIR{}, fmt.Errorf("source has %d activities, denominator requires %d", len(activities), denominator.Total)
	}
	byActivity := denominator.CellByActivity()
	seen := make(map[string]bool, len(activities))
	nodes := make([]ActivityIR, 0, len(activities))
	for _, activity := range activities {
		cell, ok := byActivity[activity.name]
		if !ok {
			return SemanticIR{}, fmt.Errorf("activity %q is not in denominator", activity.name)
		}
		if seen[activity.name] {
			return SemanticIR{}, fmt.Errorf("activity %q is duplicated", activity.name)
		}
		seen[activity.name] = true
		nodes = append(nodes, ActivityIR{
			ID:         "gooo://resolution-lattice/activity/" + kebab(activity.name),
			Name:       activity.name,
			SourceLine: activity.line,
			MetricID:   cell.MetricID,
			Artifact:   cell.Artifact,
			Evaluator:  cell.Evaluator,
			Resolution: cell.Resolution,
			From:       cell.From,
			To:         cell.To,
		})
	}
	for _, cell := range denominator.Cells {
		if !seen[cell.Activity] {
			return SemanticIR{}, fmt.Errorf("denominator activity %q is absent from source", cell.Activity)
		}
	}
	return SemanticIR{
		Schema:           IRSchema,
		SourcePath:       sourcePath,
		SourceDigest:     Digest(source),
		ResolutionLadder: append([]string(nil), ResolutionLevels...),
		Nodes:            nodes,
	}, nil
}

func ParseIR(path string) (SemanticIR, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SemanticIR{}, err
	}
	var ir SemanticIR
	if err := json.Unmarshal(data, &ir); err != nil {
		return SemanticIR{}, err
	}
	if ir.Schema != IRSchema {
		return SemanticIR{}, errors.New("unsupported semantic IR schema")
	}
	return ir, nil
}

func LoadMeta(sourcePath, contractPath, irPath string) (Meta, error) {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return Meta{}, err
	}
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		return Meta{}, err
	}
	ir, err := ParseIR(irPath)
	if err != nil {
		return Meta{}, err
	}
	sourceDigest := Digest(source)
	if ir.SourceDigest != sourceDigest || len(ir.Nodes) != 12 || !sameStrings(ir.ResolutionLadder, ResolutionLevels) {
		return Meta{}, errors.New("semantic IR is not bound to the current Gooo source")
	}
	return Meta{
		SourcePath:     ir.SourcePath,
		SourceDigest:   sourceDigest,
		ContractDigest: Digest(contract),
		ToolDigest:     ToolDigest,
		Bindings:       BindingsFromIR(ir),
	}, nil
}

func BindingsFromIR(ir SemanticIR) map[string]Binding {
	bindings := make(map[string]Binding, len(ir.Nodes))
	for _, node := range ir.Nodes {
		bindings[node.Name] = Binding{
			Activity:   node.Name,
			SourcePath: ir.SourcePath,
			IRNode:     node.ID,
			MetricID:   node.MetricID,
			Artifact:   node.Artifact,
			Evaluator:  node.Evaluator,
			Resolution: node.Resolution,
			From:       node.From,
			To:         node.To,
		}
	}
	return bindings
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func CanonicalDigest(data []byte) string {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return Digest(data)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return Digest(data)
	}
	return Digest(canonical)
}

type parsedActivity struct {
	name string
	line int
}

func parseActivities(source string) []parsedActivity {
	lines := strings.Split(source, "\n")
	activities := make([]parsedActivity, 0)
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "activity ") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "activity "))
		open := strings.IndexByte(rest, '(')
		if open <= 0 {
			continue
		}
		activities = append(activities, parsedActivity{
			name: strings.TrimSpace(rest[:open]),
			line: index + 1,
		})
	}
	return activities
}

func kebab(value string) string {
	var builder strings.Builder
	for index, r := range value {
		if r >= 'A' && r <= 'Z' {
			if index > 0 {
				builder.WriteByte('-')
			}
			builder.WriteRune(r + ('a' - 'A'))
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
