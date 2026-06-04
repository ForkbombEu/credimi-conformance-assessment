package report

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi-conformance-assessment/internal/facts"
	"github.com/forkbombeu/credimi-conformance-assessment/internal/rules"
	"github.com/forkbombeu/credimi-conformance-assessment/internal/sot"
)

func Render(af facts.AssessmentFacts, tests []sot.FlatTest, results map[int]rules.Result) string {
	byID := map[int]sot.FlatTest{}
	for _, t := range tests {
		byID[t.Number] = t
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "# Credimi Conformance Assessment — %s\n\n", af.Fixture.Name)
	b.WriteString("## Passed tests digest\n\n")
	b.WriteString("| # | Actor | Test | Test result |\n|---:|---|---|---|\n")
	ids := make([]int, 0, len(results))
	for id := range results {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		t := byID[id]
		fmt.Fprintf(&b, "| %d | %s | %s | %s |\n", id, esc(t.Actor), esc(t.Test), esc(results[id].Text))
	}
	fmt.Fprintf(&b, "\n**Passed tests count:** %d\n\n", len(ids))
	b.WriteString("## Assessment summary\n\n")
	b.WriteString("This deterministic assessment was generated from the fixture Temporal input/output, extracted artifacts, the Credimi source-of-truth flat conformance list, and declarative taxonomy assessment rules. It is not an official certification result.\n\n")
	fmt.Fprintf(&b, "- Fixture: `%s`\n- Temporal input present: `%t`\n- Temporal output present: `%t`\n- Credential-offer artifacts: `%d`\n- Presentation-request artifacts: `%d`\n- Issuer metadata fetched: `%t`\n\n", af.Fixture.Name, af.Workflow.TemporalInputPresent, af.Workflow.TemporalOutputPresent, len(af.CredentialOffers), len(af.Presentations), af.Issuer.MetadataFetched)
	b.WriteString("## Workflow steps\n\n")
	if af.Workflow.Name != "" {
		fmt.Fprintf(&b, "- Workflow name: `%s`\n", esc(af.Workflow.Name))
	}
	if af.Workflow.WorkflowID != "" {
		fmt.Fprintf(&b, "- Workflow ID: `%s`\n", esc(af.Workflow.WorkflowID))
	}
	if af.Workflow.RunID != "" {
		fmt.Fprintf(&b, "- Workflow run ID: `%s`\n", esc(af.Workflow.RunID))
	}
	b.WriteString("- Wallet visible-error check: conservative black-box interpretation only; no internal cryptographic validation is inferred without explicit verifier/callback evidence.\n\n")
	b.WriteString("## Fixture evidence used\n\n")
	b.WriteString("| Evidence | Present |\n|---|---:|\n")
	fmt.Fprintf(&b, "| Temporal input.json | %t |\n| Temporal output.json | %t |\n| Discovered step artifacts | %t |\n| Extraction summary | %t |\n| Hashed JSON artifacts | %t |\n\n", af.Workflow.TemporalInputPresent, af.Workflow.TemporalOutputPresent, af.Evidence.StepArtifactsPresent, af.Evidence.ExtractionSummaryPresent, af.Evidence.ArtifactsHashed)
	b.WriteString("## Assessment table\n\n")
	b.WriteString("Blank **Test result** cells mean the fixture did not execute or did not sufficiently prove that test. **HITM** is intentionally left empty for human review notes.\n\n")
	b.WriteString("| # | Actor | Test | Test result | HITM | Evidence strength | Recommended execution | Standards / source references | Notes |\n|---:|---|---|---|---|---|---|---|---|\n")
	for _, t := range tests {
		rr := results[t.Number].Text
		fmt.Fprintf(&b, "| %d | %s | %s | %s |  | %s | %s | %s | %s |\n", t.Number, esc(t.Actor), esc(t.Test), esc(rr), esc(t.EvidenceStrength), esc(t.RecommendedExecution), esc(t.SourceReferences), esc(t.Notes))
	}
	return b.String()
}
func esc(s string) string { return strings.ReplaceAll(s, "\n", " ") }
