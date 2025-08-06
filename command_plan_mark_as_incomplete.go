package tasked

import (
	"fmt"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanMarkAsIncompleteCmd = &cobra.Command{
	Use:   "mark-as-incomplete <plan-name> <step-id>",
	Short: "Mark a step as incomplete (TODO)",
	Long: `Mark a step in the specified plan as incomplete (TODO status).
This changes the step status from DONE back to TODO, allowing you to track
that work still needs to be done on this step.`,
	Args: cobra.ExactArgs(2),
	RunE: RunPlanMarkAsIncomplete,
}

func RunPlanMarkAsIncomplete(cmd *cobra.Command, args []string) error {
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

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}

	// Mark the step as incomplete
	if err := plan.MarkAsIncomplete(stepID); err != nil {
		return fmt.Errorf("failed to mark step as incomplete: %w", err)
	}

	// Save the plan
	if err := p.Save(plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	fmt.Printf("Marked step '%s' in plan '%s' as incomplete\n", stepID, planName)
	return nil
}
