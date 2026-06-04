package conformance

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateUsesPipelineOutputEvidence(t *testing.T) {
	pipelineOutput := json.RawMessage(`{
		"credential_well_knowns": [
			{
				"step_id": "cred-step",
				"credential_id": "tenant/credential",
				"well_known": {
					"credential_configurations_supported": {
						"pid": {
							"format": "vc+sd-jwt",
							"proof_types_supported": {
								"jwt": {
									"proof_signing_alg_values_supported": ["ES256"]
								}
							}
						}
					}
				}
			}
		],
		"presentation_results": [
			{
				"step_id": "vp-step",
				"use_case_id": "tenant/use-case",
				"result": {
					"format": "jwt",
					"header": {"alg": "ES256"},
					"payload": {"response_type": "vp_token"}
				}
			}
		]
	}`)
	res, err := Generate(
		ReportInput{
			Fixture:        "activity-output",
			TemporalInput:  json.RawMessage(`{"name":"activity-output"}`),
			TemporalOutput: json.RawMessage(`{"workflow_id":"wf","run_id":"run","output":"COMPLETED"}`),
			PipelineOutput: pipelineOutput,
		},
		ReportOptions{SourceDir: "../../source-of-truth"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Reports) != 1 {
		t.Fatalf("reports count got %d want 1", len(res.Reports))
	}
	rep := res.Reports[0]
	if rep.Slug != "activity-output" {
		t.Fatalf("slug got %q", rep.Slug)
	}
	if rep.Markdown == "" {
		t.Fatal("markdown should be returned when out dir is empty")
	}
	if !strings.Contains(rep.Markdown, "Credential-offer artifacts: `0`") {
		t.Fatalf("unexpected credential offer count in markdown")
	}
	if !strings.Contains(rep.Markdown, "Presentation-request artifacts: `1`") {
		t.Fatalf("presentation result was not reflected in markdown")
	}
	if !strings.Contains(rep.Markdown, "Issuer metadata fetched: `true`") {
		t.Fatalf("well-known evidence was not reflected in markdown")
	}
}

func TestGenerateActivityWrapsReportResult(t *testing.T) {
	res, err := GenerateActivity(ActivityInput{
		Payload: ActivityPayload{
			ReportInput: ReportInput{
				Fixture:        "activity-input",
				TemporalInput:  json.RawMessage(`{"name":"activity-input"}`),
				TemporalOutput: json.RawMessage(`{"workflow_id":"wf","run_id":"run"}`),
			},
			ReportOptions: ReportOptions{SourceDir: "../../source-of-truth"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Output.Reports) != 1 {
		t.Fatalf("reports count got %d want 1", len(res.Output.Reports))
	}
	if res.Output.Reports[0].Slug != "activity-input" {
		t.Fatalf("slug got %q", res.Output.Reports[0].Slug)
	}
}
