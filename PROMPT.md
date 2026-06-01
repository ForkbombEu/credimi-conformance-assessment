# Codex Prompt — Credimi Conformance Assessment Generator in Go

## Goal

Implement a thin, deterministic Go program that generates `conformance-assessment.md` reports from Credimi conformance fixtures.

The program must reproduce the style and semantics of the existing manually/LLM-generated reports for the provided fixtures, using:

1. Credimi source-of-truth package **v1.1.1**:
   - `credimi-conformance-aggregation-taxonomy-v1.1.yaml`
   - `credimi-conformance-evidence-registry-v1.1.yaml`
   - `credimi-conformance-result.schema.v1.1.json`
   - `credimi-flat-conformance-test-list-v1.1.md`
2. Fixture inputs and extracted artifacts:
   - Temporal `input.json`
   - Temporal `output.json`
   - `discovered-steps.json`
   - `extraction-summary.json`
   - credential-offer extraction outputs
   - issuer `.well-known` / metadata outputs
   - presentation-request / DCQL / request-uri outputs, when present
3. The existing generated Markdown assessments as golden expected outputs.

The resulting code must be as thin as possible. Business logic must be declarative and loaded from the taxonomy YAML wherever feasible.

The intended future behavior is:

> If an updated taxonomy/evidence registry/schema is generated and it does not introduce new primitive predicates or new test types, the updated YAML/JSON files can be dropped into the existing codebase and the Go program continues to work without code changes.

---

## Architectural principle

Use this split:

> Go extracts normalized facts. YAML decides which facts imply which tests.

Do **not** hard-code individual test logic such as:

> “Test #6 passes when PID + SD-JWT + successful Wallet issuance flow.”

Instead, Go should extract facts like:

```yaml
credential.profile: PID
credential.format: SD_JWT
wallet.issuance_flow_completed: true
```

Then the taxonomy should contain a declarative rule that maps those facts to the flat test ID.

---

## Required outputs

For each fixture, generate:

```text
conformance-assessment-<fixture-slug>.md
```

The Markdown report must follow the same structure as the existing generated reports:

```markdown
# Credimi Conformance Assessment — <fixture name>

## Passed tests digest

| # | Actor | Test | Test result |
|---:|---|---|---|
| ... |

**Passed tests count:** N

## Assessment summary

...

## Workflow steps

...

## Fixture evidence used

...

## Assessment table

Blank **Test result** cells mean the fixture did not execute or did not sufficiently prove that test. **HITM** is intentionally left empty for human review notes.

| # | Actor | Test | Test result | HITM | Evidence strength | Recommended execution | Standards / source references | Notes |
|---:|---|---|---|---|---|---|---|---|
| ... full flat test list ... |
```

The **Passed tests digest must be at the top** of the file.

The full assessment table must include every row from `credimi-flat-conformance-test-list-v1.1.md`, with blank `Test result` cells for tests that were not passed.

---

## Inputs

Implement a CLI with this approximate interface:

```bash
credimi-assess \
  --source-dir ./source-of-truth \
  --fixtures-dir ./fixtures \
  --extracted-dir ./out \
  --out-dir ./assessments
```

Also support generating a single fixture:

```bash
credimi-assess \
  --source-dir ./source-of-truth \
  --fixtures-dir ./fixtures \
  --extracted-dir ./out \
  --fixture EUDI-iss-ver \
  --out-dir ./assessments
```

Expected fixture layout:

```text
fixtures/<FixtureName>/input.json
fixtures/<FixtureName>/output.json
out/<fixture-slug>/discovered-steps.json
out/<fixture-slug>/extraction-summary.json
out/<fixture-slug>/credential-offers/.../credential-offer.json
out/<fixture-slug>/credential-offers/.../credential-offer-deeplink.txt
out/<fixture-slug>/credential-offers/.../credential-offer-resolution-chain.json
out/<fixture-slug>/credential-offers/.../issuer-metadata-fetch.json
out/<fixture-slug>/credential-offers/.../well-known.json
out/<fixture-slug>/presentation-requests/.../presentation-deeplink.txt
out/<fixture-slug>/presentation-requests/.../request-uri-fetch.json
out/<fixture-slug>/presentation-requests/.../request-uri-output.json
out/<fixture-slug>/presentation-requests/.../request-uri-raw.jwt
```

The program must tolerate missing artifact groups. For example, `AgeVerification` has no credential-offer or presentation-request extraction artifacts and should still produce a valid report.

---

## Source-of-truth parsing

Load these files from `--source-dir`:

