# MVP pages and API contract

Status: teams, proposals/offers, decisions and acceptance awards are implemented,
along with `GET /health` and `/api/health`. Task builder, AI and catalog routes
remain planned. See [block 2](block2.md) for implemented canonical `/offers`
routes, +10 awards, request examples, and the shared task schema boundary.
The Postman `/proposals` routes below remain compatible aliases.

## Pages and requests

1. **Shared catalog** - frontend `/catalog`
   - `GET /api/catalog?topic=retail&readiness=ready&sort=rating_desc&limit=20&offset=0`
   - Show title, topic, confirmed rating, readiness, and a short need summary.
   - Empty filters mean all published tasks. Include published 0-39 tasks.
2. **Task details + proposal form** - frontend `/tasks/:taskId`
   - `GET /api/tasks/{task_id}` - published, confirmed card and rating.
   - `GET /api/teams` - seeded team selector; load once and cache.
   - `POST /api/tasks/{task_id}/proposals` - submit from an inline form/modal.
   - No separate application wizard or team profile page is needed.
3. **Business task list** - frontend `/business/tasks`
   - `GET /api/tasks?status=draft&limit=20&offset=0`
   - Omit status for all business demo tasks; allow draft/confirmed/published.
   - Show status, title or raw description, rating, proposal_count, and links
     to edit/review. Creating a task opens the builder.
4. **Task builder** - frontend `/business/tasks/new` or `/business/tasks/:taskId/edit`
   - `POST /api/tasks/draft` - save the initial description and topic.
   - `GET /api/tasks/{task_id}?view=editor` - resume saved work.
   - `POST /api/tasks/{task_id}/questions` - generate at least 3 questions.
   - `PUT /api/tasks/{task_id}/answers` - save answers by question ID.
   - `POST /api/tasks/{task_id}/generate-card` - create an editable AI draft.
   - `PUT /api/tasks/{task_id}` - save the entire editable card.
   - `GET /api/tasks/{task_id}/rating` - official rating plus draft preview.
   - `POST /api/tasks/{task_id}/confirm` - human confirms the current revision.
   - `POST /api/tasks/{task_id}/publish` - publish a confirmed card.
   - These are steps/tabs on one page. Questions, rating, and missing fields
     are panels, not separate pages.
5. **Business proposal review** - frontend `/business/tasks/:taskId/proposals`
   - `GET /api/tasks/{task_id}?view=editor` - task heading and context.
   - `GET /api/tasks/{task_id}/proposals?status=submitted&limit=20&offset=0`
   - `POST /api/proposals/{proposal_id}/accept`
   - `POST /api/proposals/{proposal_id}/reject`
   - Compare team, idea, plan, timeline, prototype link and decision inline.

Infrastructure: `GET /health`. No UI page required.

No XP dashboard, streaks, leaderboard, chat, notifications, calendar, login
flow, or full project tracker. The brief mentions milestone points in its
extended scenario, but explicitly says a proposal and business decision suffice
for the demo. Keep milestone tracking deferred, not a mandatory screen.

## Shared conventions

- API origin: `http://localhost:8080`; JSON request and response bodies.
- IDs are UUID strings; timestamps are UTC RFC 3339 strings.
- Task path variable `{task_id}` means the same ID through the whole lifecycle.
- Single task replies use `{"task": {...}}`; proposal replies use
  `{"proposal": {...}}`. Lists use
  `{"items": [], "total": 0, "limit": 20, "offset": 0}`.
- List defaults: limit 20, offset 0; limit 1-100, offset >= 0.
  Invalid query enum/number values return 400.
- Catalog sort: `rating_desc` (default) or `newest`. Rating ties use
  published_at descending, then id ascending for stable paging.
- Topic is a trimmed category slug, e.g. `retail`, `education`,
  `logistics`. It is the draft's industry/category. Topic filtering is exact.
- Readiness: `draft`, `workable`, `ready`, `priority`.
  Readiness `draft` is a score band; task status `draft` is unpublished.
- Demo identity for block 2: `X-Demo-Actor: team:<UUID>` on submissions,
  `X-Demo-Actor: business:<UUID>` on review/decisions. The business UUID must
  match `tasks.owner_id`. Optional `team_id` must match the header's team.
  These are local demo selectors, not authenticated production identities.
  Five demo teams are seeded once by the startup migration.
