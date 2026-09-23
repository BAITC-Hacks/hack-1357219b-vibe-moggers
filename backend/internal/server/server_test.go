package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubPinger struct {
	err error
}

func (s stubPinger) PingContext(context.Context) error {
	return s.err
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		pingError  error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "database connected",
			wantStatus: http.StatusOK,
			wantBody:   "{\"database\":\"connected\",\"status\":\"ok\"}\n",
		},
		{
			name:       "database unavailable",
			pingError:  errors.New("connection failed"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "{\"database\":\"unavailable\",\"status\":\"degraded\"}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			handler := New(stubPinger{err: tt.pingError}, logger)
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if response.Body.String() != tt.wantBody {
				t.Fatalf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", contentType)
			}
		})
	}
}
