package tasks

import (
	"testing"
	"vibe-moggers/backend/internal/rating"
)

func str(s string) *string { return &s }
func TestEditingAndConfirmation(t *testing.T) {
	task := Task{WorkingFields: rating.EmptyFields(), Revision: 1, Draft: "We have CSV data", Status: "draft"}
	if err := task.edit(EditInput{Fields: map[string]rating.Field{"context": {Value: str("Retail business"), Confirmed: true}}}); err != nil {
		t.Fatal(err)
	}
	if task.WorkingFields["context"].Confirmed {
		t.Fatal("client forged confirmation")
	}
	if err := task.confirm(ConfirmInput{Reviewed: true, FieldKeys: []string{"context"}}); err != nil {
		t.Fatal(err)
	}
	if task.ConfirmedSnapshot.Total != 10 || task.Revision != 3 {
		t.Fatalf("snapshot: %+v", task)
	}
	if err := task.edit(EditInput{Fields: map[string]rating.Field{"context": {Value: str("New context"), Confirmed: true}}}); err != nil {
		t.Fatal(err)
	}
	if task.WorkingFields["context"].Confirmed || *task.ConfirmedSnapshot.Fields["context"].Value != "Retail business" {
		t.Fatal("editing must not change confirmed snapshot")
	}
	for _, key := range []string{"successMetric", "successTarget", "acceptanceMethod"} {
		task.WorkingFields[key] = rating.Field{Value: str("10 minutes"), Confirmed: true}
	}
	if err := task.edit(EditInput{Fields: map[string]rating.Field{"successMetric": {Value: str("Error count")}}}); err != nil {
		t.Fatal(err)
	}
	if task.WorkingFields["successTarget"].Confirmed || task.WorkingFields["acceptanceMethod"].Confirmed {
		t.Fatal("metric edit must reset dependent confirmations")
	}
}
func TestSourceValidation(t *testing.T) {
	task := Task{WorkingFields: rating.EmptyFields(), Revision: 1, Draft: "We have CSV data"}
	err := task.edit(EditInput{Fields: map[string]rating.Field{"dataFormat": {Value: str("CSV"), Source: &rating.Source{ID: "draft", Quote: "CSV"}}}})
	if err != nil {
		t.Fatal(err)
	}
	err = task.edit(EditInput{Fields: map[string]rating.Field{"dataFormat": {Value: str("SQL database"), Source: &rating.Source{ID: "draft", Quote: "SQL database"}}}})
	if err == nil {
		t.Fatal("invented source accepted")
	}
}
