package facts_test

import (
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

func evaluateFixture(t *testing.T, name string) map[int]rules.Result {
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
	return rules.Evaluate(src.Taxonomy, af)
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
