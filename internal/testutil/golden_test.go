package testutil

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi-conformance-assessment/pkg/conformance"
)

func TestGeneratedAssessmentUsesRealtimeEvidenceNotFixtureSlug(t *testing.T) {
	withoutEvidence, err := conformance.Generate(
		conformance.ReportInput{
			Fixture:        "EUDI-iss-ver",
			PipelineInput:  json.RawMessage(`{"name":"EUDI-iss-ver"}`),
			PipelineOutput: json.RawMessage(`{"workflow_id":"wf","run_id":"run"}`),
		},
		conformance.ReportOptions{SourceDir: "../../source-of-truth"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if withoutEvidence.Reports[0].PassedCount != 1 {
		t.Fatalf("fixture slug should not grant golden passes, got %d", withoutEvidence.Reports[0].PassedCount)
	}
	if strings.Contains(withoutEvidence.Reports[0].Markdown, "PID credential offer processed by Wallet") {
		t.Fatalf("report still contains golden fixture wording")
	}

	evidence := json.RawMessage(`{
		"credential_offers": [{
			"credential_offer": {
				"credential_issuer": "https://issuer.example",
				"credential_configuration_ids": ["pid_sd_jwt"],
				"grants": {"authorization_code": {}}
			}
		}],
		"credential_well_knowns": [{
			"well_known": {
				"credential_issuer": "https://issuer.example",
				"credential_configurations_supported": {
					"pid_sd_jwt": {
						"format": "dc+sd-jwt",
						"vct": "urn:eudi:pid:1",
						"cryptographic_binding_methods_supported": ["jwk"],
						"proof_types_supported": {"jwt": {"proof_signing_alg_values_supported": ["ES256"]}}
					}
				}
			}
		}],
		"presentation_results": [{
			"result": {
				"header": {"alg": "ES256", "x5c": ["cert"]},
				"payload": {"response_type": "vp_token", "response_mode": "direct_post.jwt", "client_id": "x509_hash:abc"}
			}
		}]
	}`)
	withEvidence, err := conformance.Generate(
		conformance.ReportInput{
			Fixture:        "EUDI-iss-ver",
			PipelineInput:  json.RawMessage(`{"name":"runtime","global_runner_id":"pixel6-android16"}`),
			PipelineOutput: json.RawMessage(`{"workflow_id":"wf","run_id":"run","output":"Assert that \"Oups! Something went wrong\" is not visible... COMPLETED result_video screenshot"}`),
			Evidence:       evidence,
		},
		conformance.ReportOptions{SourceDir: "../../source-of-truth"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if withEvidence.Reports[0].PassedCount <= withoutEvidence.Reports[0].PassedCount {
		t.Fatalf("real evidence did not increase passed count: without=%d with=%d", withoutEvidence.Reports[0].PassedCount, withEvidence.Reports[0].PassedCount)
	}
	for _, want := range []string{
		"PASSED - real-time credential offer parsed successfully",
		"PASSED - real-time .well-known issuer metadata was supplied",
		"PASSED - real-time presentation request included x5c certificate material",
	} {
		if !strings.Contains(withEvidence.Reports[0].Markdown, want) {
			t.Fatalf("report missing %q", want)
		}
	}
	if strings.Contains(withEvidence.Reports[0].Markdown, "HITM") {
		t.Fatalf("report should not contain HITM column")
	}
	for _, want := range []string{
		"| CR-W-001 | Wallet | Wallet can receive a PID credential offer |",
		"| CR-I-065 | Issuer | Issuer exposes OpenID Credential Issuer metadata |",
	} {
		if !strings.Contains(withEvidence.Reports[0].Markdown, want) {
			t.Fatalf("report should render categorized CR display ID row %q", want)
		}
	}
}

func TestGeneratedFailureAssessmentUsesCRIDsAndNoHITM(t *testing.T) {
	res, err := conformance.Generate(
		conformance.ReportInput{Fixture: "Paradym-iss-BundesDruckerei"},
		conformance.ReportOptions{SourceDir: "../../source-of-truth", FixturesDir: "../../fixtures", ExtractedDir: "../../out"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Reports) != 1 {
		t.Fatalf("report count got %d want 1", len(res.Reports))
	}
	md := res.Reports[0].Markdown
	if strings.Contains(md, "HITM") {
		t.Fatalf("report should not contain HITM column")
	}
	for _, want := range []string{
		"| CR-W-001 | Wallet | Wallet can receive a PID credential offer | FAILED -",
		"| CR-I-080 | Issuer | Issuer can issue PID to a real Wallet | FAILED -",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("failure report should render categorized CR display ID row %q", want)
		}
	}
}
