package awards

import (
	"context"
	"database/sql"
	"fmt"
)

const AcceptancePoints = 10

// Grant participates in the caller's decision transaction. The database
// uniqueness constraint protects both retries and concurrent requests.
func Grant(ctx context.Context, tx *sql.Tx, offerID, teamID string) error {
	result, err := tx.ExecContext(ctx, `INSERT INTO point_awards (offer_id, team_id, points)
		VALUES ($1, $2, $3) ON CONFLICT (offer_id) DO NOTHING`, offerID, teamID, AcceptancePoints)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil || inserted == 0 {
		return err
	}
	result, err = tx.ExecContext(ctx, `UPDATE teams SET points = points + $1 WHERE id = $2`, AcceptancePoints, teamID)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("award team missing")
	}
	return nil
}
