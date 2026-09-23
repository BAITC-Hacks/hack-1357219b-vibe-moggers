package offers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
	"vibe-moggers/backend/internal/httpapi"
	"vibe-moggers/backend/internal/rating"
)

type Milestone struct {
	ID          string     `json:"id"`
	ProposalID  string     `json:"proposalId"`
	TeamID      string     `json:"teamId"`
	ResultText  string     `json:"resultText"`
	EvidenceURL string     `json:"evidenceUrl"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	ConfirmedAt *time.Time `json:"confirmedAt"`
	Points      int        `json:"points"`
}
type milestoneInput struct {
	ResultText  string `json:"resultText"`
	EvidenceURL string `json:"evidenceUrl"`
}

const milestoneColumns = `m.id,m.proposal_id,m.team_id,m.result_text,m.evidence_url,m.status,m.created_at,m.confirmed_at,
 COALESCE((SELECT points FROM progress_awards WHERE milestone_id=m.id),0)`

func scanMilestone(row scanner) (Milestone, error) {
	var m Milestone
	err := row.Scan(&m.ID, &m.ProposalID, &m.TeamID, &m.ResultText, &m.EvidenceURL, &m.Status, &m.CreatedAt, &m.ConfirmedAt, &m.Points)
	m.CreatedAt = m.CreatedAt.UTC()
	if m.ConfirmedAt != nil {
		v := m.ConfirmedAt.UTC()
		m.ConfirmedAt = &v
	}
	return m, err
}
func (s *Store) milestoneForProposal(ctx context.Context, id string) (*Milestone, error) {
	m, err := scanMilestone(s.db.QueryRowContext(ctx, `SELECT `+milestoneColumns+` FROM milestones m WHERE proposal_id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}
func (s *Store) SubmitMilestone(ctx context.Context, id, team string, in milestoneInput) (Milestone, error) {
	in.ResultText = strings.TrimSpace(in.ResultText)
	in.EvidenceURL = strings.TrimSpace(in.EvidenceURL)
	if in.ResultText == "" || utf8.RuneCountInString(in.ResultText) > 10000 || strings.ContainsRune(in.ResultText, '\x00') || !rating.ValidURL(in.EvidenceURL) {
		return Milestone{}, httpapi.Invalid(map[string]string{"resultText": "Provide a result up to 10000 characters.", "evidenceUrl": "An absolute HTTP(S) URL is required."})
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Milestone{}, err
	}
	defer tx.Rollback()
	var taskID string
	err = tx.QueryRowContext(ctx, `SELECT task_id FROM offers WHERE id=$1`, id).Scan(&taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return Milestone{}, httpapi.Problem(404, "proposal_not_found", "Proposal not found.")
	}
	if err != nil {
		return Milestone{}, err
	}
	if _, err = s.tasks.Lock(ctx, tx, taskID); err != nil {
		return Milestone{}, err
	}
	var owner, status string
	if err = tx.QueryRowContext(ctx, `SELECT team_id,status FROM offers WHERE id=$1 FOR UPDATE`, id).Scan(&owner, &status); err != nil {
		return Milestone{}, err
	}
	if owner != team {
		return Milestone{}, httpapi.Problem(403, "forbidden", "Only the proposal's team can submit its result.")
	}
	if status != "accepted" {
		return Milestone{}, httpapi.Problem(409, "proposal_not_accepted", "The business must first accept this proposal.")
	}
	existing, err := scanMilestone(tx.QueryRowContext(ctx, `SELECT `+milestoneColumns+` FROM milestones m WHERE proposal_id=$1`, id))
	if err == nil {
		if existing.ResultText != in.ResultText || existing.EvidenceURL != in.EvidenceURL {
			return Milestone{}, httpapi.Problem(409, "milestone_exists", "This proposal already has a milestone.")
		}
		return existing, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Milestone{}, err
	}
	mid, err := httpapi.NewID()
	if err != nil {
		return Milestone{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO milestones(id,proposal_id,team_id,result_text,evidence_url) VALUES($1,$2,$3,$4,$5)`, mid, id, team, in.ResultText, in.EvidenceURL)
	if err != nil {
		return Milestone{}, err
	}
	m, err := scanMilestone(tx.QueryRowContext(ctx, `SELECT `+milestoneColumns+` FROM milestones m WHERE m.id=$1`, mid))
	if err != nil {
		return m, err
	}
	return m, tx.Commit()
}
func (s *Store) ConfirmMilestone(ctx context.Context, id, owner string) (Milestone, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Milestone{}, err
	}
	defer tx.Rollback()
	var taskID string
	err = tx.QueryRowContext(ctx, `SELECT o.task_id FROM milestones m JOIN offers o ON o.id=m.proposal_id WHERE m.id=$1`, id).Scan(&taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return Milestone{}, httpapi.Problem(404, "milestone_not_found", "Milestone not found.")
	}
	if err != nil {
		return Milestone{}, err
	}
	task, err := s.tasks.Lock(ctx, tx, taskID)
	if err != nil {
		return Milestone{}, err
	}
	if task.OwnerID != owner {
		return Milestone{}, httpapi.Problem(403, "forbidden", "Only the task owner can confirm a result.")
	}
	m, err := scanMilestone(tx.QueryRowContext(ctx, `SELECT `+milestoneColumns+` FROM milestones m WHERE m.id=$1 FOR UPDATE OF m`, id))
	if err != nil {
		return m, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE milestones SET status='confirmed',confirmed_at=COALESCE(confirmed_at,CURRENT_TIMESTAMP) WHERE id=$1`, id)
	if err != nil {
		return m, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO progress_awards(milestone_id,team_id,points) VALUES($1,$2,10) ON CONFLICT(milestone_id) DO NOTHING`, id, m.TeamID)
	if err != nil {
		return m, err
	}
	m, err = scanMilestone(tx.QueryRowContext(ctx, `SELECT `+milestoneColumns+` FROM milestones m WHERE m.id=$1`, id))
	if err != nil {
		return m, err
	}
	return m, tx.Commit()
}
func (h *Handler) submitMilestone(w http.ResponseWriter, r *http.Request) {
	team, err := httpapi.Actor(r, "team")
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	id, err := pathID(r, "offer_id")
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	var in milestoneInput
	if err = httpapi.Decode(w, r, &in); err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	m, err := h.store.SubmitMilestone(r.Context(), id, team, in)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	httpapi.Data(w, 200, m)
}
func (h *Handler) confirmMilestone(w http.ResponseWriter, r *http.Request) {
	owner, err := httpapi.Actor(r, "business")
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	id, err := pathID(r, "milestone_id")
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	var in struct {
		Confirmed bool `json:"confirmed"`
	}
	if err = httpapi.Decode(w, r, &in); err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	if !in.Confirmed {
		httpapi.Fail(w, h.logger, httpapi.Invalid(map[string]string{"confirmed": "Explicit confirmation is required."}))
		return
	}
	m, err := h.store.ConfirmMilestone(r.Context(), id, owner)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	httpapi.Data(w, 200, m)
}
