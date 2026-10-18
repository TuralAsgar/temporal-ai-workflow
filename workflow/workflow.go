package workflow

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowStatus tracks current progress for Temporal Queries
type WorkflowStatus struct {
	Stage          string    `json:"stage"`
	LastUpdated    time.Time `json:"last_updated"`
	DraftAvailable bool      `json:"draft_available"`
	DraftLength    int       `json:"draft_length"`
	ErrorMessage   string    `json:"error_message,omitempty"`
}

// AIResearchWorkflow coordinates the end-to-end multi-stage AI research pipeline
func AIResearchWorkflow(ctx workflow.Context, input AIWorkflowInput) (*AIWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	workflowStartTime := workflow.Now(ctx)
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID

	logger.Info("Starting AIResearchWorkflow", "WorkflowID", workflowID, "Topic", input.Topic)

	// State tracked in workflow memory (reconstructed deterministically on replay)
	currentStatus := WorkflowStatus{
		Stage:       "Initializing",
		LastUpdated: workflow.Now(ctx),
	}

	// Register Temporal Query handler so clients/UI can inspect live workflow state
	err := workflow.SetQueryHandler(ctx, "getStatus", func() (WorkflowStatus, error) {
		return currentStatus, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register getStatus query handler: %w", err)
	}

	// Configure Activity Execution Options & Retry Policy
	// Notice: MaximumAttempts is set to 100 (or unlimited) so if Activity 2 fails deliberately,
	// Temporal will keep retrying every few seconds while preserving Activity 1's state!
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 1.5,
			MaximumInterval:    8 * time.Second,
			MaximumAttempts:    100, // Keeps retrying so you can fix code live without failing workflow
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	var activities *AIActivities

	// -------------------------------------------------------------------------
	// STAGE 1: Research & Draft Generation (LLM Activity 1)
	// -------------------------------------------------------------------------
	currentStatus.Stage = "Stage 1: Generating Draft with Gemini LLM"
	currentStatus.LastUpdated = workflow.Now(ctx)
	logger.Info("Executing Activity 1: GenerateDraftActivity", "Topic", input.Topic)

	var draftOutput ResearchDraftOutput
	err = workflow.ExecuteActivity(ctx, activities.GenerateDraftActivity, ResearchDraftInput{
		Topic:          input.Topic,
		TargetAudience: input.TargetAudience,
	}).Get(ctx, &draftOutput)
	if err != nil {
		currentStatus.Stage = "Failed at Stage 1"
		currentStatus.ErrorMessage = err.Error()
		return nil, fmt.Errorf("GenerateDraftActivity failed: %w", err)
	}

	currentStatus.DraftAvailable = true
	currentStatus.DraftLength = len(draftOutput.RawDraft)
	logger.Info("Activity 1 Succeeded! Output stored in Temporal history.",
		"DraftChars", currentStatus.DraftLength)

	// -------------------------------------------------------------------------
	// STAGE 2: Polish & Executive Summary (LLM Activity 2)
	// ⚠️ IF THE DELIBERATE ERROR IS UNCOMMENTED, EXECUTION PAUSES HERE AND RETRIES!
	// ⚠️ STAGE 1 RESULT (draftOutput) REMAINS COMPLETELY SAFE IN TEMPORAL HISTORY!
	// -------------------------------------------------------------------------
	currentStatus.Stage = "Stage 2: Polishing Report with Gemini LLM"
	currentStatus.LastUpdated = workflow.Now(ctx)
	logger.Info("Executing Activity 2: PolishReportActivity")

	var polishOutput PolishReportOutput
	err = workflow.ExecuteActivity(ctx, activities.PolishReportActivity, PolishReportInput{
		Draft: draftOutput,
	}).Get(ctx, &polishOutput)
	if err != nil {
		currentStatus.Stage = "Failed at Stage 2"
		currentStatus.ErrorMessage = err.Error()
		return nil, fmt.Errorf("PolishReportActivity failed: %w", err)
	}

	logger.Info("Activity 2 Succeeded!")

	// -------------------------------------------------------------------------
	// STAGE 3: Publish Report to File System
	// -------------------------------------------------------------------------
	currentStatus.Stage = "Stage 3: Publishing Final Report"
	currentStatus.LastUpdated = workflow.Now(ctx)

	outputFilePath := fmt.Sprintf("output/report_%s.md", workflowID)
	var publishOutput PublishOutput
	err = workflow.ExecuteActivity(ctx, activities.PublishFileActivity, PublishInput{
		Topic:          input.Topic,
		PolishedReport: polishOutput,
		Draft:          draftOutput,
		OutputFilePath: outputFilePath,
	}).Get(ctx, &publishOutput)
	if err != nil {
		currentStatus.Stage = "Failed at Stage 3"
		currentStatus.ErrorMessage = err.Error()
		return nil, fmt.Errorf("PublishFileActivity failed: %w", err)
	}

	totalDuration := workflow.Now(ctx).Sub(workflowStartTime).Round(time.Millisecond).String()
	currentStatus.Stage = "Workflow Completed Successfully"
	currentStatus.LastUpdated = workflow.Now(ctx)

	logger.Info("AIResearchWorkflow Completed Successfully",
		"OutputFile", publishOutput.FilePath,
		"TotalDuration", totalDuration)

	return &AIWorkflowResult{
		WorkflowID:    workflowID,
		Topic:         input.Topic,
		OutputFile:    publishOutput.FilePath,
		TotalDuration: totalDuration,
		Status:        "SUCCESS",
	}, nil
}
