package facts

type AssessmentFacts struct {
	Fixture              FixtureFacts           `json:"fixture"`
	Workflow             WorkflowFacts          `json:"workflow"`
	Evidence             EvidenceFacts          `json:"evidence"`
	Steps                []StepFacts            `json:"steps"`
	CredentialOffers     []CredentialOfferFacts `json:"credential_offers"`
	Presentations        []PresentationFacts    `json:"presentations"`
	IssuanceAttempts     []AttemptFacts         `json:"issuance_attempts"`
	PresentationAttempts []AttemptFacts         `json:"presentation_attempts"`
	Wallet               WalletFacts            `json:"wallet"`
	Issuer               IssuerFacts            `json:"issuer"`
	Verifier             VerifierFacts          `json:"verifier"`
	Environment          EnvironmentFacts       `json:"environment"`
	Conformance          ConformanceFacts       `json:"conformance"`
}
type FixtureFacts struct{ Name, Slug string }
type WorkflowFacts struct {
	TemporalInputPresent, TemporalOutputPresent, HasScreenshotsOrVideos bool
	HasCompletedSteps, HasFailures                                      bool
	WorkflowID, RunID, Name                                             string
	Steps                                                               []string
}
type EvidenceFacts struct {
	StepArtifactsPresent, ArtifactsHashed bool
	ExtractionSummaryPresent              bool
}
type StepFacts struct {
	ID, Use, ActionID, CredentialID, UseCaseID, Status, StatusSource string
	OutputPresent, OutputEmpty, OutputNonEmpty                       bool
	References                                                       []string
}
type CredentialOfferFacts struct {
	Exists                                                                          bool
	StepID                                                                          string
	ConfigurationID, IssuerURL, GrantType, RawFormat, VCT, Doctype, Profile, Format string
	IsPID, IsSDJWT, IsMdoc                                                          bool
	SigningAlgorithms, ProofSigningAlgorithms, BindingMethods                       []string
}
type PresentationFacts struct {
	Exists, RequestURIFetched, JWTSigned, HasX5C                 bool
	StepID                                                       string
	JWTAlg, ResponseType, ResponseMode, ClientID, ClientIDScheme string
	DCQLFormats, VCTValues, ClaimPaths                           []string
}
type AttemptFacts struct {
	ProducerStepID, ConsumerStepID, ConsumerStatus, ConsumerStatusSource string
	Profile, Format, GrantType, IssuerURL                                string
	IsPID, IsSDJWT, IsMdoc, IsOpenID4VP, HasX5C                          bool
}
type WalletFacts struct{ IssuanceFlowCompleted, PresentationFlowCompleted, PresentationShareCompleted, NoVisibleError, RanOnPhysicalAndroid bool }
type IssuerFacts struct {
	MetadataFetched, MetadataAdvertisesPID, MetadataAdvertisesSDJWT, MetadataAdvertisesMdoc, MetadataAdvertisesJWKBinding, MetadataAdvertisesDIDBinding bool
	MetadataHasX5C, OfferedConfigurationPresent, OfferIssuerMatchesMetadataIssuer                                                                       bool
	MetadataFormat, MetadataContentType, IssuerURL                                                                                                      string
	ConfigurationIDs                                                                                                                                    []string
	Configurations                                                                                                                                      []CredentialConfigurationFacts
	MetadataAdvertisesSigningAlgorithms                                                                                                                 []string
}
type CredentialConfigurationFacts struct {
	ID, Format, VCT, Doctype string
	IsPID, IsSDJWT, IsMdoc   bool
}
type VerifierFacts struct{ CallbackResultPresent bool }
type ConformanceFacts struct {
	WEBuildWalletChecksCompleted bool
	WEBuildWalletCheckCount      int
}
type EnvironmentFacts struct{ Runner string }
