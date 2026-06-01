# Design note

The generator uses a stable contract between code and taxonomy rules.

## Extracted by Go

Go extracts normalized, protocol-shaped facts from fixture directories:

- fixture identity and deterministic slug;
- Temporal input/output presence and selected workflow identifiers;
- credential-offer presence, issuer URL, configuration IDs, and grant type;
- issuer metadata presence and advertised formats/algorithms/binding hints;
- presentation request presence, request-URI fetch evidence, and JWT header or
  payload fields when present;
- conservative Wallet black-box flow evidence from Temporal output and extracted
  artifacts.

Go does not decide that a specific conformance test passed because a specific
fixture name or protocol profile was observed. It only exposes facts to the rule
evaluator and renders the rows selected by the taxonomy.

## Decided by taxonomy

The taxonomy extension contains:

- `normalization_rules` for profile and format classification; and
- `assessment_rules` mapping facts to flat test IDs and deterministic result
  text.

The evaluator supports generic Boolean composition and primitive comparisons.
This keeps business semantics in YAML so a refreshed source-of-truth package can
change mappings and result text without changing the Go renderer or CLI.

## Conservative policy

Reports intentionally avoid overclaiming. Successful Wallet automation is treated
as black-box interoperability evidence only. Verifier-side cryptographic
acceptance, credential signature validation, revocation, trust-list behavior, and
manual/CAB-style assurance remain blank unless explicit artifacts support those
claims.
