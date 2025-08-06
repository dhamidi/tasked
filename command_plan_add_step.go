package tasked

import (
	"fmt"
	"strings"

	"github.com/dhamidi/tasked/planner"
	"github.com/spf13/cobra"
)

var PlanAddStepCmd = &cobra.Command{
	Use:   "add-step [--after step-id] [--references ref1,ref2] <plan-name> <step-id> <description> <acceptance-criteria> ...",
	Short: "Add a new step to a plan",
	Long: `Add a new step to an existing plan. The step can be positioned after a specific
step using the --after flag. If no --after flag is provided, the step will be added
at the end of the plan.

References can be added using the --references flag with comma-separated values.`,
	Args: cobra.MinimumNArgs(3),
	RunE: RunPlanAddStep,
}

var afterStepID string
var referencesFlag string

func init() {
	PlanAddStepCmd.Flags().StringVar(&afterStepID, "after", "", "ID of the step after which to insert the new step")
	PlanAddStepCmd.Flags().StringVar(&referencesFlag, "references", "", "Comma-separated list of references (URLs or other reference strings)")
}

func RunPlanAddStep(cmd *cobra.Command, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("requires at least 3 arguments: plan-name, step-id, description")
	}

	planName := args[0]
	stepID := args[1]
	description := args[2]
	acceptanceCriteria := args[3:]

	// Get the database file path from settings
	dbPath := GlobalSettings.GetDatabaseFile()

	// Initialize the planner
	p, err := planner.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize planner: %w", err)
	}
	defer p.Close()

	// Get the existing plan
	plan, err := p.Get(planName)
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}

	// Check if step ID already exists
	for _, step := range plan.Steps {
		if step.ID() == stepID {
			return fmt.Errorf("step with ID '%s' already exists in plan '%s'", stepID, planName)
		}
	}

	// Find the insertion position
	insertIndex := len(plan.Steps) // Default to end
	if afterStepID != "" {
		found := false
		for i, step := range plan.Steps {
			if step.ID() == afterStepID {
				insertIndex = i + 1
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("step with ID '%s' not found in plan '%s'", afterStepID, planName)
		}
	}

	// Parse references from comma-separated string
	var references []string
	if referencesFlag != "" {
		references = strings.Split(referencesFlag, ",")
		// Trim whitespace from each reference
		for i, ref := range references {
			references[i] = strings.TrimSpace(ref)
		}
	}

	// Add the step at the end first (AddStep always appends)
	plan.AddStep(stepID, description, acceptanceCriteria, references)

	// If we need to insert it in a specific position (not at the end), reorder
	if afterStepID != "" && insertIndex < len(plan.Steps)-1 {
		// Create new order that puts our step in the right position
		var newOrder []string

		// Add all steps before the insertion point
		for i := 0; i < insertIndex; i++ {
			newOrder = append(newOrder, plan.Steps[i].ID())
		}

		// Add our new step
		newOrder = append(newOrder, stepID)

		// Add all steps after the insertion point (excluding our step which is at the end)
		for i := insertIndex; i < len(plan.Steps)-1; i++ {
			newOrder = append(newOrder, plan.Steps[i].ID())
		}

		plan.Reorder(newOrder)
	}

	// Save the updated plan
	if err := p.Save(plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	fmt.Printf("Added step '%s' to plan '%s'\n", stepID, planName)
	return nil
}
