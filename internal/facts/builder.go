package facts

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
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
		_ = applyEvidence(&af, b)
	}
	if b, err := os.ReadFile(filepath.Join(f.Dir, "output.json")); err == nil {
		applyTemporalOutput(&af, b)
		_ = applyEvidence(&af, b)
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
		case "credential_offers.json", "credential-offers.json", "credential_well_knowns.json", "credential-well-knowns.json", "presentation_results.json", "presentation-results.json":
			if b, e := os.ReadFile(path); e == nil {
				_ = applyEvidence(&af, b)
			}
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
	if strings.Contains(strings.ToLower(s), "pixel") {
		af.Wallet.RanOnPhysicalAndroid = true
	}
	extractConformanceInputFacts(af, s)
	applyWorkflowDefinition(af, b)
}
func applyTemporalOutput(af *AssessmentFacts, b []byte) {
	af.Workflow.TemporalOutputPresent = true
	extractOutputFacts(af, b)
	applyPipelineOutput(af, b)
}
func applyEvidence(af *AssessmentFacts, b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		var items []json.RawMessage
		if err := json.Unmarshal(b, &items); err != nil {
			return err
		}
		for _, item := range items {
			if err := applyEvidence(af, item); err != nil {
				return err
			}
		}
		return nil
	}
	if hasJSON(m["payload"]) {
		if err := applyEvidence(af, m["payload"]); err != nil {
			return err
		}
	}
	if hasJSON(m["evidence"]) {
		if err := applyEvidence(af, m["evidence"]); err != nil {
			return err
		}
	}
	used := false
	for _, raw := range rawArray(m, "credential_offers") {
		af.CredentialOffers = appendCredentialOffer(af.CredentialOffers, readCredentialOfferEvidenceBytes(raw))
		used = true
	}
	for _, raw := range rawArray(m, "credential_offer_resolution_chains") {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if hasJSON(item["CredentialOffer"]) {
			af.CredentialOffers = appendCredentialOffer(af.CredentialOffers, readOfferBytes(item["CredentialOffer"]))
			used = true
		}
		if hasJSON(item["credential_offer"]) {
			af.CredentialOffers = appendCredentialOffer(af.CredentialOffers, readOfferBytes(item["credential_offer"]))
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
		if co.IssuerURL != "" && af.Issuer.IssuerURL != "" && co.IssuerURL == af.Issuer.IssuerURL {
			af.Issuer.OfferIssuerMatchesMetadataIssuer = true
		}
		if cfg, ok := selectedCredentialConfiguration(*co, af.Issuer.Configurations); ok {
			af.Issuer.OfferedConfigurationPresent = true
			co.Format = firstNonEmpty(co.Format, cfg.Format)
			co.VCT = firstNonEmpty(co.VCT, cfg.VCT)
			co.Doctype = firstNonEmpty(co.Doctype, cfg.Doctype)
			co.IsPID = co.IsPID || cfg.IsPID
			co.IsSDJWT = co.IsSDJWT || cfg.IsSDJWT
			co.IsMdoc = co.IsMdoc || cfg.IsMdoc
		} else if co.ConfigurationID != "" && stringIn(co.ConfigurationID, af.Issuer.ConfigurationIDs) {
			af.Issuer.OfferedConfigurationPresent = true
		}
	}
	buildAttempts(af)
	if len(af.IssuanceAttempts) > 0 {
		af.Wallet.IssuanceFlowCompleted = anyAttemptStatus(af.IssuanceAttempts, "succeeded")
	} else if len(af.CredentialOffers) > 0 {
		af.Wallet.IssuanceFlowCompleted = af.Workflow.TemporalOutputPresent && af.Workflow.HasCompletedSteps && af.Wallet.NoVisibleError
	}
	if len(af.PresentationAttempts) > 0 {
		af.Wallet.PresentationFlowCompleted = anyAttemptStatus(af.PresentationAttempts, "succeeded")
		af.Wallet.PresentationShareCompleted = af.Wallet.PresentationFlowCompleted
	} else if len(af.Presentations) > 0 {
		af.Wallet.PresentationFlowCompleted = af.Workflow.TemporalOutputPresent && af.Workflow.HasCompletedSteps && af.Wallet.NoVisibleError
		af.Wallet.PresentationShareCompleted = af.Wallet.PresentationFlowCompleted
	}
}
func applyWorkflowDefinition(af *AssessmentFacts, b []byte) {
	root := asMap(parseJSON(b))
	if payload := asMap(root["payload"]); len(payload) > 0 {
		root = payload
	}
	wf := asMap(root["workflow_definition"])
	if len(wf) == 0 {
		wf = root
	}
	if name := stringValue(wf, "name"); name != "" {
		af.Workflow.Name = name
	}
	for _, raw := range anySlice(wf["steps"]) {
		m := asMap(raw)
		step := StepFacts{ID: stringValue(m, "id"), Use: stringValue(m, "use")}
		with := asMap(m["with"])
		payload := asMap(with["payload"])
		step.ActionID = firstNonEmpty(stringValue(with, "action_id"), stringValue(payload, "action_id"))
		step.CredentialID = firstNonEmpty(stringValue(with, "credential_id"), stringValue(payload, "credential_id"))
		step.UseCaseID = firstNonEmpty(stringValue(with, "use_case_id"), stringValue(payload, "use_case_id"))
		step.References = referencedStepIDs(raw)
		if step.ID != "" {
			upsertStep(af, step)
		}
	}
}

