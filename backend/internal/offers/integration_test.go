package offers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/lib/pq"

	"vibe-moggers/backend/internal/database"
	"vibe-moggers/backend/internal/httpapi"
	"vibe-moggers/backend/internal/offers"
	"vibe-moggers/backend/internal/server"
	"vibe-moggers/backend/internal/tasks"
	"vibe-moggers/backend/internal/teams"
)

const owner = "10000000-0000-4000-8000-000000000001"
const stranger = "10000000-0000-4000-8000-000000000002"

type fixture struct {
	db      *sql.DB
	handler http.Handler
	dsn     string
}

func handler(db *sql.DB, access tasks.Access) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return server.New(db, logger, teams.NewHandler(db, logger).Register,
		offers.NewHandler(offers.NewStore(db, access), logger).Register)
}

func setup(t *testing.T) fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests in an isolated schema")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a postgres:// URL")
	}
	admin, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	id, err := httpapi.NewID()
	if err != nil {
		t.Fatal(err)
	}
	schema := "block2_test_" + strings.ReplaceAll(id, "-", "")
	if _, err := admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`); err != nil {
			t.Error(err)
		}
	})
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for i := 0; i < 2; i++ {
		if err := database.Migrate(context.Background(), db); err != nil {
			t.Fatal(err)
		}
	}
	var seeded int
	if err := db.QueryRow(`SELECT count(*) FROM teams`).Scan(&seeded); err != nil || seeded != 5 {
		t.Fatalf("migrations must seed exactly five teams once: count=%d, err=%v", seeded, err)
	}
	return fixture{db: db, handler: handler(db, tasks.SQLAccess{}), dsn: parsed.String()}
}

func request(h http.Handler, method, path, actor, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if actor != "" {
		r.Header.Set("X-Demo-Actor", actor)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func (f fixture) call(t *testing.T, method, path, actor, body string, want int) []byte {
	t.Helper()
	w := request(f.handler, method, path, actor, body)
	if w.Code != want {
		t.Fatalf("%s %s: HTTP %d, want %d: %s", method, path, w.Code, want, w.Body)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected JSON response")
	}
	return w.Body.Bytes()
}

func (f fixture) task(t *testing.T, published bool) string {
	t.Helper()
	id, err := httpapi.NewID()
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.db.Exec(`INSERT INTO tasks(id, owner_id, status, published_at)
		VALUES ($1, $2, CASE WHEN $3 THEN 'published' ELSE 'draft' END,
		CASE WHEN $3 THEN CURRENT_TIMESTAMP ELSE NULL END)`, id, owner, published)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (f fixture) team(t *testing.T) string {
	t.Helper()
	body := f.call(t, "POST", "/api/teams", "", `{"name":"Student team"}`, 201)
	var response struct {
		Team teams.Team `json:"team"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if response.Team.Points != 0 || response.Team.Name != "Student team" || !httpapi.UUID(response.Team.ID) {
		t.Fatalf("unexpected profile: %+v", response.Team)
	}
	return response.Team.ID
}

