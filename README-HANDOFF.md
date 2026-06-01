# Credimi Conformance Assessment Generator — Codex Handoff

This package contains the Codex prompt, Credimi conformance source-of-truth v1.1.1, six fixtures with extracted artifacts, and six generated golden conformance assessments.

Use `PROMPT.md` as the implementation brief.

## Contents

- `PROMPT.md` — Codex prompt for implementing the Go assessment generator.
- `source-of-truth/` — unpacked Credimi conformance source-of-truth v1.1.1.
- `source-packages/` — original source-of-truth zip.
- `fixtures/` — six Temporal input/output fixture directories.
- `out/` — extracted credential-offer, issuer metadata, presentation request, DCQL/JWT artifacts.
- `golden-assessments/` — six expected Markdown reports generated from the fixtures.
- `input-bundles/` — original uploaded all-inputs/outputs fixture zip.

## Golden outputs

The implementation should generate reports matching `golden-assessments/*.md` for the six included fixtures.
