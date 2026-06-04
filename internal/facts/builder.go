package facts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"credimi-conformance-assessment/internal/fixture"
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
	evidenceInput json.RawMessage,
	evidenceOutput json.RawMessage,
) (AssessmentFacts, error) {
	if name == "" {
		name = "inline-assessment"
	}
	af := AssessmentFacts{}
	af.Fixture.Name, af.Fixture.Slug = name, fixture.Slug(name)
	if hasJSON(pipelineInput) {
		applyTemporalInput(&af, pipelineInput)
	}
	if hasJSON(pipelineOutput) {
		applyTemporalOutput(&af, pipelineOutput)
	}
	if hasJSON(evidenceInput) {
		if err := applyEvidenceInput(&af, evidenceInput); err != nil {
			return AssessmentFacts{}, err
		}
		markHashed(&af, evidenceInput)
	}
	if hasJSON(evidenceOutput) {
		if err := applyEvidenceOutput(&af, evidenceOutput); err != nil {
			return AssessmentFacts{}, err
		}
		markHashed(&af, evidenceOutput)
	}
	finalize(&af)
	return af, nil
}

func applyTemporalInput(af *AssessmentFacts, b []byte) {
	af.Workflow.TemporalInputPresent = true
	af.Workflow.Name = stringValue(parseJSON(b), "name", "workflow_name")
}
func applyTemporalOutput(af *AssessmentFacts, b []byte) {
	af.Workflow.TemporalOutputPresent = true
	extractOutputFacts(af, b)
}
func applyEvidenceInput(af *AssessmentFacts, b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	if hasJSON(m["discovered_steps"]) {
		af.Evidence.StepArtifactsPresent = true
	}
	if hasJSON(m["extraction_summary"]) {
		af.Evidence.ExtractionSummaryPresent = true
	}
	for _, raw := range rawArray(m, "credential_offers") {
		af.CredentialOffers = append(af.CredentialOffers, readOfferBytes(raw))
	}
	for _, key := range []string{"well_known", "issuer_metadata"} {
		for _, raw := range rawArray(m, key) {
			mergeIssuer(af, readWellKnownBytes(raw))
		}
	}
	for _, key := range []string{"presentation_requests", "request_uri_outputs"} {
		for _, raw := range rawArray(m, key) {
			af.Presentations = append(af.Presentations, readPresentationBytes(raw))
		}
	}
	return nil
}
func applyEvidenceOutput(af *AssessmentFacts, b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	for _, raw := range rawArray(m, "credential_well_knowns") {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if hasJSON(item["well_known"]) {
			mergeIssuer(af, readWellKnownBytes(item["well_known"]))
		}
		if hasJSON(item["fetch"]) {
			markHashed(af, item["fetch"])
		}
	}
	for _, raw := range rawArray(m, "presentation_results") {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if hasJSON(item["result"]) {
			af.Presentations = append(af.Presentations, readPresentationBytes(item["result"]))
		}
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
	if len(af.CredentialOffers) > 0 {
		af.Wallet.IssuanceFlowCompleted = af.Workflow.TemporalOutputPresent && af.Wallet.NoVisibleError
	}
	if len(af.Presentations) > 0 {
		af.Wallet.PresentationFlowCompleted = af.Workflow.TemporalOutputPresent && af.Wallet.NoVisibleError
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
	af.Wallet.NoVisibleError = !strings.Contains(s, "Oups! Something went wrong")
	af.Workflow.HasScreenshotsOrVideos = strings.Contains(s, "screenshot") || strings.Contains(s, "video")
	af.Workflow.WorkflowID = findJSONString(s, "workflow_id")
	af.Workflow.RunID = findJSONString(s, "run_id")
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
func readOfferBytes(b []byte) CredentialOfferFacts {
	m := asMap(parseJSON(b))
	co := CredentialOfferFacts{Exists: true}
	co.IssuerURL = stringValue(m, "credential_issuer")
	if ids, ok := m["credential_configuration_ids"].([]any); ok && len(ids) > 0 {
		co.ConfigurationID = toString(ids[0])
	}
	if grants := asMap(m["grants"]); len(grants) > 0 {
		for k := range grants {
			co.GrantType = k
			break
		}
	}
	return co
}
func readWellKnown(path string) IssuerFacts { b, _ := os.ReadFile(path); return readWellKnownBytes(b) }
func readWellKnownBytes(b []byte) IssuerFacts {
	s := string(b)
	is := IssuerFacts{MetadataFetched: true, MetadataFormat: "JSON"}
	if strings.Count(s, ".") >= 2 && !strings.HasPrefix(strings.TrimSpace(s), "{") {
		is.MetadataFormat = "JWT"
	}
	if strings.Contains(s, "dc+sd-jwt") || strings.Contains(s, "vc+sd-jwt") {
		is.MetadataAdvertisesSDJWT = true
	}
	if strings.Contains(s, "mso_mdoc") {
		is.MetadataAdvertisesMdoc = true
	}
	if strings.Contains(strings.ToLower(s), "pid") {
		is.MetadataAdvertisesPID = true
	}
	if strings.Contains(s, "did:jwk") {
		is.MetadataAdvertisesJWKBinding = true
	}
	if strings.Contains(s, "did:key") {
		is.MetadataAdvertisesDIDBinding = true
	}
	for _, alg := range []string{"ES256", "ES384", "ES512", "EdDSA", "RS256"} {
		if strings.Contains(s, alg) {
			is.MetadataAdvertisesSigningAlgorithms = append(is.MetadataAdvertisesSigningAlgorithms, alg)
		}
	}
	return is
}
func mergeIssuer(af *AssessmentFacts, is IssuerFacts) {
	af.Issuer.MetadataFetched = af.Issuer.MetadataFetched || is.MetadataFetched
	af.Issuer.MetadataFormat = is.MetadataFormat
	af.Issuer.MetadataAdvertisesPID = af.Issuer.MetadataAdvertisesPID || is.MetadataAdvertisesPID
	af.Issuer.MetadataAdvertisesSDJWT = af.Issuer.MetadataAdvertisesSDJWT || is.MetadataAdvertisesSDJWT
	af.Issuer.MetadataAdvertisesMdoc = af.Issuer.MetadataAdvertisesMdoc || is.MetadataAdvertisesMdoc
	af.Issuer.MetadataAdvertisesJWKBinding = af.Issuer.MetadataAdvertisesJWKBinding || is.MetadataAdvertisesJWKBinding
	af.Issuer.MetadataAdvertisesDIDBinding = af.Issuer.MetadataAdvertisesDIDBinding || is.MetadataAdvertisesDIDBinding
	af.Issuer.MetadataAdvertisesSigningAlgorithms = appendUnique(af.Issuer.MetadataAdvertisesSigningAlgorithms, is.MetadataAdvertisesSigningAlgorithms...)
}
func readPresentation(path string) PresentationFacts {
	b, _ := os.ReadFile(path)
	return readPresentationBytes(b)
}
func readPresentationBytes(b []byte) PresentationFacts {
	s := string(b)
	p := PresentationFacts{Exists: true, RequestURIFetched: true}
	p.JWTSigned = strings.Contains(s, "\"alg\"")
	p.HasX5C = strings.Contains(s, "\"x5c\"")
	p.JWTAlg = findJSONString(s, "alg")
	p.ResponseType = findJSONString(s, "response_type")
	p.ResponseMode = findJSONString(s, "response_mode")
	p.ClientID = findJSONString(s, "client_id")
	return p
}
func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
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
