package planner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolInfo represents information about a planner tool
type ToolInfo struct {
	Tool    mcp.Tool
	Handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// MakePlannerToolHandler returns a single tool handler that provides access to all planner operations.
// This replaces the previous 14 separate tools with a single "manage_plan" tool that uses action parameters.
//
// Action mappings from old tools:
// - create_plan → add_steps action (creates plan if it doesn't exist)
// - get_plan → inspect action
// - list_plans → list_plans action
// - save_plan → removed (saving happens automatically)
// - remove_plans → remove_plans action
// - compact_plans → compact_plans action
// - add_step → add_steps action
// - remove_steps → remove_steps action
// - reorder_steps → reorder_steps action
// - mark_step_completed/mark_step_incomplete → set_status action
// - inspect_plan → inspect action
// - get_next_step → get_next_step action
// - is_plan_completed → is_completed action
func MakePlannerToolHandler(databasePath string) (ToolInfo, error) {
	planner, err := New(databasePath)
	if err != nil {
		return ToolInfo{}, fmt.Errorf("failed to initialize planner: %w", err)
	}

	// Create the unified manage_plan tool
	tool := mcp.NewTool("manage_plan",
		mcp.WithDescription("Manage plans and their steps with various operations. Steps can include references to relevant files, URLs, or documentation."),

		// Required parameters
		mcp.WithString("plan_name", mcp.Required(), mcp.Description("Name of the plan to operate on")),
		mcp.WithString("action", mcp.Required(), mcp.Enum(
			"add_steps",
			"inspect",
			"list_plans",
			"remove_plans",
			"compact_plans",
			"remove_steps",
			"reorder_steps",
			"set_status",
			"get_next_step",
			"is_completed",
		), mcp.Description("Action to perform")),

		// Conditional parameters based on action
		mcp.WithString("step_id", mcp.Description("ID of the step (required for set_status, single step operations)")),
		mcp.WithString("description", mcp.Description("Description of the step (required for add_steps when adding single step)")),
		mcp.WithArray("acceptance_criteria", mcp.WithStringItems(), mcp.Description("Acceptance criteria for the step (for add_steps)")),
		mcp.WithArray("references", mcp.WithStringItems(), mcp.Description("References for the step (for add_steps) - URLs, file paths, or other resource identifiers (1-5 items)")),
		mcp.WithArray("step_ids", mcp.WithStringItems(), mcp.Description("IDs of steps (required for remove_steps)")),
		mcp.WithArray("step_order", mcp.WithStringItems(), mcp.Description("New order of step IDs (required for reorder_steps)")),
		mcp.WithArray("plan_names", mcp.WithStringItems(), mcp.Description("Names of plans to remove (required for remove_plans)")),
		mcp.WithString("status", mcp.Enum("completed", "incomplete"), mcp.Description("Status to set for step (required for set_status)")),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleManagePlan(ctx, req, planner)
	}

	return ToolInfo{Tool: tool, Handler: handler}, nil
}

// handleManagePlan is the main handler that dispatches to specific action handlers
func handleManagePlan(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	action, err := req.RequireString("action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch action {
	case "add_steps":
		return handleAddSteps(ctx, req, p)
	case "inspect":
		return handleInspectPlan(ctx, req, p)
	case "list_plans":
		return handleListPlans(ctx, req, p)
	case "remove_plans":
		return handleRemovePlans(ctx, req, p)
	case "compact_plans":
		return handleCompactPlans(ctx, req, p)
	case "remove_steps":
		return handleRemoveSteps(ctx, req, p)
	case "reorder_steps":
		return handleReorderSteps(ctx, req, p)
	case "set_status":
		return handleSetStatus(ctx, req, p)
	case "get_next_step":
		return handleGetNextStep(ctx, req, p)
	case "is_completed":
		return handleIsPlanCompleted(ctx, req, p)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unknown action: %s", action)), nil
	}
}

// Action handlers

func handleAddSteps(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get or create the plan
	plan, err := p.Get(planName)
	if err != nil {
		// If plan doesn't exist, create it
		plan, err = p.Create(planName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create plan: %s", err.Error())), nil
		}
	}

	// Add single step using individual parameters
	stepID, err := req.RequireString("step_id")
	if err != nil {
		return mcp.NewToolResultError("step_id required"), nil
	}

	description, err := req.RequireString("description")
	if err != nil {
		return mcp.NewToolResultError("description required"), nil
	}

	acceptanceCriteria := req.GetStringSlice("acceptance_criteria", []string{})
	references := req.GetStringSlice("references", []string{})
	plan.AddStep(stepID, description, acceptanceCriteria, references)

	// Save the plan
	err = p.Save(plan)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":    plan.ID,
		"steps": len(plan.Steps),
	})

	return mcp.NewToolResultText(string(result)), nil
}

