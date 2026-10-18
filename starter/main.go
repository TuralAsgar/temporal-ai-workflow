package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.temporal.io/sdk/client"

	"temporal-ai-demo/config"
	wf "temporal-ai-demo/workflow"
)

func main() {
	config.LoadEnv(".env")

	topicFlag := flag.String("topic", "Why Temporal is the Ultimate Orchestrator for Multi-Step AI Agents", "Topic for AI research")
	audienceFlag := flag.String("audience", "Senior Software Engineers and AI Architects", "Target audience for the report")
	flag.Parse()

	temporalHost := os.Getenv("TEMPORAL_HOST_PORT")
	if temporalHost == "" {
		temporalHost = client.DefaultHostPort
	}

	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("❌ Unable to connect to Temporal: %v\n"+
			"👉 Make sure Temporal dev server is running: 'temporal server start-dev'\n", err)
	}
	defer c.Close()

	workflowID := fmt.Sprintf("ai-research-%d", time.Now().Unix())
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: wf.TaskQueueName,
	}

	input := wf.AIWorkflowInput{
		Topic:          *topicFlag,
		TargetAudience: *audienceFlag,
	}

	fmt.Println("╔═════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║               STARTING TEMPORAL AI RESEARCH WORKFLOW                    ║")
	fmt.Println("╚═════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("📋 Topic:            %s\n", input.Topic)
	fmt.Printf("🎯 Target Audience:  %s\n", input.TargetAudience)
	fmt.Printf("🆔 Workflow ID:      %s\n", workflowID)
	fmt.Printf("🌐 View in Web UI:   http://localhost:8233/namespaces/default/workflows/%s\n", workflowID)
	fmt.Println("─────────────────────────────────────────────────────────────────────────")

	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, wf.AIResearchWorkflow, input)
	if err != nil {
		log.Fatalf("❌ Failed to start workflow: %v", err)
	}

	fmt.Println("⏳ Workflow launched! Monitoring status via Temporal queries...")

	// Periodically query workflow state while waiting for result
	doneCh := make(chan struct{})
	var result wf.AIWorkflowResult
	var workflowErr error

	go func() {
		workflowErr = we.Get(ctx, &result)
		close(doneCh)
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastStage := ""
	for {
		select {
		case <-doneCh:
			goto Finished
		case <-ticker.C:
			// Query workflow for live status
			queryRes, err := c.QueryWorkflow(ctx, we.GetID(), we.GetRunID(), "getStatus")
			if err == nil {
				var status wf.WorkflowStatus
				if err := queryRes.Get(&status); err == nil {
					if status.Stage != lastStage {
						fmt.Printf("  📍 Current Status: %s\n", status.Stage)
						lastStage = status.Stage
					}
				}
			}
		}
	}

Finished:
	if workflowErr != nil {
		fmt.Printf("\n❌ Workflow failed with error: %v\n", workflowErr)
		return
	}

	fmt.Println("\n─────────────────────────────────────────────────────────────────────────")
	fmt.Println("🎉 WORKFLOW COMPLETED SUCCESSFULLY!")
	fmt.Println("─────────────────────────────────────────────────────────────────────────")
	fmt.Printf("📁 Output Report:    %s\n", result.OutputFile)
	fmt.Printf("⏱️  Total Duration:   %s\n", result.TotalDuration)
	fmt.Printf("🌐 Temporal History: http://localhost:8233/namespaces/default/workflows/%s\n", workflowID)
	fmt.Println("─────────────────────────────────────────────────────────────────────────")
}
