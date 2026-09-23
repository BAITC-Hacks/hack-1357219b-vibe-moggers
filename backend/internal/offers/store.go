package offers

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"vibe-moggers/backend/internal/awards"
	"vibe-moggers/backend/internal/httpapi"
	"vibe-moggers/backend/internal/tasks"
)

type Offer struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"task_id"`
	TeamID        string     `json:"team_id"`
	TeamName      string     `json:"team_name"`
	SolutionIdea  string     `json:"solution_idea"`
	Plan          string     `json:"plan"`
	Timeline      string     `json:"timeline"`
	PrototypeLink string     `json:"prototype_link"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	DecidedAt     *time.Time `json:"decided_at"`
}

type Input struct {
	TeamID        string `json:"team_id"`
	SolutionIdea  string `json:"solution_idea"`
	Plan          string `json:"plan"`
	Timeline      string `json:"timeline"`
	PrototypeLink string `json:"prototype_link"`
}

type Store struct {
	db    *sql.DB
	tasks tasks.Access
}

func NewStore(db *sql.DB, taskAccess tasks.Access) *Store {
	return &Store{db: db, tasks: taskAccess}
}

const selectOffer = `SELECT o.id, o.task_id, o.team_id, t.name, o.solution_idea,
	o.plan, o.timeline, o.prototype_link, o.status, o.created_at, o.decided_at
	FROM offers o JOIN teams t ON t.id = o.team_id`

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Offer, error) {
	var offer Offer
	err := row.Scan(&offer.ID, &offer.TaskID, &offer.TeamID, &offer.TeamName,
		&offer.SolutionIdea, &offer.Plan, &offer.Timeline, &offer.PrototypeLink,
		&offer.Status, &offer.CreatedAt, &offer.DecidedAt)
	offer.CreatedAt = offer.CreatedAt.UTC()
	if offer.DecidedAt != nil {
		utc := offer.DecidedAt.UTC()
		offer.DecidedAt = &utc
	}
	return offer, err
}

func (s *Store) Create(ctx context.Context, taskID, teamID string, input Input) (Offer, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Offer{}, err
	}
	defer tx.Rollback()
	task, err := s.tasks.Lock(ctx, tx, taskID)
	if err != nil {
		return Offer{}, err
	}
	if !task.Published {
		return Offer{}, httpapi.Problem(409, "task_not_published", "Only published tasks accept offers.")
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM teams WHERE id = $1)`, teamID).Scan(&exists); err != nil {
		return Offer{}, err
	}
	if !exists {
		return Offer{}, httpapi.Problem(404, "team_not_found", "Team not found.")
	}
	id, err := httpapi.NewID()
	if err != nil {
		return Offer{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO offers
		(id, task_id, team_id, solution_idea, plan, timeline, prototype_link)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, id, taskID, teamID,
		input.SolutionIdea, input.Plan, input.Timeline, input.PrototypeLink)
	if err != nil {
		return Offer{}, err
	}
	offer, err := scan(tx.QueryRowContext(ctx, selectOffer+` WHERE o.id = $1`, id))
	if err != nil {
		return Offer{}, err
	}
	return offer, tx.Commit()
}

func (s *Store) List(ctx context.Context, taskID, ownerID, status string, page httpapi.Page) ([]Offer, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	task, err := s.tasks.Lock(ctx, tx, taskID)
	if err != nil {
		return nil, 0, err
	}
	if task.OwnerID != ownerID {
		return nil, 0, httpapi.Problem(403, "forbidden", "Only the task owner can review offers.")
	}
	var total int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM offers WHERE task_id = $1 AND ($2 = '' OR status = $2)`, taskID, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := tx.QueryContext(ctx, selectOffer+` WHERE o.task_id = $1 AND ($2 = '' OR o.status = $2)
		ORDER BY o.created_at DESC, o.id ASC LIMIT $3 OFFSET $4`, taskID, status, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Offer, 0)
	for rows.Next() {
		offer, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, offer)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, tx.Commit()
}

func (s *Store) Decide(ctx context.Context, id, ownerID, status string) (Offer, error) {
	if status != "accepted" && status != "rejected" {
		return Offer{}, httpapi.Invalid(map[string]string{"status": "Use accepted or rejected."})
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Offer{}, err
	}
	defer tx.Rollback()
	var taskID string
	err = tx.QueryRowContext(ctx, `SELECT task_id FROM offers WHERE id = $1`, id).Scan(&taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return Offer{}, httpapi.Problem(404, "offer_not_found", "Offer not found.")
	}
	if err != nil {
		return Offer{}, err
	}
	// All writers lock task first, then offer, then team. This serializes
	// competing decisions and keeps work_status consistent without deadlocks.
	task, err := s.tasks.Lock(ctx, tx, taskID)
	if err != nil {
		return Offer{}, err
	}
	if task.OwnerID != ownerID {
		return Offer{}, httpapi.Problem(403, "forbidden", "Only the task owner can decide on offers.")
	}
	offer, err := scan(tx.QueryRowContext(ctx, selectOffer+` WHERE o.id = $1 FOR UPDATE OF o`, id))
	if err != nil {
		return Offer{}, err
	}
	if offer.Status != status {
		_, err := tx.ExecContext(ctx, `UPDATE offers SET status = $1, decided_at = CURRENT_TIMESTAMP WHERE id = $2`, status, id)
		if err != nil {
			return Offer{}, err
		}
	}
	if status == "accepted" {
		if err := awards.Grant(ctx, tx, id, offer.TeamID); err != nil {
			return Offer{}, err
		}
	}
	if err := s.tasks.RefreshWorkStatus(ctx, tx, taskID); err != nil {
		return Offer{}, err
	}
	offer, err = scan(tx.QueryRowContext(ctx, selectOffer+` WHERE o.id = $1`, id))
	if err != nil {
		return Offer{}, err
	}
	return offer, tx.Commit()
}
