package facts_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/forkbombeu/credimi-conformance-assessment/internal/facts"
	"github.com/forkbombeu/credimi-conformance-assessment/internal/fixture"
	"github.com/forkbombeu/credimi-conformance-assessment/internal/rules"
	"github.com/forkbombeu/credimi-conformance-assessment/internal/sot"
)

func TestBuildUsesSelectedCredentialConfigurationForWalletFormat(t *testing.T) {
	results := evaluateFixture(t, "EUDI-iss-ver")
	requirePassed(t, results, 6)
	requireNotPassed(t, results, 7)
	requirePassed(t, results, 75)
	requirePassed(t, results, 76)
	requirePassed(t, results, 107)
}

func TestBuildReadsWrappedIssuerMetadata(t *testing.T) {
	results := evaluateFixture(t, "EUDI-iss2")
	requirePassed(t, results, 6)
	requireNotPassed(t, results, 7)
	requirePassed(t, results, 14)
	requirePassed(t, results, 75)
	requirePassed(t, results, 76)
}

func TestBuildKeepsMdocAndSDJWTSelectedOffersExclusive(t *testing.T) {
	results := evaluateFixture(t, "Multipaz")
	requireNotPassed(t, results, 6)
	requirePassed(t, results, 7)
	requirePassed(t, results, 75)
	requirePassed(t, results, 76)
}

func TestBuildMapsWEBuildConformanceChecksWithoutGrantingDIDMetadata(t *testing.T) {
	results := evaluateFixture(t, "eudiw-checks-5x")
	requirePassed(t, results, 6)
	requireNotPassed(t, results, 7)
	requireNotPassed(t, results, 15)
	requirePassed(t, results, 75)
	requirePassed(t, results, 76)
	requirePassed(t, results, 162)
}

func TestBuildMarksCompressedEmptyConsumerOutputAsFailed(t *testing.T) {
	af, results := evaluateFixtureFacts(t, "Paradym-iss-BundesDruckerei")
	if len(af.IssuanceAttempts) != 1 {
		t.Fatalf("issuance attempts got %d want 1: %#v", len(af.IssuanceAttempts), af.IssuanceAttempts)
	}
	attempt := af.IssuanceAttempts[0]
	if attempt.ProducerStepID != "animo-bundesdruckerei-pid-issuer-0003" || attempt.ConsumerStepID != "get-bundesdruckerei-pid-0004" || attempt.ConsumerStatus != "failed" {
		t.Fatalf("unexpected issuance attempt: %#v", attempt)
	}
	requireFailed(t, results, 1)
	requireFailed(t, results, 5)
	requireFailed(t, results, 6)
	requireFailed(t, results, 11)
	requireFailed(t, results, 14)
	requireFailed(t, results, 80)
	requirePassed(t, results, 65)
	requirePassed(t, results, 67)
	requirePassed(t, results, 69)
	requirePassed(t, results, 74)
}

func TestBuildMarksMultipazSDJWTEmptyConsumerOutputAsFailed(t *testing.T) {
	af, results := evaluateFixtureFacts(t, "multipaz-sd-jwt-fail")
	if len(af.IssuanceAttempts) != 1 {
		t.Fatalf("issuance attempts got %d want 1: %#v", len(af.IssuanceAttempts), af.IssuanceAttempts)
	}
	attempt := af.IssuanceAttempts[0]
	if attempt.ConsumerStepID != "get-multipaz-pid-sd-jwt-urn-eudi-pid-1-lee-tom-0003" || attempt.ConsumerStatus != "failed" {
		t.Fatalf("unexpected issuance attempt: %#v", attempt)
	}
	for _, id := range []int{1, 4, 6, 11, 14, 80} {
		requireFailed(t, results, id)
	}
}