- OpenAI credentials stay on the backend. Never send them from the frontend or
  put them in Postman. AI responses include `ai: {"provider":"openai",
  "fallback_used":false}`; a stub uses provider `stub`, fallback_used true.
- Server should allow the agreed frontend origin with CORS when implementing
  these routes. CORS is still pending; block 2 domain routes are implemented.

## Lifecycle and confirmation

A draft starts with an empty card, score 0, revision 1, status `draft`.
Saving answers, generating questions/card, or editing increments revision.
Returning already saved questions does not increment it. Editing an unpublished
confirmed task returns its status to draft while preserving the prior confirmed
snapshot. Confirm and publish do not change the content revision.
Mutations of an existing task include its last `revision`; stale values
return 409. Confirm/publish also require revision and return the latest task.

AI generation and manual editing never silently confirm facts. The editor
shows a preview score, labelled "pending confirmation". Confirm copies the
current card into the confirmed snapshot, recalculates the official rating,
and records confirmed_revision. For an unpublished task this sets status
`confirmed`. Publish requires confirmed_revision == revision and changes
status to `published`. There is no minimum score for publication or proposals.

For subsequent edits, retain the last confirmed snapshot and official rating.
The editor sees the new working card and preview; public readers see only the
confirmed snapshot. On the next explicit confirmation, the public card and its
rating update together. An already published task stays published. This avoids
showing unconfirmed AI changes in the catalog.

Repeated confirm/publish on the same current revision are harmless 200 replies.
Question generation is an explicit initial action: once saved questions exist,
return them without regenerating IDs or losing answers. Regeneration of
questions is deferred. Card generation is allowed to replace the working card
only after a deliberate user action; the UI should warn if manual edits exist.

## Request bodies and responses

### Task creation and retrieval

`POST /api/tasks/draft` -> 201
```json
{"raw_description":"Customers cannot easily compare our services.","topic":"retail"}
```
raw_description: 1-10000 trimmed characters; topic: 1-80 characters.
Returns a task with empty strings for all card fields, empty questions,
rating score 0, preview score 0, and revision 1.

`GET /api/tasks/{task_id}?view=editor` -> 200
returns the workspace task shape below, including saved questions/answers.
The default GET is public and returns 404 for unpublished tasks. It uses the
confirmed snapshot and omits raw_description, working card and questions.
Task lists return summary objects with id, title, raw_description (business
list only), topic, status, rating, proposal_count, created_at, published_at.

Workspace task response shape:
```json
{
  "task": {
    "id": "UUID",
    "raw_description": "Customers cannot easily compare our services.",
    "topic": "retail",
    "status": "draft",
    "revision": 1,
    "confirmed_revision": null,
    "card": {
      "title": "", "context": "", "need": "", "users": "", "data": "",
      "constraints": "", "expected_result": "", "success_criteria": "",
      "contact": "", "interaction_format": ""
    },
    "questions": [],
    "rating": {
      "score": 0, "level": "draft", "breakdown": [], "missing_fields": []
    },
    "rating_preview": {
      "score": 0, "level": "draft", "breakdown": [], "missing_fields": []
    },
    "proposal_count": 0,
    "created_at": "2026-09-23T08:00:00Z",
    "updated_at": "2026-09-23T08:00:00Z",
    "published_at": null
  }
}
```
The empty breakdown/missing_fields arrays above abbreviate the response;
actual ratings always contain all seven breakdown categories and every
missing card field, including title. `rating` always describes confirmed
content; `rating_preview` describes the working card.

### Questions, answers and card

`POST /api/tasks/{task_id}/questions` -> 200
```json
{"revision":1}
```
Returns task wrapper plus ai metadata; task.questions has at least 3 objects:
`{"id":"UUID","field":"users","question":"Who will use this?","answer":""}`.
field is a card field name; IDs are persistent, not array indices.

`PUT /api/tasks/{task_id}/answers` -> 200
```json
{"revision":2,"answers":[{"question_id":"UUID","answer":"Our retail customers."}]}
```
Upserts supplied answers and preserves omitted ones; unknown or duplicate
question IDs return 422. At least one answer entry, at most 10000 characters
per answer; an empty answer clears it. Returns task wrapper.

