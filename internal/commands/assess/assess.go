package assess

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/forkbombeu/credimi-conformance-assessment/internal/cli"
	"github.com/forkbombeu/credimi-conformance-assessment/internal/config"
	"github.com/forkbombeu/credimi-conformance-assessment/pkg/conformance"
)

// Run executes the assess CLI command.
func Run(args []string) error {
	fs := flag.NewFlagSet("assess", flag.ExitOnError)
	envPath := fs.String("env", ".env", "path to .env config")
	inputJSON := fs.String("input-json", "", "path to JSON assessment input")
	fixtureName := fs.String("fixture", "", "fixture name for inline JSON input or legacy fixture selection")
	fixturesDir := fs.String("fixtures-dir", "", "legacy fixture directory")
	pipelineDir := fs.String("pipeline-dir", "", "legacy extracted pipeline artifact directory")
	sourceDir := fs.String("source-dir", "", "legacy source-of-truth directory override")
	extractedDir := fs.String("extracted-dir", "", "legacy extracted pipeline artifact directory")
	outDir := fs.String("out-dir", "", "legacy output directory override")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, cli.ASCIIArt)
	}

	cfg := config.Load(*envPath)
	input := conformance.ReportInput{Fixture: *fixtureName}
	opts := conformance.ReportOptions{SourceDir: cfg.SourceDir, OutDir: cfg.OutDir}
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
		req, err := conformance.LoadInput(path)
		if err != nil {
			return err
		}
		if req.Fixture == "" {
			req.Fixture = input.Fixture
		}
		input = req
	}

	res, err := conformance.Generate(input, opts)
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
