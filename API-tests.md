# API Test Curls

Start the API server first:

```bash
go run . api
```

Each command sources `.env` so `API_PORT` controls the target port.

## Minimal Inline Report

```bash
set -a; . ./.env; set +a; curl -s "http://localhost:${API_PORT}/assessments" -H 'Content-Type: application/json' --data-raw '{"fixture":"EUDI-iss-ver","pipeline_input":{"name":"EUDI issuer verification"},"pipeline_output":{"workflow-id":"example-workflow-id","workflow-run-id":"example-run-id","output":"COMPLETED"},"evidence":{"credential_offers":[{"step_id":"credential-step","credential_id":"tenant/credential","credential_offer":{"credential_issuer":"https://issuer.example","credential_configuration_ids":["pid_sd_jwt"],"grants":{"urn:ietf:params:oauth:grant-type:pre-authorized_code":{}}}}],"credential_well_knowns":[{"step_id":"credential-step","credential_id":"tenant/credential","well_known":{"credential_endpoint":"https://issuer.example/credential","credential_configurations_supported":{"pid_sd_jwt":{"format":"vc+sd-jwt","proof_types_supported":{"jwt":{"proof_signing_alg_values_supported":["ES256"]}}}}}}],"presentation_results":[]}}'
```

## Conservative Report Without Evidence

```bash
set -a; . ./.env; set +a; curl -s "http://localhost:${API_PORT}/assessments" -H 'Content-Type: application/json' --data-raw '{"fixture":"AgeVerification","pipeline_input":{"name":"Age verification"},"pipeline_output":{"workflow-id":"example-workflow-id","workflow-run-id":"example-run-id","output":"COMPLETED"}}'
```