func (f fixture) offer(t *testing.T, taskID, teamID string, legacy bool) offers.Offer {
	t.Helper()
	path := "/api/tasks/" + taskID + "/offers"
	if legacy {
		path = "/api/tasks/" + taskID + "/proposals"
	}
	body := f.call(t, "POST", path, "team:"+teamID,
		fmt.Sprintf(`{"team_id":%q,"solution_idea":"Сравнение услуг","plan":"1. Собрать данные\n2. Сделать прототип\n3. Проверить","timeline":"One week","prototype_link":"https://example.org/demo"}`, teamID), 201)
	var response struct {
		Offer    offers.Offer `json:"offer"`
		Proposal offers.Offer `json:"proposal"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	result, want := response.Offer, "pending"
	if legacy {
		result, want = response.Proposal, "submitted"
	}
	if result.Status != want || result.TeamID != teamID || result.TaskID != taskID || result.DecidedAt != nil {
		t.Fatalf("unexpected new offer: %+v", result)
	}
	return result
}

func (f fixture) decide(t *testing.T, id, status string) {
	t.Helper()
	f.call(t, "PATCH", "/api/offers/"+id+"/status", "business:"+owner, fmt.Sprintf(`{"status":%q}`, status), 200)
}

func (f fixture) points(t *testing.T, id string, want int) {
	t.Helper()
	body := f.call(t, "GET", "/api/teams/"+id, "", "", 200)
	var response struct {
		Team teams.Team `json:"team"`
	}
	if err := json.Unmarshal(body, &response); err != nil || response.Team.Points != want {
		t.Fatalf("team points: %s, want %d, err=%v", body, want, err)
	}
}

func (f fixture) state(t *testing.T, taskID, want string) {
	t.Helper()
	var publication, work string
	if err := f.db.QueryRow(`SELECT status, work_status FROM tasks WHERE id = $1`, taskID).Scan(&publication, &work); err != nil {
		t.Fatal(err)
	}
	if publication != "published" || work != want {
		t.Fatalf("task status=%s, work_status=%s; want published/%s", publication, work, want)
	}
}

func TestBlock2Postgres(t *testing.T) {
	f := setup(t)
	t.Run("multiple teams, retries, aliases and persistence", func(t *testing.T) {
		task := f.task(t, true)
		a, b := f.team(t), f.team(t)
		first, second := f.offer(t, task, a, false), f.offer(t, task, b, true)
		f.decide(t, first.ID, "accepted")
		for i := 0; i < 3; i++ {
			f.decide(t, first.ID, "accepted")
		}
		f.call(t, "POST", "/api/proposals/"+second.ID+"/accept", "business:"+owner, "", 200)
		f.points(t, a, 10)
		f.points(t, b, 10)
		f.state(t, task, "in_progress")
		third := f.offer(t, task, a, false) // Work in progress still accepts offers.
		body := f.call(t, "GET", "/api/tasks/"+task+"/offers?status=accepted&limit=1", "business:"+owner, "", 200)
		var list struct {
			Items []offers.Offer `json:"items"`
			Total int            `json:"total"`
		}
		if err := json.Unmarshal(body, &list); err != nil || list.Total != 2 || len(list.Items) != 1 {
			t.Fatalf("filtered pagination: %s, err=%v", body, err)
		}
		body = f.call(t, "GET", "/api/tasks/"+task+"/proposals?status=submitted", "business:"+owner, "", 200)
		if err := json.Unmarshal(body, &list); err != nil || list.Total != 1 || list.Items[0].ID != third.ID || list.Items[0].Status != "submitted" {
			t.Fatalf("other offers must remain pending: %s, err=%v", body, err)
		}
		body = f.call(t, "GET", "/api/tasks/"+task+"/offers?offset=999", "business:"+owner, "", 200)
		if err := json.Unmarshal(body, &list); err != nil || list.Total != 3 || list.Items == nil || len(list.Items) != 0 {
			t.Fatalf("empty page must keep total: %s", body)
		}
		freshDB, err := database.Open(f.dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer freshDB.Close()
		fresh := fixture{db: freshDB, handler: handler(freshDB, tasks.SQLAccess{})}
		fresh.points(t, a, 10)
		fresh.state(t, task, "in_progress")
		fresh.decide(t, first.ID, "accepted")
		fresh.points(t, a, 10)
	})
	t.Run("reject all and correct decisions without repeat awards", func(t *testing.T) {
		task, team := f.task(t, true), f.team(t)
		a, b := f.offer(t, task, team, false), f.offer(t, task, team, false)
		f.decide(t, a.ID, "rejected")
		f.decide(t, b.ID, "rejected")
		f.points(t, team, 0)
		f.state(t, task, "open")
		f.decide(t, a.ID, "accepted")
		f.call(t, "POST", "/api/proposals/"+a.ID+"/reject", "business:"+owner, "", 200)
		f.state(t, task, "open")
		f.decide(t, a.ID, "accepted")
		f.points(t, team, 10)
		f.state(t, task, "in_progress")
	})
	t.Run("ownership, publication and validation", func(t *testing.T) {
		task, team := f.task(t, true), f.team(t)
		offer := f.offer(t, task, team, false)
		for _, actor := range []string{"business:" + stranger, "team:" + team} {
			f.call(t, "GET", "/api/tasks/"+task+"/offers", actor, "", 403)
			f.call(t, "PATCH", "/api/offers/"+offer.ID+"/status", actor, `{"status":"accepted"}`, 403)
		}
		f.call(t, "PATCH", "/api/offers/"+offer.ID+"/status", "business:"+owner, `{"status":"pending"}`, 422)
		f.call(t, "GET", "/api/tasks/"+task+"/offers?limit=0", "business:"+owner, "", 400)
		f.call(t, "GET", "/api/tasks/"+task+"/offers?status=unknown", "business:"+owner, "", 400)
		valid := `{"solution_idea":"Idea","plan":"Build and test","timeline":"Week"}`
		f.call(t, "POST", "/api/tasks/"+f.task(t, false)+"/offers", "team:"+team, valid, 409)
		f.call(t, "POST", "/api/tasks/"+task+"/offers", "team:"+stranger, valid, 404)
		f.call(t, "POST", "/api/tasks/"+stranger+"/offers", "team:"+team, valid, 404)
		f.call(t, "PATCH", "/api/offers/"+stranger+"/status", "business:"+owner, `{"status":"accepted"}`, 404)
		f.call(t, "GET", "/api/teams/"+stranger, "", "", 404)
		f.call(t, "POST", "/api/teams", "", `{"name":" ","points":100}`, 400)
		f.call(t, "POST", "/api/teams", "", `{"name":" "}`, 422)
		f.call(t, "POST", "/api/teams", "", `{"name":"bad\u0000name"}`, 422)
		f.call(t, "GET", "/api/teams?offset=999999", "", "", 200)
		f.points(t, team, 0)
		f.state(t, task, "open")
	})
	t.Run("concurrent acceptance is awarded once per offer", func(t *testing.T) {
		task, team := f.task(t, true), f.team(t)
		a, b := f.offer(t, task, team, false), f.offer(t, task, team, false)
		var wg sync.WaitGroup
		failures := make(chan string, 16)
		for i := 0; i < 16; i++ {
			id := a.ID
			if i%2 == 1 {
				id = b.ID
			}
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				w := request(f.handler, "PATCH", "/api/offers/"+id+"/status", "business:"+owner, `{"status":"accepted"}`)
				if w.Code != 200 {
					failures <- w.Body.String()
				}
			}(id)
		}
		wg.Wait()
		close(failures)
		for failure := range failures {
			t.Error(failure)
		}
		f.points(t, team, 20)
		f.state(t, task, "in_progress")
		var count int
		if err := f.db.QueryRow(`SELECT count(*) FROM point_awards WHERE team_id = $1`, team).Scan(&count); err != nil || count != 2 {
			t.Fatalf("award count = %d, err=%v", count, err)
		}
	})
	t.Run("late failure rolls back status award points and task", func(t *testing.T) {
		task, team := f.task(t, true), f.team(t)
		offer := f.offer(t, task, team, false)
		broken := fixture{db: f.db, handler: handler(f.db, failingTaskAccess{})}
		body := broken.call(t, "PATCH", "/api/offers/"+offer.ID+"/status", "business:"+owner, `{"status":"accepted"}`, 500)
		if strings.Contains(string(body), "injected") {
			t.Fatal("internal error leaked")
		}
		f.points(t, team, 0)
		f.state(t, task, "open")
		var status string
		var awards int
		if err := f.db.QueryRow(`SELECT status FROM offers WHERE id=$1`, offer.ID).Scan(&status); err != nil || status != "pending" {
			t.Fatalf("offer rollback: %s, %v", status, err)
		}
		if err := f.db.QueryRow(`SELECT count(*) FROM point_awards WHERE offer_id=$1`, offer.ID).Scan(&awards); err != nil || awards != 0 {
			t.Fatalf("award rollback: %d, %v", awards, err)
		}
		f.decide(t, offer.ID, "accepted")
		f.points(t, team, 10)
	})
}

type failingTaskAccess struct{ tasks.SQLAccess }

func (a failingTaskAccess) RefreshWorkStatus(ctx context.Context, tx *sql.Tx, id string) error {
	if err := a.SQLAccess.RefreshWorkStatus(ctx, tx, id); err != nil {
		return err
	}
	return errors.New("injected failure after task update")
}
