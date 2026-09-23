package tasks

import (
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"vibe-moggers/backend/internal/httpapi"
	"vibe-moggers/backend/internal/rating"
)

type Answer struct {
	ID         string `json:"id"`
	QuestionID string `json:"questionId"`
	Text       string `json:"text"`
}
type Snapshot struct {
	Title    string                  `json:"title"`
	Industry string                  `json:"industry"`
	Fields   map[string]rating.Field `json:"fields"`
	rating.Result
	Timestamp time.Time `json:"timestamp"`
}
type Task struct {
	ID                string                  `json:"id"`
	OwnerID           string                  `json:"ownerId"`
	Title             string                  `json:"title"`
	Industry          string                  `json:"industry"`
	Draft             string                  `json:"draft"`
	Answers           []Answer                `json:"answers"`
	WorkingFields     map[string]rating.Field `json:"workingFields"`
	ConfirmedSnapshot *Snapshot               `json:"confirmedSnapshot"`
	Revision          int                     `json:"revision"`
	ConfirmedRevision *int                    `json:"confirmedRevision"`
	PublishedAt       *time.Time              `json:"publishedAt"`
	CreatedAt         time.Time               `json:"createdAt"`
	Status            string                  `json:"status"`
	WorkStatus        string                  `json:"workStatus"`
	Preview           rating.Result           `json:"preview"`
	Published         bool                    `json:"-"`
}
type PublicTask struct {
	ID string `json:"id"`
	*Snapshot
	PublishedAt *time.Time `json:"publishedAt"`
	WorkStatus  string     `json:"workStatus"`
}

func (t Task) Public() PublicTask {
	return PublicTask{ID: t.ID, Snapshot: t.ConfirmedSnapshot, PublishedAt: t.PublishedAt, WorkStatus: t.WorkStatus}
}

type CreateInput struct {
	Draft    string `json:"draft"`
	Title    string `json:"title"`
	Industry string `json:"industry"`
}
type EditInput struct {
	Revision int                     `json:"revision"`
	Title    *string                 `json:"title"`
	Industry *string                 `json:"industry"`
	Answers  *[]Answer               `json:"answers"`
	Fields   map[string]rating.Field `json:"fields"`
}
type ConfirmInput struct {
	Revision  int      `json:"revision"`
	FieldKeys []string `json:"fieldKeys"`
	Reviewed  bool     `json:"reviewed"`
}

