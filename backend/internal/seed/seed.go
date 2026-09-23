package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"vibe-moggers/backend/internal/rating"
	"vibe-moggers/backend/internal/tasks"
)

const OwnerID = "10000000-0000-4000-8000-000000000001"

type Fixtures struct {
	ScoreWeights map[string]int                        `json:"scoreWeights"`
	Drafts       []struct{ ID, Text, Industry string } `json:"drafts"`
	Cards        []struct {
		ID, OwnerID, Title, Industry string
		Fields                       map[string]rating.Field
		ExpectedScore                int
		ExpectedReadiness            string
		PublishedAt                  time.Time
	} `json:"cards"`
	Teams []struct {
		ID, Name                        string
		Interests, Skills, Technologies []string
	} `json:"teams"`
	Proposals []struct {
		ID, TaskID, TeamID, Idea string
		Plan                     []string
		DurationDays             int
		PrototypeURL, Status     string
		CreatedAt                time.Time
	} `json:"proposals"`
}

func ID(kind string, n int) string {
	prefix := map[string]string{"team": "00000000", "task": "20000000", "draft": "30000000", "proposal": "40000000"}[kind]
	return fmt.Sprintf("%s-0000-4000-8000-%012d", prefix, n)
}
func fixtureID(raw, kind, prefix string) (string, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(raw, prefix))
	if err != nil || n < 1 || n > 5 || raw != fmt.Sprintf("%s%d", prefix, n) {
		return "", fmt.Errorf("invalid fixture ID %q", raw)
	}
	return ID(kind, n), nil
}
func Load(ctx context.Context, db *sql.DB, reader io.Reader) error {
	var f Fixtures
	if err := json.NewDecoder(io.LimitReader(reader, 2<<20)).Decode(&f); err != nil {
		return err
	}
	if len(f.Drafts) != 5 || len(f.Cards) != 5 || len(f.Teams) != 5 || len(f.Proposals) != 5 {
		return fmt.Errorf("expected exactly five of each fixture type")
	}
	for _, d := range rating.Definitions {
		if f.ScoreWeights[d.Key] != d.Weight {
			return fmt.Errorf("fixture weight mismatch: %s", d.Key)
		}
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(1357220)`); err != nil {
		return err
	}
	for _, team := range f.Teams {
		id, e := fixtureID(team.ID, "team", "team-")
		if e != nil {
			return e
		}
		interests, _ := json.Marshal(team.Interests)
		skills, _ := json.Marshal(team.Skills)
		tech, _ := json.Marshal(team.Technologies)
		_, err = tx.ExecContext(ctx, `INSERT INTO teams(id,name,interests,skills,technologies) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO NOTHING`, id, team.Name, string(interests), string(skills), string(tech))
		if err != nil {
			return err
		}
	}
	empty, _ := json.Marshal(rating.EmptyFields())
	for i, d := range f.Drafts {
		id, e := fixtureID(d.ID, "draft", "draft-")
		if e != nil {
			return e
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO tasks(id,owner_id,title,industry,draft,working_fields) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(id) DO NOTHING`, id, OwnerID, fmt.Sprintf("Draft %d", i+1), d.Industry, d.Text, string(empty))
		if err != nil {
			return err
		}
	}
	for _, c := range f.Cards {
		id, e := fixtureID(c.ID, "task", "seed-task-")
		if e != nil {
			return e
		}
		if c.OwnerID != "demo-business-1" {
			return fmt.Errorf("unknown fixture owner")
		}
		if len(c.Fields) != len(rating.Definitions) {
			return fmt.Errorf("invalid fixture fields: %s", c.ID)
		}
		result := rating.Calculate(c.Fields, false)
		if result.Total != c.ExpectedScore || result.Readiness != c.ExpectedReadiness {
			return fmt.Errorf("fixture %s: score=%d readiness=%s, want %d/%s", c.ID, result.Total, result.Readiness, c.ExpectedScore, c.ExpectedReadiness)
		}
		for key, field := range c.Fields {
			if !rating.Known(key) || field.Confirmed && !rating.Eligible(key, field.Value) {
				return fmt.Errorf("invalid confirmed fixture field %s", key)
			}
		}
		snap := tasks.Snapshot{Title: c.Title, Industry: c.Industry, Fields: c.Fields, Result: result, Timestamp: c.PublishedAt}
		raw, _ := json.Marshal(snap)
		fields, _ := json.Marshal(c.Fields)
		_, err = tx.ExecContext(ctx, `INSERT INTO tasks(id,owner_id,title,industry,draft,working_fields,confirmed_snapshot,revision,confirmed_revision,status,published_at,score,readiness)
   VALUES($1,$2,$3,$4,'Synthetic demonstration card',$5,$6,1,1,'published',$7,$8,$9) ON CONFLICT(id) DO NOTHING`, id, OwnerID, c.Title, c.Industry, string(fields), string(raw), c.PublishedAt, result.Total, result.Readiness)
		if err != nil {
			return err
		}
	}
	for _, p := range f.Proposals {
		id, e := fixtureID(p.ID, "proposal", "proposal-")
		if e != nil {
			return e
		}
		task, e := fixtureID(p.TaskID, "task", "seed-task-")
		if e != nil {
			return e
		}
		team, e := fixtureID(p.TeamID, "team", "team-")
		if e != nil {
			return e
		}
		if p.Status != "pending" || p.DurationDays < 1 || len(p.Plan) == 0 {
			return fmt.Errorf("invalid fixture proposal")
		}
		steps, _ := json.Marshal(p.Plan)
		_, err = tx.ExecContext(ctx, `INSERT INTO offers(id,task_id,team_id,solution_idea,plan,timeline,prototype_link,status,created_at,plan_steps,duration_days)
   VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT(id) DO NOTHING`, id, task, team, p.Idea, strings.Join(p.Plan, "\n"), fmt.Sprintf("%d days", p.DurationDays), p.PrototypeURL, p.Status, p.CreatedAt, string(steps), p.DurationDays)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
