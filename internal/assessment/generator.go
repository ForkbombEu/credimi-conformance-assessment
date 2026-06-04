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
	sourceoftruth "credimi-conformance-assessment/source-of-truth"
)

type Options struct {
	SourceDir      string
	TemporalData   string
	OutDir         string
	Fixture        string
	PipelineInput  json.RawMessage
	PipelineOutput json.RawMessage
	EvidenceInput  json.RawMessage
	EvidenceOutput json.RawMessage
	FixturesDir    string
	ExtractedDir   string
}

type Request struct {
	Fixture        string          `json:"fixture"`
	PipelineInput  json.RawMessage `json:"pipeline_input"`
	PipelineOutput json.RawMessage `json:"pipeline_output"`
	EvidenceInput  json.RawMessage `json:"evidence_input"`
	EvidenceOutput json.RawMessage `json:"evidence_output"`
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
	src, err := loadSource(opts.SourceDir)
	if err != nil {
		return Result{}, err
	}
	if hasInlineInput(opts) {
		af, err := facts.BuildInline(
			opts.Fixture,
			opts.PipelineInput,
			opts.PipelineOutput,
			opts.EvidenceInput,
			opts.EvidenceOutput,
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

func loadSource(sourceDir string) (*sot.Source, error) {
	if sourceDir != "" {
		return sot.Load(sourceDir)
	}
	return sot.LoadFS(sourceoftruth.FS)
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
	if len(req.PipelineInput) > 0 {
		opts.PipelineInput = req.PipelineInput
	}
	if len(req.PipelineOutput) > 0 {
		opts.PipelineOutput = req.PipelineOutput
	}
	if len(req.EvidenceInput) > 0 {
		opts.EvidenceInput = req.EvidenceInput
	}
	if len(req.EvidenceOutput) > 0 {
		opts.EvidenceOutput = req.EvidenceOutput
	}
	return opts
}
func hasInlineInput(opts Options) bool {
	return len(opts.PipelineInput) > 0 ||
		len(opts.PipelineOutput) > 0 ||
		len(opts.EvidenceInput) > 0 ||
		len(opts.EvidenceOutput) > 0
}
