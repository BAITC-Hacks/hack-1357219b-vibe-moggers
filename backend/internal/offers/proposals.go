package offers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"vibe-moggers/backend/internal/httpapi"
)

type Proposal struct {
	ID           string     `json:"id"`
	TaskID       string     `json:"taskId"`
	TeamID       string     `json:"teamId"`
	TeamName     string     `json:"teamName"`
	Idea         string     `json:"idea"`
	Plan         []string   `json:"plan"`
	DurationDays *int       `json:"durationDays"`
	PrototypeURL string     `json:"prototypeUrl"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	DecidedAt    *time.Time `json:"decidedAt"`
	Milestone    *Milestone `json:"milestone"`
}
type proposalInput struct {
	Idea         string   `json:"idea"`
	Plan         []string `json:"plan"`
	DurationDays int      `json:"durationDays"`
	PrototypeURL string   `json:"prototypeUrl"`
}

func asProposal(o Offer) Proposal {
	return Proposal{ID: o.ID, TaskID: o.TaskID, TeamID: o.TeamID, TeamName: o.TeamName, Idea: o.SolutionIdea, Plan: o.PlanSteps, DurationDays: o.DurationDays, PrototypeURL: o.PrototypeLink, Status: o.Status, CreatedAt: o.CreatedAt, DecidedAt: o.DecidedAt}
}
func (h *Handler) createProposal(w http.ResponseWriter, r *http.Request) {
	actor, err := httpapi.Actor(r, "team")
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	id, err := pathID(r, "task_id")
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	var in proposalInput
	if err = httpapi.Decode(w, r, &in); err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	if len(in.Plan) < 1 || len(in.Plan) > 30 || in.DurationDays < 1 || in.DurationDays > 3650 {
		httpapi.Fail(w, h.logger, httpapi.Invalid(map[string]string{"plan": "Use 1-30 nonempty steps.", "durationDays": "Use 1-3650 days."}))
		return
	}
	for i, step := range in.Plan {
		in.Plan[i] = strings.TrimSpace(step)
		if in.Plan[i] == "" {
			httpapi.Fail(w, h.logger, httpapi.Invalid(map[string]string{"plan": "Steps must not be blank."}))
			return
		}
	}
	input := Input{SolutionIdea: in.Idea, Plan: strings.Join(in.Plan, "\n"), Timeline: fmt.Sprintf("%d days", in.DurationDays), PrototypeLink: in.PrototypeURL, PlanSteps: in.Plan, DurationDays: &in.DurationDays}
	if err = validate(&input, actor); err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	o, err := h.store.Create(r.Context(), id, actor, input)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	httpapi.Data(w, 201, asProposal(o))
}
func (h *Handler) listProposals(w http.ResponseWriter, r *http.Request) {
	role, actor, err := httpapi.Identity(r)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	id, err := pathID(r, "task_id")
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
	switch status {
	case "", "pending", "accepted", "rejected":
	default:
		httpapi.Fail(w, h.logger, httpapi.Problem(400, "invalid_query", "Unknown proposal status."))
		return
	}
	items, total, err := h.store.ListForActor(r.Context(), id, role, actor, status, page)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	out := []Proposal{}
	for _, o := range items {
		p := asProposal(o)
		p.Milestone, err = h.store.milestoneForProposal(r.Context(), o.ID)
		if err != nil {
			httpapi.Fail(w, h.logger, err)
			return
		}
		out = append(out, p)
	}
	httpapi.List(w, out, total, page)
}
func (h *Handler) decision(w http.ResponseWriter, r *http.Request) {
	owner, err := httpapi.Actor(r, "business")
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	id, err := pathID(r, "offer_id")
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	var in struct {
		Decision string `json:"decision"`
	}
	if err = httpapi.Decode(w, r, &in); err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	o, err := h.store.Decide(r.Context(), id, owner, in.Decision)
	if err != nil {
		httpapi.Fail(w, h.logger, err)
		return
	}
	httpapi.Data(w, 200, asProposal(o))
}
