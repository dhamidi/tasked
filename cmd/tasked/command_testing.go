package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test [test-name]",
	Short: "Run integration tests for the tasked MCP server",
	Long: `Run integration tests against the tasked MCP server. The test command starts 
an MCP server subprocess and runs various tool calls to verify functionality.

Available test scenarios:
  default    Test the manage_plan tool with comprehensive scenarios`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	testName := "default"
	if len(args) > 0 {
		testName = args[0]
	}

	switch testName {
	case "default":
		return runDefaultTest()
	default:
		return fmt.Errorf("unknown test scenario: %s", testName)
	}
}

func runDefaultTest() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create temporary database file
	tempDir := os.TempDir()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	tempDB := filepath.Join(tempDir, fmt.Sprintf("test-%s.db", timestamp))
	defer os.Remove(tempDB)

	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create MCP client - connect to tasked subprocess
	mcpClient, err := client.NewStdioMCPClient(execPath, os.Environ(), "mcp", "--database-file", tempDB)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}
	defer mcpClient.Close()

	// Initialize client
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "tasked-test",
				Version: "1.0.0",
			},
		},
	}
	_, err = mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// Run the test scenario
	return runManagePlanTestScenario(ctx, mcpClient)
}

func runManagePlanTestScenario(ctx context.Context, c *client.Client) error {
	testPlan := "test-plan"

	// Test 1: add_steps - Create plan with 3 steps
	logToolCall("add_steps", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-1",
		"description":         "First test step",
		"acceptance_criteria": []string{"Complete the first task"},
	})
	result, err := callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-1",
		"description":         "First test step",
		"acceptance_criteria": []string{"Complete the first task"},
	})
	if err != nil {
		failTest("Failed to add first step: %v", err)
	}
	assertSuccess(result, "add_steps step-1")

	logToolCall("add_steps", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-2",
		"description":         "Second test step",
		"acceptance_criteria": []string{"Complete the second task"},
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-2",
		"description":         "Second test step",
		"acceptance_criteria": []string{"Complete the second task"},
	})
	if err != nil {
		failTest("Failed to add second step: %v", err)
	}
	assertSuccess(result, "add_steps step-2")

	logToolCall("add_steps", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-3",
		"description":         "Third test step",
		"acceptance_criteria": []string{"Complete the third task"},
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-3",
		"description":         "Third test step",
		"acceptance_criteria": []string{"Complete the third task"},
	})
	if err != nil {
		failTest("Failed to add third step: %v", err)
	}
	assertSuccess(result, "add_steps step-3")

	// Test 2: inspect - Verify plan structure and content
	logToolCall("inspect", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "inspect",
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "inspect",
	})
	if err != nil {
		failTest("Failed to inspect plan: %v", err)
	}
	assertSuccess(result, "inspect")
	inspectData := parseJSONResult(getResultText(result))
	steps, ok := inspectData["steps"].([]interface{})
	if !ok || len(steps) != 3 {
		failTest("Expected 3 steps in plan, got %d", len(steps))
	}

	// Test 3: get_next_step - Check first incomplete step
	logToolCall("get_next_step", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "get_next_step",
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "get_next_step",
	})
	if err != nil {
		failTest("Failed to get next step: %v", err)
	}
	assertSuccess(result, "get_next_step")
	nextStep := parseJSONResult(getResultText(result))
	firstStepID := nextStep["id"].(string)
	if firstStepID != "step-1" {
		failTest("Expected next step to be 'step-1', got '%s'", firstStepID)
	}

	// Test 4: set_status - Mark first step as DONE
	logToolCall("set_status", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "set_status",
		"step_id":   "step-1",
		"status":    "completed",
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "set_status",
		"step_id":   "step-1",
		"status":    "completed",
	})
	if err != nil {
		failTest("Failed to set status: %v", err)
	}
	assertSuccess(result, "set_status step-1 completed")

	// Test 5: get_next_step - Verify next step changed
	logToolCall("get_next_step", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "get_next_step",
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "get_next_step",
	})
	if err != nil {
		failTest("Failed to get next step after completion: %v", err)
	}
	assertSuccess(result, "get_next_step after completion")
	nextStep = parseJSONResult(getResultText(result))
	secondStepID := nextStep["id"].(string)
	if secondStepID != "step-2" {
		failTest("Expected next step to be 'step-2' after completing step-1, got '%s'", secondStepID)
	}

	// Test 6: list_plans - Verify plan exists in list
	logToolCall("list_plans", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "list_plans",
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "list_plans",
	})
	if err != nil {
		failTest("Failed to list plans: %v", err)
	}
	assertSuccess(result, "list_plans")
	planList := parseJSONResultAsArray(getResultText(result))
	if planList == nil {
		failTest("Expected list of plans, got nil")
	}
	planFound := false
	for _, p := range planList {
		if planMap, ok := p.(map[string]interface{}); ok {
			if planMap["name"] == testPlan {
				planFound = true
				break
			}
		}
	}
	if !planFound {
		failTest("Plan '%s' not found in plan list", testPlan)
	}

	// Test 7: remove_steps - Remove one specific step
	logToolCall("remove_steps", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "remove_steps",
		"step_ids":  []string{"step-3"},
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "remove_steps",
		"step_ids":  []string{"step-3"},
	})
	if err != nil {
		failTest("Failed to remove step: %v", err)
	}
	assertSuccess(result, "remove_steps step-3")

	// Test 8: reorder_steps - Change step order
	logToolCall("reorder_steps", map[string]interface{}{
		"plan_name":  testPlan,
		"action":     "reorder_steps",
		"step_order": []string{"step-2", "step-1"},
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name":  testPlan,
		"action":     "reorder_steps",
		"step_order": []string{"step-2", "step-1"},
	})
	if err != nil {
		failTest("Failed to reorder steps: %v", err)
	}
	assertSuccess(result, "reorder_steps")

	// Test 9: is_completed - Check completion status (should be false)
	logToolCall("is_completed", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "is_completed",
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "is_completed",
	})
	if err != nil {
		failTest("Failed to check completion status: %v", err)
	}
	assertSuccess(result, "is_completed")
	completionData := parseJSONResult(getResultText(result))
	isCompleted, ok := completionData["completed"].(bool)
	if !ok || isCompleted {
		failTest("Expected plan to be incomplete, got completed=%v", isCompleted)
	}

	// Test 10: set_status - Mark remaining step as DONE
	logToolCall("set_status", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "set_status",
		"step_id":   "step-2",
		"status":    "completed",
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "set_status",
		"step_id":   "step-2",
		"status":    "completed",
	})
	if err != nil {
		failTest("Failed to set status for step-2: %v", err)
	}
	assertSuccess(result, "set_status step-2 completed")

	// Test 11: is_completed - Verify now completed
	logToolCall("is_completed", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "is_completed",
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "is_completed",
	})
	if err != nil {
		failTest("Failed to check final completion status: %v", err)
	}
	assertSuccess(result, "is_completed final")
	completionData = parseJSONResult(getResultText(result))
	isCompleted, ok = completionData["completed"].(bool)
	if !ok || !isCompleted {
		failTest("Expected plan to be completed, got completed=%v", isCompleted)
	}

	// Test 12: compact_plans - Cleanup completed plans
	logToolCall("compact_plans", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "compact_plans",
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "compact_plans",
	})
	if err != nil {
		failTest("Failed to compact plans: %v", err)
	}
	assertSuccess(result, "compact_plans")

	log.Printf("✓ All tests passed successfully")
	return nil
}

func callTool(ctx context.Context, c *client.Client, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
}

func logToolCall(action string, args map[string]interface{}) {
	argsJSON, _ := json.MarshalIndent(args, "", "  ")
	log.Printf("→ Tool call: %s\n%s", action, string(argsJSON))
}

func assertSuccess(result *mcp.CallToolResult, operation string) {
	if result.IsError {
		text := getResultText(result)
		failTest("Operation '%s' failed: %s", operation, text)
	}
}

func getResultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	
	if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
		return textContent.Text
	}
	
	return fmt.Sprintf("%v", result.Content[0])
}

func parseJSONResult(text string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		failTest("Failed to parse JSON result: %v", err)
	}
	return result
}

func parseJSONResultAsArray(text string) []interface{} {
	var result []interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		failTest("Failed to parse JSON array result: %v", err)
	}
	return result
}

func failTest(format string, args ...interface{}) {
	log.Printf("✗ Test failed: "+format, args...)
	os.Exit(1)
}
