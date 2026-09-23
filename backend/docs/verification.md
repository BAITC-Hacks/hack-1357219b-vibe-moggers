# Backend Verification

Verified locally on 2026-09-23 using Go, Windows and PostgreSQL 16 in Docker.

| Check | Result |
| --- | --- |
| `go test ./... -count=1 -timeout=60s` with TEST_DATABASE_URL | PASS, including real PostgreSQL integration suites |
| `go test ./... -count=3 -shuffle=on -cover -timeout=60s` | PASS on three randomized runs |
| `go vet ./...` | PASS |
| `go build ./...` and API executable build | PASS |
| Postman YAML smoke runner | PASS: 24 HTTP requests and 87 response assertions |
| Actual API restart | Task retained score 95; team retained 10 progress points |
| Seed importer, invoked twice in integration tests | Five drafts/cards/teams/proposals, no duplicates or overwritten edits |
| Concurrent milestone confirmations | 16 requests, one award, 10 points |
| Injected database failure | Milestone status and award rolled back together |
| OpenAI local HTTP fake | Structured request, source validation, malformed output, retry and timeout tests passed |
| Masked key setup script | PowerShell syntax validated; `.env` is Git-ignored |
| pgAdmin startup configuration | Replaced the rejected `.local` login with `admin@example.com`; login page returned HTTP 200 |
| `git diff --check` | PASS |

The collection was executed by `scripts/check-postman.cjs`, not through the
Postman UI. The runner reads the real YAML files and executes their current
response-assertion subset. The application does not require Node or Python.

Restart proof from the local demonstration database:

```json
{
  "health": {"aiMode":"fallback","database":"connected","status":"ok"},
  "taskId":"b29a0a65-45fb-4903-a17b-8f4b78ffa1a4",
  "scoreAfterAPIRestart":95,
  "teamPointsAfterAPIRestart":10
}
```

The smoke-created task and proposal history remain as demo data. The temporary
API process was stopped after verification; PostgreSQL and pgAdmin remain up.

## Not verified

- A real OpenAI request: no local API key was set. `go run ./cmd/check-ai` correctly
  exited with instructions to configure live mode. Model entitlement and quota
  depend on the hackathon key. Mock tests do not establish real provider access.
- `go test -race ./...`: unavailable with this Windows toolchain's CGO disabled
  and no C compiler. Concurrent database tests did run and pass, but they are not
  a substitute for the Go race detector.
- Frontend/browser integration: no frontend application exists in this checkout.

No commit or push was performed. Review and branch selection remain with the user.
