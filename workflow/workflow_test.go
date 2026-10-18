package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *UnitTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *UnitTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

// Test_SuccessfulWorkflow tests normal execution of the AI workflow
func (s *UnitTestSuite) Test_SuccessfulWorkflow() {
	var activities *AIActivities

	s.env.OnActivity(activities.GenerateDraftActivity, mock.Anything, mock.Anything).Return(
		&ResearchDraftOutput{
			Topic:       "Test Topic",
			RawDraft:    "Test raw draft content from Gemini",
			ModelUsed:   "gemini-3.8-flash",
			ExecutionID: "test-exec-1",
		}, nil)

	s.env.OnActivity(activities.PolishReportActivity, mock.Anything, mock.Anything).Return(
		&PolishReportOutput{
			ExecutiveSummary: "Test Summary",
			RefinedContent:   "Test Refined Content",
			ModelUsed:        "gemini-3.8-flash",
		}, nil)

	s.env.OnActivity(activities.PublishFileActivity, mock.Anything, mock.Anything).Return(
		&PublishOutput{
			FilePath: "output/test_report.md",
			FileSize: 128,
		}, nil)

	input := AIWorkflowInput{
		Topic:          "Test Topic",
		TargetAudience: "Engineers",
	}

	s.env.ExecuteWorkflow(AIResearchWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result AIWorkflowResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal("SUCCESS", result.Status)
	s.Equal("output/test_report.md", result.OutputFile)
}

// Test_ActivityRetryOnFailure verifies that when an activity fails,
// Temporal retries it according to the retry policy.
func (s *UnitTestSuite) Test_ActivityRetryOnFailure() {
	var activities *AIActivities

	// Step 1: Draft succeeds
	s.env.OnActivity(activities.GenerateDraftActivity, mock.Anything, mock.Anything).Return(
		&ResearchDraftOutput{
			Topic:       "Test Topic",
			RawDraft:    "Draft text",
			ModelUsed:   "gemini-3.8-flash",
			ExecutionID: "test-exec-2",
		}, nil)

	// Step 2: Fails twice, then succeeds on 3rd attempt
	callCount := 0
	s.env.OnActivity(activities.PolishReportActivity, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, input PolishReportInput) (*PolishReportOutput, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("simulated transient failure in PolishReportActivity")
			}
			return &PolishReportOutput{
				ExecutiveSummary: "Recovered summary",
				RefinedContent:   "Recovered refined content",
				ModelUsed:        "gemini-3.8-flash",
			}, nil
		})

	// Step 3: Publish succeeds
	s.env.OnActivity(activities.PublishFileActivity, mock.Anything, mock.Anything).Return(
		&PublishOutput{
			FilePath: "output/test_report.md",
			FileSize: 256,
		}, nil)

	input := AIWorkflowInput{
		Topic:          "Test Topic",
		TargetAudience: "Engineers",
	}

	s.env.ExecuteWorkflow(AIResearchWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.Equal(3, callCount, "PolishReportActivity should have retried and succeeded on attempt 3")
}

func TestWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}
