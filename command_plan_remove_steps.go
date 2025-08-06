package tasked

import (
	"fmt"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanRemoveStepsCmd = &cobra.Command{
	Use:   "remove-steps <plan-name> <step-id> [step-id]...",
	Short: "Remove steps from a plan",
	Long: `Remove one or more steps from a plan by their step IDs. This will delete
the specified steps and their acceptance criteria from the plan. The operation
is permanent and cannot be undone.`,
	Args: cobra.MinimumNArgs(2),
	RunE: RunPlanRemoveSteps,
}

func RunPlanRemoveSteps(cmd *cobra.Command, args []string) error {
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

	// Get the plan from the database
	plan, err := p.Get(planName)
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}

	// Track which steps were found and removed
	stepsFound := make(map[string]bool)
	for _, step := range plan.Steps {
		for _, stepID := range stepIDs {
			if step.ID() == stepID {
				stepsFound[stepID] = true
				break
			}
		}
	}

	// Remove the steps from the plan
	plan.RemoveSteps(stepIDs)

	// Save the updated plan to the database
	err = p.Save(plan)
	if err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	// Report success/failure for each step
	hasErrors := false
	for _, stepID := range stepIDs {
		if stepsFound[stepID] {
			fmt.Printf("Removed step '%s' from plan '%s'\n", stepID, planName)
		} else {
			fmt.Printf("Step '%s' not found in plan '%s'\n", stepID, planName)
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("one or more steps could not be removed")
	}

	return nil
}