```text
credimi-conformance-aggregation-taxonomy-v1.1.yaml
credimi-conformance-evidence-registry-v1.1.yaml
credimi-conformance-result.schema.v1.1.json
credimi-flat-conformance-test-list-v1.1.md
```

The flat test list is the atomic row vocabulary. Parse it into rows with at least:

```go
type FlatTest struct {
    Number               int
    Actor                string
    Test                 string
    EvidenceStrength     string
    RecommendedExecution string
    SourceReferences     string
    Notes                string
}
```

The generator must preserve the source references and notes in the output table.

---

## Declarative rule layer in taxonomy

The current taxonomy may not yet contain enough executable assessment rules. Extend it in a backward-compatible way by adding a section such as:

```yaml
assessment_rules:
  - rule_id: wallet-receive-pid-offer
    test_id: 1
    result_text: "PASSED — PID credential offer processed by Wallet"
    strength: definitive
    when:
      all:
        - fact: credential_offer.exists
          equals: true
        - fact: credential.profile
          equals: PID
        - fact: wallet.issuance_flow_completed
          equals: true

  - rule_id: wallet-receive-authorization-code-offer
    test_id: 4
    result_text: "PASSED — authorization_code grant observed and issuance completed"
    strength: definitive
    when:
      all:
        - fact: credential_offer.grant_type
          equals: authorization_code
        - fact: wallet.issuance_flow_completed
          equals: true

  - rule_id: wallet-receive-pid-sd-jwt
    test_id: 6
    result_text: "PASSED — PID configuration uses dc+sd-jwt / urn:eudi:pid:1"
    strength: definitive
    when:
      all:
        - fact: credential.profile
          equals: PID
        - fact: credential.format
          equals: SD_JWT
        - fact: wallet.issuance_flow_completed
          equals: true
```

The exact YAML structure can be refined, but the evaluator must support at least:

```yaml
when:
  all: []
  any: []
  not: {}
```

Primitive comparisons should support:

```yaml
equals
not_equals
contains
contains_any
exists
matches_regex
lte
gte
```

Also support rules that generate a result text using simple templates:

```yaml
result_text: "PASSED — selected configuration {{ credential.configuration_id }} uses {{ credential.raw_format }}"
```

Keep the templating minimal and deterministic. Missing template variables must fail the rule or render as a clear placeholder only in debug mode, never silently.

---

## Normalized facts model

Go must build a normalized fact model from the fixture artifacts. This fact model is the stable contract between code and taxonomy rules.

Implement something close to this:

```go
type AssessmentFacts struct {
    Fixture FixtureFacts `json:"fixture"`
    Workflow WorkflowFacts `json:"workflow"`
    Evidence EvidenceFacts `json:"evidence"`
    CredentialOffers []CredentialOfferFacts `json:"credential_offers"`
    Presentations []PresentationFacts `json:"presentations"`
    Wallet WalletFacts `json:"wallet"`
    Issuer IssuerFacts `json:"issuer"`
    Verifier VerifierFacts `json:"verifier"`
    Environment EnvironmentFacts `json:"environment"`
}
```

The rule evaluator may expose facts as flattened paths, for example:

```text
credential_offer.exists
credential_offer.count
credential_offer.success_count
credential_offer.grant_type
credential.configuration_id
credential.profile
credential.format
credential.raw_format
credential.vct
credential.doctype
credential.signing_algorithms
credential.proof_signing_algorithms
credential.binding_methods
credential.issuer_url
issuer.metadata_fetched
issuer.metadata_format
issuer.metadata_content_type
issuer.metadata_advertises_pid
issuer.metadata_advertises_sd_jwt
issuer.metadata_advertises_mdoc
issuer.metadata_advertises_signing_algorithms
issuer.metadata_advertises_jwk_binding
issuer.metadata_advertises_did_binding
presentation.exists
presentation.request_uri_fetched
presentation.jwt_signed
presentation.jwt_alg
presentation.has_x5c
presentation.response_type
presentation.response_mode
presentation.dcql.exists
presentation.dcql.format
presentation.dcql.vct_values
presentation.dcql.claim_paths
presentation.client_id
presentation.client_id_scheme
wallet.issuance_flow_completed
wallet.presentation_flow_completed
wallet.presentation_share_completed
wallet.no_visible_error
wallet.ran_on_physical_android
workflow.temporal_input_present
workflow.temporal_output_present
workflow.has_screenshots_or_videos
evidence.step_artifacts_present
evidence.artifacts_hashed
```

