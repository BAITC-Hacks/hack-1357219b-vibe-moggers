package offers

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestValidationBeforeDatabase(t *testing.T) {
	const team = "00000000-0000-4000-8000-000000000001"
	const path = "/api/tasks/10000000-0000-4000-8000-000000000001/offers"
	valid := `{"solution_idea":"Idea","plan":"1. Build\n2. Test","timeline":"One week"}`
	mux := http.NewServeMux()
	NewHandler(nil, slog.New(slog.NewTextHandler(io.Discard, nil))).Register(mux)
	for _, tt := range []struct {
		name, actor, path, body string
		status                  int
	}{
		{"missing actor", "", path, valid, 401},
		{"invalid actor", "team:invalid", path, valid, 401},
		{"wrong role", "business:" + team, path, valid, 403},
		{"invalid task ID", "team:" + team, "/api/tasks/invalid/offers", valid, 404},
		{"malformed JSON", "team:" + team, path, `{`, 400},
		{"unknown field", "team:" + team, path, `{"points":100}`, 400},
		{"extra JSON", "team:" + team, path, valid + `{}`, 400},
		{"missing fields", "team:" + team, path, `{}`, 422},
		{"impersonated team", "team:" + team, path, `{"team_id":"00000000-0000-4000-8000-000000000002"}`, 403},
		{"oversized body", "team:" + team, path, `{"solution_idea":"` + strings.Repeat("a", 140000) + `"}`, 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			r.Header.Set("X-Demo-Actor", tt.actor)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.status, w.Body)
			}
		})
	}
}

func TestOfferContentValidation(t *testing.T) {
	for _, tt := range []struct {
		name  string
		edit  func(*Input)
		valid bool
	}{
		{"optional prototype", func(*Input) {}, true},
		{"https prototype", func(i *Input) { i.PrototypeLink = "https://example.org/prototype" }, true},
		{"blank idea", func(i *Input) { i.SolutionIdea = " \n " }, false},
		{"long timeline", func(i *Input) { i.Timeline = strings.Repeat("д", 501) }, false},
		{"javascript link", func(i *Input) { i.PrototypeLink = "javascript:alert(1)" }, false},
		{"relative link", func(i *Input) { i.PrototypeLink = "/prototype" }, false},
		{"invalid link", func(i *Input) { i.PrototypeLink = "https://%" }, false},
		{"NUL in plan", func(i *Input) { i.Plan = "a\x00b" }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := Input{SolutionIdea: "Идея", Plan: "1. Собрать\n2. Проверить", Timeline: "Неделя"}
			tt.edit(&input)
			if err := validate(&input, "team"); (err == nil) != tt.valid {
				t.Fatalf("valid = %v, error = %v", tt.valid, err)
			}
		})
	}
}
