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

// MakePlannerToolHandler returns a tool handler function that provides access to all planner operations.
// It also returns a slice of tools that should be registered with the MCP server.
func MakePlannerToolHandler(databasePath string) ([]ToolInfo, error) {
	planner, err := New(databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize planner: %w", err)
	}

	// Define all planner tools with their handlers
	tools := []ToolInfo{
		{createPlanTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreatePlan(ctx, req, planner)
		}},
		{getPlanTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetPlan(ctx, req, planner)
		}},
		{listPlansTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListPlans(ctx, req, planner)
		}},
		{savePlanTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleSavePlan(ctx, req, planner)
		}},
		{removePlansTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleRemovePlans(ctx, req, planner)
		}},
		{compactPlansTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCompactPlans(ctx, req, planner)
		}},
		{addStepTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleAddStep(ctx, req, planner)
		}},
		{removeStepsTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleRemoveSteps(ctx, req, planner)
		}},
		{reorderStepsTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleReorderSteps(ctx, req, planner)
		}},
		{markStepCompletedTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleMarkStepCompleted(ctx, req, planner)
		}},
		{markStepIncompleteTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleMarkStepIncomplete(ctx, req, planner)
		}},
		{inspectPlanTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleInspectPlan(ctx, req, planner)
		}},
		{getNextStepTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetNextStep(ctx, req, planner)
		}},
		{isPlanCompletedTool(), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleIsPlanCompleted(ctx, req, planner)
		}},
	}

	return tools, nil
}

