package workflow

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
)

// AIActivities holds dependencies for Temporal activities (e.g. Gemini AI Client)
type AIActivities struct {
	gemini *GeminiClient
}

// NewAIActivities constructor
func NewAIActivities(gemini *GeminiClient) *AIActivities {
	return &AIActivities{
		gemini: gemini,
	}
}

// GenerateDraftActivity is Activity 1: Uses Gemini to generate initial research & draft outline.
// This is an expensive/time-consuming LLM call. Once completed, Temporal records the output in
// the workflow execution history so it NEVER needs to run again if later activities fail!
func (a *AIActivities) GenerateDraftActivity(ctx context.Context, input ResearchDraftInput) (*ResearchDraftOutput, error) {
	activityInfo := activity.GetInfo(ctx)
	log.Printf("\n[Activity 1: Draft] 🚀 Generating draft for topic: %q (Attempt #%d, WorkflowID: %s)",
		input.Topic, activityInfo.Attempt, activityInfo.WorkflowExecution.ID)

	prompt := fmt.Sprintf(
		"You are an expert AI research assistant. Provide an in-depth, structured research draft on the following topic:\n"+
			"Topic: %s\n"+
			"Target Audience: %s\n\n"+
			"Include key points, detailed analysis, and architectural highlights.",
		input.Topic, input.TargetAudience,
	)

	draftText, err := a.gemini.GenerateText(ctx, prompt)
	if err != nil {
		log.Printf("[Activity 1: Draft] ❌ Failed to call LLM: %v", err)
		return nil, fmt.Errorf("GenerateDraftActivity failed: %w", err)
	}

	log.Printf("[Activity 1: Draft] ✅ Draft successfully created! (%d characters)", len(draftText))
	log.Println("[Activity 1: Draft] 💾 [TEMPORAL PERSISTENCE] Output is now safely saved in Temporal event history.")

	return &ResearchDraftOutput{
		Topic:       input.Topic,
		RawDraft:    draftText,
		GeneratedAt: time.Now(),
		ModelUsed:   a.gemini.GetModelName(),
		ExecutionID: activityInfo.WorkflowExecution.ID,
	}, nil
}

// PolishReportActivity is Activity 2: Takes the raw draft and uses Gemini to polish, format,
// and synthesize an executive summary.
//
// ⚠️ THIS ACTIVITY CONTAINS THE DELIBERATE ERROR SWITCH FOR THE DEMO!
func (a *AIActivities) PolishReportActivity(ctx context.Context, input PolishReportInput) (*PolishReportOutput, error) {
	activityInfo := activity.GetInfo(ctx)
	log.Printf("\n[Activity 2: Polish] 🎨 Polishing and refining draft for topic: %q (Attempt #%d)",
		input.Draft.Topic, activityInfo.Attempt)

	// ==================================================================================
	// 💥💥💥 DELIBERATE ERROR SWITCH (FOR DEMO PURPOSES) 💥💥💥
	//
	// TO SEE TEMPORAL STATE PERSISTENCE IN ACTION:
	// 1. UNCOMMENT the return error statement below:
	//
	// return nil, fmt.Errorf("💥 DELIBERATE DEMO ERROR: Database connection failed / API rate-limit exceeded! Notice that Activity 1 (Draft) is ALREADY SAVED in Temporal history and will NOT re-run. Comment this line back out and restart the worker to recover!")
	//
	// 2. Run the workflow. Activity 1 finishes and saves its output.
	// 3. Activity 2 will fail here and enter Temporal's automatic retry loop.
	// 4. Open Temporal Web UI (http://localhost:8233) to inspect the saved state & error.
	// 5. Comment the line back out, save, and restart the worker.
	// 6. Temporal resumes and completes Activity 2 and 3 WITHOUT re-running Activity 1!
	// ==================================================================================

	// Alternate runtime switch via env variable TRIGGER_ERROR=true (for headless scripts/demos)
	if os.Getenv("TRIGGER_ERROR") == "true" {
		log.Println("[Activity 2: Polish] 💥 TRIGGER_ERROR=true detected in environment!")
		return nil, fmt.Errorf("💥 DELIBERATE DEMO ERROR (triggered via TRIGGER_ERROR=true): Simulated crash in PolishReportActivity! Temporal is saving state and retrying...")
	}

	prompt := fmt.Sprintf(
		"You are a principal technical editor. Given this research draft, refine it into an executive-ready report.\n"+
			"Include an Executive Summary, Key Insights, and clear formatted markdown sections.\n\n"+
			"Original Topic: %s\n"+
			"Draft Content:\n%s",
		input.Draft.Topic, input.Draft.RawDraft,
	)

	polishedText, err := a.gemini.GenerateText(ctx, prompt)
	if err != nil {
		log.Printf("[Activity 2: Polish] ❌ Failed to polish report: %v", err)
		return nil, fmt.Errorf("PolishReportActivity failed: %w", err)
	}

	execSummary := "Executive Summary generated based on synthesized research."
	lines := strings.Split(polishedText, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "## Executive Summary") || strings.HasPrefix(trimmed, "Executive Summary:") {
			execSummary = trimmed
			break
		}
	}

	log.Printf("[Activity 2: Polish] ✅ Report successfully polished! (%d characters)", len(polishedText))
	return &PolishReportOutput{
		ExecutiveSummary: execSummary,
		RefinedContent:   polishedText,
		PolishedAt:       time.Now(),
		ModelUsed:        a.gemini.GetModelName(),
	}, nil
}

// PublishFileActivity is Activity 3: Saves the final polished report to disk.
func (a *AIActivities) PublishFileActivity(ctx context.Context, input PublishInput) (*PublishOutput, error) {
	activityInfo := activity.GetInfo(ctx)
	log.Printf("\n[Activity 3: Publish] 📝 Publishing report to file (Attempt #%d)...", activityInfo.Attempt)

	outDir := filepath.Dir(input.OutputFilePath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	content := fmt.Sprintf(
		"# %s\n\n"+
			"> **Generated via Temporal + Google AI Studio Gemini Workflow**  \n"+
			"> **Model:** %s | **Date:** %s  \n\n"+
			"%s\n\n"+
			"---\n"+
			"### Workflow Metadata\n"+
			"- **Workflow Execution ID:** %s\n"+
			"- **Draft Created:** %s\n"+
			"- **Polished At:** %s\n",
		input.Topic,
		input.PolishedReport.ModelUsed,
		time.Now().Format("2006-01-02 15:04:05 MST"),
		input.PolishedReport.RefinedContent,
		activityInfo.WorkflowExecution.ID,
		input.Draft.GeneratedAt.Format(time.RFC3339),
		input.PolishedReport.PolishedAt.Format(time.RFC3339),
	)

	if err := os.WriteFile(input.OutputFilePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write published report to %s: %w", input.OutputFilePath, err)
	}

	fileInfo, err := os.Stat(input.OutputFilePath)
	var fileSize int64
	if err == nil {
		fileSize = fileInfo.Size()
	}

	log.Printf("[Activity 3: Publish] 🎉 Successfully published to: %s (%d bytes)", input.OutputFilePath, fileSize)
	return &PublishOutput{
		FilePath:    input.OutputFilePath,
		PublishedAt: time.Now(),
		FileSize:    fileSize,
	}, nil
}
