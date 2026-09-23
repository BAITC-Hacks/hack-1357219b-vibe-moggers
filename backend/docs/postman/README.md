# Hackathon API collection

This is a Postman v3 collection containing 18 ready-to-send request definitions
for the agreed backend contract, including both public and editor task reads.

## Open in Postman 12

In your existing workspace, use Local View and look for **Hackathon API**.
The collection is registered in `backend/docs/.postman/resources.yaml`.

If you need to reopen the folder, use **Files > Open folder** and select
`backend/docs`. Open `postman/collections/Hackathon API` in the file tree.
Postman v3 uses a folder of YAML files, not a single JSON import.

Base URL: `http://localhost:8080`. Change the collection's `base_url` variable
if your Go server uses another port.

## Send requests

- Start with **01 - Health > Health**. PostgreSQL and the Go API must be running.
- After the planned endpoints are implemented, run folders **02** through **04**
  in order. Create draft captures `task_id`; questions capture question IDs;
  saving changes updates `revision`; List teams captures `team_id`;
  Submit proposal captures `proposal_id`.
- Use **05 - Proposal review > Accept proposal** or **Reject proposal** for the
  same proposal. Running both leaves it rejected.
- The card and answers use synthetic demo data. Adjust answers to match the
  questions actually returned by AI.
- Browse catalog already includes rating sorting. To filter, add
  `topic=retail` and `readiness=priority` to its query parameters.

Only `GET /health` works today. Creating this collection does not implement the
planned routes, which currently return 404. Health returns 503 if its database
connection becomes unavailable.

For additional validation, reuse the requests with a title-only card to check
low-readiness publication, or submit proposals from different teams and accept
each in turn to check that multiple teams can be selected.

The shared API contract is [api-request-list.md](../api-request-list.md).
