// Package tasks implements the task builder and the SQL boundary for proposals.
package tasks

import (
	"context"
	"database/sql"
	"errors"

	"vibe-moggers/backend/internal/httpapi"
)

type Access interface {
	Lock(context.Context, *sql.Tx, string) (Task, error)
	RefreshWorkStatus(context.Context, *sql.Tx, string) error
}

type SQLAccess struct{}

// Lock serializes decisions on one task, including decisions on different
// offers. Keep this lock until awards and work_status have committed.
func (SQLAccess) Lock(ctx context.Context, tx *sql.Tx, id string) (Task, error) {
	var task Task
	err := tx.QueryRowContext(ctx, `SELECT id, owner_id, status = 'published' AND published_at IS NOT NULL
		FROM tasks WHERE id = $1 FOR UPDATE`, id).Scan(&task.ID, &task.OwnerID, &task.Published)
	if errors.Is(err, sql.ErrNoRows) {
		return task, httpapi.Problem(404, "task_not_found", "Task not found.")
	}
	return task, err
}

func (SQLAccess) RefreshWorkStatus(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx, `UPDATE tasks SET work_status = CASE
		WHEN EXISTS (SELECT 1 FROM offers WHERE task_id = $1 AND status = 'accepted')
		THEN 'in_progress' ELSE 'open' END, updated_at = CURRENT_TIMESTAMP WHERE id = $1`, id)
	return err
}
