package tasked

import (
	"fmt"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanNewCmd = &cobra.Command{
	Use:   "new <plan-name>",
	Short: "Create a new empty plan",
	Long: `Create a new empty plan with the specified name. The plan will be created
in the database and can then be populated with steps using other plan commands.`,
	Args: cobra.ExactArgs(1),
	RunE: RunPlanNew,
}

func RunPlanNew(cmd *cobra.Command, args []string) error {
	planName := args[0]

	// Get the database file path from settings
	dbPath := GlobalSettings.GetDatabaseFile()
	
	// Initialize the planner
	p, err := planner.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize planner: %w", err)
	}
	defer p.Close()

	// Create the new plan
	plan, err := p.Create(planName)
	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}

	// Save the plan to the database
	if err := p.Save(plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	fmt.Printf("Created plan '%s'\n", planName)
	return nil
}
