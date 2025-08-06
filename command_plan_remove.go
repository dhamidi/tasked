package tasked

import (
	"fmt"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanRemoveCmd = &cobra.Command{
	Use:   "remove <plan-name> [plan-name...]",
	Short: "Remove one or more plans",
	Long: `Remove one or more plans by name. This will permanently delete the plans
and all their associated steps and acceptance criteria from the database.`,
	Args: cobra.MinimumNArgs(1),
	RunE: RunPlanRemove,
}

func RunPlanRemove(cmd *cobra.Command, args []string) error {
	planNames := args

	// Get the database file path from settings
	dbPath := GlobalSettings.GetDatabaseFile()
	
	// Initialize the planner
	p, err := planner.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize planner: %w", err)
	}
	defer p.Close()

	// Remove the plans
	results := p.Remove(planNames)

	// Report success/failure for each plan individually
	hasErrors := false
	for _, planName := range planNames {
		if err, exists := results[planName]; exists && err != nil {
			fmt.Printf("Failed to remove plan '%s': %v\n", planName, err)
			hasErrors = true
		} else {
			fmt.Printf("Removed plan '%s'\n", planName)
		}
	}

	if hasErrors {
		return fmt.Errorf("one or more plans could not be removed")
	}

	return nil
}
