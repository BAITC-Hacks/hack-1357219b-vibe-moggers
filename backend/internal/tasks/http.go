package tasks

import (
	"log/slog"
	"net/http"
	"strings"
	"vibe-moggers/backend/internal/httpapi"
)

type Handler struct {
	store  *Store
	logger *slog.Logger
}

func NewHandler(store *Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/tasks", h.create)
	mux.HandleFunc("GET /api/tasks/{task_id}", h.get)
	mux.HandleFunc("PUT /api/tasks/{task_id}/card", h.mutate("edit"))
	mux.HandleFunc("POST /api/tasks/{task_id}/confirm", h.mutate("confirm"))
	mux.HandleFunc("POST /api/tasks/{task_id}/publish", h.mutate("publish"))
	mux.HandleFunc("GET /api/tasks", h.list(false))
	mux.HandleFunc("GET /api/business/tasks", h.list(true))
}
func taskID(r *http.Request) (string, error) {
	id := r.PathValue("task_id")
	if !httpapi.UUID(id) {
		return "", httpapi.Problem(404, "task_not_found", "Task not found.")
	}
	return strings.ToLower(id), nil
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	owner, err := httpapi.Actor(r, "business")
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	var in CreateInput
	if err = httpapi.Decode(w, r, &in); err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	t, err := h.store.Create(r.Context(), owner, in)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	httpapi.Data(w, 201, t)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := taskID(r)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	role, actor := "", ""
	if r.Header.Get("X-Demo-Actor") != "" {
		role, actor, err = httpapi.Identity(r)
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
	}
	t, err := h.store.Get(r.Context(), id)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	if role == "business" && actor == t.OwnerID {
		httpapi.Data(w, 200, t)
		return
	}
	if t.PublishedAt == nil || t.ConfirmedSnapshot == nil {
		httpapi.Fail(w, h.logger, httpapi.Problem(404, "task_not_found", "Task not found."))
		return
	}
	httpapi.Data(w, 200, t.Public())
}
func (h *Handler) mutate(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, err := httpapi.Actor(r, "business")
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		id, err := taskID(r)
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		var t Task
		switch action {
		case "edit":
			var in EditInput
			err = httpapi.Decode(w, r, &in)
			if err == nil {
				t, err = h.store.Edit(r.Context(), id, owner, in)
			}
		case "confirm":
			var in ConfirmInput
			err = httpapi.Decode(w, r, &in)
			if err == nil {
				t, err = h.store.Confirm(r.Context(), id, owner, in)
			}
		case "publish":
			var in struct {
				Revision int `json:"revision"`
			}
			err = httpapi.Decode(w, r, &in)
			if err == nil {
				t, err = h.store.Publish(r.Context(), id, owner, in.Revision)
			}
		}
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		httpapi.Data(w, 200, t)
	}
}
func (h *Handler) list(owned bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner := ""
		var err error
		if owned {
			owner, err = httpapi.Actor(r, "business")
			if err != nil {
				httpapi.Fail(w, h.logger, err)
				return
			}
		}
		page, err := httpapi.Pagination(r)
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		readiness := r.URL.Query().Get("readiness")
		switch readiness {
		case "", "draft", "working", "ready", "priority":
		default:
			httpapi.Fail(w, h.logger, httpapi.Problem(400, "invalid_query", "Unknown readiness."))
			return
		}
		items, total, err := h.store.List(r.Context(), owner, r.URL.Query().Get("industry"), readiness, page)
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		if owned {
			httpapi.List(w, items, total, page)
			return
		}
		public := []PublicTask{}
		for _, t := range items {
			public = append(public, t.Public())
		}
		httpapi.List(w, public, total, page)
	}
}
