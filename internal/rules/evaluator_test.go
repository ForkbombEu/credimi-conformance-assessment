package rules

import (
	"testing"

	"credimi-conformance-assessment/internal/facts"
)

func TestEvaluatePredicates(t *testing.T) {
	af := facts.AssessmentFacts{}
	af.Fixture.Slug = "demo"
	tax := Taxonomy{AssessmentRules: []Rule{
		{RuleID: "all", TestID: 1, ResultText: "ok", When: Condition{All: []Condition{{Fact: "fixture.slug", Equals: "demo"}, {Fact: "credential_offer.count", GTE: 0}}}},
		{RuleID: "not", TestID: 2, ResultText: "ok", When: Condition{Not: &Condition{Fact: "fixture.slug", Equals: "other"}}},
		{RuleID: "regex", TestID: 3, ResultText: "fixture {{ fixture.slug }}", When: Condition{Fact: "fixture.slug", MatchesRegex: "^de"}},
		{RuleID: "missing-template", TestID: 4, ResultText: "{{ missing.value }}", When: Condition{Fact: "fixture.slug", Equals: "demo"}},
	}}
	got := Evaluate(tax, af)
	if len(got) != 3 || got[1].Text != "ok" || got[2].Text != "ok" || got[3].Text != "fixture demo" {
		t.Fatalf("unexpected results: %#v", got)
	}
	if _, ok := got[4]; ok {
		t.Fatalf("rule with missing template variable should not pass")
	}
}
