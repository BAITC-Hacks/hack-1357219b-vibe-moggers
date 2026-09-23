# Frontend API Contract

Base: `http://127.0.0.1:8080/api`. JSON request bodies. Success: `{"data":...}`.
Lists: `{"data":{"items":[],"total":0,"limit":20,"offset":0}}`.
Errors: `{"error":{"code":"...","message":"...","fields":{}}}`; `fields` is optional.
Pagination accepts `limit=1..100`, `offset>=0`. Identifiers are UUIDs; dates are UTC.
Protected requests use `X-Demo-Actor: business:UUID` or `team:UUID`. These are local
demo roles, not production authentication. Never send the OpenAI key to these routes.

## 1. Task Builder: /tasks/new and /tasks/:id/edit

| Request | Body/purpose |
| --- | --- |
| `POST /tasks` | `{draft,title,industry}`; business actor; returns private Task, revision=1 |
| `GET /tasks/:id` | Owner gets editable Task; others get only the published snapshot, or 404 |
| `POST /ai/analyze` | `{stage:"clarify",sources:[{id:"draft",text:"..."}],currentFields:{}}` |
| `PUT /tasks/:id/card` | `{revision,title?,industry?,answers?,fields?}` saves answers/fields, returns Task and preview |
| `POST /ai/analyze` | Same schema with `stage:"assemble"`, saved answers as additional sources |
| `POST /tasks/:id/confirm` | `{revision,fieldKeys:["context",...],reviewed:true}`; updates snapshot and revision |
| `POST /tasks/:id/publish` | `{revision}`; requires current reviewed snapshot, title and industry; no minimum score |

`draft` is required, max 20000 characters. Title max 200, industry max 100.
Store the latest returned `revision` after every mutation. A stale one returns 409.
Publish is idempotent at the current confirmed revision and does not increment it.
No separate questions, answers, generate-card or rating endpoints are necessary.

`answers` replaces the answer list when supplied (max 50):

```json
[{"id":"answer-1","questionId":"q1","text":"Sales are available as CSV."}]
```

Answer IDs must be unique and cannot be `draft`. Empty answer text means skipped.
Omitted title/industry/answers stay unchanged. `fields` patches only supplied keys;
use `value:null` to clear a field. Each field is:

```json
{"value":"CSV","confirmed":false,"source":{"id":"answer-1","quote":"CSV"}}
```

Manual fields use `source:null`. Client-supplied confirmation is never trusted.
AI suggestions are `{field,value,sourceId}`: map them to fields with
`source:{id:sourceId,quote:value}`, show them for review, then save through PUT.
AI never writes the database or confirms facts. A provided source must contain
the field value after whitespace normalization. Changing a sourced answer removes
stale source confirmation; editing successMetric resets target/method confirmation.

Field keys: `context`, `need`, `dataSource`, `dataFormat`, `dataAccess`, `deliverable`,
`deliveryFormat`, `successMetric`, `successTarget`, `acceptanceMethod`, `deadline`,
`constraints`, `users`, `usageScenario`, `contact`, `feedbackFormat`.

Task returns `workingFields`, `answers`, `draft`, `revision`, `confirmedRevision`,
`confirmedSnapshot`, `preview`, `status`, `workStatus`, and timestamps.
`preview` and the snapshot include `total`, `readiness`, seven `scoreBreakdown`
groups, `missingFields`, and `nextActions` with `possibleGain`.
Confirm accepts an empty `fieldKeys:[]` for an explicitly reviewed empty card.
Unconfirmed values never appear in the public snapshot. Draft edits leave the
last public snapshot unchanged; confirming a published task updates it atomically.

AI returns `questions`, `fieldSuggestions`, `warnings`, `mode`, `durationMs`.
Display fallback clearly and keep manual editing available. `clarify` returns
3-5 questions. Sources are untrusted data; the server checks source attribution.

## 2. Catalog: /catalog