func TestBuildLetsSuccessfulPresentationOverrideFailedGenericPresentation(t *testing.T) {
	af, results := evaluateFixtureFacts(t, "EUDIW-fails-1-verification")
	if len(af.IssuanceAttempts) != 2 {
		t.Fatalf("issuance attempts got %d want 2: %#v", len(af.IssuanceAttempts), af.IssuanceAttempts)
	}
	if len(af.PresentationAttempts) != 2 {
		t.Fatalf("presentation attempts got %d want 2: %#v", len(af.PresentationAttempts), af.PresentationAttempts)
	}
	var failedPresentation bool
	for _, attempt := range af.PresentationAttempts {
		if attempt.ConsumerStepID == "verifycredential-pid-formeu-issuer-eudiw-dev-0009" && attempt.ConsumerStatus == "failed" {
			failedPresentation = true
		}
	}
	if !failedPresentation {
		t.Fatalf("failed mdoc presentation attempt not found: %#v", af.PresentationAttempts)
	}
	requirePassed(t, results, 1)
	requirePassed(t, results, 4)
	requirePassed(t, results, 6)
	requirePassed(t, results, 7)
	requirePassed(t, results, 28)
	requirePassed(t, results, 29)
	requirePassed(t, results, 30)
	requirePassed(t, results, 37)
}

func TestBuildInlineReadsDirectPipelineOutputStepMap(t *testing.T) {
	b, err := os.ReadFile("../../fixtures/multipaz-sd-jwt-fail/temporal-input-output.json")
	if err != nil {
		t.Fatal(err)
	}
	var captured []struct {
		Payload struct {
			WorkflowDefinition json.RawMessage `json:"workflow_definition"`
			PipelineOutput     json.RawMessage `json:"pipeline_output"`
			Evidence           json.RawMessage `json:"evidence"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(b, &captured); err != nil {
		t.Fatal(err)
	}
	if len(captured) != 1 {
		t.Fatalf("captured payload count got %d want 1", len(captured))
	}

	af, err := facts.BuildInline(
		"multipaz-sd-jwt-fail",
		captured[0].Payload.WorkflowDefinition,
		captured[0].Payload.PipelineOutput,
		captured[0].Payload.Evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(af.IssuanceAttempts) != 1 {
		t.Fatalf("issuance attempts got %d want 1: %#v", len(af.IssuanceAttempts), af.IssuanceAttempts)
	}
	attempt := af.IssuanceAttempts[0]
	if attempt.ConsumerStepID != "get-multipaz-pid-sd-jwt-urn-eudi-pid-1-lee-tom-0003" || attempt.ConsumerStatus != "failed" {
		t.Fatalf("unexpected issuance attempt: %#v", attempt)
	}

	src, err := sot.Load("../../source-of-truth")
	if err != nil {
		t.Fatal(err)
	}
	results := rules.Evaluate(src.Taxonomy, af)
	for _, id := range []int{1, 4, 6, 11, 14, 80} {
		requireFailed(t, results, id)
	}
}

func evaluateFixture(t *testing.T, name string) map[int]rules.Result {
	t.Helper()
	_, results := evaluateFixtureFacts(t, name)
	return results
}

func evaluateFixtureFacts(t *testing.T, name string) (facts.AssessmentFacts, map[int]rules.Result) {
	t.Helper()
	fixtures, err := fixture.List("../../fixtures", "../../out", name)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 1 {
		t.Fatalf("fixture count got %d want 1", len(fixtures))
	}
	af, err := facts.Build(fixtures[0])
	if err != nil {
		t.Fatal(err)
	}
	src, err := sot.Load("../../source-of-truth")
	if err != nil {
		t.Fatal(err)
	}
	return af, rules.Evaluate(src.Taxonomy, af)
}

func requirePassed(t *testing.T, results map[int]rules.Result, id int) {
	t.Helper()
	if _, ok := results[id]; !ok {
		t.Fatalf("test %d did not pass; results: %#v", id, results)
	}
}

func requireNotPassed(t *testing.T, results map[int]rules.Result, id int) {
	t.Helper()
	if result, ok := results[id]; ok {
		t.Fatalf("test %d unexpectedly passed with %q", id, result.Text)
	}
}

func requireFailed(t *testing.T, results map[int]rules.Result, id int) {
	t.Helper()
	result, ok := results[id]
	if !ok {
		t.Fatalf("test %d did not produce a result; results: %#v", id, results)
	}
	if result.Status != "failed" {
		t.Fatalf("test %d status got %q want failed; result: %#v", id, result.Status, result)
	}
}