`POST /api/tasks/{task_id}/generate-card` -> 200
```json
{"revision":3}
```
Requires saved questions and at least one nonempty answer, else 409. Uses only
the draft and saved answers; unknown facts remain empty strings. Returns
task wrapper plus ai metadata. Invalid AI output must be validated and handled
with a labelled local fallback, or a 502 if no valid fallback can be produced;
timeouts return 504. Never persist a partial invalid card.

`PUT /api/tasks/{task_id}` -> 200
```json
{
  "revision":4,
  "topic":"retail",
  "card":{
    "title":"Service comparison tool",
    "context":"Customers ask staff to explain service differences.",
    "need":"Let customers compare services before contacting staff.",
    "users":"Retail customers",
    "data":"An anonymized service catalog with names and prices",
    "constraints":"Web prototype in one week; no customer personal data",
    "expected_result":"A working comparison prototype",
    "success_criteria":"Five test users can compare three services without help",
    "contact":"demo-business@example.com",
    "interaction_format":"Weekly video review and written feedback within two days"
  }
}
```
All 10 card keys are required strings; empty strings are valid unknown values.
Title maximum 200 characters; other fields maximum 10000.
Returns task wrapper including rating_preview. No additional recalculate
request is necessary after a save.

### Rating and publishing

`GET /api/tasks/{task_id}/rating` -> 200
```json
{
  "task_id":"UUID",
  "revision":5,
  "confirmed_revision":null,
  "rating":{
    "score":0,"level":"draft",
    "breakdown":[
      {"key":"context_need","earned":0,"max":20},
      {"key":"data","earned":0,"max":20},
      {"key":"expected_result","earned":0,"max":15},
      {"key":"success_criteria","earned":0,"max":15},
      {"key":"constraints","earned":0,"max":10},
      {"key":"users","earned":0,"max":10},
      {"key":"business_connection","earned":0,"max":10}
    ],
    "missing_fields":["title","context","need","users","data","constraints","expected_result","success_criteria","contact","interaction_format"]
  },
  "rating_preview":{
    "score":100,"level":"priority",
    "breakdown":[
      {"key":"context_need","earned":20,"max":20},
      {"key":"data","earned":20,"max":20},
      {"key":"expected_result","earned":15,"max":15},
      {"key":"success_criteria","earned":15,"max":15},
      {"key":"constraints","earned":10,"max":10},
      {"key":"users","earned":10,"max":10},
      {"key":"business_connection","earned":10,"max":10}
    ],
    "missing_fields":[]
  }
}
```
Deterministic MVP scoring: context 10 + need 10; data 20; expected_result 15;
success_criteria 15; constraints 10; users 10; contact 5 + interaction_format 5.
Each field earns its points when trimmed content is nonempty and confirmed.
This is a transparent completeness rubric, not a claim that arbitrary filled
text is high quality. Human confirmation is required for official points.
Title is required to confirm/publish but earns no points.
Missing fields belong to the corresponding official or preview card.

Bands: 0-39 draft, 40-69 workable, 70-89 ready, 90-100 priority.

`POST /api/tasks/{task_id}/confirm` -> 200
```json
{"revision":5}
```
Requires a nonempty title; other fields may remain unknown. Returns task
wrapper with confirmed_revision equal to revision and updated official rating.

`POST /api/tasks/{task_id}/publish` -> 200
```json
{"revision":5}
```
Requires confirmation of the current revision; otherwise 409. Returns task.

### Teams and proposals

`GET /api/teams` -> 200 list of the five demo teams:
`{"id":"UUID","name":"Team Alpha","interests":["retail"],"skills":["backend"],
"technologies":["Go","PostgreSQL"],"points":0,"created_at":"2026-09-23T08:00:00Z"}`.
Same list envelope/pagination defaults. `POST /api/teams` accepts `{"name":"..."}`
and creates a zero-point profile. `GET /api/teams/{team_id}` returns `{"team":{...}}`
including its current points.

`POST /api/tasks/{task_id}/proposals` -> 201
```json
{
  "team_id":"UUID",
  "solution_idea":"A searchable comparison table for the service catalog",
  "plan":"Import sample services, build comparison view, test with five users",
  "timeline":"One week",
  "prototype_link":"https://example.com/prototype"
}
```
Task must be published (409 otherwise); team must exist (404 otherwise).
solution_idea and plan: 1-10000 trimmed characters; timeline: 1-500.
prototype_link: empty string or absolute HTTP(S) URL, maximum 2048 characters.
A prototype is optional at first submission, but seeded examples include links.
Unlimited proposals, including multiple proposals from the same team; disable
the submit button while a request is pending to reduce accidental duplicates.