When there are multiple credential offers or presentation requests, the engine must evaluate rules over the set. A rule passes if at least one evidence item satisfies the rule unless the rule explicitly requires all.

---

## Extraction logic belongs in Go

Go is allowed to contain generic protocol/extraction logic. This includes:

### Temporal extraction

From `input.json`:

- workflow name
- steps
- step IDs
- step `use` type
- action/use-case/credential IDs
- deeplink parameter wiring such as `${{step-id.outputs}}`
- runner/device metadata where available

From `output.json`:

- workflow ID / run ID
- per-step outputs
- `flow_output`
- test run URLs
- result video URLs
- screenshot URLs
- generated deeplinks

### Credential-offer extraction

From credential-offer artifacts:

- offer exists / parses
- scheme: `openid-credential-offer`, `haip-vci`, etc.
- `credential_issuer`
- `credential_configuration_ids`
- grant type: `authorization_code`, `pre-authorized_code`, etc.
- `issuer_state` or pre-authorized code presence, without leaking secrets into reports
- resolution chain status
- fetched metadata status, content type, hash

### Issuer metadata extraction

From `.well-known` / `well-known.json`:

- metadata fetched
- metadata format: JSON or JWT
- issuer metadata JWT header, if present
- signing algorithm, if present
- `x5c`, if present
- `credential_configurations_supported`
- for each selected credential configuration:
  - format
  - vct
  - doctype
  - scope
  - credential signing algorithms
  - proof signing algorithms
  - binding methods
  - claim paths

### Presentation request extraction

From presentation artifacts:

- presentation deeplink exists
- scheme: `openid4vp`, `haip-vp`, etc.
- `client_id`
- `client_id` scheme, e.g. `x509_hash`
- `request_uri`
- `request_uri_method`
- request URI fetched
- JWT header: `typ`, `alg`, `x5c`
- payload:
  - `response_type`
  - `response_mode`
  - `response_uri`
  - `nonce`
  - `state`
  - `aud`
  - `client_metadata`
  - `vp_formats_supported`
  - `dcql_query`
  - credential formats
  - `vct_values`
  - requested claim paths

### Maestro/mobile-output extraction

From `flow_output`:

- step completed or failed
- app launched
- deeplink opened
- visible failure strings such as `Oups! Something went wrong`
- assertions that failure string is not visible
- relevant positive UI evidence:
  - `Add`
  - `Authorize`
  - `Review & Send`
  - `Share`
  - `Close`
- presentation flow completion
- issuance flow completion

Be careful: these are black-box UI observations. Do not overclaim internal cryptographic validation unless the fixture contains verifier-side or negative-fixture evidence.

---

## Classification / mapping rules

The program must infer metadata dimensions deterministically using declarative mappings loaded from the taxonomy where possible.

Add/consume taxonomy mappings such as:

```yaml
normalization_rules:
  credential_profile:
    PID:
      any:
        - field: credential.configuration_id
          contains: "pid"
        - field: credential.vct
          equals: "urn:eudi:pid:1"
        - field: credential.doctype
          equals: "eu.europa.ec.eudi.pid.1"
  credential_format:
    SD_JWT:
      any:
        - field: credential.raw_format
          equals: "dc+sd-jwt"
        - field: credential.raw_format
          equals: "vc+sd-jwt"
    MDOC:
      any:
        - field: credential.raw_format
          equals: "mso_mdoc"
  grant_type:
    authorization_code:
      field: credential_offer.grants.authorization_code
      exists: true
    pre-authorized_code:
      field: credential_offer.grants.pre-authorized_code
      exists: true
  signing_algorithm_family:
    ECDSA:
      any:
        - field: credential.signing_algorithms
          contains_any: ["ES256", "ES384", "ES512", -7, -35, -36]
    EdDSA:
      any:
        - field: credential.signing_algorithms
          contains_any: ["EdDSA", "Ed25519"]
  key_strength:
    P256:
      any:
        - field: credential.signing_algorithms
          contains_any: ["ES256", -7]
        - field: credential.curves
          contains_any: ["P-256", 1]
```

Hard-coded Go mappings should be limited to raw extraction and generic helpers. Test semantics and profile mappings should be data-driven.

---

## Do not overclaim

The generated report must be conservative.

Examples:

- If issuer metadata advertises ES256 and the Wallet flow completes, it may mark:
  - Wallet can complete issuance for a credential signed with ES256 / P-256
  - Issuer metadata advertises supported signing algorithms

- It must **not** mark:
  - Wallet cryptographically verified the issuer signature
  - Verifier validated SD-JWT disclosures
  - Verifier accepted a valid presentation