| Request | Purpose |
| --- | --- |
| `GET /tasks?industry=retail&readiness=priority&limit=20&offset=0` | Published cards, optional filters |

Default order: score descending, first publication time ascending, UUID ascending.
Readiness: `draft` 0-39, `working` 40-69, `ready` 70-89, `priority` 90-100.
Here `draft` is a readiness band, not publication status. Published zero-score
cards remain visible and accept proposals. Remove query filters to reset them.
Public items contain `id,title,industry,fields,total,readiness,scoreBreakdown,
missingFields,nextActions,timestamp,publishedAt,workStatus`, not private answers.

## 3. Task Details: /tasks/:id

| Request | Purpose |
| --- | --- |
| `GET /tasks/:id` | Public card; omit business-owner header when requesting the public view |
| `GET /teams` | Seed profiles for the demo role picker |
| `POST /tasks/:id/proposals` | Team submits `{idea,plan:["step"],durationDays:7,prototypeUrl:"https://example.org"}` |
| `GET /tasks/:id/proposals` | Team sees only its own; owner sees all |
| `POST /proposals/:id/milestone` | Accepted team submits `{resultText,evidenceUrl:"https://example.org/result"}` |

Idea max 10000 characters; plan 1-30 nonempty steps, 10000 joined characters;
durationDays is an integer 1-3650. Prototype URL is optional (empty string allowed).
Milestone result max 10000 characters, evidence URL required. URLs must be absolute
HTTP(S), without credentials. The server never fetches submitted links.
Milestone submission returns 200, including an identical retry; different content
for an existing milestone returns 409. One milestone per accepted proposal.

Proposal fields: `id,taskId,teamId,teamName,idea,plan,durationDays,prototypeUrl,
status,createdAt,decidedAt,milestone`. Status: pending/accepted/rejected.
List responses include the milestone and awarded points, or `milestone:null`.
Legacy stored offers can have `durationDays:null`; their original timeline remains
available via the compatibility `/offers` API.

## 4. Business Workspace: /business/tasks/:id

| Request | Purpose |
| --- | --- |
| `GET /business/tasks` | Current owner's drafts and published tasks |
| `GET /tasks/:id` | Private working card and confirmed snapshot |
| `GET /tasks/:id/proposals?status=pending` | Review proposals; optional status and pagination |
| `POST /proposals/:id/decision` | `{decision:"accepted"}` or `{decision:"rejected"}` |
| `POST /milestones/:id/confirm` | `{confirmed:true}` verifies result and grants 10 once |
| `GET /teams/:id` | Profile and total verified-progress points |

Several proposals may be accepted, or all rejected. Other pending proposals do
not change automatically. Repeating the same decision is safe; reversing a final
decision returns 409. Accepting a proposal awards **no points**. Confirmation of
its milestone grants exactly 10, including concurrent retries. Points derive from
the unique milestone award records, not the old acceptance-award table.

## Operational and compatibility routes

`GET /health` (under `/api`) checks DB connectivity and reports configured AI mode.
The root `/health` is an alias. It does not make a paid AI request.
`POST /teams` with `{name}` remains available for local demo profiles.

Old `/tasks/:id/offers` GET/POST, `PATCH /offers/:id/status` and
`POST /proposals/:id/accept|reject` remain compatibility routes. Legacy create uses
`solution_idea,plan` (string), `timeline,prototype_link,team_id?`; optional team_id
must equal the selected actor. Single legacy responses retain `offer`/`proposal`
wrappers; list responses use the standard data envelope. Final-decision and
milestone-only points rules apply to every route. New frontend code should use
the primary routes above.

HTTP codes: 400 invalid JSON/query, 401 missing/invalid demo actor, 403 forbidden,
404 unavailable/missing resource, 409 stale revision/state conflict, 422 invalid
fields, 500 internal failure, 503 DB unavailable or demo mode disabled.
