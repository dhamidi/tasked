package main

import (
	"fmt"

	"github.com/dhamidi/tasked"
	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanInspectCmd = &cobra.Command{
	Use:   "inspect <plan-name>",
	Short: "Display detailed plan information",
	Long: `Display detailed information about a plan including all its steps, their status,
and acceptance criteria. This provides a comprehensive view of the plan's current state.`,
	Args: cobra.ExactArgs(1),
	RunE: RunPlanInspect,
}

func RunPlanInspect(cmd *cobra.Command, args []string) error {
	planName := args[0]

	// Get the database file path from settings
	dbPath := tasked.GlobalSettings.GetDatabaseFile()
	
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

	// Display the plan details using the Inspect method
	fmt.Print(plan.Inspect())
	return nil
}
