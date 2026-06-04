// Package conformance exposes Credimi conformance report generation for CLIs,
// HTTP handlers, and future Temporal activities.
package conformance

import (
	"encoding/json"
	"fmt"
	"os"

	"credimi-conformance-assessment/internal/assessment"
)

// ReportInput is the stable request payload for generating conformance reports.
type ReportInput struct {
	Fixture        string          `json:"fixture,omitempty"`
	TemporalInput  json.RawMessage `json:"temporal_input,omitempty"`
	TemporalOutput json.RawMessage `json:"temporal_output,omitempty"`
	PipelineInput  json.RawMessage `json:"pipeline_input,omitempty"`
	PipelineOutput json.RawMessage `json:"pipeline_output,omitempty"`
}

// ReportOptions configures source material and optional filesystem output.
type ReportOptions struct {
	SourceDir    string `json:"source_dir,omitempty"`
	FixturesDir  string `json:"fixtures_dir,omitempty"`
	ExtractedDir string `json:"extracted_dir,omitempty"`
	OutDir       string `json:"out_dir,omitempty"`
}

// Report describes one generated assessment.
type Report struct {
	Fixture     string `json:"fixture"`
	Slug        string `json:"slug"`
	Path        string `json:"path,omitempty"`
	PassedCount int    `json:"passed_count"`
	Markdown    string `json:"markdown,omitempty"`
}

// ReportResult is the generated report output.
type ReportResult struct {
	Reports []Report `json:"reports"`
}

// ActivityPayload mirrors the payload shape used by Credimi activities.
type ActivityPayload struct {
	ReportInput
	ReportOptions
}

// ActivityInput mirrors credimi-2 workflowengine.ActivityInput without taking a
// dependency on Temporal or credimi-2.
type ActivityInput struct {
	Payload ActivityPayload   `json:"payload,omitempty"`
	Config  map[string]string `json:"config,omitempty"`
}

// ActivityResult mirrors credimi-2 workflowengine.ActivityResult.
type ActivityResult struct {
	Output ReportResult `json:"output,omitempty"`
	Log    []string     `json:"log,omitempty"`
}

// LoadInput reads a ReportInput JSON file.
func LoadInput(path string) (ReportInput, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ReportInput{}, fmt.Errorf("read report input: %w", err)
	}
	var input ReportInput
	if err := json.Unmarshal(b, &input); err != nil {
		return ReportInput{}, fmt.Errorf("decode report input: %w", err)
	}
	return input, nil
}

// Generate returns reports for inline payloads or fixture directories.
func Generate(input ReportInput, opts ReportOptions) (ReportResult, error) {
	res, err := assessment.Generate(toInternalOptions(input, opts))
	if err != nil {
		return ReportResult{}, err
	}
	return fromInternalResult(res), nil
}

// GenerateActivity is a small adapter for embedding this package in a Temporal
// activity implementation.
func GenerateActivity(input ActivityInput) (ActivityResult, error) {
	res, err := Generate(input.Payload.ReportInput, input.Payload.ReportOptions)
	if err != nil {
		return ActivityResult{}, err
	}
	return ActivityResult{Output: res}, nil
}

func toInternalOptions(input ReportInput, opts ReportOptions) assessment.Options {
	return assessment.Options{
		SourceDir:      opts.SourceDir,
		OutDir:         opts.OutDir,
		Fixture:        input.Fixture,
		TemporalInput:  input.TemporalInput,
		TemporalOutput: input.TemporalOutput,
		PipelineInput:  input.PipelineInput,
		PipelineOutput: input.PipelineOutput,
		FixturesDir:    opts.FixturesDir,
		ExtractedDir:   opts.ExtractedDir,
	}
}

func fromInternalResult(res assessment.Result) ReportResult {
	out := ReportResult{Reports: make([]Report, 0, len(res.Reports))}
	for _, rep := range res.Reports {
		out.Reports = append(out.Reports, Report{
			Fixture:     rep.Fixture,
			Slug:        rep.Slug,
			Path:        rep.Path,
			PassedCount: rep.PassedCount,
			Markdown:    rep.Markdown,
		})
	}
	return out
}
