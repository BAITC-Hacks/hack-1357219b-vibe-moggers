package rating

import "testing"

func ptr(s string) *string { return &s }
func full() map[string]Field {
	f := EmptyFields()
	for _, d := range Definitions {
		f[d.Key] = Field{Value: ptr("Specific business information"), Confirmed: true}
	}
	f["deadline"] = Field{Value: ptr("14 days"), Confirmed: true}
	f["contact"] = Field{Value: ptr("demo@example.org"), Confirmed: true}
	f["successTarget"] = Field{Value: ptr("10 minutes per list"), Confirmed: true}
	return f
}
func TestScoreRules(t *testing.T) {
	f := full()
	r := Calculate(f, false)
	if r.Total != 100 || r.Readiness != "priority" || len(r.ScoreBreakdown) != 7 {
		t.Fatalf("full: %+v", r)
	}
	for k, v := range f {
		v.Confirmed = false
		f[k] = v
	}
	if Calculate(f, false).Total != 0 || Calculate(f, true).Total != 100 {
		t.Fatal("confirmation must be required; preview predicts confirmation")
	}
	for _, v := range []string{"", " ", "-", "N/A", "later", "не знаю", "позже"} {
		f["context"] = Field{Value: ptr(v), Confirmed: true}
		if Eligible("context", f["context"].Value) {
			t.Errorf("placeholder eligible: %q", v)
		}
	}
	f = full()
	metric := f["successMetric"]
	metric.Confirmed = false
	f["successMetric"] = metric
	if Calculate(f, false).Total != 85 {
		t.Fatal("success dependencies must lose all 15 points")
	}
	if Calculate(f, true).Total != 100 {
		t.Fatal("preview dependency")
	}
	metric.Value = nil
	f["successMetric"] = metric
	if Calculate(f, true).Total != 85 {
		t.Fatal("missing metric must block dependent preview")
	}
	for _, a := range Calculate(f, false).NextActions {
		if (a.Field == "successTarget" || a.Field == "acceptanceMethod") && a.PossibleGain != 0 {
			t.Fatal("cannot promise dependent gain")
		}
	}
}
func TestBands(t *testing.T) {
	for score, want := range map[int]string{0: "draft", 39: "draft", 40: "working", 69: "working", 70: "ready", 89: "ready", 90: "priority", 100: "priority"} {
		if got := Band(score); got != want {
			t.Errorf("%d=%s want %s", score, got, want)
		}
	}
}
func TestFieldValidation(t *testing.T) {
	for _, tc := range []struct {
		key, value string
		want       bool
	}{
		{"deadline", "2026-12-31", true}, {"deadline", "2026-02-30", false}, {"deadline", "14 календарных дней", true}, {"deadline", "soon", false},
		{"contact", "demo@example.org", true}, {"contact", "https://example.org/contact", true}, {"contact", "+7 777 123 4567", true}, {"contact", "ask me", false},
		{"successTarget", "10", false}, {"successTarget", "10 minutes", true}, {"successTarget", "all tests pass", true}, {"successTarget", "better", false},
		{"notAField", "anything", false},
	} {
		if got := Eligible(tc.key, ptr(tc.value)); got != tc.want {
			t.Errorf("%s %q=%v", tc.key, tc.value, got)
		}
	}
}
