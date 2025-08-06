package tasked

import (
	"fmt"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanNextStepCmd = &cobra.Command{
	Use:   "next-step <plan-name>",
	Short: "Show the next incomplete step in a plan",
	Long: `Display the next incomplete step in a plan. Shows the step ID, description,
and acceptance criteria. If all steps are completed, indicates the plan is done.`,
	Args: cobra.ExactArgs(1),
	RunE: RunPlanNextStep,
}

func RunPlanNextStep(cmd *cobra.Command, args []string) error {
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

	// Get the next step
	nextStep := plan.NextStep()
	if nextStep == nil {
		fmt.Printf("Plan '%s' is completed - all steps are done!\n", planName)
		return nil
	}

	// Display the next step details
	fmt.Printf("Next step: %s\n", nextStep.ID())
	fmt.Printf("Status: %s\n", nextStep.Status())
	fmt.Printf("\n%s\n", nextStep.Description())

	if len(nextStep.AcceptanceCriteria()) > 0 {
		fmt.Printf("\nAcceptance Criteria:\n")
		for i, criterion := range nextStep.AcceptanceCriteria() {
			fmt.Printf("%d. %s\n", i+1, criterion)
		}
	}

	return nil
}