func textOK(s string, max int, required bool) bool {
	return (!required || strings.TrimSpace(s) != "") && utf8.RuneCountInString(s) <= max && !strings.ContainsRune(s, '\x00')
}
func validateCreate(in *CreateInput) error {
	in.Draft = strings.TrimSpace(in.Draft)
	in.Title = strings.TrimSpace(in.Title)
	in.Industry = strings.TrimSpace(in.Industry)
	fields := map[string]string{}
	if !textOK(in.Draft, 20000, true) {
		fields["draft"] = "Required; maximum 20000 characters."
	}
	if !textOK(in.Title, 200, false) {
		fields["title"] = "Maximum 200 characters."
	}
	if !textOK(in.Industry, 100, false) {
		fields["industry"] = "Maximum 100 characters."
	}
	if len(fields) > 0 {
		return httpapi.Invalid(fields)
	}
	return nil
}
func validSource(f rating.Field, sources map[string]string) bool {
	if f.Source == nil {
		return true
	}
	source, ok := sources[f.Source.ID]
	return ok && f.Value != nil && rating.Normalize(f.Source.Quote) != "" && rating.Normalize(*f.Value) == rating.Normalize(f.Source.Quote) && strings.Contains(rating.Normalize(source), rating.Normalize(f.Source.Quote))
}
func (t *Task) edit(in EditInput) error {
	errs := map[string]string{}
	if in.Title != nil {
		t.Title = strings.TrimSpace(*in.Title)
		if !textOK(t.Title, 200, false) {
			errs["title"] = "Maximum 200 characters."
		}
	}
	if in.Industry != nil {
		t.Industry = strings.TrimSpace(*in.Industry)
		if !textOK(t.Industry, 100, false) {
			errs["industry"] = "Maximum 100 characters."
		}
	}
	if in.Answers != nil {
		t.Answers = *in.Answers
	}
	if t.Answers == nil {
		t.Answers = []Answer{}
	}
	sources := map[string]string{"draft": t.Draft}
	seen := map[string]bool{"draft": true}
	if len(t.Answers) > 50 {
		errs["answers"] = "Maximum 50 answers."
	}
	for i, a := range t.Answers {
		a.ID = strings.TrimSpace(a.ID)
		a.Text = strings.TrimSpace(a.Text)
		t.Answers[i] = a
		if !textOK(a.ID, 100, true) || !textOK(a.QuestionID, 100, false) || !textOK(a.Text, 10000, false) || seen[a.ID] {
			errs["answers"] = "Use unique nonempty IDs, valid text and at most 10000 characters per answer."
		}
		seen[a.ID] = true
		sources[a.ID] = a.Text
	}
	metricChanged := false
	for key, f := range in.Fields {
		if !rating.Known(key) {
			errs["fields."+key] = "Unknown field."
			continue
		}
		if f.Value != nil {
			s := strings.TrimSpace(*f.Value)
			f.Value = &s
			if s == "" {
				f.Value = nil
			} else if !textOK(s, 10000, false) {
				errs["fields."+key] = "Maximum 10000 characters; NUL is not allowed."
			}
		}
		if !validSource(f, sources) {
			errs["fields."+key+".source"] = "Source must contain the exact field value."
		}
		old := t.WorkingFields[key]
		changed := !reflect.DeepEqual(old.Value, f.Value) || !reflect.DeepEqual(old.Source, f.Source)
		f.Confirmed = old.Confirmed && !changed // Confirmation is never trusted from the request body.
		if changed && key == "successMetric" {
			metricChanged = true
		}
		t.WorkingFields[key] = f
	}
	for key, f := range t.WorkingFields {
		if !validSource(f, sources) {
			f.Source = nil
			f.Confirmed = false
			t.WorkingFields[key] = f
			if key == "successMetric" {
				metricChanged = true
			}
		}
	}
	if metricChanged {
		for _, key := range []string{"successTarget", "acceptanceMethod"} {
			f := t.WorkingFields[key]
			f.Confirmed = false
			t.WorkingFields[key] = f
		}
	}
	if len(errs) > 0 {
		return httpapi.Invalid(errs)
	}
	t.Revision++
	if t.PublishedAt == nil {
		t.Status = "draft"
	}
	return nil
}
func (t *Task) confirm(in ConfirmInput) error {
	if !in.Reviewed {
		return httpapi.Invalid(map[string]string{"reviewed": "Explicit human review is required."})
	}
	if in.FieldKeys == nil {
		return httpapi.Invalid(map[string]string{"fieldKeys": "Provide an array, including an empty array for unknown fields."})
	}
	seen := map[string]bool{}
	for _, key := range in.FieldKeys {
		if !rating.Known(key) || seen[key] {
			return httpapi.Invalid(map[string]string{"fieldKeys": "Use unique known field keys."})
		}
		seen[key] = true
		f := t.WorkingFields[key]
		if !rating.Eligible(key, f.Value) {
			return httpapi.Invalid(map[string]string{"fields." + key: "Cannot confirm an empty, placeholder or invalid value."})
		}
		f.Confirmed = true
		t.WorkingFields[key] = f
	}
	fields := rating.EmptyFields()
	for key, f := range t.WorkingFields {
		if f.Confirmed && rating.Eligible(key, f.Value) {
			fields[key] = f
		}
	}
	t.ConfirmedSnapshot = &Snapshot{Title: t.Title, Industry: t.Industry, Fields: fields, Result: rating.Calculate(fields, false), Timestamp: time.Now().UTC()}
	t.Revision++
	rev := t.Revision
	t.ConfirmedRevision = &rev
	if t.PublishedAt == nil {
		t.Status = "confirmed"
	} else if t.Title == "" || t.Industry == "" {
		return httpapi.Invalid(map[string]string{"title": "Published cards need a title and industry."})
	}
	return nil
}
