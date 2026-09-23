# Backend

Go API backed by PostgreSQL.

## Frontend API contract

See [the page-by-page request list](docs/api-request-list.md) for the five MVP
screens, request/response shapes, rating rules and demo flow. Open
[the Postman v3 collection](docs/postman/README.md) in Postman's Local View to
exercise the agreed routes as they are implemented. Only `GET /health` is implemented
today; all `/api/*` requests are planned and currently return 404.

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