func handleInspectPlan(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Check if this is a detailed inspection or simple get
	// For compatibility, return detailed JSON format like the old get_plan
	steps := make([]map[string]interface{}, len(plan.Steps))
	for i, step := range plan.Steps {
		steps[i] = map[string]interface{}{
			"id":                  step.ID(),
			"description":         step.Description(),
			"status":              step.Status(),
			"acceptance_criteria": step.AcceptanceCriteria(),
			"references":          step.References(),
		}
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":    plan.ID,
		"steps": steps,
	})

	return mcp.NewToolResultText(string(result)), nil
}

func handleListPlans(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	plans, err := p.List()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, _ := json.Marshal(plans)
	return mcp.NewToolResultText(string(result)), nil
}

func handleRemovePlans(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planNames, err := req.RequireStringSlice("plan_names")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	results := p.Remove(planNames)

	// Convert results to a JSON-serializable format
	jsonResults := make(map[string]string)
	for name, err := range results {
		if err != nil {
			jsonResults[name] = err.Error()
		} else {
			jsonResults[name] = "success"
		}
	}

	result, _ := json.Marshal(jsonResults)
	return mcp.NewToolResultText(string(result)), nil
}

func handleCompactPlans(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	err := p.Compact()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText("Completed plans compacted successfully"), nil
}

func handleRemoveSteps(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stepIDs, err := req.RequireStringSlice("step_ids")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Remove the steps
	removedCount := plan.RemoveSteps(stepIDs)

	// Save the plan
	err = p.Save(plan)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Removed %d steps from plan '%s'", removedCount, planName)), nil
}

func handleReorderSteps(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stepOrder, err := req.RequireStringSlice("step_order")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Reorder the steps
	plan.Reorder(stepOrder)

	// Save the plan
	err = p.Save(plan)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Steps reordered in plan '%s'", planName)), nil
}

func handleSetStatus(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stepID, err := req.RequireString("step_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	status, err := req.RequireString("status")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Set the status
	switch status {
	case "completed":
		err = plan.MarkAsCompleted(stepID)
	case "incomplete":
		err = plan.MarkAsIncomplete(stepID)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("invalid status: %s (must be 'completed' or 'incomplete')", status)), nil
	}

	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Save the plan
	err = p.Save(plan)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Step '%s' marked as %s in plan '%s'", stepID, status, planName)), nil
}

func handleGetNextStep(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	nextStep := plan.NextStep()
	if nextStep == nil {
		return mcp.NewToolResultText("No incomplete steps found"), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":                  nextStep.ID(),
		"description":         nextStep.Description(),
		"status":              nextStep.Status(),
		"acceptance_criteria": nextStep.AcceptanceCriteria(),
		"references":          nextStep.References(),
	})

	return mcp.NewToolResultText(string(result)), nil
}

func handleIsPlanCompleted(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	isCompleted := plan.IsCompleted()
	result, _ := json.Marshal(map[string]bool{
		"completed": isCompleted,
	})

	return mcp.NewToolResultText(string(result)), nil
}
