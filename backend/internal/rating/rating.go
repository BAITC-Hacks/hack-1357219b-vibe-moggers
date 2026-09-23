package rating

import (
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type Source struct {
	ID    string `json:"id"`
	Quote string `json:"quote"`
}
type Field struct {
	Value     *string `json:"value"`
	Confirmed bool    `json:"confirmed"`
	Source    *Source `json:"source"`
}
type Definition struct {
	Key, Label, Group, GroupLabel string
	Weight                        int
}

var Definitions = []Definition{
	{"context", "Describe the current process", "context", "Context and need", 10},
	{"need", "Describe the business need", "context", "Context and need", 10},
	{"dataSource", "Identify available data", "data", "Data and materials", 10},
	{"dataFormat", "Specify the data format", "data", "Data and materials", 5},
	{"dataAccess", "Explain how the team gets data", "data", "Data and materials", 5},
	{"deliverable", "Specify the deliverable", "result", "Expected result", 10},
	{"deliveryFormat", "Specify the delivery format", "result", "Expected result", 5},
	{"successMetric", "Define a success metric", "success", "Success criteria", 5},
	{"successTarget", "Set a measurable acceptance target", "success", "Success criteria", 5},
	{"acceptanceMethod", "Explain how results will be checked", "success", "Success criteria", 5},
	{"deadline", "Provide a date or duration", "constraints", "Constraints", 5},
	{"constraints", "Describe constraints", "constraints", "Constraints", 5},
	{"users", "Identify the users", "users", "Users", 5},
	{"usageScenario", "Describe the usage scenario", "users", "Users", 5},
	{"contact", "Provide an email, URL or phone", "contact", "Business contact", 5},
	{"feedbackFormat", "Explain the feedback process", "contact", "Business contact", 5},
}

type Breakdown struct {
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	Earned        int      `json:"earned"`
	Max           int      `json:"max"`
	MissingFields []string `json:"missingFields"`
}
type Action struct {
	Field        string `json:"field"`
	Action       string `json:"action"`
	PossibleGain int    `json:"possibleGain"`
}
type Result struct {
	Total          int         `json:"total"`
	Readiness      string      `json:"readiness"`
	ScoreBreakdown []Breakdown `json:"scoreBreakdown"`
	MissingFields  []string    `json:"missingFields"`
	NextActions    []Action    `json:"nextActions"`
}

func Known(key string) bool {
	for _, d := range Definitions {
		if d.Key == key {
			return true
		}
	}
	return false
}
func EmptyFields() map[string]Field {
	out := map[string]Field{}
	for _, d := range Definitions {
		out[d.Key] = Field{}
	}
	return out
}
func Normalize(s string) string { return strings.Join(strings.Fields(s), " ") }
func Band(score int) string {
	switch {
	case score >= 90:
		return "priority"
	case score >= 70:
		return "ready"
	case score >= 40:
		return "working"
	default:
		return "draft"
	}
}

var duration = regexp.MustCompile(`(?i)\b[1-9][0-9]*\s*(hours?|days?|weeks?|months?|час|дн|день|дней|сут|нед|месяц|календарн|рабоч)`)
var target = regexp.MustCompile(`[0-9]+\s*[^\s0-9.,]`)
var phone = regexp.MustCompile(`^\+?[0-9 ()-]{7,25}$`)

func Eligible(key string, value *string) bool {
	if value == nil || !Known(key) {
		return false
	}
	s := strings.TrimSpace(*value)
	if s == "" || utf8.RuneCountInString(s) > 10000 || strings.ContainsRune(s, '\x00') {
		return false
	}
	switch strings.ToLower(s) {
	case "-", "n/a", "na", "none", "null", "unknown", "later", "tbd", "don't know", "не знаю", "позже", "неизвестно", "нет данных":
		return false
	}
	switch key {
	case "deadline":
		if _, err := time.Parse("2006-01-02", s); err == nil {
			return true
		}
		return duration.MatchString(s)
	case "successTarget":
		if target.MatchString(s) {
			return true
		}
		lower := strings.ToLower(s)
		for _, phrase := range []string{"all tests pass", "no errors", "without errors", "all records", "все тесты", "без ошибок", "все заявки", "все записи"} {
			if strings.Contains(lower, phrase) {
				return true
			}
		}
		return false
	case "contact":
		if a, e := mail.ParseAddress(s); e == nil && a.Address == s {
			return true
		}
		if ValidURL(s) {
			return true
		}
		digits := 0
		for _, r := range s {
			if r >= '0' && r <= '9' {
				digits++
			}
		}
		return phone.MatchString(s) && digits >= 7 && digits <= 15
	}
	return true
}
func ValidURL(s string) bool {
	u, e := url.Parse(s)
	return e == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil && !strings.ContainsAny(s, " \t\r\n\x00") && len(s) <= 2048
}

func Calculate(fields map[string]Field, preview bool) Result {
	r := Result{ScoreBreakdown: []Breakdown{}, MissingFields: []string{}, NextActions: []Action{}}
	metric := fields["successMetric"]
	metricOK := Eligible("successMetric", metric.Value) && (preview || metric.Confirmed)
	for _, d := range Definitions {
		if len(r.ScoreBreakdown) == 0 || r.ScoreBreakdown[len(r.ScoreBreakdown)-1].Key != d.Group {
			r.ScoreBreakdown = append(r.ScoreBreakdown, Breakdown{Key: d.Group, Label: d.GroupLabel, MissingFields: []string{}})
		}
		group := &r.ScoreBreakdown[len(r.ScoreBreakdown)-1]
		group.Max += d.Weight
		f := fields[d.Key]
		ok := Eligible(d.Key, f.Value) && (preview || f.Confirmed)
		dependency := d.Key == "successTarget" || d.Key == "acceptanceMethod"
		if dependency && !metricOK {
			ok = false
		}
		if ok {
			group.Earned += d.Weight
			r.Total += d.Weight
		} else {
			group.MissingFields = append(group.MissingFields, d.Key)
			r.MissingFields = append(r.MissingFields, d.Key)
			gain := d.Weight
			action := d.Label
			if dependency && !metricOK {
				gain = 0
				action = "Confirm a valid successMetric, then: " + action
			}
			r.NextActions = append(r.NextActions, Action{Field: d.Key, Action: action, PossibleGain: gain})
		}
	}
	r.Readiness = Band(r.Total)
	return r
}
