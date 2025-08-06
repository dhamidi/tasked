package tasked

import (
	"fmt"
	"os"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanIsCompletedCmd = &cobra.Command{
	Use:   "is-completed <plan-name>",
	Short: "Check if a plan is completed",
	Long: `Check if a plan is completed by verifying that all steps have been finished.
Returns "true" if all steps are completed, "false" otherwise.
Exit code 0 indicates completed, exit code 1 indicates incomplete.`,
	Args: cobra.ExactArgs(1),
	RunE: RunPlanIsCompleted,
}

func RunPlanIsCompleted(cmd *cobra.Command, args []string) error {
	planName := args[0]

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

	// Check if the plan is completed by using NextStep()
	// If NextStep() returns nil, the plan is completed
	nextStep := plan.NextStep()
	isCompleted := nextStep == nil

	if isCompleted {
		fmt.Println("true")
		os.Exit(0)
	} else {
		fmt.Println("false")
		os.Exit(1)
	}

	return nil
}
