package workflow

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"
)

// GeminiClient wraps the Google AI Studio Gemini API client
type GeminiClient struct {
	client    *genai.Client
	modelName string
	isLive    bool
}

// NewGeminiClient initializes a Gemini client with Google AI Studio API key
func NewGeminiClient(ctx context.Context) (*GeminiClient, error) {
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-3.8-flash"
	}

	if apiKey == "" {
		log.Println("─────────────────────────────────────────────────────────────────────────")
		log.Println("⚠️  GEMINI_API_KEY not set!")
		log.Println("👉 Using Simulated LLM Mode (Rich mock responses for demo).")
		log.Println("💡 To use real Google AI Studio: set GEMINI_API_KEY in .env or your shell.")
		log.Println("   Get a free key at: https://aistudio.google.com/app/apikey")
		log.Println("─────────────────────────────────────────────────────────────────────────")
		return &GeminiClient{
			modelName: modelName + " (simulated)",
			isLive:    false,
		}, nil
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Google GenAI client: %w", err)
	}

	log.Printf("🤖 Google GenAI client initialized with model: %s (LIVE AI Studio)\n", modelName)
	return &GeminiClient{
		client:    client,
		modelName: modelName,
		isLive:    true,
	}, nil
}

// GenerateText sends a prompt to Gemini (or simulates if live key not present)
func (g *GeminiClient) GenerateText(ctx context.Context, prompt string) (string, error) {
	if !g.isLive {
		// Realistic delay to simulate network & LLM generation latency
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return "", ctx.Err()
		}

		if strings.Contains(prompt, "Polish and format") {
			return "## Executive Summary\n\nThis refined report synthesizes the key insights gathered during the initial AI draft phase. Orchestrating LLM workloads with Temporal provides fault-tolerance, state durability, and human-in-the-loop workflows without data loss.\n\n## Refined In-Depth Analysis\n\n1. **Deterministic State Replay**: Temporal decouples state persistence from business logic.\n2. **Zero-Token Waste**: When intermediate steps crash, completed LLM steps are never re-billed.\n3. **Production Reliability**: Microservices calling AI APIs gain automatic backoff, jitter, and retry policies.\n\n*(Generated via Simulated LLM Mode. Set GEMINI_API_KEY for live Google AI Studio generation)*", nil
		}

		return "### AI Draft Outline\n\n- Key Point 1: LLM APIs are inherently prone to transient timeouts and rate limits.\n- Key Point 2: Temporal workflows persist the exact outputs of each LLM call as event history.\n- Key Point 3: Developers can patch bugs live while Temporal retries the activity in the background.\n\nDetailed Draft Paragraphs:\nModern AI applications are shifting from simple prompt-response patterns to long-running agentic pipelines. In these pipelines, executing multiple sequential LLM calls can take dozens of seconds and cost non-trivial token budgets. Temporal acts as the durable execution layer that guarantees progress.\n\n*(Generated via Simulated LLM Mode. Set GEMINI_API_KEY for live Google AI Studio generation)*", nil
	}

	resp, err := g.client.Models.GenerateContent(ctx, g.modelName, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("Gemini API call failed: %w", err)
	}

	return resp.Text(), nil
}

// GetModelName returns the active model name
func (g *GeminiClient) GetModelName() string {
	return g.modelName
}
