package conformance

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateUsesPipelineEvidence(t *testing.T) {
	evidence := json.RawMessage(`{
		"credential_offers": [
			{
				"step_id": "cred-step",
				"credential_id": "tenant/credential",
				"credential_offer": {
					"credential_issuer": "https://issuer.example",
					"credential_configuration_ids": ["pid"],
					"grants": {
						"urn:ietf:params:oauth:grant-type:pre-authorized_code": {}
					}
				}
			}
		],
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
			Fixture:        "pipeline-output",
			PipelineInput:  json.RawMessage(`{"name":"pipeline-output"}`),
			PipelineOutput: json.RawMessage(`{"workflow-id":"wf","workflow-run-id":"run","output":"COMPLETED"}`),
			Evidence:       evidence,
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
	if rep.Slug != "pipeline-output" {
		t.Fatalf("slug got %q", rep.Slug)
	}
	if rep.Markdown == "" {
		t.Fatal("markdown should be returned when out dir is empty")
	}
	if !strings.Contains(rep.Markdown, "Credential-offer artifacts: `1`") {
		t.Fatalf("unexpected credential offer count in markdown")
	}
	if !strings.Contains(rep.Markdown, "Presentation-request artifacts: `1`") {
		t.Fatalf("presentation result was not reflected in markdown")
	}
	if !strings.Contains(rep.Markdown, "Issuer metadata fetched: `true`") {
		t.Fatalf("well-known evidence was not reflected in markdown")
	}
}

func TestGenerateUsesEmbeddedSourceByDefault(t *testing.T) {
	t.Chdir(t.TempDir())

	res, err := Generate(
		ReportInput{
			Fixture:        "embedded-source",
			PipelineInput:  json.RawMessage(`{"name":"embedded-source"}`),
			PipelineOutput: json.RawMessage(`{"workflow_id":"wf","run_id":"run"}`),
		},
		ReportOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Reports) != 1 {
		t.Fatalf("reports count got %d want 1", len(res.Reports))
	}
	if res.Reports[0].Slug != "embedded-source" {
		t.Fatalf("slug got %q", res.Reports[0].Slug)
	}
	if res.Reports[0].Markdown == "" {
		t.Fatal("markdown should be generated from embedded source")
	}
}
