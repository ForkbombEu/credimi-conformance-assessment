# Credimi Conformance Assessment Generator

This repository contains a thin, deterministic Go CLI that generates Credimi
`conformance-assessment-<fixture-slug>.md` reports from the supplied fixture
bundle and source-of-truth package.

The implementation follows the handoff principle:

> Go extracts normalized facts. The taxonomy decides which facts imply which
> flat conformance tests.

## Usage

Generate assessments for every fixture:

```bash
go run ./cmd/credimi-assess \
  --source-dir ./source-of-truth \
  --fixtures-dir ./fixtures \
  --extracted-dir ./out \
  --out-dir ./assessments
```

Generate a single fixture:

```bash
go run ./cmd/credimi-assess \
  --source-dir ./source-of-truth \
  --fixtures-dir ./fixtures \
  --extracted-dir ./out \
  --fixture EUDI-iss-ver \
  --out-dir ./assessments
```

The CLI writes one Markdown report per selected fixture.

## Inputs

The generator expects the handoff layout:

- `source-of-truth/credimi-flat-conformance-test-list-v1.1.md`
- `source-of-truth/credimi-conformance-aggregation-taxonomy-v1.1.yaml`
- `fixtures/<FixtureName>/input.json`
- `fixtures/<FixtureName>/output.json`
- `out/<fixture-slug>/...` extracted artifacts

Artifact groups are optional. For example, `AgeVerification` has no credential
offer or presentation request artifacts and still produces a valid report.

## Design note: Go logic vs taxonomy logic

Go code is intentionally limited to generic mechanics:

- parse the flat test-list table into the atomic row vocabulary;
- discover fixtures in deterministic slug order;
- read Temporal input/output and extracted artifact files;
- build a normalized fact map such as `fixture.slug`,
  `workflow.temporal_input_present`, `credential_offer.exists`,
  `issuer.metadata_fetched`, and `presentation.exists`;
- evaluate declarative rule predicates (`all`, `any`, `not`, `equals`,
  `not_equals`, `contains`, `contains_any`, `exists`, `matches_regex`, `lte`,
  `gte`);
- render a stable Markdown report with the passed-test digest at the top and a
  full flat-list assessment table.

The source-of-truth taxonomy YAML owns conformance semantics. This repository
extends `credimi-conformance-aggregation-taxonomy-v1.1.yaml` with
backward-compatible `normalization_rules` and `assessment_rules` sections. The
assessment rules map extracted facts to flat test IDs and result text. Updating
those YAML rules does not require Go changes unless new primitive predicates or
new artifact extraction needs are introduced.

## Testing

Run all tests:

```bash
go test ./...
```

The golden test runs the generator over all six supplied fixtures and compares
semantic output against `golden-assessments/*.md`:

- the passed-test set;
- the passed-test count;
- the non-empty result text for each passed row;
- required Markdown sections and conservative blank-row policy text.

The initial renderer is deterministic but does not try to byte-for-byte clone the
older manually generated reports; it preserves the required conformance result
semantics while emitting the normalized report structure requested in the
handoff.