unless the fixture contains explicit verifier callback/result evidence, invalid negative-fixture evidence, or external validation output.

Presentation request generation and Wallet sharing can be marked as passed when the fixture proves them. Verifier-side validation of the returned VP must remain blank unless callback/result evidence exists.

---

## Existing fixture expectations

Use these existing reports as golden references:

```text
mock-conformance-assessment-eudi-iss2.md
conformance-assessment-eudi-iss-ver.md
conformance-assessment-age-verification.md
conformance-assessment-multipaz.md
conformance-assessment-talao-iss-cred13.md
conformance-assessment-eudiw-checks-5x.md
```

Expected high-level pass counts from current reports:

```text
EUDI-iss2: 19 passed tests
EUDI-iss-ver: 30 passed tests
AgeVerification: 1 passed test
Multipaz: 18 passed tests
Talao-iss-cred13: 13 passed tests
eudiw-checks-5x: 23 passed tests
```

Implement tests that compare generated output against these reports.

Recommended testing strategy:

1. Unit tests for extraction:
   - credential offer parsing
   - issuer metadata parsing
   - presentation request parsing
   - Maestro flow-output interpretation
   - Temporal step wiring
2. Unit tests for rule evaluation:
   - `all`
   - `any`
   - `not`
   - equality
   - contains
   - regex
   - missing-field behavior
3. Golden tests for each fixture:
   - compare passed test IDs
   - compare passed test count
   - compare passed digest rows
   - optionally compare full Markdown snapshot after normalizing volatile dates/UUIDs if necessary

Do not require exact byte-for-byte full Markdown equality at first if source references contain unstable whitespace. At minimum, full digest and pass/blank mapping must match.

---

## Report rendering rules

The renderer must:

- keep `Passed tests digest` at the top;
- include all flat-list tests in the assessment table;
- preserve flat-list test order;
- leave `HITM` blank;
- leave `Test result` blank unless a rule passed;
- include evidence summary sections;
- avoid secrets and full tokens in the report;
- truncate long raw JWTs or raw deeplinks if ever displayed;
- include hashes, URLs, fixture file names, and workflow IDs where useful;
- avoid claiming official certification.

The report must explicitly say:

```markdown
Blank **Test result** cells mean the fixture did not execute or did not sufficiently prove that test. **HITM** is intentionally left empty for human review notes.
```

---

## Suggested Go package layout

Use a simple package layout, not an over-engineered framework:

```text
cmd/credimi-assess/main.go
internal/sot/loader.go
internal/sot/flatlist.go
internal/fixture/loader.go
internal/extract/temporal.go
internal/extract/credential_offer.go
internal/extract/issuer_metadata.go
internal/extract/presentation.go
internal/extract/maestro.go
internal/facts/model.go
internal/facts/builder.go
internal/rules/model.go
internal/rules/evaluator.go
internal/report/markdown.go
internal/testutil/golden.go
```

Keep dependencies minimal:

- standard library where possible
- YAML parser: `gopkg.in/yaml.v3`
- JSON Schema validation only if needed for validating output objects; do not block Markdown report generation on schema validation unless requested

Avoid embedding business logic in the report renderer.

---

## Determinism requirements

The generator must be deterministic:

- stable ordering of fixtures
- stable ordering of evidence items
- stable ordering of passed tests by flat-list number
- stable Markdown output
- stable slug generation
- stable handling of missing values
- no wall-clock timestamps in the report unless supplied by fixture evidence
- no network access during report generation

---

## Acceptance criteria

The implementation is acceptable when:

1. It loads source-of-truth v1.1.1 files.
2. It loads the fixture bundle layout.
3. It generates one Markdown assessment per fixture.
4. It can skip or select fixtures by CLI option.
5. It reproduces the passed-test digest for the six current reports.
6. It marks only defensible tests as passed.
7. Most test-to-fact mapping logic is in taxonomy YAML, not Go.
8. The Go code remains generic: extract facts, evaluate declarative rules, render report.
9. Updating taxonomy YAML without new primitive predicates does not require code changes.

---

## Implementation advice

Start with the thin vertical slice:

1. Parse the flat test list into rows.
2. Load fixture `input.json` and `output.json`.
3. Extract a minimal fact model.
4. Add a small `assessment_rules` section to the taxonomy for the currently passed tests.
5. Generate the digest and full table.
6. Make `EUDI-iss2` pass as golden.
7. Add presentation extraction and make `EUDI-iss-ver` pass.
8. Add the remaining four fixtures.
9. Refactor only after the golden tests are stable.

