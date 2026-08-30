package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-resolution-lattice/internal/lattice"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "compile":
		return compile(args[1:], stdout, stderr)
	case "resolve":
		return resolve(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, "gooo-lattice/v1")
		return 0
	default:
		usage(stderr)
		return 2
	}
}

func compile(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("compile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourcePath := flags.String("source", "examples/resolution-lattice/main.gooo", "Gooo source")
	contractPath := flags.String("contract", "contracts/resolution-lattice-denominator-v1.json", "fixed denominator")
	outputPath := flags.String("output", "semantic-ir.json", "generated semantic IR")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "compile accepts flags only")
		return 2
	}
	source, err := os.ReadFile(*sourcePath)
	if err != nil {
		fmt.Fprintf(stderr, "read Gooo source: %v\n", err)
		return 1
	}
	denominator, err := lattice.LoadDenominator(*contractPath)
	if err != nil {
		fmt.Fprintf(stderr, "read denominator: %v\n", err)
		return 1
	}
	ir, err := lattice.CompileSource(*sourcePath, source, denominator)
	if err != nil {
		fmt.Fprintf(stderr, "compile Gooo source: %v\n", err)
		return 1
	}
	if err := writeJSON(*outputPath, ir); err != nil {
		fmt.Fprintf(stderr, "write semantic IR: %v\n", err)
		return 1
	}
	if err := writeJSON(stdout, ir); err != nil {
		fmt.Fprintf(stderr, "emit semantic IR: %v\n", err)
		return 1
	}
	return 0
}

func resolve(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("resolve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	inputPath := flags.String("input", "", "JSON claim input")
	outputDir := flags.String("output-dir", "", "caller-owned output directory")
	sourcePath := flags.String("source", "examples/resolution-lattice/main.gooo", "Gooo source")
	contractPath := flags.String("contract", "contracts/resolution-lattice-denominator-v1.json", "fixed denominator")
	irPath := flags.String("ir", "semantic-ir.json", "generated semantic IR")
	jsonOutput := flags.Bool("json", false, "emit the report as JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *inputPath == "" || *outputDir == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "resolve requires --input, --output-dir, and no positional arguments")
		return 2
	}
	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	meta, err := lattice.LoadMeta(*sourcePath, *contractPath, *irPath)
	if err != nil {
		fmt.Fprintf(stderr, "load meta binding: %v\n", err)
		return 1
	}
	report := lattice.ResolveJSON(raw, meta)
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "create output directory: %v\n", err)
		return 1
	}
	if err := writeJSON(filepath.Join(*outputDir, "report.json"), report); err != nil {
		fmt.Fprintf(stderr, "write report: %v\n", err)
		return 1
	}
	receipts := make([]lattice.GeneratedReceipt, 0, len(report.Edges))
	for _, edge := range report.Edges {
		receipts = append(receipts, edge.Receipt)
	}
	receiptBundle := struct {
		Schema   string                      `json:"schema"`
		CaseID   string                      `json:"case_id"`
		Receipts []lattice.GeneratedReceipt `json:"receipts"`
	}{
		Schema:   "gooo/resolution-lattice/receipts/v1",
		CaseID:   report.CaseID,
		Receipts: receipts,
	}
	if err := writeJSON(filepath.Join(*outputDir, "receipts.json"), receiptBundle); err != nil {
		fmt.Fprintf(stderr, "write receipts: %v\n", err)
		return 1
	}
	if *jsonOutput {
		if err := writeJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "emit report: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "%s case=%s state=%s claims=%d edges=%d\n", report.Decision, report.CaseID, report.State, len(report.Claims), len(report.Edges))
	return 0
}

func writeJSON(target any, value any) error {
	var writer io.Writer
	switch typed := target.(type) {
	case string:
		parent := filepath.Dir(typed)
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
		file, err := os.Create(typed)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	case io.Writer:
		writer = typed
	default:
		return fmt.Errorf("unsupported JSON target %T", target)
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func usage(stderr io.Writer) {
	fmt.Fprintln(stderr, "usage: gooo-lattice compile --source PATH --contract PATH --output PATH")
	fmt.Fprintln(stderr, "       gooo-lattice resolve --input PATH --output-dir PATH --source PATH --contract PATH --ir PATH")
}
