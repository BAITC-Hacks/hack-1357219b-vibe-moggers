// Package httpapi contains the shared JSON contract and local demo identity.
package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func Problem(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func Invalid(fields map[string]string) *Error {
	return &Error{Status: 422, Code: "validation_error", Message: "Some fields are invalid.", Fields: fields}
}

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func Fail(w http.ResponseWriter, logger *slog.Logger, err error) {
	var problem *Error
	if !errors.As(err, &problem) {
		logger.Error("request failed", "error", err)
		problem = Problem(500, "internal_error", "An unexpected error occurred.")
	}
	JSON(w, problem.Status, map[string]any{"error": problem})
}

func Decode(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return Problem(400, "invalid_json", "Expected a JSON object with known fields.")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Problem(400, "invalid_json", "Expected a single JSON object.")
	}
	return nil
}

func UUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	return err == nil
}

func NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// Actor is a local demo selector, NOT authenticated identity. Replace this
// resolver with session/token middleware before exposing the app publicly.
func Actor(r *http.Request, role string) (string, error) {
	value := r.Header.Get("X-Demo-Actor")
	if value == "" {
		return "", Problem(401, "actor_required", "Select a demo actor using X-Demo-Actor.")
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || !UUID(parts[1]) {
		return "", Problem(401, "invalid_actor", "Expected business:UUID or team:UUID.")
	}
	if parts[0] != role {
		return "", Problem(403, "forbidden", "This action is unavailable to this role.")
	}
	return strings.ToLower(parts[1]), nil
}

type Page struct {
	Limit  int
	Offset int
}

func Pagination(r *http.Request) (Page, error) {
	p := Page{Limit: 20}
	for key, dest := range map[string]*int{"limit": &p.Limit, "offset": &p.Offset} {
		if raw, exists := r.URL.Query()[key]; exists {
			if len(raw) != 1 {
				return p, Problem(400, "invalid_query", "Invalid pagination.")
			}
			value, err := strconv.Atoi(raw[0])
			if err != nil {
				return p, Problem(400, "invalid_query", "Invalid pagination.")
			}
			*dest = value
		}
	}
	if p.Limit < 1 || p.Limit > 100 || p.Offset < 0 {
		return p, Problem(400, "invalid_query", "limit must be 1-100 and offset must be nonnegative.")
	}
	return p, nil
}

func List(w http.ResponseWriter, items any, total int, page Page) {
	JSON(w, 200, map[string]any{"items": items, "total": total, "limit": page.Limit, "offset": page.Offset})
}
