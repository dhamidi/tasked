package tasked

import (
	"fmt"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanReorderStepsCmd = &cobra.Command{
	Use:   "reorder-steps <plan-name> <step-id> [step-id]...",
	Short: "Reorder steps in a plan",
	Long: `Reorder the steps in a plan according to the provided step-id sequence.
Steps are placed in the order specified, with any remaining steps appended
at the end in their original relative order.`,
	Args: cobra.MinimumNArgs(2),
	RunE: RunPlanReorderSteps,
}

func RunPlanReorderSteps(cmd *cobra.Command, args []string) error {
	planName := args[0]
	stepIDs := args[1:]

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

	// Validate all step IDs exist in the plan
	existingStepIDs := make(map[string]bool)
	for _, step := range plan.Steps {
		existingStepIDs[step.ID()] = true
	}

	for _, stepID := range stepIDs {
		if !existingStepIDs[stepID] {
			return fmt.Errorf("step with ID '%s' not found in plan '%s'", stepID, planName)
		}
	}

	// Reorder the steps
	plan.Reorder(stepIDs)

	// Save the plan
	if err := p.Save(plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	fmt.Printf("Reordered steps in plan '%s'\n", planName)
	return nil
}
