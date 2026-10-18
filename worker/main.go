package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"temporal-ai-demo/config"
	wf "temporal-ai-demo/workflow"
)

func main() {
	// Load environment variables from .env file if present
	config.LoadEnv(".env")

	temporalHost := os.Getenv("TEMPORAL_HOST_PORT")
	if temporalHost == "" {
		temporalHost = client.DefaultHostPort // "localhost:7233"
	}

	fmt.Println("╔═════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║               TEMPORAL + GOOGLE AI STUDIO DEMO WORKER                   ║")
	fmt.Println("╚═════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("📡 Connecting to Temporal Server at: %s\n", temporalHost)

	// Create the Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("❌ Unable to connect to Temporal Server: %v\n"+
			"👉 Make sure Temporal dev server is running: 'temporal server start-dev'\n", err)
	}
	defer c.Close()

	// Initialize Google AI Studio Gemini client
	ctx := context.Background()
	geminiClient, err := wf.NewGeminiClient(ctx)
	if err != nil {
		log.Fatalf("❌ Unable to create Gemini AI client: %v", err)
	}

	aiActivities := wf.NewAIActivities(geminiClient)

	// Create a worker that listens on the task queue
	w := worker.New(c, wf.TaskQueueName, worker.Options{})

	// Register Workflow & Activities
	w.RegisterWorkflow(wf.AIResearchWorkflow)
	w.RegisterActivity(aiActivities.GenerateDraftActivity)
	w.RegisterActivity(aiActivities.PolishReportActivity)
	w.RegisterActivity(aiActivities.PublishFileActivity)

	fmt.Printf("👷 Worker listening on Task Queue: %q\n", wf.TaskQueueName)
	fmt.Println("🌐 Temporal Web UI available at: http://localhost:8233")
	fmt.Println("─────────────────────────────────────────────────────────────────────────")
	fmt.Println("💡 DEMO TIP:")
	fmt.Println("   To test error recovery and state persistence:")
	fmt.Println("   1. Open 'workflow/activities.go'")
	fmt.Println("   2. In 'PolishReportActivity', uncomment the deliberate error line.")
	fmt.Println("   3. Run the starter to see Activity 1 save its state and Activity 2 retry.")
	fmt.Println("   4. Re-comment the line and restart worker to watch it resume seamlessly!")
	fmt.Println("─────────────────────────────────────────────────────────────────────────")

	// Start listening to the Task Queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("❌ Worker failed: %v", err)
	}
}
