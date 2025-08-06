package tasked

import (
	"fmt"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all plans with their status and task counts",
	Long: `List all existing plans showing their names, completion status (DONE/TODO),
and task count information. This provides a quick overview of all plans in the database.`,
	RunE: RunPlanList,
}

func RunPlanList(cmd *cobra.Command, args []string) error {
	// Get the database file path from settings
	dbPath := GlobalSettings.GetDatabaseFile()

	// Initialize the planner
	p, err := planner.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize planner: %w", err)
	}
	defer p.Close()

	// Get all plans using the List method
	plans, err := p.List()
	if err != nil {
		return fmt.Errorf("failed to list plans: %w", err)
	}

	// Handle empty list gracefully
	if len(plans) == 0 {
		fmt.Println("No plans found.")
		return nil
	}

	// Format and display the output
	for _, plan := range plans {
		status := plan.Status
		if plan.TotalTasks == 0 {
			fmt.Printf("%s [%s] (no tasks)\n", plan.Name, status)
		} else {
			fmt.Printf("%s [%s] (%d/%d tasks completed)\n",
				plan.Name, status, plan.CompletedTasks, plan.TotalTasks)
		}
	}

	return nil
}
