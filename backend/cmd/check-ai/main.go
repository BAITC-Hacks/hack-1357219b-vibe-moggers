package main

import (
	"context"
	"fmt"
	"os"
	"vibe-moggers/backend/internal/ai"
	"vibe-moggers/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if cfg.AIMode != "live" {
		fmt.Fprintln(os.Stderr, "Set AI_MODE=live and OPENAI_API_KEY in backend/.env first.")
		os.Exit(1)
	}
	result, err := ai.New(cfg.AIMode, cfg.OpenAIKey, cfg.OpenAIModel).Analyze(context.Background(), ai.Input{Stage: "clarify", Sources: []ai.Source{{ID: "draft", Text: "Our shop needs a stock planning prototype. We have sales data in CSV."}}})
	if err != nil || result.Mode != "live" {
		fmt.Fprintln(os.Stderr, "Live AI check failed. Verify the key, model access, quota and network. The key was not printed.")
		os.Exit(1)
	}
	fmt.Printf("OpenAI success: model=%s mode=%s questions=%d durationMs=%d\n", cfg.OpenAIModel, result.Mode, len(result.Questions), result.DurationMS)
}
