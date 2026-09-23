package ai

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
	"vibe-moggers/backend/internal/httpapi"
	"vibe-moggers/backend/internal/rating"
)

//go:embed prompt.txt
var prompt string

type Source struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type Input struct {
	Stage         string                  `json:"stage"`
	Sources       []Source                `json:"sources"`
	CurrentFields map[string]rating.Field `json:"currentFields"`
}
type Question struct {
	ID        string   `json:"id"`
	FieldKeys []string `json:"fieldKeys"`
	Text      string   `json:"text"`
}
type Suggestion struct {
	Field    string `json:"field"`
	Value    string `json:"value"`
	SourceID string `json:"sourceId"`
}
type Result struct {
	Questions        []Question   `json:"questions"`
	FieldSuggestions []Suggestion `json:"fieldSuggestions"`
	Warnings         []string     `json:"warnings"`
	Mode             string       `json:"mode"`
	DurationMS       int64        `json:"durationMs"`
}
type Service struct {
	mode, key, model, endpoint string
	client                     *http.Client
	timeout                    time.Duration
}

func New(mode, key, model string) *Service {
	return &Service{mode: mode, key: key, model: model, endpoint: "https://api.openai.com/v1/responses", client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, timeout: 15 * time.Second}
}
func validateInput(in Input) error {
	if in.Stage != "clarify" && in.Stage != "assemble" {
		return httpapi.Invalid(map[string]string{"stage": "Use clarify or assemble."})
	}
	if len(in.Sources) < 1 || len(in.Sources) > 51 {
		return httpapi.Invalid(map[string]string{"sources": "Use 1-51 source objects."})
	}
	seen := map[string]bool{}
	total := 0
	for _, s := range in.Sources {
		total += utf8.RuneCountInString(s.Text)
		if strings.TrimSpace(s.ID) == "" || len(s.ID) > 100 || seen[s.ID] || strings.TrimSpace(s.Text) == "" || utf8.RuneCountInString(s.Text) > 20000 || strings.ContainsRune(s.ID+s.Text, '\x00') {
			return httpapi.Invalid(map[string]string{"sources": "Use unique IDs and nonempty texts up to 20000 characters each."})
		}
		seen[s.ID] = true
	}
	if total > 60000 {
		return httpapi.Invalid(map[string]string{"sources": "Maximum total 60000 characters."})
	}
	for key, f := range in.CurrentFields {
		if !rating.Known(key) || f.Value != nil && (utf8.RuneCountInString(*f.Value) > 10000 || strings.ContainsRune(*f.Value, '\x00')) {
			return httpapi.Invalid(map[string]string{"currentFields": "Use known field keys and valid values."})
		}
	}
	return nil
}
func fallback(in Input, warning string) Result {
	groups := []Question{
		{ID: "q1", FieldKeys: []string{"context", "need"}, Text: "How does the process work today, and what needs to change?"},
		{ID: "q2", FieldKeys: []string{"dataSource", "dataFormat", "dataAccess"}, Text: "What data is available, in which format, and how can the team access it?"},
		{ID: "q3", FieldKeys: []string{"deliverable", "deliveryFormat"}, Text: "What should the team deliver, and in which format?"},
		{ID: "q4", FieldKeys: []string{"successMetric", "successTarget", "acceptanceMethod"}, Text: "What measurable result counts as success, and how will you verify it?"},
		{ID: "q5", FieldKeys: []string{"deadline", "constraints"}, Text: "What deadline and technical or access constraints apply?"},
		{ID: "q6", FieldKeys: []string{"users", "usageScenario"}, Text: "Who will use the result, and in which situation?"},
		{ID: "q7", FieldKeys: []string{"contact", "feedbackFormat"}, Text: "Who is the contact, and how will the team receive feedback?"},
	}
	selected := []Question{}
	used := map[string]bool{}
	for _, q := range groups {
		for _, key := range q.FieldKeys {
			if !rating.Eligible(key, in.CurrentFields[key].Value) {
				selected = append(selected, q)
				used[q.ID] = true
				break
			}
		}
		if len(selected) == 5 {
			break
		}
	}
	for _, q := range groups {
		if len(selected) >= 3 {
			break
		}
		if !used[q.ID] {
			selected = append(selected, q)
		}
	}
	return Result{Questions: selected, FieldSuggestions: []Suggestion{}, Warnings: []string{warning}, Mode: "fallback"}
}
func (s *Service) Analyze(ctx context.Context, in Input) (Result, error) {
	if err := validateInput(in); err != nil {
		return Result{}, err
	}
	start := time.Now()
	if s.mode != "live" {
		r := fallback(in, "Fallback mode: prepared questions; no AI-generated facts.")
		r.DurationMS = time.Since(start).Milliseconds()
		return r, nil
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var result Result
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		var retry bool
		result, retry, err = s.request(ctx, in)
		if err == nil || !retry || ctx.Err() != nil {
			break
		}
		if attempt == 0 {
			select {
			case <-time.After(200 * time.Millisecond):
			case <-ctx.Done():
			}
		}
	}
	if err != nil {
		result = fallback(in, "AI unavailable or returned invalid output. Continue with prepared questions.")
	}
	result.DurationMS = time.Since(start).Milliseconds()
	return result, nil
}
func object(properties map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}
func schema() map[string]any {
	keys := []string{}
	for _, d := range rating.Definitions {
		keys = append(keys, d.Key)
	}
	str := map[string]any{"type": "string"}
	key := map[string]any{"type": "string", "enum": keys}
	q := object(map[string]any{"id": str, "fieldKeys": map[string]any{"type": "array", "items": key}, "text": str}, "id", "fieldKeys", "text")
	suggestion := object(map[string]any{"field": key, "value": str, "sourceId": str}, "field", "value", "sourceId")
	return object(map[string]any{"questions": map[string]any{"type": "array", "items": q}, "fieldSuggestions": map[string]any{"type": "array", "items": suggestion}, "warnings": map[string]any{"type": "array", "items": str}}, "questions", "fieldSuggestions", "warnings")
}
func (s *Service) request(ctx context.Context, in Input) (Result, bool, error) {
	input, _ := json.Marshal(in)
	payload := map[string]any{"model": s.model, "store": false, "instructions": prompt, "input": string(input), "max_output_tokens": 2500, "text": map[string]any{"format": map[string]any{"type": "json_schema", "name": "task_analysis", "strict": true, "schema": schema()}}}
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{}, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, false, err
	}
	req.Header.Set("Authorization", "Bearer "+s.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, true, errors.New("AI request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Result{}, resp.StatusCode == 429 || resp.StatusCode >= 500, fmt.Errorf("AI HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return Result{}, false, errors.New("invalid AI response size")
	}
	var envelope struct {
		Status string `json:"status"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err = json.Unmarshal(raw, &envelope); err != nil || envelope.Status != "completed" {
		return Result{}, false, errors.New("incomplete AI response")
	}
	text := ""
	for _, o := range envelope.Output {
		if o.Type != "message" {
			continue
		}
		for _, c := range o.Content {
			if c.Type == "refusal" {
				return Result{}, false, errors.New("AI refusal")
			}
			if c.Type == "output_text" {
				text += c.Text
			}
		}
	}
	r, err := parse(text, in)
	return r, false, err
}
func parse(text string, in Input) (Result, error) {
	var body struct {
		Questions        []Question   `json:"questions"`
		FieldSuggestions []Suggestion `json:"fieldSuggestions"`
		Warnings         []string     `json:"warnings"`
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return Result{}, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return Result{}, errors.New("extra AI JSON")
	}
	if body.Questions == nil || body.FieldSuggestions == nil || body.Warnings == nil || len(body.Questions) > 5 || (in.Stage == "clarify" && len(body.Questions) < 3) || len(body.FieldSuggestions) > 16 || len(body.Warnings) > 20 {
		return Result{}, errors.New("invalid AI response shape")
	}
	seen := map[string]bool{}
	for _, q := range body.Questions {
		if strings.TrimSpace(q.ID) == "" || len(q.ID) > 100 || seen[q.ID] || strings.TrimSpace(q.Text) == "" || utf8.RuneCountInString(q.Text) > 2000 || len(q.FieldKeys) == 0 || strings.ContainsRune(q.Text, '\x00') {
			return Result{}, errors.New("invalid AI questions")
		}
		seen[q.ID] = true
		for _, key := range q.FieldKeys {
			if !rating.Known(key) {
				return Result{}, errors.New("unknown AI question field")
			}
		}
	}
	sources := map[string]string{}
	for _, s := range in.Sources {
		sources[s.ID] = rating.Normalize(s.Text)
	}
	out := Result{Questions: body.Questions, FieldSuggestions: []Suggestion{}, Warnings: []string{}, Mode: "live"}
	// Provider warnings are untrusted text too. Bound their size before returning them.
	for _, w := range body.Warnings {
		if utf8.RuneCountInString(w) <= 2000 && !strings.ContainsRune(w, '\x00') {
			out.Warnings = append(out.Warnings, w)
		}
	}
	seen = map[string]bool{}
	for _, f := range body.FieldSuggestions {
		value := rating.Normalize(f.Value)
		source, ok := sources[f.SourceID]
		if !rating.Known(f.Field) || seen[f.Field] || !ok || value == "" || utf8.RuneCountInString(value) > 10000 || !strings.Contains(source, value) {
			out.Warnings = append(out.Warnings, "Ignored an invalid or unsupported field suggestion.")
			continue
		}
		f.Value = value
		out.FieldSuggestions = append(out.FieldSuggestions, f)
		seen[f.Field] = true
	}
	return out, nil
}
func (s *Service) Register(logger *slog.Logger) func(*http.ServeMux) {
	return func(mux *http.ServeMux) {
		mux.HandleFunc("POST /api/ai/analyze", func(w http.ResponseWriter, r *http.Request) {
			if _, err := httpapi.Actor(r, "business"); err != nil {
				httpapi.Fail(w, logger, err)
				return
			}
			var in Input
			if err := httpapi.Decode(w, r, &in); err != nil {
				httpapi.Fail(w, logger, err)
				return
			}
			out, err := s.Analyze(r.Context(), in)
			if err != nil {
				httpapi.Fail(w, logger, err)
				return
			}
			httpapi.Data(w, 200, out)
		})
	}
}
