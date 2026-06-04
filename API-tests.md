# API Test Curls

Start the API server first:

```bash
go run . api
```

Each command sources `.env` so `API_PORT` controls the target port.
The fixture-only requests below use the checked-in `fixtures/`, `out/`, and `source-of-truth/` directories on the API server.
Keep `OUT_DIR` empty when you want the API response to include Markdown inline in `.reports[0].markdown`.

## Extract Markdown From The JSON Response

`curl` alone cannot select and unescape a JSON string field. Pipe the response through `jq -r` to write only the Markdown body to a file:

```bash
set -a; . ./.env; set +a; curl -s "http://localhost:${API_PORT}/assessments" -H "Content-Type: application/json" --data-raw "{\"fixture\":\"EUDI-iss-ver\"}" | jq -r ".reports[0].markdown" > conformance-assessment-eudi-iss-ver.md
```

Use the same pattern with any fixture below by changing the `fixture` value and output filename.

## AgeVerification

```bash
set -a; . ./.env; set +a; curl -s "http://localhost:${API_PORT}/assessments" -H "Content-Type: application/json" --data-raw "{\"fixture\":\"AgeVerification\"}"
```

## EUDI-iss-ver

```bash
set -a; . ./.env; set +a; curl -s "http://localhost:${API_PORT}/assessments" -H "Content-Type: application/json" --data-raw "{\"fixture\":\"EUDI-iss-ver\"}"
```

## EUDI-iss2

```bash
set -a; . ./.env; set +a; curl -s "http://localhost:${API_PORT}/assessments" -H "Content-Type: application/json" --data-raw "{\"fixture\":\"EUDI-iss2\"}"
```

## eudiw-checks-5x

```bash
set -a; . ./.env; set +a; curl -s "http://localhost:${API_PORT}/assessments" -H "Content-Type: application/json" --data-raw "{\"fixture\":\"eudiw-checks-5x\"}"
```

## Multipaz

```bash
set -a; . ./.env; set +a; curl -s "http://localhost:${API_PORT}/assessments" -H "Content-Type: application/json" --data-raw "{\"fixture\":\"Multipaz\"}"
```

## Talao-iss-cred13

```bash
set -a; . ./.env; set +a; curl -s "http://localhost:${API_PORT}/assessments" -H "Content-Type: application/json" --data-raw "{\"fixture\":\"Talao-iss-cred13\"}"
```
