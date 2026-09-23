package app_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"vibe-moggers/backend/internal/app"
	"vibe-moggers/backend/internal/config"
	"vibe-moggers/backend/internal/database"
	"vibe-moggers/backend/internal/offers"
	"vibe-moggers/backend/internal/rating"
	"vibe-moggers/backend/internal/seed"
	"vibe-moggers/backend/internal/tasks"
	"vibe-moggers/backend/internal/testutil"
)

const business = "business:" + seed.OwnerID
const team = "team:00000000-0000-4000-8000-000000000001"
const team2 = "team:00000000-0000-4000-8000-000000000002"
const stranger = "business:10000000-0000-4000-8000-000000000002"

func str(s string) *string { return &s }
func handler(db *sql.DB) http.Handler {
	return app.New(db, config.Config{AIMode: "fallback", DemoMode: true}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
func request(h http.Handler, method, path, actor string, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	if actor != "" {
		r.Header.Set("X-Demo-Actor", actor)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func call(t *testing.T, h http.Handler, method, path, actor string, body any, status int, dest any) {
	t.Helper()
	w := request(h, method, path, actor, body)
	if w.Code != status {
		t.Fatalf("%s %s got %d want %d: %s", method, path, w.Code, status, w.Body)
	}
	var envelope struct {
		Data  json.RawMessage `json:"data"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if status < 400 {
		if len(envelope.Data) == 0 {
			t.Fatal("missing data envelope")
		}
		if dest != nil {
			if err := json.Unmarshal(envelope.Data, dest); err != nil {
				t.Fatal(err)
			}
		}
	} else if len(envelope.Error) == 0 {
		t.Fatal("missing error envelope")
	}
}
func loadSeeds(t *testing.T, db *sql.DB) {
	t.Helper()
	f, err := os.Open("../../../docs/fixtures/seed.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = seed.Load(context.Background(), db, f); err != nil {
		t.Fatal(err)
	}
}
func fields95() map[string]rating.Field {
	f := rating.EmptyFields()
	for _, d := range rating.Definitions {
		if d.Key != "feedbackFormat" {
			f[d.Key] = rating.Field{Value: str("Specific business requirement"), Confirmed: true}
		}
	}
	f["deadline"] = rating.Field{Value: str("14 days")}
	f["contact"] = rating.Field{Value: str("demo@example.org")}
	f["successTarget"] = rating.Field{Value: str("10 minutes per list")}
	return f
}

type catalog struct {
	Items []tasks.PublicTask `json:"items"`
	Total int                `json:"total"`
}

func TestFullPostgresFlow(t *testing.T) {
	db, dsn := testutil.Postgres(t)
	loadSeeds(t, db)
	loadSeeds(t, db)
	h := handler(db)
	var list catalog
	call(t, h, "GET", "/api/tasks", "", nil, 200, &list)
	if list.Total != 5 {
		t.Fatalf("seed catalog=%d", list.Total)
	}
	for i, want := range []int{90, 80, 60, 30, 20} {
		if list.Items[i].Total != want {
			t.Fatalf("seed score[%d]=%d want%d", i, list.Items[i].Total, want)
		}
	}
	var counts struct{ Total int }
	call(t, h, "GET", "/api/business/tasks", business, nil, 200, &counts)
	if counts.Total != 10 {
		t.Fatal("expected 5 drafts and 5 published cards")
	}
	call(t, h, "POST", "/api/tasks", "", nil, 401, nil)
	call(t, h, "POST", "/api/tasks", team, nil, 403, nil)
	call(t, h, "POST", "/api/tasks", business, nil, 400, nil)
	var task tasks.Task
	call(t, h, "POST", "/api/tasks", business, map[string]any{"draft": "Our shop needs better stock planning", "title": "Stock planning", "industry": "retail"}, 201, &task)
	path := "/api/tasks/" + task.ID
	call(t, h, "GET", path, "", nil, 404, nil)
	call(t, h, "GET", path, stranger, nil, 404, nil)
	call(t, h, "PUT", path+"/card", stranger, map[string]any{"revision": task.Revision}, 403, nil)
	initial := map[string]rating.Field{}
	for _, key := range []string{"context", "need", "dataSource", "deliverable"} {
		initial[key] = rating.Field{Value: str("Specific business requirement"), Confirmed: true}
	}
	call(t, h, "PUT", path+"/card", business, tasks.EditInput{Revision: task.Revision, Fields: initial}, 200, &task)
	if task.Preview.Total != 40 || task.ConfirmedSnapshot != nil {
		t.Fatal("preview must not be confirmed")
	}
	for _, f := range task.WorkingFields {
		if f.Confirmed {
			t.Fatal("client confirmation trusted")
		}
	}
	call(t, h, "POST", path+"/publish", business, map[string]any{"revision": task.Revision}, 409, nil)
	call(t, h, "POST", path+"/confirm", business, tasks.ConfirmInput{Revision: 1, Reviewed: true, FieldKeys: []string{}}, 409, nil)
	call(t, h, "POST", path+"/confirm", business, tasks.ConfirmInput{Revision: task.Revision, Reviewed: false, FieldKeys: []string{}}, 422, nil)
	call(t, h, "POST", path+"/confirm", business, tasks.ConfirmInput{Revision: task.Revision, Reviewed: true, FieldKeys: []string{"contact"}}, 422, nil)
	call(t, h, "POST", path+"/confirm", business, tasks.ConfirmInput{Revision: task.Revision, Reviewed: true, FieldKeys: []string{"context", "need", "dataSource", "deliverable"}}, 200, &task)
	if task.ConfirmedSnapshot.Total != 40 || task.ConfirmedRevision == nil || *task.ConfirmedRevision != task.Revision {
		t.Fatal("bad confirmation")
	}
	call(t, h, "POST", path+"/publish", business, map[string]any{"revision": task.Revision}, 200, &task)
	call(t, h, "POST", path+"/publish", business, map[string]any{"revision": task.Revision}, 200, &task)
	publishedAt := *task.PublishedAt
	var public tasks.PublicTask
	call(t, h, "GET", path, "", nil, 200, &public)
	if public.Total != 40 {
		t.Fatal("published score")
	}
	response := request(h, "GET", path, team, nil).Body.String()
	for _, private := range []string{"\"draft\"", "\"answers\"", "\"ownerId\"", "\"workingFields\""} {
		if strings.Contains(response, private) {
			t.Fatalf("private data leaked: %s", private)
		}
	}
	full := fields95()
	call(t, h, "PUT", path+"/card", business, tasks.EditInput{Revision: task.Revision, Title: str("Improved stock planner"), Industry: str("services"), Fields: full}, 200, &task)
	if task.Preview.Total != 95 {
		t.Fatalf("preview=%d", task.Preview.Total)
	}
	call(t, h, "GET", path, "", nil, 200, &public)
	if public.Total != 40 || public.Title != "Stock planning" || public.Industry != "retail" {
		t.Fatal("working edits leaked into public card")
	}
	call(t, h, "GET", "/api/tasks?industry=retail", "", nil, 200, &list)
	found := false
	for _, p := range list.Items {
		if p.ID == task.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("catalog filtered on unconfirmed industry")
	}
	keys := []string{}
	for _, d := range rating.Definitions {
		if d.Key != "feedbackFormat" {
			keys = append(keys, d.Key)
		}
	}
	call(t, h, "POST", path+"/confirm", business, tasks.ConfirmInput{Revision: task.Revision, Reviewed: true, FieldKeys: keys}, 200, &task)
	if task.ConfirmedSnapshot.Total != 95 || !task.PublishedAt.Equal(publishedAt) {
		t.Fatal("reconfirmation must update score but preserve first publication timestamp")
	}
	call(t, h, "GET", "/api/tasks", "", nil, 200, &list)
	if list.Items[0].ID != task.ID {
		t.Fatal("95-point card should lead catalog")
	}
	call(t, h, "GET", "/api/tasks?readiness=draft", "", nil, 200, &list)
	if list.Total != 2 {
		t.Fatal("low-score cards missing")
	}
	call(t, h, "GET", "/api/tasks?readiness=invalid", "", nil, 400, nil)
	call(t, h, "GET", "/api/tasks?offset=999", "", nil, 200, &list)
	if list.Total != 6 || list.Items == nil || len(list.Items) != 0 {
		t.Fatal("empty page/total incorrect")
	}
	call(t, h, "GET", "/api/tasks?industry=absent", "", nil, 200, &list)
	if list.Total != 0 || list.Items == nil {
		t.Fatal("empty filter")
	}
	var aiResult struct {
		Mode      string
		Questions []any
	}
	call(t, h, "POST", "/api/ai/analyze", business, map[string]any{"stage": "clarify", "sources": []map[string]string{{"id": "draft", "text": "Shop planning"}}, "currentFields": map[string]any{}}, 200, &aiResult)
	if aiResult.Mode != "fallback" || len(aiResult.Questions) < 3 {
		t.Fatal("fallback unavailable")
	}
	// Exercise the same handler through a real HTTP connection, then reopen the DB.
	httpServer := httptest.NewServer(h)
	resp, err := http.Get(httpServer.URL + "/api/tasks/" + task.ID)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	httpServer.Close()
	if resp.StatusCode != 200 {
		t.Fatal("HTTP smoke failed")
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	fresh, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	call(t, handler(fresh), "GET", path, "", nil, 200, &public)
	if public.Total != 95 {
		t.Fatal("score did not persist across database reopen")
	}
	t.Log("Verified: 5/5/5/5 seed data, 40 -> 95, public snapshot isolation, filters, fallback, real HTTP and persistence")
}
func TestMilestoneTransactions(t *testing.T) {
	db, dsn := testutil.Postgres(t)
	loadSeeds(t, db)
	h := handler(db)
	// The lowest-rated published card still accepts proposals.
	path := "/api/tasks/" + seed.ID("task", 5) + "/proposals"
	input := map[string]any{"idea": "Build a prototype", "plan": []string{"Inspect data", "Build and verify"}, "durationDays": 7, "prototypeUrl": "https://example.org/demo"}
	var first, second offers.Proposal
	call(t, h, "POST", path, team, input, 201, &first)
	call(t, h, "POST", path, team2, input, 201, &second)
	var proposals struct {
		Items []offers.Proposal
		Total int
	}
	call(t, h, "GET", path, team, nil, 200, &proposals)
	if proposals.Total != 1 || proposals.Items[0].TeamID != first.TeamID {
		t.Fatal("team sees another team's proposal")
	}
	call(t, h, "GET", path, stranger, nil, 403, nil)
	milestonePath := "/api/proposals/" + first.ID + "/milestone"
	result := map[string]string{"resultText": "The prototype passes the agreed checks", "evidenceUrl": "https://example.org/result"}
	call(t, h, "POST", milestonePath, team, result, 409, nil)
	for _, p := range []offers.Proposal{first, second} {
		call(t, h, "POST", "/api/proposals/"+p.ID+"/decision", business, map[string]string{"decision": "accepted"}, 200, nil)
	}
	call(t, h, "POST", "/api/proposals/"+first.ID+"/decision", business, map[string]string{"decision": "accepted"}, 200, nil)
	call(t, h, "POST", "/api/proposals/"+first.ID+"/decision", business, map[string]string{"decision": "rejected"}, 409, nil)
	call(t, h, "POST", milestonePath, team2, result, 403, nil)
	var points struct{ Points int }
	teamPath := "/api/teams/" + first.TeamID
	call(t, h, "GET", teamPath, "", nil, 200, &points)
	if points.Points != 0 {
		t.Fatal("acceptance incorrectly awarded points")
	}
	var milestone, retry offers.Milestone
	call(t, h, "POST", milestonePath, team, result, 200, &milestone)
	call(t, h, "POST", milestonePath, team, result, 200, &retry)
	if milestone.ID != retry.ID {
		t.Fatal("duplicate milestone")
	}
	call(t, h, "POST", milestonePath, team, map[string]string{"resultText": "Changed result", "evidenceUrl": "https://example.org/result"}, 409, nil)
	confirm := "/api/milestones/" + milestone.ID + "/confirm"
	call(t, h, "POST", confirm, stranger, map[string]bool{"confirmed": true}, 403, nil)
	call(t, h, "POST", confirm, team, map[string]bool{"confirmed": true}, 403, nil)
	call(t, h, "POST", confirm, business, map[string]bool{"confirmed": false}, 422, nil)
	// A DB failure after status update must roll back the whole confirmation.
	_, err := db.Exec(`CREATE FUNCTION fail_test_award() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected private failure'; END $$;
 CREATE TRIGGER fail_test_award BEFORE INSERT ON progress_awards FOR EACH ROW EXECUTE FUNCTION fail_test_award()`)
	if err != nil {
		t.Fatal(err)
	}
	failed := request(h, "POST", confirm, business, map[string]bool{"confirmed": true})
	if failed.Code != 500 || strings.Contains(failed.Body.String(), "injected") {
		t.Fatal("incorrect failure response")
	}
	var status string
	if err = db.QueryRow(`SELECT status FROM milestones WHERE id=$1`, milestone.ID).Scan(&status); err != nil || status != "submitted" {
		t.Fatal("confirmation did not roll back")
	}
	if _, err = db.Exec(`DROP TRIGGER fail_test_award ON progress_awards; DROP FUNCTION fail_test_award()`); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan string, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := request(h, "POST", confirm, business, map[string]bool{"confirmed": true})
			if w.Code != 200 {
				failures <- w.Body.String()
			}
		}()
	}
	wg.Wait()
	close(failures)
	for e := range failures {
		t.Error(e)
	}
	call(t, h, "GET", teamPath, "", nil, 200, &points)
	if points.Points != 10 {
		t.Fatalf("points=%d want 10", points.Points)
	}
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM progress_awards WHERE milestone_id=$1`, milestone.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("award duplicated")
	}
	call(t, h, "GET", path, team, nil, 200, &proposals)
	if proposals.Items[0].Milestone == nil || proposals.Items[0].Milestone.Points != 10 {
		t.Fatal("milestone missing from proposal view")
	}
	fresh, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	call(t, handler(fresh), "GET", teamPath, "", nil, 200, &points)
	if points.Points != 10 {
		t.Fatal("points not durable")
	}
	t.Log("Verified: low-score proposals, multiple accepted teams, team visibility, final decisions, atomic rollback, 16 concurrent confirmations -> one 10-point award")
}
func TestConcurrentRevisionAndZeroScore(t *testing.T) {
	db, _ := testutil.Postgres(t)
	h := handler(db)
	var task tasks.Task
	call(t, h, "POST", "/api/tasks", business, map[string]string{"draft": "Unclear business request", "title": "Needs clarification", "industry": "retail"}, 201, &task)
	path := "/api/tasks/" + task.ID
	call(t, h, "POST", path+"/confirm", business, tasks.ConfirmInput{Revision: task.Revision, Reviewed: true, FieldKeys: []string{}}, 200, &task)
	call(t, h, "POST", path+"/publish", business, map[string]int{"revision": task.Revision}, 200, &task)
	if task.ConfirmedSnapshot.Total != 0 {
		t.Fatal("zero-score publication")
	}
	rev := task.Revision
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := request(h, "PUT", path+"/card", business, tasks.EditInput{Revision: rev, Title: str(fmt.Sprintf("Edit %d", i))})
			codes <- w.Code
		}(i)
	}
	wg.Wait()
	close(codes)
	seen := map[int]int{}
	for code := range codes {
		seen[code]++
	}
	if seen[200] != 1 || seen[409] != 1 {
		t.Fatalf("revision race: %v", seen)
	}
}
