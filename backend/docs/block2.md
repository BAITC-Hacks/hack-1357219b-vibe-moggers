# Integration Notes

The Task Builder now uses the same PostgreSQL `tasks` table as the teammate's
teams/offers modules. Existing migrations 001 and 002 are unchanged; migration
003 adds builder fields, structured proposal data, milestones and progress awards.

`tasks.Access` remains the transaction boundary. Writers lock task, then proposal,
then milestone when relevant. Task edits and confirmations lock the same task row
and reject stale revisions. SQL parameters remain bound, never string-built from
user values. New backend wiring lives in `internal/app/app.go`.

Canonical proposal requests use `/api/tasks/:id/proposals` and
`POST /api/proposals/:id/decision` with SPEC field names and `{data:...}` responses.
The older `/offers` endpoints and accept/reject aliases remain for compatibility.
Refer to [the API contract](api-request-list.md) for their exact differences.

The former `awards.Grant` acceptance hook is removed. Accept/reject updates work
status but never awards points. Final decisions cannot be reversed. An accepted
team submits one milestone; the owner confirms it, creating one unique
`progress_awards` row transactionally with the milestone status. Team profiles
calculate points from those rows. The legacy `point_awards` table and `teams.points`
column are retained without deleting or rewriting historical data, but neither
is used for current progress totals.

Use `go run ./cmd/seed` instead of inserting partial task fixtures manually.
Existing stub task records are preserved; records without a confirmed snapshot
are excluded from the public catalog until reviewed through the builder.

Tests retain coverage for multi-team selection, rejecting all, ownership,
unpublished tasks, retry concurrency and rollback. `internal/app/integration_test.go`
adds task creation, real scoring, snapshot isolation, milestone confirmation and
durability. See [backend setup](../README.md) for test commands.
