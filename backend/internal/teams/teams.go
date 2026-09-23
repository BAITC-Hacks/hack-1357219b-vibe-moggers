package teams

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"vibe-moggers/backend/internal/httpapi"
)

type Team struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Points       int       `json:"points"`
	Interests    []string  `json:"interests"`
	Skills       []string  `json:"skills"`
	Technologies []string  `json:"technologies"`
	CreatedAt    time.Time `json:"created_at"`
}

type Handler struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewHandler(db *sql.DB, logger *slog.Logger) *Handler {
	return &Handler{db: db, logger: logger}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/teams", h.create)
	mux.HandleFunc("GET /api/teams", h.list)
	mux.HandleFunc("GET /api/teams/{team_id}", h.get)
}

const columns = `id, name, points, interests, skills, technologies, created_at`

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Team, error) {
	var team Team
	var interests, skills, technologies []byte
	if err := row.Scan(&team.ID, &team.Name, &team.Points, &interests, &skills, &technologies, &team.CreatedAt); err != nil {
		return team, err
	}
	for _, field := range []struct {
		raw  []byte
		dest *[]string
	}{{interests, &team.Interests}, {skills, &team.Skills}, {technologies, &team.Technologies}} {
		if err := json.Unmarshal(field.raw, field.dest); err != nil {
			return team, err
		}
	}
	team.CreatedAt = team.CreatedAt.UTC()
	return team, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := httpapi.Decode(w, r, &input); err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if n := utf8.RuneCountInString(input.Name); n < 1 || n > 200 || strings.ContainsRune(input.Name, '\x00') {
		httpapi.Fail(w, h.logger, httpapi.Invalid(map[string]string{"name": "Required, maximum 200 characters; NUL is not allowed."}))
		return
	}
	id, err := httpapi.NewID()
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	team, err := scan(h.db.QueryRowContext(r.Context(), `INSERT INTO teams(id, name) VALUES ($1, $2) RETURNING `+columns, id, input.Name))
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	httpapi.JSON(w, 201, map[string]any{"team": team})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("team_id")
	if !httpapi.UUID(id) {
		httpapi.Fail(w, h.logger, httpapi.Problem(404, "team_not_found", "Team not found."))
		return
	}
	team, err := scan(h.db.QueryRowContext(r.Context(), `SELECT `+columns+` FROM teams WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		err = httpapi.Problem(404, "team_not_found", "Team not found.")
	}
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	httpapi.JSON(w, 200, map[string]any{"team": team})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, err := httpapi.Pagination(r)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	items, total, err := h.listPage(r.Context(), page)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	httpapi.List(w, items, total, page)
}

func (h *Handler) listPage(ctx context.Context, page httpapi.Page) ([]Team, int, error) {
	tx, err := h.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	var total int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM teams`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+columns+` FROM teams ORDER BY created_at, id LIMIT $1 OFFSET $2`, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Team, 0)
	for rows.Next() {
		team, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, team)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, tx.Commit()
}
