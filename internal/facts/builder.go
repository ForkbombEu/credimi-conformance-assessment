package facts

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi-conformance-assessment/internal/fixture"
)

func Build(f fixture.Fixture) (AssessmentFacts, error) {
	af := AssessmentFacts{}
	af.Fixture.Name, af.Fixture.Slug = f.Name, f.Slug
	if b, err := os.ReadFile(filepath.Join(f.Dir, "input.json")); err == nil {
		applyTemporalInput(&af, b)
	}
	if b, err := os.ReadFile(filepath.Join(f.Dir, "output.json")); err == nil {
		applyTemporalOutput(&af, b)
	}
	if _, err := os.Stat(filepath.Join(f.ExtractedDir, "discovered-steps.json")); err == nil {
		af.Evidence.StepArtifactsPresent = true
	}
	if _, err := os.Stat(filepath.Join(f.ExtractedDir, "extraction-summary.json")); err == nil {
		af.Evidence.ExtractionSummaryPresent = true
	}
	_ = filepath.WalkDir(f.ExtractedDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := d.Name()
		switch base {
		case "credential-offer.json":
			af.CredentialOffers = append(af.CredentialOffers, readOffer(path))
		case "well-known.json", ".well-known.json":
			mergeIssuer(&af, readWellKnown(path))
		case "request-uri-output.json":
			af.Presentations = append(af.Presentations, readPresentation(path))
		}
		if strings.HasSuffix(base, ".json") {
			if b, e := os.ReadFile(path); e == nil {
				markHashed(&af, b)
			}
		}
		return nil
	})
	finalize(&af)
	return af, nil
}

func BuildInline(
	name string,
	pipelineInput json.RawMessage,
	pipelineOutput json.RawMessage,
	evidence json.RawMessage,
) (AssessmentFacts, error) {
	if name == "" {
		name = "inline-assessment"
	}
	af := AssessmentFacts{}
	af.Fixture.Name, af.Fixture.Slug = name, fixture.Slug(name)
	if hasJSON(pipelineInput) {
		applyTemporalInput(&af, pipelineInput)
		_ = applyEvidence(&af, pipelineInput)
	}
	if hasJSON(pipelineOutput) {
		applyTemporalOutput(&af, pipelineOutput)
		_ = applyEvidence(&af, pipelineOutput)
	}
	if hasJSON(evidence) {
		if err := applyEvidence(&af, evidence); err != nil {
			return AssessmentFacts{}, err
		}
		markHashed(&af, evidence)
	}
	finalize(&af)
	return af, nil
}