func applyPipelineOutput(af *AssessmentFacts, b []byte) {
	root := asMap(parseJSON(b))
	if payload := asMap(root["payload"]); len(payload) > 0 {
		root = payload
	}
	if po := asMap(root["pipeline_output"]); len(po) > 0 {
		applyPipelineOutputMap(af, po, "pipeline_output")
	}
	failed := failureStepID(root)
	if failed != "" {
		markStepStatus(af, failed, "failed", "explicit_failure", false, true)
	}
	if details := failureDetailsMap(root); len(details) > 0 {
		applyPipelineOutputMap(af, details, "failure_details")
	}
}

func applyPipelineOutputMap(af *AssessmentFacts, m map[string]any, source string) {
	for id, raw := range m {
		item := asMap(raw)
		if len(item) == 0 {
			continue
		}
		outputs, ok := item["outputs"]
		if !ok {
			continue
		}
		empty := isEmptyOutput(outputs)
		status := "succeeded"
		statusSource := source
		if empty && stepUse(af, id) == "mobile-automation" {
			status = "failed"
			statusSource = "empty_outputs"
		}
		markStepStatus(af, id, status, statusSource, true, empty)
		if s, ok := outputs.(string); ok {
			if co, ok := credentialOfferFromDeeplink(s); ok {
				co.StepID = id
				af.CredentialOffers = appendCredentialOffer(af.CredentialOffers, co)
			}
			if strings.Contains(s, "request_uri=") {
				p := readPresentationBytes([]byte(s))
				p.StepID = id
				af.Presentations = appendPresentation(af.Presentations, p)
			}
		}
	}
}

func credentialOfferFromDeeplink(s string) (CredentialOfferFacts, bool) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return CredentialOfferFacts{}, false
	}
	offer := u.Query().Get("credential_offer")
	if offer == "" {
		return CredentialOfferFacts{}, false
	}
	return readOfferBytes([]byte(offer)), true
}

