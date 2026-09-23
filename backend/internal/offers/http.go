package offers

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

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
	mux.HandleFunc("POST /api/proposals/{offer_id}/milestone", h.submitMilestone)
	mux.HandleFunc("POST /api/milestones/{milestone_id}/confirm", h.confirmMilestone)
	mux.HandleFunc("POST /api/tasks/{task_id}/offers", h.create(false))
	mux.HandleFunc("GET /api/tasks/{task_id}/offers", h.list(false))
	mux.HandleFunc("PATCH /api/offers/{offer_id}/status", h.decide("", false))
	// Compatibility with the existing frontend/Postman contract. Both route
	// families use the same rows and transactional decision implementation.
	mux.HandleFunc("POST /api/tasks/{task_id}/proposals", h.createProposal)
	mux.HandleFunc("GET /api/tasks/{task_id}/proposals", h.listProposals)
	mux.HandleFunc("POST /api/proposals/{offer_id}/decision", h.decision)
	mux.HandleFunc("POST /api/proposals/{offer_id}/accept", h.decide("accepted", true))
	mux.HandleFunc("POST /api/proposals/{offer_id}/reject", h.decide("rejected", true))
}

func validate(input *Input, actorID string) error {
	fields := make(map[string]string)
	input.TeamID = strings.ToLower(strings.TrimSpace(input.TeamID))
	if input.TeamID != "" && input.TeamID != actorID {
		return httpapi.Problem(403, "forbidden", "team_id must match the selected demo team.")
	}
	for _, field := range []struct {
		key   string
		value *string
		max   int
	}{{"solution_idea", &input.SolutionIdea, 10000}, {"plan", &input.Plan, 10000}, {"timeline", &input.Timeline, 500}} {
		*field.value = strings.TrimSpace(*field.value)
		n := utf8.RuneCountInString(*field.value)
		if n == 0 || n > field.max || strings.ContainsRune(*field.value, '\x00') {
			fields[field.key] = "Required and must fit the documented character limit; NUL is not allowed."
		}
	}
	input.PrototypeLink = strings.TrimSpace(input.PrototypeLink)
	if input.PrototypeLink != "" {
		parsed, err := url.Parse(input.PrototypeLink)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.User != nil || strings.ContainsAny(input.PrototypeLink, " \t\r\n") || utf8.RuneCountInString(input.PrototypeLink) > 2048 {
			fields["prototype_link"] = "Use an absolute HTTP(S) URL, maximum 2048 characters, or an empty string."
		}
	}
	if len(fields) != 0 {
		return httpapi.Invalid(fields)
	}
	return nil
}

func pathID(r *http.Request, key string) (string, error) {
	id := r.PathValue(key)
	if !httpapi.UUID(id) {
		return "", httpapi.Problem(404, "not_found", "Resource not found.")
	}
	return strings.ToLower(id), nil
}

func present(offer Offer, legacy bool) Offer {
	if legacy && offer.Status == "pending" {
		offer.Status = "submitted"
	}
	return offer
}

func reply(w http.ResponseWriter, status int, offer Offer, legacy bool) {
	key := "offer"
	if legacy {
		key = "proposal"
	}
	httpapi.JSON(w, status, map[string]any{key: present(offer, legacy)})
}

func (h *Handler) create(legacy bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actorID, err := httpapi.Actor(r, "team")
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		taskID, err := pathID(r, "task_id")
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		var input Input
		if err := httpapi.Decode(w, r, &input); err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		if err := validate(&input, actorID); err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		offer, err := h.store.Create(r.Context(), taskID, actorID, input)
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		reply(w, 201, offer, legacy)
	}
}

func (h *Handler) list(legacy bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ownerID, err := httpapi.Actor(r, "business")
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		taskID, err := pathID(r, "task_id")
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		page, err := httpapi.Pagination(r)
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		status := r.URL.Query().Get("status")
		pending := "pending"
		if legacy {
			pending = "submitted"
		}
		if status != "" && status != pending && status != "accepted" && status != "rejected" {
			httpapi.Fail(w, h.logger, httpapi.Problem(400, "invalid_query", "Unknown offer status."))
			return
		}
		if status == "submitted" {
			status = "pending"
		}
		items, total, err := h.store.List(r.Context(), taskID, ownerID, status, page)
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		for i := range items {
			items[i] = present(items[i], legacy)
		}
		httpapi.List(w, items, total, page)
	}
}

func (h *Handler) decide(status string, legacy bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ownerID, err := httpapi.Actor(r, "business")
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		id, err := pathID(r, "offer_id")
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		decision := status
		if !legacy {
			var input struct {
				Status string `json:"status"`
			}
			if err := httpapi.Decode(w, r, &input); err != nil {
				httpapi.Fail(w, h.logger, err)
				return
			}
			decision = input.Status
		}
		offer, err := h.store.Decide(r.Context(), id, ownerID, decision)
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		reply(w, 200, offer, legacy)
	}
}