func applyTemporalInput(af *AssessmentFacts, b []byte) {
	af.Workflow.TemporalInputPresent = true
	s := string(b)
	af.Workflow.Name = firstJSONString(s, "name", "workflow_name")
	if af.Workflow.Name == "" {
		af.Workflow.Name = firstJSONString(s, "workflowDefinitionName")
	}
	if strings.Contains(strings.ToLower(s), "android") || strings.Contains(strings.ToLower(s), "pixel") {
		af.Wallet.RanOnPhysicalAndroid = true
	}
}
func applyTemporalOutput(af *AssessmentFacts, b []byte) {
	af.Workflow.TemporalOutputPresent = true
	extractOutputFacts(af, b)
}
func applyEvidence(af *AssessmentFacts, b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	used := false
	for _, raw := range rawArray(m, "credential_offers") {
		af.CredentialOffers = append(af.CredentialOffers, readCredentialOfferEvidenceBytes(raw))
		used = true
	}
	for _, raw := range rawArray(m, "credential_offer_resolution_chains") {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if hasJSON(item["CredentialOffer"]) {
			af.CredentialOffers = append(af.CredentialOffers, readOfferBytes(item["CredentialOffer"]))
			used = true
		}
		if hasJSON(item["credential_offer"]) {
			af.CredentialOffers = append(af.CredentialOffers, readOfferBytes(item["credential_offer"]))
			used = true
		}
		if hasJSON(item["IssuerMetadata"]) {
			mergeIssuer(af, readWellKnownBytes(item["IssuerMetadata"]))
			used = true
		}
		if hasJSON(item["issuer_metadata"]) {
			mergeIssuer(af, readWellKnownBytes(item["issuer_metadata"]))
			used = true
		}
	}
	for _, key := range []string{"credential_well_knowns", "well_knowns", "well_known"} {
		for _, raw := range rawArray(m, key) {
			var item map[string]json.RawMessage
			if err := json.Unmarshal(raw, &item); err == nil {
				switch {
				case hasJSON(item["well_known"]):
					mergeIssuer(af, readWellKnownBytes(item["well_known"]))
				case hasJSON(item["IssuerMetadata"]):
					mergeIssuer(af, readWellKnownBytes(item["IssuerMetadata"]))
				case hasJSON(item["issuer_metadata"]):
					mergeIssuer(af, readWellKnownBytes(item["issuer_metadata"]))
				default:
					mergeIssuer(af, readWellKnownBytes(raw))
				}
				if hasJSON(item["fetch"]) {
					markHashed(af, item["fetch"])
				}
			} else {
				mergeIssuer(af, readWellKnownBytes(raw))
			}
			used = true
		}
	}
	for _, raw := range rawArray(m, "presentation_results") {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(raw, &item); err == nil {
			switch {
			case hasJSON(item["result"]):
				af.Presentations = append(af.Presentations, readPresentationBytes(item["result"]))
			case hasJSON(item["request_uri_output"]):
				af.Presentations = append(af.Presentations, readPresentationBytes(item["request_uri_output"]))
			default:
				af.Presentations = append(af.Presentations, readPresentationBytes(raw))
			}
		} else {
			af.Presentations = append(af.Presentations, readPresentationBytes(raw))
		}
		used = true
	}
	if used {
		af.Evidence.StepArtifactsPresent = true
		af.Evidence.ExtractionSummaryPresent = true
		markHashed(af, b)
	}
	return nil
}

