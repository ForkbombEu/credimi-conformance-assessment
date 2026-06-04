package assessment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"credimi-conformance-assessment/internal/facts"
	"credimi-conformance-assessment/internal/fixture"
	"credimi-conformance-assessment/internal/report"
	"credimi-conformance-assessment/internal/rules"
	"credimi-conformance-assessment/internal/sot"
)

type Options struct {
	SourceDir      string
	TemporalData   string
	OutDir         string
	Fixture        string
	TemporalInput  json.RawMessage
	TemporalOutput json.RawMessage
	PipelineInput  json.RawMessage
	PipelineOutput json.RawMessage
	FixturesDir    string
	ExtractedDir   string
}

type Request struct {
	Fixture        string          `json:"fixture"`
	TemporalInput  json.RawMessage `json:"temporal_input"`
	TemporalOutput json.RawMessage `json:"temporal_output"`
	PipelineInput  json.RawMessage `json:"pipeline_input"`
	PipelineOutput json.RawMessage `json:"pipeline_output"`
}

type Report struct {
	Fixture     string `json:"fixture"`
	Slug        string `json:"slug"`
	Path        string `json:"path,omitempty"`
	PassedCount int    `json:"passed_count"`
	Markdown    string `json:"markdown,omitempty"`
}

type Result struct {
	Reports []Report `json:"reports"`
}

func LoadRequest(path string) (Request, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Request{}, err
	}
	var req Request
	if err := json.Unmarshal(b, &req); err != nil {
		return Request{}, err
	}
	return req, nil
}

func Generate(opts Options) (Result, error) {
	opts = withDefaults(opts)
	src, err := sot.Load(opts.SourceDir)
	if err != nil {
		return Result{}, err
	}
	if hasInlineInput(opts) {
		af, err := facts.BuildInline(
			opts.Fixture,
			opts.TemporalInput,
			opts.TemporalOutput,
			opts.PipelineInput,
			opts.PipelineOutput,
		)
		if err != nil {
			return Result{}, err
		}
		return renderReports(opts, src, []facts.AssessmentFacts{af})
	}
	fs, err := fixture.List(opts.FixturesDir, opts.ExtractedDir, opts.Fixture)
	if err != nil {
		return Result{}, err
	}
	afs := make([]facts.AssessmentFacts, 0, len(fs))
	for _, f := range fs {
		af, err := facts.Build(f)
		if err != nil {
			return Result{}, err
		}
		afs = append(afs, af)
	}
	return renderReports(opts, src, afs)
}

func renderReports(opts Options, src *sot.Source, afs []facts.AssessmentFacts) (Result, error) {
	if opts.OutDir != "" {
		if err := os.MkdirAll(opts.OutDir, 0755); err != nil {
			return Result{}, err
		}
	}
	res := Result{Reports: make([]Report, 0, len(afs))}
	for _, af := range afs {
		evaluated := rules.Evaluate(src.Taxonomy, af)
		md := report.Render(af, src.FlatTests, evaluated)
		rep := Report{Fixture: af.Fixture.Name, Slug: af.Fixture.Slug, PassedCount: len(evaluated)}
		if opts.OutDir == "" {
			rep.Markdown = md
		} else {
			name := fmt.Sprintf("conformance-assessment-%s.md", af.Fixture.Slug)
			rep.Path = filepath.Join(opts.OutDir, name)
			if err := os.WriteFile(rep.Path, []byte(md), 0644); err != nil {
				return Result{}, err
			}
		}
		res.Reports = append(res.Reports, rep)
	}
	sort.Slice(res.Reports, func(i, j int) bool { return res.Reports[i].Slug < res.Reports[j].Slug })
	return res, nil
}

func withDefaults(opts Options) Options {
	if opts.SourceDir == "" {
		opts.SourceDir = "./source-of-truth"
	}
	if opts.FixturesDir == "" {
		opts.FixturesDir = "./fixtures"
	}
	if opts.ExtractedDir == "" {
		opts.ExtractedDir = "./out"
	}
	return opts
}
func ApplyRequest(opts Options, req Request) Options {
	if req.Fixture != "" {
		opts.Fixture = req.Fixture
	}
	if len(req.TemporalInput) > 0 {
		opts.TemporalInput = req.TemporalInput
	}
	if len(req.TemporalOutput) > 0 {
		opts.TemporalOutput = req.TemporalOutput
	}
	if len(req.PipelineInput) > 0 {
		opts.PipelineInput = req.PipelineInput
	}
	if len(req.PipelineOutput) > 0 {
		opts.PipelineOutput = req.PipelineOutput
	}
	return opts
}
func hasInlineInput(opts Options) bool {
	return len(opts.TemporalInput) > 0 ||
		len(opts.TemporalOutput) > 0 ||
		len(opts.PipelineInput) > 0 ||
		len(opts.PipelineOutput) > 0
}
