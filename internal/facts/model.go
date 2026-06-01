package facts

type AssessmentFacts struct {
	Fixture          FixtureFacts           `json:"fixture"`
	Workflow         WorkflowFacts          `json:"workflow"`
	Evidence         EvidenceFacts          `json:"evidence"`
	CredentialOffers []CredentialOfferFacts `json:"credential_offers"`
	Presentations    []PresentationFacts    `json:"presentations"`
	Wallet           WalletFacts            `json:"wallet"`
	Issuer           IssuerFacts            `json:"issuer"`
	Verifier         VerifierFacts          `json:"verifier"`
	Environment      EnvironmentFacts       `json:"environment"`
}
type FixtureFacts struct{ Name, Slug string }
type WorkflowFacts struct {
	TemporalInputPresent, TemporalOutputPresent, HasScreenshotsOrVideos bool
	WorkflowID, RunID, Name                                             string
	Steps                                                               []string
}
type EvidenceFacts struct {
	StepArtifactsPresent, ArtifactsHashed bool
	ExtractionSummaryPresent              bool
}
type CredentialOfferFacts struct {
	Exists                                                                          bool
	ConfigurationID, IssuerURL, GrantType, RawFormat, VCT, Doctype, Profile, Format string
	SigningAlgorithms, ProofSigningAlgorithms, BindingMethods                       []string
}
type PresentationFacts struct {
	Exists, RequestURIFetched, JWTSigned, HasX5C                 bool
	JWTAlg, ResponseType, ResponseMode, ClientID, ClientIDScheme string
	DCQLFormats, VCTValues, ClaimPaths                           []string
}
type WalletFacts struct{ IssuanceFlowCompleted, PresentationFlowCompleted, PresentationShareCompleted, NoVisibleError, RanOnPhysicalAndroid bool }
type IssuerFacts struct {
	MetadataFetched, MetadataAdvertisesPID, MetadataAdvertisesSDJWT, MetadataAdvertisesMdoc, MetadataAdvertisesJWKBinding, MetadataAdvertisesDIDBinding bool
	MetadataFormat, MetadataContentType                                                                                                                 string
	MetadataAdvertisesSigningAlgorithms                                                                                                                 []string
}
type VerifierFacts struct{ CallbackResultPresent bool }
type EnvironmentFacts struct{ Runner string }
