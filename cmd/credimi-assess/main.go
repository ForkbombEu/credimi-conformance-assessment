package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"credimi-conformance-assessment/internal/facts"
	"credimi-conformance-assessment/internal/fixture"
	"credimi-conformance-assessment/internal/report"
	"credimi-conformance-assessment/internal/rules"
	"credimi-conformance-assessment/internal/sot"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "credimi-assess:", err)
		os.Exit(1)
	}
}
func run() error {
	sourceDir := flag.String("source-dir", "./source-of-truth", "")
	fixturesDir := flag.String("fixtures-dir", "./fixtures", "")
	extractedDir := flag.String("extracted-dir", "./out", "")
	outDir := flag.String("out-dir", "./assessments", "")
	selected := flag.String("fixture", "", "")
	flag.Parse()
	src, err := sot.Load(*sourceDir)
	if err != nil {
		return err
	}
	fs, err := fixture.List(*fixturesDir, *extractedDir, *selected)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		return err
	}
	for _, f := range fs {
		af, err := facts.Build(f)
		if err != nil {
			return err
		}
		res := rules.Evaluate(src.Taxonomy, af)
		md := report.Render(af, src.FlatTests, res)
		name := fmt.Sprintf("conformance-assessment-%s.md", f.Slug)
		if err := os.WriteFile(filepath.Join(*outDir, name), []byte(md), 0644); err != nil {
			return err
		}
	}
	return nil
}
