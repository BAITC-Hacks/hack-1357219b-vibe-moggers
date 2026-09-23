# Backend

Go API backed by PostgreSQL.

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
