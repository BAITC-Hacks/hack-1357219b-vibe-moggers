package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"vibe-moggers/backend/internal/httpapi"
	"vibe-moggers/backend/internal/rating"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

const columns = `id, owner_id, title, industry, draft, answers, working_fields, confirmed_snapshot,
 revision, confirmed_revision, published_at, created_at, status, work_status`

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Task, error) {
	t := Task{}
	var answers, fields, snapshot []byte
	err := row.Scan(&t.ID, &t.OwnerID, &t.Title, &t.Industry, &t.Draft, &answers, &fields, &snapshot, &t.Revision, &t.ConfirmedRevision, &t.PublishedAt, &t.CreatedAt, &t.Status, &t.WorkStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return t, httpapi.Problem(404, "task_not_found", "Task not found.")
	}
	if err != nil {
		return t, err
	}
	t.WorkingFields = rating.EmptyFields()
	if err = json.Unmarshal(answers, &t.Answers); err != nil {
		return t, err
	}
	if err = json.Unmarshal(fields, &t.WorkingFields); err != nil {
		return t, err
	}
	if len(snapshot) > 0 {
		if err = json.Unmarshal(snapshot, &t.ConfirmedSnapshot); err != nil {
			return t, err
		}
	}
	t.CreatedAt = t.CreatedAt.UTC()
	if t.PublishedAt != nil {
		v := t.PublishedAt.UTC()
		t.PublishedAt = &v
	}
	t.Preview = rating.Calculate(t.WorkingFields, true)
	return t, nil
}
func (s *Store) Create(ctx context.Context, owner string, in CreateInput) (Task, error) {
	if err := validateCreate(&in); err != nil {
		return Task{}, err
	}
	id, err := httpapi.NewID()
	if err != nil {
		return Task{}, err
	}
	fields, _ := json.Marshal(rating.EmptyFields())
	return scan(s.db.QueryRowContext(ctx, `INSERT INTO tasks(id,owner_id,title,industry,draft,working_fields) VALUES($1,$2,$3,$4,$5,$6) RETURNING `+columns, id, owner, in.Title, in.Industry, in.Draft, string(fields)))
}
func (s *Store) Get(ctx context.Context, id string) (Task, error) {
	return scan(s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM tasks WHERE id=$1`, id))
}
func persist(ctx context.Context, tx *sql.Tx, t Task) error {
	answers, err := json.Marshal(t.Answers)
	if err != nil {
		return err
	}
	fields, err := json.Marshal(t.WorkingFields)
	if err != nil {
		return err
	}
	var snapshot any
	score := 0
	readiness := "draft"
	if t.ConfirmedSnapshot != nil {
		data, e := json.Marshal(t.ConfirmedSnapshot)
		if e != nil {
			return e
		}
		snapshot = string(data)
		score = t.ConfirmedSnapshot.Total
		readiness = t.ConfirmedSnapshot.Readiness
	}
	_, err = tx.ExecContext(ctx, `UPDATE tasks SET title=$2,industry=$3,answers=$4,working_fields=$5,confirmed_snapshot=$6,
 revision=$7,confirmed_revision=$8,published_at=$9,status=$10,score=$11,readiness=$12,updated_at=CURRENT_TIMESTAMP WHERE id=$1`,
		t.ID, t.Title, t.Industry, string(answers), string(fields), snapshot, t.Revision, t.ConfirmedRevision, t.PublishedAt, t.Status, score, readiness)
	return err
}
func (s *Store) mutate(ctx context.Context, id, owner string, revision int, change func(*Task) error) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()
	t, err := scan(tx.QueryRowContext(ctx, `SELECT `+columns+` FROM tasks WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return t, err
	}
	if t.OwnerID != owner {
		return Task{}, httpapi.Problem(403, "forbidden", "Only the task owner can edit it.")
	}
	if revision != t.Revision {
		return Task{}, httpapi.Problem(409, "stale_revision", "Reload the task before saving; its revision changed.")
	}
	if err = change(&t); err != nil {
		return Task{}, err
	}
	if err = persist(ctx, tx, t); err != nil {
		return Task{}, err
	}
	t.Preview = rating.Calculate(t.WorkingFields, true)
	return t, tx.Commit()
}
func (s *Store) Edit(ctx context.Context, id, owner string, in EditInput) (Task, error) {
	return s.mutate(ctx, id, owner, in.Revision, func(t *Task) error { return t.edit(in) })
}
func (s *Store) Confirm(ctx context.Context, id, owner string, in ConfirmInput) (Task, error) {
	return s.mutate(ctx, id, owner, in.Revision, func(t *Task) error { return t.confirm(in) })
}
func (s *Store) Publish(ctx context.Context, id, owner string, revision int) (Task, error) {
	return s.mutate(ctx, id, owner, revision, func(t *Task) error {
		if t.ConfirmedSnapshot == nil || t.ConfirmedRevision == nil || *t.ConfirmedRevision != t.Revision {
			return httpapi.Problem(409, "confirmation_required", "Confirm the current revision before publishing.")
		}
		if t.Title == "" || t.Industry == "" {
			return httpapi.Invalid(map[string]string{"title": "Title and industry are required to publish."})
		}
		if t.PublishedAt == nil {
			now := time.Now().UTC()
			t.PublishedAt = &now
			t.Status = "published"
		}
		return nil
	})
}
func (s *Store) List(ctx context.Context, owner, industry, readiness string, page httpapi.Page) ([]Task, int, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	where := ` WHERE (($1 = '' AND published_at IS NOT NULL AND confirmed_snapshot IS NOT NULL) OR owner_id::text=$1)
 AND ($2='' OR CASE WHEN $1='' THEN confirmed_snapshot->>'industry' ELSE industry END=$2) AND ($3='' OR readiness=$3)`
	var total int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM tasks`+where, owner, industry, readiness).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+columns+` FROM tasks`+where+` ORDER BY score DESC,published_at ASC NULLS LAST,id ASC LIMIT $4 OFFSET $5`, owner, industry, readiness, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		t, e := scan(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, t)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, tx.Commit()
}
