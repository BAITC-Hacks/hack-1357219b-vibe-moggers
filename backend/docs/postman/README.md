# Postman Collection

This repository keeps the existing Postman v3 folder/YAML format. Open
`backend/docs` in Postman's Local View; `.postman/resources.yaml` registers
`postman/collections/Hackathon API`. These are not JSON import files.

Start PostgreSQL, run `go run ./cmd/seed`, and start `go run ./cmd/api` from backend.
The `base_url` collection variable defaults to `http://127.0.0.1:8080`.
No OpenAI key belongs in Postman; it is read only by the Go process.

Run folders in order:

1. Health.
2. Business tasks and team profiles; capture the demo team UUIDs.
3. Builder: create, ask, save, assemble, edit, confirm and publish a 40-point card.
4. Catalog/details and a team proposal.
5. Review: accept the first proposal, create and reject a separate second proposal.
6. Progress: submit a result, confirm it and inspect the team's 10 points.
7. Improve the card to 95, reconfirm and inspect its catalog position.

Response scripts save task/proposal/milestone IDs and revisions automatically.
Requests use deterministic manual card fields so both fallback and live AI runs
can complete. In a real UI, map and review returned AI suggestions before saving.
Final proposal decisions cannot be reversed. Repeated milestone confirmation
does not award additional points. Repeating the entire collection creates another
task and another milestone award; compare the per-milestone points, not a global
team total of exactly 10 after multiple runs.

See [the page-by-page contract](../api-request-list.md) for payloads and errors.

## Optional collection smoke test

With Node.js and Python + PyYAML installed, `node scripts/check-postman.cjs` from
backend sends these YAML requests to the running local API and executes the
assertion subset used in their response scripts. It does not automate Postman.
This creates a demonstration task/proposals/milestone, so run it only locally.
Set `BASE_URL` to use a different local port; remote hosts are rejected.
The app itself does not depend on Node or Python.