func rawArray(m map[string]json.RawMessage, key string) []json.RawMessage {
	raw := m[key]
	if !hasJSON(raw) {
		return nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}
	return []json.RawMessage{raw}
}
func hasJSON(b []byte) bool {
	s := strings.TrimSpace(string(b))
	return s != "" && s != "null"
}
func finalize(af *AssessmentFacts) {
	for i := range af.CredentialOffers {
		co := &af.CredentialOffers[i]
		if co.ConfigurationID != "" && stringIn(co.ConfigurationID, af.Issuer.ConfigurationIDs) {
			af.Issuer.OfferedConfigurationPresent = true
		}
		co.IsPID = co.IsPID || af.Issuer.MetadataAdvertisesPID
		co.IsSDJWT = co.IsSDJWT || af.Issuer.MetadataAdvertisesSDJWT
		co.IsMdoc = co.IsMdoc || af.Issuer.MetadataAdvertisesMdoc
	}
	if len(af.CredentialOffers) > 0 {
		af.Wallet.IssuanceFlowCompleted = af.Workflow.TemporalOutputPresent && af.Workflow.HasCompletedSteps && af.Wallet.NoVisibleError
	}
	if len(af.Presentations) > 0 {
		af.Wallet.PresentationFlowCompleted = af.Workflow.TemporalOutputPresent && af.Workflow.HasCompletedSteps && af.Wallet.NoVisibleError
		af.Wallet.PresentationShareCompleted = af.Wallet.PresentationFlowCompleted
	}
}
func markHashed(af *AssessmentFacts, b []byte) {
	h := sha256.Sum256(b)
	_ = hex.EncodeToString(h[:])
	af.Evidence.ArtifactsHashed = true
}
func parseJSON(b []byte) any     { var v any; _ = json.Unmarshal(b, &v); return v }
func asMap(v any) map[string]any { m, _ := v.(map[string]any); return m }
func stringValue(v any, keys ...string) string {
	m := asMap(v)
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			return s
		}
	}
	return ""
}
func extractOutputFacts(af *AssessmentFacts, b []byte) {
	s := string(b)
	ls := strings.ToLower(s)
	af.Workflow.HasCompletedSteps = strings.Contains(s, "COMPLETED") || strings.Contains(ls, "\"status\":\"ok\"") || strings.Contains(ls, "\"status\": \"ok\"")
	af.Workflow.HasFailures = strings.Contains(s, "FAILED") || strings.Contains(ls, "\"status\":\"error\"") || strings.Contains(ls, "\"status\": \"error\"") || strings.Contains(ls, "exception") || strings.Contains(ls, "traceback")
	af.Wallet.NoVisibleError = af.Workflow.HasCompletedSteps && !af.Workflow.HasFailures
	af.Workflow.HasScreenshotsOrVideos = strings.Contains(ls, "screenshot") || strings.Contains(ls, "video")
	af.Workflow.WorkflowID = firstJSONString(s, "workflow_id", "workflow-id", "workflowId")
	af.Workflow.RunID = firstJSONString(s, "run_id", "workflow-run-id", "workflowRunId")
	if strings.Contains(strings.ToLower(s), "android") || strings.Contains(strings.ToLower(s), "pixel") {
		af.Wallet.RanOnPhysicalAndroid = true
	}
}
func firstJSONString(s string, keys ...string) string {
	for _, key := range keys {
		if value := findJSONString(s, key); value != "" {
			return value
		}
	}
	return ""
}
func findJSONString(s, k string) string {
	idx := strings.Index(s, "\""+k+"\"")
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(k)+2:]
	c := strings.Index(rest, ":")
	if c < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[c+1:])
	if !strings.HasPrefix(rest, "\"") {
		return ""
	}
	rest = rest[1:]
	e := strings.Index(rest, "\"")
	if e < 0 {
		return ""
	}
	return rest[:e]
}
func readOffer(path string) CredentialOfferFacts { b, _ := os.ReadFile(path); return readOfferBytes(b) }
func readCredentialOfferEvidenceBytes(b []byte) CredentialOfferFacts {
	var item map[string]json.RawMessage
	if err := json.Unmarshal(b, &item); err == nil && hasJSON(item["credential_offer"]) {
		return readOfferBytes(item["credential_offer"])
	}
	return readOfferBytes(b)
}
func readOfferBytes(b []byte) CredentialOfferFacts {
	m := asMap(parseJSON(b))
	co := CredentialOfferFacts{Exists: true}
	co.IssuerURL = stringValue(m, "credential_issuer", "credentialIssuer")
	if ids, ok := m["credential_configuration_ids"].([]any); ok && len(ids) > 0 {
		co.ConfigurationID = toString(ids[0])
	}
	co.Format = stringValue(m, "format")
	co.VCT = stringValue(m, "vct")
	co.Doctype = stringValue(m, "doctype")
	if grants := asMap(m["grants"]); len(grants) > 0 {
		for k := range grants {
			co.GrantType = k
			break
		}
	}
	s := string(b)
	ls := strings.ToLower(s + " " + co.ConfigurationID + " " + co.Format + " " + co.VCT + " " + co.Doctype)
	co.IsPID = strings.Contains(ls, "pid") || strings.Contains(ls, "person identification")
	co.IsSDJWT = strings.Contains(ls, "sd-jwt") || strings.Contains(ls, "dc+sd-jwt") || strings.Contains(ls, "vc+sd-jwt")
	co.IsMdoc = strings.Contains(ls, "mso_mdoc") || strings.Contains(ls, "mdoc")
	return co
}
func readWellKnown(path string) IssuerFacts { b, _ := os.ReadFile(path); return readWellKnownBytes(b) }
func readWellKnownBytes(b []byte) IssuerFacts {
	s := string(b)
	analysis := s + " " + decodedCompactJWT(s)
	is := IssuerFacts{MetadataFetched: true, MetadataFormat: "JSON"}
	trimmed := strings.TrimSpace(s)
	if strings.Count(trimmed, ".") >= 2 && !strings.HasPrefix(trimmed, "{") {
		is.MetadataFormat = "JWT"
	}
	ls := strings.ToLower(analysis)
	is.IssuerURL = firstJSONString(analysis, "credential_issuer", "issuer", "iss")
	is.MetadataAdvertisesSDJWT = strings.Contains(ls, "dc+sd-jwt") || strings.Contains(ls, "vc+sd-jwt") || strings.Contains(ls, "sd-jwt")
	is.MetadataAdvertisesMdoc = strings.Contains(ls, "mso_mdoc") || strings.Contains(ls, "mdoc")
	is.MetadataAdvertisesPID = strings.Contains(ls, "pid") || strings.Contains(ls, "person identification")
	is.MetadataAdvertisesJWKBinding = strings.Contains(ls, "did:jwk") || strings.Contains(ls, "\"jwk\"")
	is.MetadataAdvertisesDIDBinding = strings.Contains(ls, "did:key") || strings.Contains(ls, "did:web")
	is.MetadataHasX5C = strings.Contains(ls, "\"x5c\"")
	for _, alg := range []string{"ES256", "ES384", "ES512", "EdDSA", "RS256", "ES256K"} {
		if strings.Contains(analysis, alg) {
			is.MetadataAdvertisesSigningAlgorithms = append(is.MetadataAdvertisesSigningAlgorithms, alg)
		}
	}
	is.ConfigurationIDs = appendUnique(nil, mapKeys(asMap(asMap(parseJSON(b))["credential_configurations_supported"]))...)
	return is
}
func mergeIssuer(af *AssessmentFacts, is IssuerFacts) {
	af.Issuer.MetadataFetched = af.Issuer.MetadataFetched || is.MetadataFetched
	if is.MetadataFormat != "" {
		af.Issuer.MetadataFormat = is.MetadataFormat
	}
	if is.IssuerURL != "" {
		af.Issuer.IssuerURL = is.IssuerURL
	}
	af.Issuer.MetadataAdvertisesPID = af.Issuer.MetadataAdvertisesPID || is.MetadataAdvertisesPID
	af.Issuer.MetadataAdvertisesSDJWT = af.Issuer.MetadataAdvertisesSDJWT || is.MetadataAdvertisesSDJWT
	af.Issuer.MetadataAdvertisesMdoc = af.Issuer.MetadataAdvertisesMdoc || is.MetadataAdvertisesMdoc
	af.Issuer.MetadataAdvertisesJWKBinding = af.Issuer.MetadataAdvertisesJWKBinding || is.MetadataAdvertisesJWKBinding
	af.Issuer.MetadataAdvertisesDIDBinding = af.Issuer.MetadataAdvertisesDIDBinding || is.MetadataAdvertisesDIDBinding
	af.Issuer.MetadataHasX5C = af.Issuer.MetadataHasX5C || is.MetadataHasX5C
	af.Issuer.ConfigurationIDs = appendUnique(af.Issuer.ConfigurationIDs, is.ConfigurationIDs...)
	af.Issuer.MetadataAdvertisesSigningAlgorithms = appendUnique(af.Issuer.MetadataAdvertisesSigningAlgorithms, is.MetadataAdvertisesSigningAlgorithms...)
}
func readPresentation(path string) PresentationFacts {
	b, _ := os.ReadFile(path)
	return readPresentationBytes(b)
}
func readPresentationBytes(b []byte) PresentationFacts {
	s := string(b)
	analysis := s + " " + decodedCompactJWT(s)
	p := PresentationFacts{Exists: true, RequestURIFetched: true}
	ls := strings.ToLower(analysis)
	p.JWTSigned = strings.Contains(ls, "\"alg\"") || strings.Contains(ls, "jwt")
	p.HasX5C = strings.Contains(ls, "\"x5c\"")
	p.JWTAlg = firstJSONString(analysis, "alg")
	p.ResponseType = firstJSONString(analysis, "response_type")
	p.ResponseMode = firstJSONString(analysis, "response_mode")
	p.ClientID = firstJSONString(analysis, "client_id")
	p.ClientIDScheme = firstJSONString(analysis, "client_id_scheme")
	return p
}
func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%f", x), "0"), ".")
	default:
		return ""
	}
}
func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
func stringIn(s string, values []string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
func decodedCompactJWT(s string) string {
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) < 2 {
		return ""
	}
	var out []string
	for _, part := range parts[:2] {
		b, err := base64.RawURLEncoding.DecodeString(part)
		if err == nil {
			out = append(out, string(b))
		}
	}
	return strings.Join(out, " ")
}

func appendUnique(a []string, vals ...string) []string {
	seen := map[string]bool{}
	for _, x := range a {
		seen[x] = true
	}
	for _, v := range vals {
		if !seen[v] {
			a = append(a, v)
			seen[v] = true
		}
	}
	return a
}
