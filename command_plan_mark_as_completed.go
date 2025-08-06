package tasked

import (
	"fmt"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanMarkAsCompletedCmd = &cobra.Command{
	Use:   "mark-as-completed <plan-name> <step-id>",
	Short: "Mark a step as completed",
	Long: `Mark a specific step in a plan as completed (DONE status).
This will update the step's status to DONE and persist the change to the database.`,
	Args: cobra.ExactArgs(2),
	RunE: RunPlanMarkAsCompleted,
}

func RunPlanMarkAsCompleted(cmd *cobra.Command, args []string) error {
	planName := args[0]
	stepID := args[1]

	// Get the database file path from settings
	dbPath := GlobalSettings.GetDatabaseFile()

	// Initialize the planner
	p, err := planner.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize planner: %w", err)
	}
	defer p.Close()

	// Get the plan from the database
	plan, err := p.Get(planName)
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}

	// Mark the step as completed
	err = plan.MarkAsCompleted(stepID)
	if err != nil {
		return fmt.Errorf("failed to mark step as completed: %w", err)
	}

	// Save the changes to the database
	err = p.Save(plan)
	if err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	fmt.Printf("Step '%s' in plan '%s' marked as completed\n", stepID, planName)
	return nil
}