func failureStepID(root map[string]any) string {
	failure := asMap(root["failure"])
	for _, m := range []map[string]any{failure, asMap(failure["cause"])} {
		app := asMap(m["applicationFailureInfo"])
		details := asMap(app["details"])
		payloads := anySlice(details["payloads"])
		if len(payloads) > 0 {
			if s := toString(payloads[0]); s != "" {
				return s
			}
			if nested := anySlice(payloads[0]); len(nested) > 0 {
				if s := toString(nested[0]); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func failureDetailsMap(root map[string]any) map[string]any {
	failure := asMap(root["failure"])
	for _, m := range []map[string]any{asMap(failure["cause"]), failure} {
		app := asMap(m["applicationFailureInfo"])
		details := asMap(app["details"])
		payloads := anySlice(details["payloads"])
		if len(payloads) > 1 {
			return asMap(payloads[1])
		}
	}
	return nil
}

func buildAttempts(af *AssessmentFacts) {
	for _, co := range af.CredentialOffers {
		if co.StepID == "" {
			continue
		}
		consumer := consumerFor(af.Steps, co.StepID, "mobile-automation")
		if consumer.ID == "" || consumer.Status == "" {
			continue
		}
		af.IssuanceAttempts = append(af.IssuanceAttempts, AttemptFacts{ProducerStepID: co.StepID, ConsumerStepID: consumer.ID, ConsumerStatus: consumer.Status, ConsumerStatusSource: consumer.StatusSource, Profile: co.Profile, Format: co.Format, GrantType: co.GrantType, IssuerURL: co.IssuerURL, IsPID: co.IsPID, IsSDJWT: co.IsSDJWT, IsMdoc: co.IsMdoc})
	}
	for _, producer := range af.Steps {
		if producer.Use != "use-case-verification-deeplink" {
			continue
		}
		consumer := consumerFor(af.Steps, producer.ID, "mobile-automation")
		if consumer.ID == "" || consumer.Status == "" {
			continue
		}
		text := strings.ToLower(producer.ID + " " + producer.UseCaseID + " " + producer.ActionID)
		af.PresentationAttempts = append(af.PresentationAttempts, AttemptFacts{ProducerStepID: producer.ID, ConsumerStepID: consumer.ID, ConsumerStatus: consumer.Status, ConsumerStatusSource: consumer.StatusSource, Format: formatFromText(text), IsSDJWT: strings.Contains(text, "sd-jwt"), IsMdoc: strings.Contains(text, "mdoc"), IsOpenID4VP: true})
	}
}

func consumerFor(steps []StepFacts, producerID, use string) StepFacts {
	producerIndex := -1
	for i, step := range steps {
		if step.ID == producerID {
			producerIndex = i
		}
		if step.Use == use && stringIn(producerID, step.References) {
			return step
		}
	}
	if producerIndex >= 0 {
		for _, step := range steps[producerIndex+1:] {
			if step.Use == use {
				return step
			}
			if step.Use == "credential-offer" || step.Use == "use-case-verification-deeplink" {
				break
			}
		}
	}
	return StepFacts{}
}

func anyAttemptStatus(attempts []AttemptFacts, status string) bool {
	for _, attempt := range attempts {
		if attempt.ConsumerStatus == status {
			return true
		}
	}
	return false
}

func upsertStep(af *AssessmentFacts, step StepFacts) {
	for i := range af.Steps {
		if af.Steps[i].ID == step.ID {
			old := af.Steps[i]
			if step.Use != "" {
				old.Use = step.Use
			}
			if step.ActionID != "" {
				old.ActionID = step.ActionID
			}
			if step.CredentialID != "" {
				old.CredentialID = step.CredentialID
			}
			if step.UseCaseID != "" {
				old.UseCaseID = step.UseCaseID
			}
			if len(step.References) > 0 {
				old.References = appendUnique(old.References, step.References...)
			}
			af.Steps[i] = old
			return
		}
	}
	af.Steps = append(af.Steps, step)
}

func markStepStatus(af *AssessmentFacts, id, status, source string, outputPresent, outputEmpty bool) {
	upsertStep(af, StepFacts{ID: id})
	for i := range af.Steps {
		if af.Steps[i].ID == id {
			if status != "" {
				af.Steps[i].Status = status
			}
			if source != "" {
				af.Steps[i].StatusSource = source
			}
			af.Steps[i].OutputPresent = af.Steps[i].OutputPresent || outputPresent
			af.Steps[i].OutputEmpty = outputEmpty
			af.Steps[i].OutputNonEmpty = outputPresent && !outputEmpty
			return
		}
	}
}

func stepUse(af *AssessmentFacts, id string) string {
	for _, step := range af.Steps {
		if step.ID == id {
			return step.Use
		}
	}
	return ""
}

func referencedStepIDs(v any) []string {
	var values []string
	collectStrings(v, &values)
	var out []string
	for _, value := range values {
		for {
			start := strings.Index(value, "${{")
			if start < 0 {
				break
			}
			rest := value[start+3:]
			end := strings.Index(rest, ".outputs")
			if end < 0 {
				break
			}
			out = appendUnique(out, strings.TrimSpace(rest[:end]))
			value = rest[end+len(".outputs"):]
		}
	}
	return out
}

func collectStrings(v any, out *[]string) {
	switch x := v.(type) {
	case string:
		*out = append(*out, x)
	case []any:
		for _, item := range x {
			collectStrings(item, out)
		}
	case map[string]any:
		for _, item := range x {
			collectStrings(item, out)
		}
	}
}

func isEmptyOutput(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(x) == ""
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	default:
		return false
	}
}

func formatFromText(s string) string {
	if strings.Contains(s, "mdoc") {
		return "MDOC"
	}
	if strings.Contains(s, "sd-jwt") {
		return "SD_JWT"
	}
	return ""
}

func appendCredentialOffer(items []CredentialOfferFacts, co CredentialOfferFacts) []CredentialOfferFacts {
	for _, item := range items {
		if item.StepID != "" && item.StepID == co.StepID {
			return items
		}
	}
	return append(items, co)
}

func appendPresentation(items []PresentationFacts, p PresentationFacts) []PresentationFacts {
	for _, item := range items {
		if item.StepID != "" && item.StepID == p.StepID {
			return items
		}
	}
	return append(items, p)
}

func stringFromRaw(raw json.RawMessage) string {
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

func selectedCredentialConfiguration(co CredentialOfferFacts, configs []CredentialConfigurationFacts) (CredentialConfigurationFacts, bool) {
	for _, cfg := range configs {
		if cfg.ID == co.ConfigurationID {
			return cfg, true
		}
	}
	return CredentialConfigurationFacts{}, false
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
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
	if strings.Contains(strings.ToLower(s), "pixel") {
		af.Wallet.RanOnPhysicalAndroid = true
	}
	extractConformanceOutputFacts(af, s)
}
func extractConformanceInputFacts(af *AssessmentFacts, s string) {
	if strings.Contains(s, "conformance-check") && strings.Contains(s, "WEBUILD-") {
		af.Conformance.WEBuildWalletCheckCount = strings.Count(s, "\"use\":\"conformance-check\"")
	}
}

func extractConformanceOutputFacts(af *AssessmentFacts, s string) {
	if af.Conformance.WEBuildWalletCheckCount == 0 {
		return
	}
	deeplinkCount := strings.Count(s, "credential_offer_uri=")
	completedWalletFlows := strings.Count(s, "Oups! Something went wrong")
	if deeplinkCount >= af.Conformance.WEBuildWalletCheckCount && completedWalletFlows >= af.Conformance.WEBuildWalletCheckCount {
		af.Conformance.WEBuildWalletChecksCompleted = true
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
		co := readOfferBytes(item["credential_offer"])
		co.StepID = stringFromRaw(item["step_id"])
		return co
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
	root := asMap(parseJSON(b))
	metadata := root
	if payload := asMap(root["payload"]); len(payload) > 0 {
		if _, ok := payload["credential_configurations_supported"]; ok || stringValue(payload, "credential_issuer", "issuer", "iss", "sub") != "" {
			metadata = payload
		}
	}
	ls := strings.ToLower(analysis)
	is.IssuerURL = stringValue(metadata, "credential_issuer", "issuer", "iss", "sub")
	if is.IssuerURL == "" {
		is.IssuerURL = firstJSONString(analysis, "credential_issuer", "issuer", "iss", "sub")
	}
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
	is.Configurations = readCredentialConfigurations(asMap(metadata["credential_configurations_supported"]))
	for _, cfg := range is.Configurations {
		is.ConfigurationIDs = appendUnique(is.ConfigurationIDs, cfg.ID)
	}
	return is
}

func readCredentialConfigurations(configs map[string]any) []CredentialConfigurationFacts {
	out := make([]CredentialConfigurationFacts, 0, len(configs))
	for id, raw := range configs {
		m := asMap(raw)
		cfg := CredentialConfigurationFacts{
			ID:      id,
			Format:  stringValue(m, "format"),
			VCT:     stringValue(m, "vct"),
			Doctype: stringValue(m, "doctype"),
		}
		b, _ := json.Marshal(raw)
		ls := strings.ToLower(id + " " + cfg.Format + " " + cfg.VCT + " " + cfg.Doctype + " " + string(b))
		cfg.IsPID = strings.Contains(ls, "pid") || strings.Contains(ls, "person identification")
		cfg.IsSDJWT = strings.Contains(ls, "sd-jwt") || strings.Contains(ls, "dc+sd-jwt") || strings.Contains(ls, "vc+sd-jwt")
		cfg.IsMdoc = strings.Contains(ls, "mso_mdoc") || strings.Contains(ls, "mdoc")
		out = append(out, cfg)
	}
	return out
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
	af.Issuer.Configurations = appendUniqueConfigurations(af.Issuer.Configurations, is.Configurations...)
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
	root := asMap(parseJSON(b))
	if payload := asMap(root["payload"]); len(payload) > 0 {
		p.ResponseType = firstNonEmpty(p.ResponseType, stringValue(payload, "response_type"))
		p.ResponseMode = firstNonEmpty(p.ResponseMode, stringValue(payload, "response_mode"))
		p.ClientID = firstNonEmpty(p.ClientID, stringValue(payload, "client_id"))
		p.ClientIDScheme = firstNonEmpty(p.ClientIDScheme, stringValue(payload, "client_id_scheme"))
		collectDCQL(&p, asMap(payload["dcql_query"]))
	} else {
		collectDCQL(&p, asMap(root["dcql_query"]))
	}
	return p
}

func collectDCQL(p *PresentationFacts, dcql map[string]any) {
	credentials, _ := dcql["credentials"].([]any)
	for _, rawCredential := range credentials {
		credential := asMap(rawCredential)
		if format := stringValue(credential, "format"); format != "" {
			p.DCQLFormats = appendUnique(p.DCQLFormats, format)
		}
		meta := asMap(credential["meta"])
		for _, v := range anySlice(meta["vct_values"]) {
			if s := toString(v); s != "" {
				p.VCTValues = appendUnique(p.VCTValues, s)
			}
		}
		for _, rawClaim := range anySlice(credential["claims"]) {
			claim := asMap(rawClaim)
			var parts []string
			for _, part := range anySlice(claim["path"]) {
				if s := toString(part); s != "" {
					parts = append(parts, s)
				}
			}
			if len(parts) > 0 {
				p.ClaimPaths = appendUnique(p.ClaimPaths, strings.Join(parts, "."))
			}
		}
	}
}

func anySlice(v any) []any {
	items, _ := v.([]any)
	return items
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

func appendUniqueConfigurations(a []CredentialConfigurationFacts, vals ...CredentialConfigurationFacts) []CredentialConfigurationFacts {
	seen := map[string]bool{}
	for _, cfg := range a {
		seen[cfg.ID] = true
	}
	for _, cfg := range vals {
		if !seen[cfg.ID] {
			a = append(a, cfg)
			seen[cfg.ID] = true
		}
	}
	return a
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
