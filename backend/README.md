# Backend

Go API backed by PostgreSQL.

## Frontend API contract

See [the page-by-page request list](docs/api-request-list.md) for the five MVP
screens, request/response shapes, rating rules and demo flow. Open
[the Postman v3 collection](docs/postman/README.md) in Postman's Local View to
exercise the agreed routes as they are implemented.

Implemented: health (`/health` and `/api/health`), team creation/list/profile,
offers, owner decisions, and one-time +10 acceptance awards. Existing `/proposals`
routes are supported as aliases. Task creation, AI, cards and catalog (block 1)
are still pending in this checkout.

See [block 2 integration and request examples](docs/block2.md) for the exact
routes, demo identity headers, task schema boundary, and local test fixture.

## Local setup

Requirements: Go 1.23+ and Docker Desktop.

1. Start PostgreSQL and pgAdmin:

   ```powershell
   docker compose up -d
   ```

2. Set the local environment variables in PowerShell:

   ```powershell
   $env:PORT = "8080"
   $env:DATABASE_URL = "postgres://hackathon:hackathon@localhost:5433/gamified_tasks?sslmode=disable"
   ```

3. Start the API:

   ```powershell
   go run ./cmd/api
   ```

   Startup applies embedded SQL migrations once and seeds five demo teams.
   It does not create published demo tasks; use the SQL fixture in `docs/block2.md`
   until block 1 can create and publish them.

4. Verify the API and database connection:

   ```powershell
   Invoke-RestMethod http://localhost:8080/health
   ```

   Expected response:

   ```json
   {"database":"connected","status":"ok"}
   ```

pgAdmin is available at `http://localhost:5050`. Log in with
`admin@hackathon.local` / `admin`, then register a server using host `postgres`,
port `5432`, database `gamified_tasks`, and username/password
`hackathon` / `hackathon`.

The container maps PostgreSQL to host port `5433` to avoid conflicts with a
locally installed PostgreSQL server. Connections from pgAdmin use the internal
Docker port `5432`.

Configuration values are documented in `.env.example`. The application reads
environment variables directly and does not automatically load a `.env` file.

## Verification

```powershell
go test ./...
go vet ./...
go build ./cmd/api
```

The PostgreSQL integration suite is enabled explicitly:

```powershell
$env:TEST_DATABASE_URL = "postgres://hackathon:hackathon@localhost:5433/gamified_tasks?sslmode=disable"
go test -count=1 -v ./...
```

It creates and drops a uniquely named test schema, not the application's tables.
The database user needs permission to create schemas. Without `TEST_DATABASE_URL`,
integration tests are reported as skipped. The suite checks two accepted teams,
all rejected, ownership, unpublished tasks, concurrent retries, persistence across
connections, migration idempotence, and rollback after a late transaction error.
