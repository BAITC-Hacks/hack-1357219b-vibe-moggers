package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStrictJSONObject(t *testing.T) {
	for _, body := range []string{"null", "[]", "42", "", "{} {}", `{"unknown":true}`, `{"name":"` + strings.Repeat("x", 129<<10) + `"}`} {
		var input struct {
			Name string `json:"name"`
		}
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		if err := Decode(httptest.NewRecorder(), r, &input); err == nil {
			t.Fatalf("invalid body accepted (%d bytes)", len(body))
		}
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := Decode(httptest.NewRecorder(), httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Valid"}`)), &input); err != nil || input.Name != "Valid" {
		t.Fatal("valid object rejected")
	}
}
