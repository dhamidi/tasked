package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test [test-name]",
	Short: "Run integration tests for the tasked MCP server",
	Long: `Run comprehensive integration tests for tasked functionality through different interfaces.

The test command provides two distinct test scenarios that validate the same core functionality
through different access methods:

TEST SCENARIOS:

default (MCP Integration Testing)
  Tests the manage_plan MCP tool by connecting to a tasked subprocess via stdio protocol.
  This scenario validates MCP server functionality and tool integration for AI agents.
  
  What it tests:
  - MCP server initialization and tool registration  
  - manage_plan tool with all actions (add_steps, inspect, get_next_step, etc.)
  - JSON response parsing and data validation
  - MCP protocol compliance and error handling
  
  Use this when: Testing MCP integration, AI agent compatibility, or server functionality

plan-subcommand (CLI Functionality Testing)  
  Tests plan subcommands by directly invoking the tasked binary with CLI arguments.
  This scenario validates command-line interface and user workflow functionality.
  
  What it tests:
  - All plan subcommands (new, add-step, list, inspect, etc.)
  - Command-line argument parsing and validation
  - Text output formatting and user experience
  - Database operations and state management
  - Advanced features like --after flag for step insertion
  
  Use this when: Testing CLI workflows, command behavior, or user interface

USAGE EXAMPLES:

  # Run MCP integration tests (default scenario)
  tasked test
  tasked test default
  
  # Run CLI functionality tests  
  tasked test plan-subcommand
  
TECHNICAL DETAILS:

Both scenarios test identical functionality but through different interfaces:
- Same database operations (create plans, add steps, mark completion, etc.)
- Same validation logic and error conditions
- Different interaction methods (MCP tools vs CLI commands)
- Different output formats (JSON vs formatted text)

The tests use temporary databases and are fully self-contained with automatic cleanup.`,
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
	case "plan-subcommand":
		return runPlanSubcommandTest()
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

	// Test 1: add_steps - Create plan with 3 steps, including references
	logToolCall("add_steps", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-1",
		"description":         "First test step",
		"acceptance_criteria": []string{"Complete the first task"},
		"references":          []string{"doc-1", "spec-A"},
	})
	result, err := callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-1",
		"description":         "First test step",
		"acceptance_criteria": []string{"Complete the first task"},
		"references":          []string{"doc-1", "spec-A"},
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
		"references":          []string{"guide-B"},
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-2",
		"description":         "Second test step",
		"acceptance_criteria": []string{"Complete the second task"},
		"references":          []string{"guide-B"},
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

	// Test 1b: add_steps - Test step with multiple references
	logToolCall("add_steps", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-multi-refs",
		"description":         "Step with multiple references",
		"acceptance_criteria": []string{"Test multiple references functionality"},
		"references":          []string{"ref-X", "ref-Y", "ref-Z"},
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name":           testPlan,
		"action":              "add_steps",
		"step_id":             "step-multi-refs",
		"description":         "Step with multiple references",
		"acceptance_criteria": []string{"Test multiple references functionality"},
		"references":          []string{"ref-X", "ref-Y", "ref-Z"},
	})
	if err != nil {
		failTest("Failed to add step with multiple references: %v", err)
	}
	assertSuccess(result, "add_steps step-multi-refs")

	// Test 2: inspect - Verify plan structure and content including references
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
	if !ok || len(steps) != 4 {
		failTest("Expected 4 steps in plan, got %d", len(steps))
	}
	
	// Verify references in step-1 (should have multiple references)
	step1 := steps[0].(map[string]interface{})
	references1, ok := step1["references"].([]interface{})
	if !ok || len(references1) != 2 {
		failTest("Expected 2 references in step-1, got %v", references1)
	}
	if references1[0].(string) != "doc-1" || references1[1].(string) != "spec-A" {
		failTest("Expected step-1 references ['doc-1', 'spec-A'], got %v", references1)
	}
	
	// Verify references in step-2 (should have single reference)
	step2 := steps[1].(map[string]interface{})
	references2, ok := step2["references"].([]interface{})
	if !ok || len(references2) != 1 {
		failTest("Expected 1 reference in step-2, got %v", references2)
	}
	if references2[0].(string) != "guide-B" {
		failTest("Expected step-2 reference 'guide-B', got %v", references2[0])
	}
	
	// Verify step-3 has no references (empty array or nil)
	step3 := steps[2].(map[string]interface{})
	references3, exists := step3["references"]
	if exists {
		if refs, ok := references3.([]interface{}); ok && len(refs) > 0 {
			failTest("Expected step-3 to have no references, got %v", refs)
		}
	}
	
	// Verify step-multi-refs has multiple references
	stepMulti := steps[3].(map[string]interface{})
	referencesMulti, ok := stepMulti["references"].([]interface{})
	if !ok || len(referencesMulti) != 3 {
		failTest("Expected 3 references in step-multi-refs, got %v", referencesMulti)
	}
	expectedRefs := []string{"ref-X", "ref-Y", "ref-Z"}
	for i, expectedRef := range expectedRefs {
		if referencesMulti[i].(string) != expectedRef {
			failTest("Expected step-multi-refs reference[%d] to be '%s', got '%s'", i, expectedRef, referencesMulti[i])
		}
	}

	// Test 3: get_next_step - Check first incomplete step and verify references
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
	
	// Verify references are included in get_next_step response
	nextStepRefs, ok := nextStep["references"].([]interface{})
	if !ok || len(nextStepRefs) != 2 {
		failTest("Expected 2 references in next step response, got %v", nextStepRefs)
	}
	if nextStepRefs[0].(string) != "doc-1" || nextStepRefs[1].(string) != "spec-A" {
		failTest("Expected next step references ['doc-1', 'spec-A'], got %v", nextStepRefs)
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

	// Test 5: get_next_step - Verify next step changed and check different references
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
	
	// Verify step-2 has different references than step-1
	step2Refs, ok := nextStep["references"].([]interface{})
	if !ok || len(step2Refs) != 1 {
		failTest("Expected 1 reference in step-2 next step response, got %v", step2Refs)
	}
	if step2Refs[0].(string) != "guide-B" {
		failTest("Expected step-2 next step reference 'guide-B', got %v", step2Refs[0])
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

	// Test 7: remove_steps - Remove multiple steps including one with references
	logToolCall("remove_steps", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "remove_steps",
		"step_ids":  []string{"step-3", "step-multi-refs"},
	})
	result, err = callTool(ctx, c, "manage_plan", map[string]interface{}{
		"plan_name": testPlan,
		"action":    "remove_steps",
		"step_ids":  []string{"step-3", "step-multi-refs"},
	})
	if err != nil {
		failTest("Failed to remove steps: %v", err)
	}
	assertSuccess(result, "remove_steps step-3 and step-multi-refs")

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

// execCommand executes a command with the current executable and returns stdout, stderr, exit code, and error
func execCommand(args []string, databaseFile string) (string, string, int, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Add database file parameter if provided
	if databaseFile != "" {
		args = append([]string{"--database-file", databaseFile}, args...)
	}

	cmd := exec.Command(execPath, args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	exitCode := 0
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode = exitError.ExitCode()
	} else if err != nil {
		return "", "", -1, err
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

// logPlanCommand logs the plan command being executed, similar to logToolCall
func logPlanCommand(command string, args []string) {
	allArgs := append([]string{command}, args...)
	log.Printf("→ Plan command: %s", strings.Join(allArgs, " "))
}

// assertCommandSuccess validates that a command succeeded (exit code 0)
func assertCommandSuccess(stdout, stderr string, exitCode int, operation string) {
	if exitCode != 0 {
		failTest("Command '%s' failed with exit code %d\nStdout: %s\nStderr: %s",
			operation, exitCode, stdout, stderr)
	}
}

// assertCommandOutput validates that command output contains expected content
func assertCommandOutput(stdout string, expected []string, operation string) {
	for _, exp := range expected {
		if !strings.Contains(stdout, exp) {
			failTest("Command '%s' output missing expected content '%s'\nActual output: %s",
				operation, exp, stdout)
		}
	}
}

// execPlanCommand is a wrapper specifically for plan subcommands that handles database file automatically
func execPlanCommand(subcommand string, args []string, databaseFile string) (string, error) {
	commandArgs := append([]string{"plan", subcommand}, args...)
	logPlanCommand("plan", append([]string{subcommand}, args...))

	stdout, stderr, exitCode, err := execCommand(commandArgs, databaseFile)
	if err != nil {
		return "", fmt.Errorf("failed to execute plan %s: %w", subcommand, err)
	}

	assertCommandSuccess(stdout, stderr, exitCode, fmt.Sprintf("plan %s", subcommand))
	return stdout, nil
}

func runPlanSubcommandTest() error {
	// Create temporary database file
	tempDir := os.TempDir()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	tempDB := filepath.Join(tempDir, fmt.Sprintf("test-plan-subcommand-%s.db", timestamp))
	defer os.Remove(tempDB)

	testPlan := "test-plan"

	// Test 1: plan new - Create a test plan
	stdout, err := execPlanCommand("new", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to create test plan: %w", err)
	}
	assertCommandOutput(stdout, []string{"Created plan"}, "plan new")

	// Test 2: plan add-step - Add multiple steps with acceptance criteria and references
	stdout, err = execPlanCommand("add-step", []string{testPlan, "step-1", "First test step", "Complete the first task", "--references", "doc-1,spec-A"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to add step-1: %w", err)
	}
	assertCommandOutput(stdout, []string{"Added step"}, "plan add-step step-1")

	stdout, err = execPlanCommand("add-step", []string{testPlan, "step-2", "Second test step", "Complete the second task", "--references", "guide-B"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to add step-2: %w", err)
	}
	assertCommandOutput(stdout, []string{"Added step"}, "plan add-step step-2")

	stdout, err = execPlanCommand("add-step", []string{testPlan, "step-3", "Third test step", "Complete the third task"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to add step-3: %w", err)
	}
	assertCommandOutput(stdout, []string{"Added step"}, "plan add-step step-3")

	// Test 2b: Test --after flag for plan add-step
	stdout, err = execPlanCommand("add-step", []string{testPlan, "step-1.5", "Middle step", "Complete the middle task", "--after", "step-1"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to add step-1.5 after step-1: %w", err)
	}
	assertCommandOutput(stdout, []string{"Added step"}, "plan add-step step-1.5 --after step-1")

	// Test 3: plan list - Verify plan appears in list with proper format
	stdout, err = execPlanCommand("list", []string{}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to list plans: %w", err)
	}
	// Validate that the plan appears and has proper format
	assertCommandOutput(stdout, []string{testPlan, "4 tasks"}, "plan list format")

	// Test 4: plan inspect - Verify plan structure and detailed content including references
	stdout, err = execPlanCommand("inspect", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to inspect plan: %w", err)
	}
	// Check for step IDs, descriptions, acceptance criteria, and references
	expectedInspectContent := []string{
		"step-1", "step-1.5", "step-2", "step-3",
		"First test step", "Middle step", "Second test step", "Third test step",
		"Complete the first task", "Complete the middle task", "Complete the second task", "Complete the third task",
		"TODO", "Acceptance Criteria:",
		"doc-1", "spec-A", "guide-B", "References:",
	}
	assertCommandOutput(stdout, expectedInspectContent, "plan inspect detailed content")

	// Validate step order after --after insertion
	lines := strings.Split(stdout, "\n")
	step1Position := -1
	step15Position := -1
	step2Position := -1
	for i, line := range lines {
		if strings.Contains(line, "step-1") && !strings.Contains(line, "step-1.5") {
			step1Position = i
		}
		if strings.Contains(line, "step-1.5") {
			step15Position = i
		}
		if strings.Contains(line, "step-2") {
			step2Position = i
		}
	}
	if step1Position >= step15Position || step15Position >= step2Position {
		return fmt.Errorf("step order incorrect after --after flag: step-1 at %d, step-1.5 at %d, step-2 at %d",
			step1Position, step15Position, step2Position)
	}

	// Test 5: plan next-step - Get first incomplete step and verify references
	stdout, err = execPlanCommand("next-step", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to get next step: %w", err)
	}
	assertCommandOutput(stdout, []string{"step-1", "doc-1", "spec-A"}, "plan next-step")

	// Test 6: plan mark-as-completed - Mark a step as done
	stdout, err = execPlanCommand("mark-as-completed", []string{testPlan, "step-1"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to mark step-1 as completed: %w", err)
	}
	assertCommandOutput(stdout, []string{"marked as completed"}, "plan mark-as-completed step-1")

	// Test 7: plan next-step - Verify next step changed to step-1.5 (no references)
	stdout, err = execPlanCommand("next-step", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to get next step after completion: %w", err)
	}
	assertCommandOutput(stdout, []string{"step-1.5"}, "plan next-step after step-1 completion")
	// step-1.5 should not contain the references from step-1
	if strings.Contains(stdout, "doc-1") || strings.Contains(stdout, "spec-A") {
		return fmt.Errorf("step-1.5 next-step output should not contain step-1 references: %s", stdout)
	}

	// Test 8: plan mark-as-incomplete - Mark step back to todo
	stdout, err = execPlanCommand("mark-as-incomplete", []string{testPlan, "step-1"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to mark step-1 as incomplete: %w", err)
	}
	assertCommandOutput(stdout, []string{"as incomplete"}, "plan mark-as-incomplete step-1")

	// Verify step-1 is now the next step again with references restored
	stdout, err = execPlanCommand("next-step", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to get next step after marking incomplete: %w", err)
	}
	assertCommandOutput(stdout, []string{"step-1", "doc-1", "spec-A"}, "plan next-step after marking step-1 incomplete")
	
	// Test references persistence: Complete step-1 and step-1.5, then check step-2 references
	stdout, err = execPlanCommand("mark-as-completed", []string{testPlan, "step-1"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to mark step-1 as completed for persistence test: %w", err)
	}
	
	stdout, err = execPlanCommand("mark-as-completed", []string{testPlan, "step-1.5"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to mark step-1.5 as completed for persistence test: %w", err)
	}
	
	// Now step-2 should be next and should have its reference
	stdout, err = execPlanCommand("next-step", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to get step-2 as next step: %w", err)
	}
	assertCommandOutput(stdout, []string{"step-2", "guide-B"}, "plan next-step step-2 with references")

	// Test 9: plan remove-steps - Remove a specific step and validate removal
	stdout, err = execPlanCommand("remove-steps", []string{testPlan, "step-3"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to remove step-3: %w", err)
	}
	assertCommandOutput(stdout, []string{"Removed"}, "plan remove-steps step-3")

	// Verify step-3 was actually removed by inspecting the plan
	stdout, err = execPlanCommand("inspect", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to inspect plan after removing step-3: %w", err)
	}
	if strings.Contains(stdout, "step-3") {
		return fmt.Errorf("step-3 was not properly removed from plan")
	}
	// Should now have 3 steps
	assertCommandOutput(stdout, []string{"step-1", "step-1.5", "step-2"}, "plan inspect after step-3 removal")

	// Test 10: plan reorder-steps - Change step order and validate
	stdout, err = execPlanCommand("reorder-steps", []string{testPlan, "step-2", "step-1.5", "step-1"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to reorder steps: %w", err)
	}
	assertCommandOutput(stdout, []string{"Reordered steps"}, "plan reorder-steps")

	// Verify the new order by inspecting
	stdout, err = execPlanCommand("inspect", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to inspect plan after reordering: %w", err)
	}

	// Parse lines to verify step order: step-2, step-1.5, step-1
	lines = strings.Split(stdout, "\n")
	step1NewPosition := -1
	step15NewPosition := -1
	step2NewPosition := -1
	for i, line := range lines {
		if strings.Contains(line, "step-1") && !strings.Contains(line, "step-1.5") {
			step1NewPosition = i
		}
		if strings.Contains(line, "step-1.5") {
			step15NewPosition = i
		}
		if strings.Contains(line, "step-2") {
			step2NewPosition = i
		}
	}
	if step2NewPosition >= step15NewPosition || step15NewPosition >= step1NewPosition {
		return fmt.Errorf("step order incorrect after reordering: step-2 at %d, step-1.5 at %d, step-1 at %d",
			step2NewPosition, step15NewPosition, step1NewPosition)
	}

	// Test 11: plan is-completed - Check completion status (should be false)
	// Note: is-completed uses exit codes to indicate status, so we use execCommand directly
	commandArgs := []string{"plan", "is-completed", testPlan}
	logPlanCommand("plan", []string{"is-completed", testPlan})
	stdout, _, exitCode, err := execCommand(commandArgs, tempDB)
	if err != nil {
		return fmt.Errorf("failed to execute is-completed command: %w", err)
	}
	// For incomplete plan, expect exit code 1 and output "false"
	if exitCode != 1 {
		return fmt.Errorf("expected exit code 1 for incomplete plan, got %d", exitCode)
	}
	assertCommandOutput(stdout, []string{"false"}, "plan is-completed (incomplete)")
	if strings.Contains(stdout, "true") {
		return fmt.Errorf("plan should be incomplete but is-completed returned true")
	}

	// Test 12: Mark all remaining steps as completed
	stdout, err = execPlanCommand("mark-as-completed", []string{testPlan, "step-2"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to mark step-2 as completed: %w", err)
	}
	assertCommandOutput(stdout, []string{"marked as completed"}, "plan mark-as-completed step-2")

	stdout, err = execPlanCommand("mark-as-completed", []string{testPlan, "step-1.5"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to mark step-1.5 as completed: %w", err)
	}
	assertCommandOutput(stdout, []string{"marked as completed"}, "plan mark-as-completed step-1.5")

	stdout, err = execPlanCommand("mark-as-completed", []string{testPlan, "step-1"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to mark step-1 as completed: %w", err)
	}
	assertCommandOutput(stdout, []string{"marked as completed"}, "plan mark-as-completed step-1")

	// Test 13: plan is-completed - Check completion status (should be true)
	// Note: is-completed uses exit codes to indicate status, so we use execCommand directly
	commandArgs = []string{"plan", "is-completed", testPlan}
	logPlanCommand("plan", []string{"is-completed", testPlan})
	stdout, _, exitCode, err = execCommand(commandArgs, tempDB)
	if err != nil {
		return fmt.Errorf("failed to execute final is-completed command: %w", err)
	}
	// For completed plan, expect exit code 0 and output "true"
	if exitCode != 0 {
		return fmt.Errorf("expected exit code 0 for completed plan, got %d", exitCode)
	}
	assertCommandOutput(stdout, []string{"true"}, "plan is-completed (completed)")
	if strings.Contains(stdout, "false") {
		return fmt.Errorf("plan should be completed but is-completed returned false")
	}

	// Test 14: Verify next-step returns nothing when plan is complete
	stdout, err = execPlanCommand("next-step", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to get next step on completed plan: %w", err)
	}
	// Should indicate no next step available
	if strings.Contains(stdout, "step-") {
		return fmt.Errorf("next-step should return no steps for completed plan, but found: %s", stdout)
	}

	// Test 15: plan remove - Cleanup the test plan
	stdout, err = execPlanCommand("remove", []string{testPlan}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to remove test plan: %w", err)
	}
	assertCommandOutput(stdout, []string{"Removed plan"}, "plan remove")

	// Test 16: Verify database cleanup - plan was removed
	stdout, err = execPlanCommand("list", []string{}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to list plans after removal: %w", err)
	}
	if strings.Contains(stdout, testPlan) {
		return fmt.Errorf("plan %s was not properly removed from database", testPlan)
	}

	// Test 17: Additional comprehensive scenario - Create second plan for more edge cases
	testPlan2 := "test-plan-2"
	stdout, err = execPlanCommand("new", []string{testPlan2}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to create second test plan: %w", err)
	}

	// Add steps with multiple acceptance criteria and multiple references
	stdout, err = execPlanCommand("add-step", []string{testPlan2, "multi-step", "Step with multiple criteria", "First criterion", "Second criterion", "Third criterion", "--references", "ref-1,ref-2,ref-3"}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to add step with multiple criteria: %w", err)
	}

	// Verify multiple acceptance criteria and references appear in inspect
	stdout, err = execPlanCommand("inspect", []string{testPlan2}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to inspect plan with multiple criteria: %w", err)
	}
	assertCommandOutput(stdout, []string{
		"multi-step",
		"Step with multiple criteria",
		"First criterion",
		"Second criterion",
		"Third criterion",
		"ref-1", "ref-2", "ref-3",
		"References:",
	}, "plan inspect with multiple acceptance criteria and references")
	
	// Test next-step with multiple references
	stdout, err = execPlanCommand("next-step", []string{testPlan2}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to get next step for multi-reference test: %w", err)
	}
	assertCommandOutput(stdout, []string{"multi-step", "ref-1", "ref-2", "ref-3"}, "plan next-step with multiple references")

	// Clean up second plan
	stdout, err = execPlanCommand("remove", []string{testPlan2}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to remove second test plan: %w", err)
	}

	// Final verification - database should be empty
	stdout, err = execPlanCommand("list", []string{}, tempDB)
	if err != nil {
		return fmt.Errorf("failed to list plans for final verification: %w", err)
	}
	// Should show empty list or "No plans found"
	if strings.Contains(stdout, "test-plan") {
		return fmt.Errorf("database not properly cleaned - found remaining test plans")
	}

	log.Printf("✓ All enhanced plan subcommand tests passed successfully")
	return nil
}

func failTest(format string, args ...interface{}) {
	log.Printf("✗ Test failed: "+format, args...)
	os.Exit(1)
}
