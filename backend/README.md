# Qadam Backend

Go 1.23+, `net/http`, `database/sql`, PostgreSQL, OpenAI Responses API. No ORM.
The backend implements the task builder, confirmed readiness scores, catalog,
proposals, manual decisions and one verified milestone per accepted proposal.
The Vue frontend is integrated through the Vite proxy. See the root README for
macOS/Linux startup and `../frontend/AI-CONTRACT.md` for chat and voice payloads.

## Start locally

Run these commands from `backend` with Docker Desktop running:

```powershell
docker compose up -d postgres pgadmin
if (!(Test-Path .env)) { Copy-Item .env.example .env }
go run ./cmd/seed
go run ./cmd/api
```

The API listens on `http://127.0.0.1:8080`. Existing environment variables override
`.env`; commands automatically load `.env` from the current working directory.
Set a different `PORT` if 8080 is already occupied. Startup applies ordered,
transactional embedded migrations. Existing migration files are not rewritten.

```powershell
Invoke-RestMethod http://127.0.0.1:8080/api/health
```

Expected in fallback mode:

```json
{"data":{"status":"ok","database":"connected","aiMode":"fallback"}}
```

PostgreSQL: host `localhost`, port `5433`, database `gamified_tasks`, username
`hackathon`, password `hackathon`. These are local demo credentials only.
pgAdmin: `http://localhost:5050`, login `admin@example.com` / `admin`.
Register its database connection using host **postgres**, port **5432**, and the
same database/user/password. The API connects through host port 5433 instead.

## OpenAI secret and model

Do not paste your API key into chat, Postman, frontend code, commands or commits.
Use the masked local prompt:

```powershell
.\scripts\set-openai.ps1 -Model gpt-4.1-mini
go run ./cmd/check-ai
```

The script preserves other `.env` entries and sets `OPENAI_API_KEY`, `OPENAI_MODEL`
and `AI_MODE=live`. `.env` is Git-ignored but remains plaintext on your machine:
do not share the file. Model access depends on the hackathon project's key.
Use the exact model ID granted by the organizers if different.
`check-ai` makes one small billable analysis request, with at most one transient
retry, and prints only model, mode, question count and duration. It exits nonzero
if the request falls back. Restart the API after changing its configuration.

The adapter uses [Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs)
with `store=false`, bounded responses, source validation and a 15-second total
timeout. [GPT-4.1 mini](https://developers.openai.com/api/docs/models/gpt-4.1-mini)
supports the required structured-output workflow; this does not guarantee access
for your specific key. Set `AI_MODE=fallback` to work without API calls.
The internal offline analyzer can return `mode=fallback` for diagnostic tests;
the HTTP endpoint returns 503 instead of exposing prepared questions as AI output.
The prompt is in `internal/ai/prompt.txt`; malformed-output examples and parser
checks are in `internal/ai/ai_test.go`.

`POST /api/ai/chat` uses Responses API with server instructions and `store=false`.
It loads only a public task snapshot or the selected business actor's own draft.
`POST /api/ai/transcribe` accepts up to 15 MiB of WebM/MP4/Ogg/WAV audio,
checks the container signature and calls Audio Transcriptions with
`OPENAI_TRANSCRIBE_MODEL` (default `gpt-4o-mini-transcribe`). Audio is not retained.
Provider failures return bounded errors without exposing provider response bodies.

## Frontend contract and demo

- [Page-by-page routes and payloads](docs/api-request-list.md)
- [Postman v3 folder collection](docs/postman/README.md)
- [Teammate integration notes](docs/block2.md)
- [Product specification](../docs/SPEC.md)

Select a demo actor using `X-Demo-Actor: business:UUID` or `team:UUID`.
Seed business: `10000000-0000-4000-8000-000000000001`.
Seed teams: `00000000-0000-4000-8000-000000000001` through `...000005`.
`GET /api/teams` returns the IDs and profiles. IDs in the original JSON fixtures
map deterministically to UUIDs; seed tasks start `20000000`, drafts `30000000`,
proposals `40000000`. Existing seed records and subsequent edits are preserved.
The five team profiles from the original migration are also preserved.

Demo: create a draft, request questions, save answers/card, confirm four 10-point
fields, publish at 40, submit/accept a proposal, submit/confirm its milestone for
10 points, then fill and confirm all fields except feedbackFormat to reach 95.
The Postman folders follow that flow. Rejection uses a separate proposal.

## Verification

```powershell
go test ./...
$env:TEST_DATABASE_URL = 'postgres://hackathon:hackathon@localhost:5433/gamified_tasks?sslmode=disable'
go test ./... -count=1 -v -timeout=60s
go vet ./...
go build ./...
```

Integration tests create and drop only randomly named test schemas. They never
reset application data. Without `TEST_DATABASE_URL`, they explicitly skip.
The DB user needs CREATE SCHEMA permission. Tests cover the 40-to-95 flow, seed
idempotence, filters, revision conflicts, private/public snapshots, multiple
accepted teams, milestone authorization, concurrent one-time awards, transaction
rollback, DB reopening, source checks, invalid AI output, retries and timeout.
Provider tests use a local HTTP fake; a real key is checked separately by `check-ai`.

Recorded results and limitations: [verification report](docs/verification.md).

## Deliberate limits

- Demo actors are not authentication. The API binds to loopback; do not expose it
  publicly. `DEMO_MODE=false` disables application routes until real auth exists.
- In development, proxy `/api` from Vite to `http://127.0.0.1:8080`; no wildcard CORS.
- No XP/streak dashboard, paid ranking, recommendations or automatic team selection.
- Existing `/offers` and accept/reject aliases remain for compatibility. New clients
  should use the documented `/proposals` routes and `{data:...}` envelopes.
- Historical acceptance awards remain stored but do not contribute to progress
  points. Only `progress_awards` from confirmed milestones count.
- The seed command is additive and idempotent. There is no destructive public reset.