// Tool definitions
func createPlanTool() mcp.Tool {
	return mcp.NewTool("create_plan",
		mcp.WithDescription("Create a new plan with the given name"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the plan to create")),
	)
}

func getPlanTool() mcp.Tool {
	return mcp.NewTool("get_plan",
		mcp.WithDescription("Retrieve a plan by name from the database"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the plan to retrieve")),
	)
}

func listPlansTool() mcp.Tool {
	return mcp.NewTool("list_plans",
		mcp.WithDescription("List all plans with summary information"),
	)
}

func savePlanTool() mcp.Tool {
	return mcp.NewTool("save_plan",
		mcp.WithDescription("Save a plan to the database"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the plan to save")),
	)
}

func removePlansTool() mcp.Tool {
	return mcp.NewTool("remove_plans",
		mcp.WithDescription("Remove plans from the database"),
		mcp.WithArray("names", mcp.Required(), mcp.WithStringItems(), mcp.Description("Names of plans to remove")),
	)
}

func compactPlansTool() mcp.Tool {
	return mcp.NewTool("compact_plans",
		mcp.WithDescription("Remove all completed plans from the database"),
	)
}

func addStepTool() mcp.Tool {
	return mcp.NewTool("add_step",
		mcp.WithDescription("Add a step to a plan"),
		mcp.WithString("plan_name", mcp.Required(), mcp.Description("Name of the plan to add step to")),
		mcp.WithString("step_id", mcp.Required(), mcp.Description("ID for the new step")),
		mcp.WithString("description", mcp.Required(), mcp.Description("Description of the step")),
		mcp.WithArray("acceptance_criteria", mcp.WithStringItems(), mcp.Description("Acceptance criteria for the step")),
	)
}

func removeStepsTool() mcp.Tool {
	return mcp.NewTool("remove_steps",
		mcp.WithDescription("Remove steps from a plan"),
		mcp.WithString("plan_name", mcp.Required(), mcp.Description("Name of the plan to remove steps from")),
		mcp.WithArray("step_ids", mcp.Required(), mcp.WithStringItems(), mcp.Description("IDs of steps to remove")),
	)
}

func reorderStepsTool() mcp.Tool {
	return mcp.NewTool("reorder_steps",
		mcp.WithDescription("Reorder steps in a plan"),
		mcp.WithString("plan_name", mcp.Required(), mcp.Description("Name of the plan to reorder steps in")),
		mcp.WithArray("step_order", mcp.Required(), mcp.WithStringItems(), mcp.Description("New order of step IDs")),
	)
}

func markStepCompletedTool() mcp.Tool {
	return mcp.NewTool("mark_step_completed",
		mcp.WithDescription("Mark a step as completed in a plan"),
		mcp.WithString("plan_name", mcp.Required(), mcp.Description("Name of the plan containing the step")),
		mcp.WithString("step_id", mcp.Required(), mcp.Description("ID of the step to mark as completed")),
	)
}

func markStepIncompleteTool() mcp.Tool {
	return mcp.NewTool("mark_step_incomplete",
		mcp.WithDescription("Mark a step as incomplete in a plan"),
		mcp.WithString("plan_name", mcp.Required(), mcp.Description("Name of the plan containing the step")),
		mcp.WithString("step_id", mcp.Required(), mcp.Description("ID of the step to mark as incomplete")),
	)
}

func inspectPlanTool() mcp.Tool {
	return mcp.NewTool("inspect_plan",
		mcp.WithDescription("Get a formatted string representation of a plan"),
		mcp.WithString("plan_name", mcp.Required(), mcp.Description("Name of the plan to inspect")),
	)
}

func getNextStepTool() mcp.Tool {
	return mcp.NewTool("get_next_step",
		mcp.WithDescription("Get the next incomplete step in a plan"),
		mcp.WithString("plan_name", mcp.Required(), mcp.Description("Name of the plan to get next step from")),
	)
}

func isPlanCompletedTool() mcp.Tool {
	return mcp.NewTool("is_plan_completed",
		mcp.WithDescription("Check if all steps in a plan are completed"),
		mcp.WithString("plan_name", mcp.Required(), mcp.Description("Name of the plan to check")),
	)
}

// Tool handlers
func handleCreatePlan(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	plan, err := p.Create(name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"id":    plan.ID,
		"steps": len(plan.Steps),
	})

	return mcp.NewToolResultText(string(result)), nil
}

func handleGetPlan(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	plan, err := p.Get(name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Convert to a JSON-serializable format
	steps := make([]map[string]interface{}, len(plan.Steps))
	for i, step := range plan.Steps {
		steps[i] = map[string]interface{}{
			"id":                 step.ID(),
			"description":        step.Description(),
			"status":             step.Status(),
			"acceptance_criteria": step.AcceptanceCriteria(),
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

func handleSavePlan(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the plan first (could be a newly created one or existing one)
	plan, err := p.Get(name)
	if err != nil {
		// If plan doesn't exist, try to create it
		plan, err = p.Create(name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("plan not found and could not create: %s", err.Error())), nil
		}
	}

	err = p.Save(plan)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Plan '%s' saved successfully", name)), nil
}

func handleRemovePlans(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	names, err := req.RequireStringSlice("names")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	results := p.Remove(names)
	
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

func handleAddStep(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stepID, err := req.RequireString("step_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	description, err := req.RequireString("description")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	acceptanceCriteria := req.GetStringSlice("acceptance_criteria", []string{})

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Add the step
	plan.AddStep(stepID, description, acceptanceCriteria)

	// Save the plan
	err = p.Save(plan)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Step '%s' added to plan '%s'", stepID, planName)), nil
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

func handleMarkStepCompleted(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stepID, err := req.RequireString("step_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Mark step as completed
	err = plan.MarkAsCompleted(stepID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Save the plan
	err = p.Save(plan)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Step '%s' marked as completed in plan '%s'", stepID, planName)), nil
}

func handleMarkStepIncomplete(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	stepID, err := req.RequireString("step_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Mark step as incomplete
	err = plan.MarkAsIncomplete(stepID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Save the plan
	err = p.Save(plan)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Step '%s' marked as incomplete in plan '%s'", stepID, planName)), nil
}

func handleInspectPlan(ctx context.Context, req mcp.CallToolRequest, p *Planner) (*mcp.CallToolResult, error) {
	planName, err := req.RequireString("plan_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Get the plan
	plan, err := p.Get(planName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	inspection := plan.Inspect()
	return mcp.NewToolResultText(inspection), nil
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
		"id":                 nextStep.ID(),
		"description":        nextStep.Description(),
		"status":             nextStep.Status(),
		"acceptance_criteria": nextStep.AcceptanceCriteria(),
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