Do not try to implement a general conformance framework first. The correct abstraction should emerge from matching the six concrete fixtures.

---

## Important semantic constraints from current assessment work

- `AgeVerification` currently proves only that raw Temporal input/output evidence exists. It does not prove EUDI credential/protocol conformance because no credential-offer or presentation-request artifacts were extracted.
- `Talao-iss-cred13` uses pre-authorized-code and non-PID credential configurations; do not mark PID/EAA-specific tests unless the taxonomy can map the credential profile defensibly.
- `Multipaz` proves PID mdoc issuance through a real Wallet flow.
- `EUDI-iss-ver` proves issuance plus Wallet-side presentation handling. It does not prove verifier-side cryptographic acceptance unless callback/result evidence is added.
- `eudiw-checks-5x` proves several WE BUILD / VCI check flows were processed by Wallet automation, but do not overclaim independent credential cryptographic validation unless corresponding validation artifacts are present.
- `EUDI-iss2` proves PID SD-JWT issuance through the Wallet flow, with ES256/P-256 evidence, not RSA.

---

## Deliverables

Produce:

1. Go source code.
2. Extended taxonomy YAML containing declarative assessment rules and normalization rules.
3. Tests using the supplied fixtures and generated assessments.
4. README with usage examples.
5. A short design note explaining which logic lives in Go and which logic lives in taxonomy.


---

## Handoff package contents and golden-test requirement

This handoff zip includes everything required to implement and test the generator:

```text
source-of-truth/
  README-conformance-artifacts-v1.1.md
  credimi-conformance-aggregation-taxonomy-v1.1.yaml
  credimi-conformance-evidence-registry-v1.1.yaml
  credimi-conformance-result.schema.v1.1.json
  credimi-flat-conformance-test-list-v1.1.md
  ...

fixtures/
  AgeVerification/
  EUDI-iss-ver/
  EUDI-iss2/
  eudiw-checks-5x/
  Multipaz/
  Talao-iss-cred13/

out/
  AgeVerification/
  eudi-iss-ver/
  eudi-iss2/
  eudiw-checks-5x/
  Multipaz/
  Talao-iss-cred13/

golden-assessments/
  conformance-assessment-age-verification.md
  conformance-assessment-eudi-iss-ver.md
  conformance-assessment-eudi-iss2.md
  conformance-assessment-eudiw-checks-5x.md
  conformance-assessment-multipaz.md
  conformance-assessment-talao-iss-cred13.md
```

The six fixtures and the six `golden-assessments/*.md` files are mandatory golden tests.

The generated Go program must be able to run against these six fixtures and produce Markdown reports that are semantically equivalent to the included golden assessments. For the first implementation, treat the golden files as expected outputs and aim for byte-for-byte equality where practical. Where byte-for-byte equality is not immediately practical, the implementation must at minimum preserve:

1. the same passed-test set;
2. the same `Passed tests digest` section at the top;
3. the same `Passed tests count`;
4. the same non-empty `Test result` cells in the assessment table;
5. blank `Test result` cells for tests not passed;
6. the same conservative evidence policy, especially:
   - do not mark verifier-side cryptographic validation as passed unless verifier callback/result evidence is present;
   - do not mark credential signature validation as passed merely because metadata advertises a signing algorithm;
   - do not infer Wallet internals beyond successful Maestro flow completion, unless explicit artifacts support it.

Expected golden-test mapping:

| Fixture | Extracted-artifact directory | Golden expected output |
|---|---|---|
| `AgeVerification` | `out/AgeVerification` | `golden-assessments/conformance-assessment-age-verification.md` |
| `EUDI-iss-ver` | `out/eudi-iss-ver` | `golden-assessments/conformance-assessment-eudi-iss-ver.md` |
| `EUDI-iss2` | `out/eudi-iss2` | `golden-assessments/conformance-assessment-eudi-iss2.md` |
| `eudiw-checks-5x` | `out/eudiw-checks-5x` | `golden-assessments/conformance-assessment-eudiw-checks-5x.md` |
| `Multipaz` | `out/Multipaz` | `golden-assessments/conformance-assessment-multipaz.md` |
| `Talao-iss-cred13` | `out/Talao-iss-cred13` | `golden-assessments/conformance-assessment-talao-iss-cred13.md` |

Add automated tests, preferably Go tests, that run the generator over all six fixtures and compare the output against the corresponding golden assessment files. Include a deterministic normalization layer for comparison only if needed, but the normal generator output must remain stable and deterministic.

