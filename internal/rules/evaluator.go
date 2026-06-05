package rules

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/forkbombeu/credimi-conformance-assessment/internal/facts"
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
			status := r.ResultStatus
			if status == "" {
				status = "passed"
			}
			if shouldReplace(res[r.TestID], res[r.TestID].TestID != 0, status, r.RuleID) {
				res[r.TestID] = Result{TestID: r.TestID, Text: txt, RuleID: r.RuleID, Strength: r.Strength, Status: status}
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
	m["workflow.has_completed_steps"] = af.Workflow.HasCompletedSteps
	m["workflow.has_failures"] = af.Workflow.HasFailures
	m["workflow.workflow_id"] = af.Workflow.WorkflowID
	m["workflow.run_id"] = af.Workflow.RunID
	m["evidence.step_artifacts_present"] = af.Evidence.StepArtifactsPresent
	m["evidence.artifacts_hashed"] = af.Evidence.ArtifactsHashed
	m["credential_offer.exists"] = len(af.CredentialOffers) > 0
	m["credential_offer.count"] = len(af.CredentialOffers)
	if len(af.CredentialOffers) > 0 {
		co := af.CredentialOffers[0]
		m["credential_offer.grant_type"] = co.GrantType
		m["credential_offer.is_pid"] = anyCredentialOffer(af.CredentialOffers, func(co facts.CredentialOfferFacts) bool { return co.IsPID })
		m["credential_offer.is_sd_jwt"] = anyCredentialOffer(af.CredentialOffers, func(co facts.CredentialOfferFacts) bool { return co.IsSDJWT })
		m["credential_offer.is_mdoc"] = anyCredentialOffer(af.CredentialOffers, func(co facts.CredentialOfferFacts) bool { return co.IsMdoc })
		m["credential_offer.has_authorization_code"] = anyCredentialOffer(af.CredentialOffers, func(co facts.CredentialOfferFacts) bool { return co.GrantType == "authorization_code" })
		m["credential_offer.has_pre_authorized_code"] = anyCredentialOffer(af.CredentialOffers, func(co facts.CredentialOfferFacts) bool {
			return co.GrantType == "urn:ietf:params:oauth:grant-type:pre-authorized_code"
		})
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
		m["presentation.dcql_claim_paths"] = p.ClaimPaths
		m["presentation.dcql_claim_path_count"] = len(p.ClaimPaths)
	}
	m["wallet.issuance_flow_completed"] = af.Wallet.IssuanceFlowCompleted
	m["wallet.presentation_flow_completed"] = af.Wallet.PresentationFlowCompleted
	m["wallet.presentation_share_completed"] = af.Wallet.PresentationShareCompleted
	m["wallet.no_visible_error"] = af.Wallet.NoVisibleError
	m["wallet.ran_on_physical_android"] = af.Wallet.RanOnPhysicalAndroid
	addAttemptFacts(m, "issuance", af.IssuanceAttempts)
	addAttemptFacts(m, "presentation", af.PresentationAttempts)
	m["issuer.metadata_fetched"] = af.Issuer.MetadataFetched
	m["issuer.metadata_format"] = af.Issuer.MetadataFormat
	m["issuer.metadata_advertises_pid"] = af.Issuer.MetadataAdvertisesPID
	m["issuer.metadata_advertises_sd_jwt"] = af.Issuer.MetadataAdvertisesSDJWT
	m["issuer.metadata_advertises_mdoc"] = af.Issuer.MetadataAdvertisesMdoc
	m["issuer.metadata_advertises_signing_algorithms"] = af.Issuer.MetadataAdvertisesSigningAlgorithms
	m["issuer.metadata_advertises_es256"] = contains(af.Issuer.MetadataAdvertisesSigningAlgorithms, "ES256")
	m["issuer.metadata_advertises_eddsa"] = contains(af.Issuer.MetadataAdvertisesSigningAlgorithms, "EdDSA")
	m["issuer.metadata_advertises_rs256"] = contains(af.Issuer.MetadataAdvertisesSigningAlgorithms, "RS256")
	m["issuer.metadata_has_x5c"] = af.Issuer.MetadataHasX5C
	m["issuer.offer_url_matches_metadata_issuer"] = af.Issuer.OfferIssuerMatchesMetadataIssuer
	m["issuer.offered_configuration_present"] = af.Issuer.OfferedConfigurationPresent
	m["issuer.metadata_advertises_jwk_binding"] = af.Issuer.MetadataAdvertisesJWKBinding
	m["issuer.metadata_advertises_did_binding"] = af.Issuer.MetadataAdvertisesDIDBinding
	m["conformance.webuild_wallet_checks_completed"] = af.Conformance.WEBuildWalletChecksCompleted
	m["conformance.webuild_wallet_check_count"] = af.Conformance.WEBuildWalletCheckCount
	return m
}
func anyCredentialOffer(offers []facts.CredentialOfferFacts, fn func(facts.CredentialOfferFacts) bool) bool {
	for _, offer := range offers {
		if fn(offer) {
			return true
		}
	}
	return false
}

func shouldReplace(old Result, exists bool, status, ruleID string) bool {
	if !exists {
		return true
	}
	if old.Status == "failed" && status == "passed" {
		return true
	}
	if old.Status == "passed" && status == "failed" {
		return false
	}
	return ruleID < old.RuleID
}

func addAttemptFacts(m map[string]any, prefix string, attempts []facts.AttemptFacts) {
	m[prefix+".attempt_count"] = len(attempts)
	for _, attempt := range attempts {
		status := attempt.ConsumerStatus
		if status == "" {
			continue
		}
		setBool(m, prefix+"."+status, true)
		if attempt.IsPID {
			setBool(m, prefix+"."+status+"_pid", true)
		}
		if attempt.IsSDJWT {
			setBool(m, prefix+"."+status+"_sd_jwt", true)
		}
		if attempt.IsMdoc {
			setBool(m, prefix+"."+status+"_mdoc", true)
		}
		if attempt.IsPID && attempt.IsSDJWT {
			setBool(m, prefix+"."+status+"_pid_sd_jwt", true)
		}
		if attempt.IsPID && attempt.IsMdoc {
			setBool(m, prefix+"."+status+"_pid_mdoc", true)
		}
		if attempt.GrantType == "authorization_code" {
			setBool(m, prefix+"."+status+"_authorization_code", true)
		}
		if attempt.GrantType == "urn:ietf:params:oauth:grant-type:pre-authorized_code" {
			setBool(m, prefix+"."+status+"_pre_authorized_code", true)
		}
		if attempt.IsOpenID4VP {
			setBool(m, prefix+"."+status+"_openid4vp", true)
		}
	}
}

func setBool(m map[string]any, key string, value bool) {
	if value {
		m[key] = true
	}
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
		present := ok && fmt.Sprint(val) != ""
		return present == *c.Exists
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

func PassedCount(m map[int]Result) int {
	count := 0
	for _, result := range m {
		if result.Status != "failed" {
			count++
		}
	}
	return count
}
