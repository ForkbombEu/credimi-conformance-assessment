package rules

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"credimi-conformance-assessment/internal/facts"
)

func Evaluate(t Taxonomy, af facts.AssessmentFacts) map[int]Result {
	flat := FlattenFacts(af)
	res := map[int]Result{}
	for _, r := range t.AssessmentRules {
		if evalCond(r.When, flat) {
			txt, ok := renderTemplate(r.ResultText, flat)
			if !ok {
				continue
			}
			if old, exists := res[r.TestID]; !exists || r.RuleID < old.RuleID {
				res[r.TestID] = Result{TestID: r.TestID, Text: txt, RuleID: r.RuleID, Strength: r.Strength}
			}
		}
	}
	return res
}
func FlattenFacts(af facts.AssessmentFacts) map[string]any {
	m := map[string]any{}
	m["fixture.name"] = af.Fixture.Name
	m["fixture.slug"] = af.Fixture.Slug
	m["workflow.temporal_input_present"] = af.Workflow.TemporalInputPresent
	m["workflow.temporal_output_present"] = af.Workflow.TemporalOutputPresent
	m["workflow.has_screenshots_or_videos"] = af.Workflow.HasScreenshotsOrVideos
	m["workflow.workflow_id"] = af.Workflow.WorkflowID
	m["workflow.run_id"] = af.Workflow.RunID
	m["evidence.step_artifacts_present"] = af.Evidence.StepArtifactsPresent
	m["evidence.artifacts_hashed"] = af.Evidence.ArtifactsHashed
	m["credential_offer.exists"] = len(af.CredentialOffers) > 0
	m["credential_offer.count"] = len(af.CredentialOffers)
	if len(af.CredentialOffers) > 0 {
		co := af.CredentialOffers[0]
		m["credential_offer.grant_type"] = co.GrantType
		m["credential.configuration_id"] = co.ConfigurationID
		m["credential.issuer_url"] = co.IssuerURL
	}
	m["presentation.exists"] = len(af.Presentations) > 0
	if len(af.Presentations) > 0 {
		p := af.Presentations[0]
		m["presentation.request_uri_fetched"] = p.RequestURIFetched
		m["presentation.jwt_signed"] = p.JWTSigned
		m["presentation.jwt_alg"] = p.JWTAlg
		m["presentation.has_x5c"] = p.HasX5C
		m["presentation.response_type"] = p.ResponseType
		m["presentation.response_mode"] = p.ResponseMode
		m["presentation.client_id"] = p.ClientID
	}
	m["wallet.issuance_flow_completed"] = af.Wallet.IssuanceFlowCompleted
	m["wallet.presentation_flow_completed"] = af.Wallet.PresentationFlowCompleted
	m["wallet.presentation_share_completed"] = af.Wallet.PresentationShareCompleted
	m["wallet.no_visible_error"] = af.Wallet.NoVisibleError
	m["issuer.metadata_fetched"] = af.Issuer.MetadataFetched
	m["issuer.metadata_format"] = af.Issuer.MetadataFormat
	m["issuer.metadata_advertises_pid"] = af.Issuer.MetadataAdvertisesPID
	m["issuer.metadata_advertises_sd_jwt"] = af.Issuer.MetadataAdvertisesSDJWT
	m["issuer.metadata_advertises_mdoc"] = af.Issuer.MetadataAdvertisesMdoc
	m["issuer.metadata_advertises_signing_algorithms"] = af.Issuer.MetadataAdvertisesSigningAlgorithms
	return m
}
func evalCond(c Condition, m map[string]any) bool {
	if len(c.All) > 0 {
		for _, x := range c.All {
			if !evalCond(x, m) {
				return false
			}
		}
		return true
	}
	if len(c.Any) > 0 {
		for _, x := range c.Any {
			if evalCond(x, m) {
				return true
			}
		}
		return false
	}
	if c.Not != nil {
		return !evalCond(*c.Not, m)
	}
	key := c.Fact
	if key == "" {
		key = c.Field
	}
	val, ok := m[key]
	if c.Exists != nil {
		return ok == *c.Exists
	}
	if !ok {
		return false
	}
	if c.Equals != nil && !eq(val, c.Equals) {
		return false
	}
	if c.NotEquals != nil && eq(val, c.NotEquals) {
		return false
	}
	if c.Contains != nil && !contains(val, c.Contains) {
		return false
	}
	if len(c.ContainsAny) > 0 {
		any := false
		for _, x := range c.ContainsAny {
			if contains(val, x) {
				any = true
				break
			}
		}
		if !any {
			return false
		}
	}
	if c.MatchesRegex != "" {
		re, err := regexp.Compile(c.MatchesRegex)
		if err != nil || !re.MatchString(fmt.Sprint(val)) {
			return false
		}
	}
	if c.LTE != nil {
		a, b := num(val), num(c.LTE)
		if a > b {
			return false
		}
	}
	if c.GTE != nil {
		a, b := num(val), num(c.GTE)
		if a < b {
			return false
		}
	}
	return true
}
func eq(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) || reflect.DeepEqual(a, b) }
func contains(a, b any) bool {
	switch x := a.(type) {
	case []string:
		for _, v := range x {
			if fmt.Sprint(v) == fmt.Sprint(b) {
				return true
			}
		}
		return false
	default:
		return strings.Contains(fmt.Sprint(a), fmt.Sprint(b))
	}
}
func num(v any) float64 { f, _ := strconv.ParseFloat(fmt.Sprint(v), 64); return f }
func renderTemplate(s string, m map[string]any) (string, bool) {
	re := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)
	ok := true
	out := re.ReplaceAllStringFunc(s, func(tok string) string {
		mm := re.FindStringSubmatch(tok)
		v, exists := m[mm[1]]
		if !exists {
			ok = false
			return tok
		}
		return fmt.Sprint(v)
	})
	return out, ok
}
func SortedResults(m map[int]Result) []Result {
	out := make([]Result, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TestID < out[j].TestID })
	return out
}
