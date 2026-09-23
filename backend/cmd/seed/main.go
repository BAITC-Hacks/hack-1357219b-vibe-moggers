package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
	"vibe-moggers/backend/internal/config"
	"vibe-moggers/backend/internal/database"
	"vibe-moggers/backend/internal/seed"
)

func run() error {
	path := flag.String("fixtures", "../docs/fixtures/seed.json", "Fixture JSON path")
	flag.Parse()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = database.Migrate(ctx, db); err != nil {
		return err
	}
	f, err := os.Open(*path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = seed.Load(ctx, db, f); err != nil {
		return err
	}
	fmt.Println("Seed complete: five drafts, five published cards, five teams and five proposals present. Existing rows preserved.")
	fmt.Println("Business actor: business:" + seed.OwnerID)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
