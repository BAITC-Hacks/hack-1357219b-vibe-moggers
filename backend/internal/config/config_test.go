package config

import "testing"

func TestConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/demo")
	t.Setenv("PORT", "8080")
	t.Setenv("AI_MODE", "fallback")
	t.Setenv("DEMO_MODE", "true")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "test-model")
	cfg, err := Load()
	if err != nil || cfg.AIMode != "fallback" || cfg.OpenAIModel != "test-model" || !cfg.DemoMode {
		t.Fatalf("load failed: %v", err)
	}
	for _, tc := range []struct{ key, value string }{{"DATABASE_URL", ""}, {"PORT", "abc"}, {"PORT", "0"}, {"AI_MODE", "unknown"}, {"AI_MODE", "live"}, {"DEMO_MODE", "maybe"}} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
	t.Setenv("AI_MODE", "live")
	t.Setenv("OPENAI_API_KEY", "test-only-secret")
	if _, err = Load(); err != nil {
		t.Fatal(err)
	}
}
