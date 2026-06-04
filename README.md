# Credimi Conformance Assessment Generator

This repository contains a thin, deterministic Go library, CLI, and REST wrapper that generates Credimi
`conformance-assessment-<fixture-slug>.md` reports from Temporal evidence,
pipeline artifacts, and the source-of-truth package.

The implementation follows the handoff principle:

> Go extracts normalized facts. The taxonomy decides which facts imply which
> flat conformance tests.

## Configuration

Runtime defaults live in `.env`. Start from the checked-in template:

```bash
cp .env.example .env
```

```dotenv
# Optional external source-of-truth directory override.
# Leave empty to use the source-of-truth files embedded in the Go module.
SOURCE_DIR=

# Optional path to a JSON request file used by the CLI when --input-json is not set.
# Leave empty when Temporal data is supplied directly by CLI JSON or REST request body.
TEMPORAL_DATA=

# Optional directory where generated Markdown reports are written.
# Leave empty to avoid file output: the CLI writes Markdown to stdout and the REST API returns it in JSON.
OUT_DIR=

# TCP port used by the REST API server and curl examples.
# The API listens on this port when started without --addr.
API_PORT=8080
```

## Input JSON

The CLI and REST API accept the same JSON shape:

```json
{
  "fixture": "EUDI-iss-ver",
  "pipeline_input": {
    "name": "EUDI issuer verification"
  },
  "pipeline_output": {
    "workflow_id": "example-workflow-id",
    "run_id": "example-run-id"
  },
  "evidence_input": {
    "discovered_steps": {},
    "extraction_summary": {},
    "credential_offers": [
      {
        "credential_issuer": "https://issuer.example",
        "credential_configuration_ids": ["pid_sd_jwt"],
        "grants": {
          "urn:ietf:params:oauth:grant-type:pre-authorized_code": {}
        }
      }
    ],
    "well_known": [
      {
        "credential_endpoint": "https://issuer.example/credential",
        "credential_configurations_supported": {
          "pid_sd_jwt": {
            "format": "vc+sd-jwt",
            "proof_types_supported": {
              "jwt": {
                "proof_signing_alg_values_supported": ["ES256"]
              }
            }
          }
        }
      }
    ],
    "presentation_requests": []
  }
}
```

`pipeline_input` and `pipeline_output` are the Credimi pipeline request and execution result. `evidence_input` is intentionally structured by artifact type instead of mirroring a full directory tree. That shape is practical for REST payloads and keeps the extraction logic deterministic. A single huge opaque evidence JSON object would work poorly for validation, streaming, provenance, and future partial re-processing; if payloads become large, prefer storing artifacts externally and sending references or a manifest.

## Library

Use `pkg/conformance` when another Go program needs to produce the report directly:

```go
package example

import (
	"encoding/json"

	"credimi-conformance-assessment/pkg/conformance"
)

func GenerateReport() (conformance.ReportResult, error) {
	return conformance.Generate(
		conformance.ReportInput{
			Fixture:        "EUDI-iss-ver",
			PipelineInput:  json.RawMessage(`{"name":"EUDI issuer verification"}`),
			PipelineOutput: json.RawMessage(`{"workflow_id":"wf","run_id":"run"}`),
			EvidenceOutput: json.RawMessage(`{
				"credential_well_knowns": [],
				"presentation_results": []
			}`),
		},
		conformance.ReportOptions{},
	)
}
```

The library only generates reports. A caller that runs inside Credimi or any workflow runtime should wrap `conformance.Generate` in its own integration code and pass the resulting `ReportResult` through its own output envelope.

When `ReportOptions.SourceDir` is empty, the library reads the source-of-truth files embedded in this module. Set `SourceDir` only when you intentionally want to override the bundled files with an external source package.

`ReportInput` separates pipeline data from conformance evidence:

- `pipeline_input`: Credimi pipeline request/workflow input.
- `pipeline_output`: Credimi pipeline execution result/workflow output.
- `evidence_input`: grouped artifact JSON used by CLI/API payloads.
- `evidence_output`: extracted evidence output with `credential_well_knowns` and `presentation_results`, matching the evidence structure produced by Credimi pipeline evidence extraction.

## CLI Usage

The repository exposes one CLI with subcommands:

```bash
go run . help
```

Generate from an input JSON file. With the default empty `OUT_DIR`, Markdown is written to stdout:

```bash
go run . assess --input-json ./assessment-input.json
```

If `TEMPORAL_DATA` is set in `.env`, the CLI can use it without `--input-json`:

```bash
go run . assess
```

Set `OUT_DIR` in `.env` to write Markdown files and print report metadata as JSON.

Legacy fixture-directory mode is still available for the checked-in sample data:

```bash
go run . assess \
  --fixtures-dir ./fixtures \
  --pipeline-dir ./out \
  --fixture EUDI-iss-ver
```

## REST API

The six checked-in fixture requests are available as copy/paste curl commands in [API-tests.md](API-tests.md).

Start the API server:

```bash
go run . api
```

Generate one assessment through the API. The curl examples read `API_PORT` from `.env`:

```bash
set -a; . ./.env; set +a; curl -s http://localhost:${API_PORT}/assessments \
  -H 'Content-Type: application/json' \
  --data @assessment-input.json
```

With the default empty `OUT_DIR`, the response includes the generated Markdown:

```json
{
  "reports": [
    {
      "fixture": "EUDI-iss-ver",
      "slug": "eudi-iss-ver",
      "passed_count": 12,
      "markdown": "# Credimi Conformance Assessment ..."
    }
  ]
}
```

If `OUT_DIR` is set in `.env`, the response returns file paths instead of embedding Markdown.

## Inputs

The generator expects:

- `SOURCE_DIR/credimi-flat-conformance-test-list-v1.1.md`
- `SOURCE_DIR/credimi-conformance-aggregation-taxonomy-v1.1.yaml`
- `pipeline_input` and `pipeline_output` JSON objects supplied by CLI JSON or REST body
- `evidence_input` JSON grouped by artifact type, or `evidence_output` JSON with extracted evidence results

Artifact groups are optional. For example, an input without credential-offer or presentation-request artifacts still produces a valid conservative report.

## Design note: Go logic vs taxonomy logic

Go code is intentionally limited to generic mechanics:

- parse the flat test-list table into the atomic row vocabulary;
- read pipeline input/output and evidence artifact JSON;
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
