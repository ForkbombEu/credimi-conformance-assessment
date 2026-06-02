package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"credimi-conformance-assessment/internal/assessment"
	"credimi-conformance-assessment/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "credimi-assess:", err)
		os.Exit(1)
	}
}
func run() error {
	envPath := flag.String("env", ".env", "path to .env config")
	inputJSON := flag.String("input-json", "", "path to JSON assessment input")
	fixtureName := flag.String("fixture", "", "fixture name for inline JSON input or legacy fixture selection")
	fixturesDir := flag.String("fixtures-dir", "", "legacy fixture directory")
	pipelineDir := flag.String("pipeline-dir", "", "legacy extracted pipeline artifact directory")
	sourceDir := flag.String("source-dir", "", "legacy source-of-truth directory override")
	extractedDir := flag.String("extracted-dir", "", "legacy extracted pipeline artifact directory")
	outDir := flag.String("out-dir", "", "legacy output directory override")
	flag.Parse()
	cfg := config.Load(*envPath)
	opts := assessment.Options{SourceDir: cfg.SourceDir, TemporalData: cfg.TemporalData, OutDir: cfg.OutDir, Fixture: *fixtureName}
	if *sourceDir != "" {
		opts.SourceDir = *sourceDir
	}
	if *outDir != "" {
		opts.OutDir = *outDir
	}
	if *fixturesDir != "" {
		opts.FixturesDir = *fixturesDir
	}
	if *pipelineDir != "" {
		opts.ExtractedDir = *pipelineDir
	}
	if *extractedDir != "" {
		opts.ExtractedDir = *extractedDir
	}
	path := *inputJSON
	if path == "" {
		path = cfg.TemporalData
	}
	if path != "" {
		req, err := assessment.LoadRequest(path)
		if err != nil {
			return err
		}
		opts = assessment.ApplyRequest(opts, req)
	}
	res, err := assessment.Generate(opts)
	if err != nil {
		return err
	}
	if opts.OutDir == "" {
		for i, rep := range res.Reports {
			if i > 0 {
				fmt.Println()
			}
			fmt.Print(rep.Markdown)
		}
		return nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}
