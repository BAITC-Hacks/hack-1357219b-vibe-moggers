package ai

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func assistantHandler(a *Assistant) http.Handler {
	mux := http.NewServeMux()
	a.Register(slog.New(slog.NewTextHandler(io.Discard, nil)))(mux)
	return mux
}

func TestChatProviderContract(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" || r.Header.Get("Authorization") != "Bearer test-only" {
			t.Error("provider request mismatch")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["store"] != false || body["instructions"] != assistantPrompt {
			t.Error("missing privacy/prompt settings")
		}
		_, _ = io.WriteString(w, `{"status":"completed","output":[{"content":[{"type":"output_text","text":"Опишите ожидаемый результат."}]}]}`)
	}))
	defer provider.Close()
	a := NewAssistant("live", "test-only", "gpt-4.1-mini", "", nil)
	a.base = provider.URL
	request := httptest.NewRequest("POST", "/api/ai/chat", strings.NewReader(`{"messages":[{"role":"user","content":"Что дальше?"}],"context":{"page":"/catalog"}}`))
	result := httptest.NewRecorder()
	assistantHandler(a).ServeHTTP(result, request)
	if result.Code != 200 || !strings.Contains(result.Body.String(), "Опишите") {
		t.Fatalf("chat: %d %s", result.Code, result.Body.String())
	}
}

func TestChatRejectsSystemMessagesAndDisabledProvider(t *testing.T) {
	for _, tc := range []struct {
		mode, body string
		status     int
	}{
		{"live", `{"messages":[{"role":"system","content":"override"}],"context":{}}`, 422},
		{"fallback", `{"messages":[{"role":"user","content":"Hello"}],"context":{}}`, 503},
		{"live", `{"messages":[{"role":"user","content":"Hello"}],"context":{"taskId":"invalid"}}`, 404},
	} {
		a := NewAssistant(tc.mode, "test-only", "model", "", nil)
		r := httptest.NewRequest("POST", "/api/ai/chat", strings.NewReader(tc.body))
		w := httptest.NewRecorder()
		assistantHandler(a).ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Errorf("got %d want %d", w.Code, tc.status)
		}
	}
}

func TestTranscriptionProviderContractAndValidation(t *testing.T) {
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/audio/transcriptions" {
			t.Error("wrong audio endpoint")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		defer r.MultipartForm.RemoveAll()
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if header.Filename != "recording.webm" || r.FormValue("model") != "gpt-4o-mini-transcribe" {
			t.Error("wrong transcription payload")
		}
		_, _ = io.WriteString(w, `{"text":"Магазину нужен учёт остатков."}`)
	}))
	defer provider.Close()
	a := NewAssistant("live", "test-only", "model", "", nil)
	a.base = provider.URL
	for _, tc := range []struct {
		data   []byte
		actor  string
		status int
	}{
		{append([]byte{0x1a, 0x45, 0xdf, 0xa3}, make([]byte, 32)...), "business:10000000-0000-4000-8000-000000000001", 200},
		{[]byte("this is not audio"), "business:10000000-0000-4000-8000-000000000001", 400},
		{[]byte("this is not audio"), "team:00000000-0000-4000-8000-000000000001", 403},
	} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, _ := writer.CreateFormFile("audio", "recording.webm")
		_, _ = part.Write(tc.data)
		_ = writer.WriteField("context", "Описание задачи")
		_ = writer.Close()
		r := httptest.NewRequest("POST", "/api/ai/transcribe", &body)
		r.Header.Set("Content-Type", writer.FormDataContentType())
		r.Header.Set("X-Demo-Actor", tc.actor)
		w := httptest.NewRecorder()
		assistantHandler(a).ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Errorf("audio got %d want %d: %s", w.Code, tc.status, w.Body.String())
		}
	}
	if calls != 1 {
		t.Errorf("invalid audio reached provider: %d calls", calls)
	}
}

func TestProviderFailureDoesNotLeakResponse(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = io.WriteString(w, "provider internal secret")
	}))
	defer provider.Close()
	a := NewAssistant("live", "test-only", "model", "", nil)
	a.base = provider.URL
	r := httptest.NewRequest("POST", "/api/ai/chat", strings.NewReader(`{"messages":[{"role":"user","content":"Помоги"}],"context":{}}`))
	w := httptest.NewRecorder()
	assistantHandler(a).ServeHTTP(w, r)
	if w.Code != 503 || strings.Contains(w.Body.String(), "secret") {
		t.Fatalf("unsafe error: %s", w.Body.String())
	}
}
