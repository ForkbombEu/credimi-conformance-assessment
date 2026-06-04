// Package conformance exposes Credimi conformance report generation.
package conformance

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/forkbombeu/credimi-conformance-assessment/internal/assessment"
)

// ReportInput is the stable request payload for generating conformance reports.
type ReportInput struct {
	Fixture        string          `json:"fixture,omitempty"`
	PipelineInput  json.RawMessage `json:"pipeline_input,omitempty"`
	PipelineOutput json.RawMessage `json:"pipeline_output,omitempty"`
	Evidence       json.RawMessage `json:"evidence,omitempty"`
}

// UnmarshalJSON accepts both the package vocabulary and Credimi runtime
// vocabulary. Runtime callers usually send temporal_input/temporal_output.
func (r *ReportInput) UnmarshalJSON(b []byte) error {
	type alias ReportInput
	var raw struct {
		alias
		TemporalInput        json.RawMessage `json:"temporal_input,omitempty"`
		TemporalInputDash    json.RawMessage `json:"temporal-input,omitempty"`
		TemporalOutput       json.RawMessage `json:"temporal_output,omitempty"`
		TemporalOutputDash   json.RawMessage `json:"temporal-output,omitempty"`
		CredentialOffers     json.RawMessage `json:"credential_offers,omitempty"`
		CredentialWellKnowns json.RawMessage `json:"credential_well_knowns,omitempty"`
		WellKnowns           json.RawMessage `json:"well_knowns,omitempty"`
		PresentationResults  json.RawMessage `json:"presentation_results,omitempty"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*r = ReportInput(raw.alias)
	if len(r.PipelineInput) == 0 {
		r.PipelineInput = firstRaw(raw.TemporalInput, raw.TemporalInputDash)
	}
	if len(r.PipelineOutput) == 0 {
		r.PipelineOutput = firstRaw(raw.TemporalOutput, raw.TemporalOutputDash)
	}
	if len(r.Evidence) == 0 {
		r.Evidence = evidenceEnvelope(raw.CredentialOffers, raw.CredentialWellKnowns, raw.WellKnowns, raw.PresentationResults)
	}
	return nil
}

func firstRaw(values ...json.RawMessage) json.RawMessage {
	for _, v := range values {
		if len(v) > 0 {
			return v
		}
	}
	return nil
}

func evidenceEnvelope(credentialOffers, credentialWellKnowns, wellKnowns, presentationResults json.RawMessage) json.RawMessage {
	m := map[string]json.RawMessage{}
	if len(credentialOffers) > 0 {
		m["credential_offers"] = credentialOffers
	}
	if len(credentialWellKnowns) > 0 {
		m["credential_well_knowns"] = credentialWellKnowns
	} else if len(wellKnowns) > 0 {
		m["credential_well_knowns"] = wellKnowns
	}
	if len(presentationResults) > 0 {
		m["presentation_results"] = presentationResults
	}
	if len(m) == 0 {
		return nil
	}
	b, _ := json.Marshal(m)
	return b
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

// Generate returns reports from the supplied runtime payload and source-of-truth rules.
func Generate(input ReportInput, opts ReportOptions) (ReportResult, error) {
	res, err := assessment.Generate(toInternalOptions(input, opts))
	if err != nil {
		return ReportResult{}, err
	}
	return fromInternalResult(res), nil
}

func toInternalOptions(input ReportInput, opts ReportOptions) assessment.Options {
	return assessment.Options{
		SourceDir:      opts.SourceDir,
		OutDir:         opts.OutDir,
		Fixture:        input.Fixture,
		PipelineInput:  input.PipelineInput,
		PipelineOutput: input.PipelineOutput,
		Evidence:       input.Evidence,
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