Proposal shape:
```json
{
  "proposal":{
    "id":"UUID","task_id":"UUID","team_id":"UUID","team_name":"Team Alpha",
    "solution_idea":"A comparison table","plan":"Build and test a prototype",
    "timeline":"One week","prototype_link":"https://example.com/prototype",
    "status":"submitted","created_at":"2026-09-23T08:00:00Z","decided_at":null
  }
}
```

`GET /api/tasks/{task_id}/proposals` -> 200 list of proposal objects.
Optional status filter: submitted/accepted/rejected. Default all; newest first.

`POST /api/proposals/{proposal_id}/accept` -> 200 proposal wrapper.
`POST /api/proposals/{proposal_id}/reject` -> 200 proposal wrapper.
No request body. Only human actions call these endpoints. Each affects just
that proposal; accepting does not reject others or close the task. Repeating a
decision is harmless; switching accepted/rejected is allowed to correct a
mistake. Pending/no selection requires no action. No automatic team ranking.

The first acceptance awards +10 to the team, once per offer. Status, award,
team points, and task `work_status` are committed in one transaction. Repeated
or concurrent acceptance does not award again. Changing accepted to rejected
retains the first-acceptance award; accepting again adds nothing. A task with
any accepted offer has `work_status=in_progress`, otherwise `open`; its
publication status stays `published` and new offers remain allowed.

## Errors

```json
{
  "error":{
    "code":"validation_error",
    "message":"Some fields are invalid.",
    "fields":{"raw_description":"Required"}
  }
}
```
400 malformed JSON/query; 401 missing/invalid demo actor; 403 wrong role/owner;
404 unknown ID/resource or unpublished public task;
409 invalid lifecycle or stale revision; 422 valid JSON with invalid field
values; 502 invalid upstream AI response; 504 AI timeout; 500 unexpected error.
fields may be omitted outside validation. Do not return database errors or keys.
Health keeps its existing separate JSON shape and uses 503 when DB is down.

## Postman and frontend handoff

Open `backend/docs` in Postman's Local View. The **Hackathon API** collection
under `postman/collections` contains 18 core requests. See
[the Postman setup guide](postman/README.md) for opening and sending them.
The collection uses Postman v3 YAML and is registered in
`.postman/resources.yaml`; no JSON import or separate environment is needed.
`base_url` defaults to `http://localhost:8080`.
The numbered folder order is the demo execution order.

Collection variables task_id, revision, question_1_id through question_3_id,
team_id and proposal_id are captured from responses. List teams also captures
second_team_id for manually testing proposals from another team.
Run the numbered happy-path folders in order after domain routes are built.
Question demo answers are generic test fixtures; edit them to match the AI's
actual questions when showing a realistic demo. The manually edited example
card is synthetic, human-provided data, not an AI claim.

Each endpoint has status assertions; captured IDs are asserted before storage.
Rating checks validate weights, sums and boundaries. The demo verifies an empty
draft at 0 and a confirmed completed card at 100, catalog rating order and
proposal submission. Accept and reject target the same captured proposal;
choose one, or run both to test changing its decision. For separate decisions,
submit another proposal and decide on the newly captured proposal_id.
Additional manual cases: publish a title-only card and submit a proposal;
accept proposals from multiple teams; submit an empty draft; and try publishing
without confirmation. These are not separate automated collection requests.
Rerunning creates fresh tasks and proposals; it does not clean up existing data.

Health, teams and proposal review/submission can run against this implementation.
Before block 1 is available, insert the published task fixture in `block2.md`
and set `task_id` and `business_id` in Postman. The full builder/catalog flow
still requires block 1. Health is 200 when connected or 503 if DB is unavailable.

Five teams are seeded; seeded drafts/cards/proposals remain a separate milestone.
List pages must support empty/loading/error states before seeding.

Suggested demo: create weak draft -> get questions -> answer -> generate card ->
manually improve card -> observe preview -> confirm -> publish -> browse catalog
-> team submits proposal -> business accepts one proposal and rejects another.
