package workflow

import "time"

// TaskQueueName is the Temporal task queue for this AI workflow demo
const TaskQueueName = "AI_RESEARCH_TASK_QUEUE"

// AIWorkflowInput is the input passed to start the workflow
type AIWorkflowInput struct {
	Topic       string `json:"topic"`
	TargetAudience string `json:"target_audience"`
}

// ResearchDraftInput is input for Activity 1 (Draft Generation)
type ResearchDraftInput struct {
	Topic          string `json:"topic"`
	TargetAudience string `json:"target_audience"`
}

// ResearchDraftOutput is the output produced by Activity 1 (Saved in Temporal History)
type ResearchDraftOutput struct {
	Topic        string    `json:"topic"`
	KeyPoints    []string  `json:"key_points"`
	RawDraft     string    `json:"raw_draft"`
	GeneratedAt  time.Time `json:"generated_at"`
	ModelUsed    string    `json:"model_used"`
	ExecutionID  string    `json:"execution_id"`
}

// PolishReportInput is input for Activity 2 (Refinement & Polishing)
type PolishReportInput struct {
	Draft ResearchDraftOutput `json:"draft"`
}

// PolishReportOutput is the output produced by Activity 2
type PolishReportOutput struct {
	ExecutiveSummary string    `json:"executive_summary"`
	RefinedContent   string    `json:"refined_content"`
	PolishedAt       time.Time `json:"polished_at"`
	ModelUsed        string    `json:"model_used"`
}

// PublishInput is input for Activity 3 (Saving to disk / publishing)
type PublishInput struct {
	Topic            string             `json:"topic"`
	PolishedReport   PolishReportOutput `json:"polished_report"`
	Draft            ResearchDraftOutput `json:"draft"`
	OutputFilePath   string             `json:"output_file_path"`
}

// PublishOutput is the final output of Activity 3
type PublishOutput struct {
	FilePath    string    `json:"file_path"`
	PublishedAt time.Time `json:"published_at"`
	FileSize    int64     `json:"file_size"`
}

// AIWorkflowResult is the final workflow return value
type AIWorkflowResult struct {
	WorkflowID    string        `json:"workflow_id"`
	Topic         string        `json:"topic"`
	OutputFile    string        `json:"output_file"`
	TotalDuration string        `json:"total_duration"`
	Status        string        `json:"status"`
}
