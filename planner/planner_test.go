package planner

import (
	"database/sql" // Import database/sql
	"fmt"
	"os"
	"path/filepath"
	"reflect" // Will be used later for deep comparisons
	"testing"
)

// Helper function to set up a temporary database for testing
func setupTestDB(t *testing.T) (*Planner, func()) {
	t.Helper()
	// Create a temporary directory for the test database
	// t.TempDir() automatically handles cleanup of the directory and its contents
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_planner.db")

	// schema.sql should be in the same directory as the test file (the planner package directory)
	schemaPath := "schema.sql"

	// Check if schema.sql exists at the expected path
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		// If running tests from project root using a pattern like ./...
		// Go sets the working dir to the package dir, so "schema.sql" should still work.
		// If it's truly not found, it's a setup error.
		t.Fatalf("schema.sql not found at %s. It should be in the planner package directory.", schemaPath)
	} else if err != nil {
		t.Fatalf("Error checking for schema.sql at %s: %v", schemaPath, err)
	}

	// Copy schema to the temp dir next to where the db will be created,
	// as New() expects it relative to the db path.
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema file %s: %v", schemaPath, err)
	}
	tmpSchemaPath := filepath.Join(tmpDir, "schema.sql") // This is where New() will look for it
	err = os.WriteFile(tmpSchemaPath, schemaContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write temporary schema file to %s: %v", tmpSchemaPath, err)
	}

	// Create a new planner instance using the temporary database path
	planner, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create new planner for testing: %v", err)
	}

	// Define a cleanup function to close the database
	// (temp dir cleanup is handled by t.TempDir)
	cleanup := func() {
		err := planner.Close()
		if err != nil {
			// Log the error but don't fail the test, as it's a cleanup step
			t.Logf("Warning: Error closing test database: %v", err)
		}
	}

	return planner, cleanup
}

// Basic test for planner creation and schema initialization
func TestNewPlanner(t *testing.T) {
	planner, cleanup := setupTestDB(t)
	defer cleanup() // Ensure cleanup runs even if test panics

	if planner.db == nil {
		t.Fatal("Planner db connection is nil after New()")
	}

	// Check if tables were created (basic check by trying to query them)
	tables := []string{"plans", "steps", "step_acceptance_criteria"}
	for _, table := range tables {
		// Using QueryRow because we don't expect results, just no error
		err := planner.db.QueryRow(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", table)).Scan(new(int))
		// We expect sql.ErrNoRows if the table is empty, which is fine.
		// Any other error indicates a problem (e.g., table doesn't exist).
		if err != nil && err != sql.ErrNoRows {
			t.Errorf("Failed to query '%s' table, schema likely not initialized correctly: %v", table, err)
		}
	}
}

// Test Create method
func TestPlanner_Create(t *testing.T) {
	planner, cleanup := setupTestDB(t)
	defer cleanup()

	planName := "test-plan-create"
	plan, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if plan == nil {
		t.Fatal("Create returned a nil plan")
	}
	if plan.ID != planName {
		t.Errorf("Create returned plan with wrong ID: got %s, want %s", plan.ID, planName)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("Create returned plan with non-empty steps: got %d, want 0", len(plan.Steps))
	}
	if !plan.isNew { // Verify isNew flag is true
		t.Errorf("Create returned plan with isNew = false, want true")
	}

	// Verify NOT in DB yet
	var count int
	err = planner.db.QueryRow("SELECT COUNT(*) FROM plans WHERE id = ?", planName).Scan(&count)
	if err != nil && err != sql.ErrNoRows { // sql.ErrNoRows is expected if not found, other errors are DB issues
		t.Fatalf("Failed to query DB after Create (expected no rows or 0 count): %v", err)
	}
	if count != 0 { // Should be 0 as it's not saved yet
		t.Errorf("Plan count in DB is wrong after Create: got %d, want 0", count)
	}

	// Test creating a plan with the same name (should not error, as it's in-memory only until save)
	// The old test expected an error because Create also saved to DB and hit a UNIQUE constraint.
	// Now, Create only makes an in-memory object.
	// The responsibility of checking for existing plans shifts to the Save method (or a pre-check if desired).
	_, err = planner.Create(planName)
	if err != nil {
		t.Errorf("Creating a second in-memory plan with the same name should not error: %v", err)
	}
}

// Test Get method (basic)
func TestPlanner_Get_Basic(t *testing.T) {
	planner, cleanup := setupTestDB(t)
	defer cleanup()

	planName := "test-plan-get"
	createdPlan, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Setup failed: Could not create plan: %v", err)
	}
	err = planner.Save(createdPlan)
	if err != nil {
		t.Fatalf("Setup failed: Could not save plan: %v", err)
	}

	plan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if plan == nil {
		t.Fatal("Get returned a nil plan")
	}
	if plan.ID != planName {
		t.Errorf("Get returned plan with wrong ID: got %s, want %s", plan.ID, planName)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("Get returned plan with non-empty steps initially: got %d, want 0", len(plan.Steps))
	}

	// Test getting non-existent plan
	_, err = planner.Get("non-existent-plan")
	if err == nil {
		t.Error("Expected error when getting non-existent plan, but got nil")
	}
}

// Test Save and Get methods together (more comprehensive)
// This test also implicitly tests the isNew flag behavior on first save.
func TestPlanner_SaveAndGet(t *testing.T) {
	planner, cleanup := setupTestDB(t)
	defer cleanup()

	planName := "test-plan-save-get"

	// 1. Create the initial plan
	plan, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !plan.isNew {
		t.Fatal("Newly created plan should have isNew = true")
	}

	// 2. Add steps to the in-memory plan
	plan.AddStep("step1", "First step description", []string{"AC1.1", "AC1.2"}, []string{"https://example.com/doc1", "https://example.com/doc2"})
	plan.AddStep("step2", "Second step", []string{"AC2.1"}, []string{"https://example.com/ref"})
	plan.AddStep("step3", "Third step", []string{"AC3.1"}, nil) // No references

	// 3. Save the plan
	err = planner.Save(plan)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if plan.isNew { // isNew should be false after a successful save
		t.Errorf("plan.isNew is true after Save, want false")
	}

	// 4. Get the plan back
	retrievedPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Get after Save failed: %v", err)
	}

	// 5. Verify the retrieved plan
	if retrievedPlan.ID != planName {
		t.Errorf("Retrieved plan ID mismatch: got %s, want %s", retrievedPlan.ID, planName)
	}
	if len(retrievedPlan.Steps) != 3 {
		t.Fatalf("Retrieved plan step count mismatch: got %d, want 3", len(retrievedPlan.Steps))
	}

	// Verify step 1
	step1 := retrievedPlan.Steps[0]
	if step1.ID() != "step1" {
		t.Errorf("Step 1 ID mismatch")
	}
	if step1.Description() != "First step description" {
		t.Errorf("Step 1 Description mismatch")
	}
	if step1.Status() != "TODO" {
		t.Errorf("Step 1 Status mismatch")
	}
	if !reflect.DeepEqual(step1.AcceptanceCriteria(), []string{"AC1.1", "AC1.2"}) {
		t.Errorf("Step 1 Acceptance Criteria mismatch: got %v", step1.AcceptanceCriteria())
	}
	if !reflect.DeepEqual(step1.References(), []string{"https://example.com/doc1", "https://example.com/doc2"}) {
		t.Errorf("Step 1 References mismatch: got %v", step1.References())
	}

	// Verify step 2
	step2 := retrievedPlan.Steps[1]
	if step2.ID() != "step2" {
		t.Errorf("Step 2 ID mismatch")
	}
	if step2.Description() != "Second step" {
		t.Errorf("Step 2 Description mismatch")
	}
	if step2.Status() != "TODO" {
		t.Errorf("Step 2 Status mismatch")
	}
	if !reflect.DeepEqual(step2.AcceptanceCriteria(), []string{"AC2.1"}) {
		t.Errorf("Step 2 Acceptance Criteria mismatch: got %v", step2.AcceptanceCriteria())
	}
	if !reflect.DeepEqual(step2.References(), []string{"https://example.com/ref"}) {
		t.Errorf("Step 2 References mismatch: got %v", step2.References())
	}

	// Verify step 3
	step3 := retrievedPlan.Steps[2]
	if step3.ID() != "step3" {
		t.Errorf("Step 3 ID mismatch")
	}
	if step3.Description() != "Third step" {
		t.Errorf("Step 3 Description mismatch")
	}
	if step3.Status() != "TODO" {
		t.Errorf("Step 3 Status mismatch")
	}
	if !reflect.DeepEqual(step3.AcceptanceCriteria(), []string{"AC3.1"}) {
		t.Errorf("Step 3 Acceptance Criteria mismatch: got %v", step3.AcceptanceCriteria())
	}
	if !reflect.DeepEqual(step3.References(), []string{}) {
		t.Errorf("Step 3 References mismatch (should be empty): got %v", step3.References())
	}

	// 6. Modify the plan (e.g., remove step, change status, reorder)
	retrievedPlan.RemoveSteps([]string{"step1"})
	// retrievedPlan.Steps[0].status = "DONE" // Mark step2 as DONE (it's now at index 0)
	err = retrievedPlan.MarkAsCompleted("step2") // Mark step2 as DONE (it's now at index 0)
	if err != nil {
		t.Fatalf("MarkAsCompleted failed: %v", err)
	}
	retrievedPlan.AddStep("step4", "Fourth step", nil, []string{"https://example.com/newref"})

	// Reorder (step4, step2, step3) - Note: step1 was removed
	retrievedPlan.Reorder([]string{"step4", "step2", "step3"})

	// 7. Save again
	err = planner.Save(retrievedPlan)
	if err != nil {
		t.Fatalf("Second Save failed: %v", err)
	}

	// 8. Get again
	finalPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}

	// 9. Verify final state
	if len(finalPlan.Steps) != 3 {
		t.Fatalf("Final plan step count mismatch: got %d, want 3", len(finalPlan.Steps))
	}

	// Check order and content
	if finalPlan.Steps[0].ID() != "step4" {
		t.Errorf("Final Step 1 ID mismatch (expected step4)")
	}
	if finalPlan.Steps[0].Status() != "TODO" {
		t.Errorf("Final Step 1 Status mismatch (expected TODO)")
	}
	if !reflect.DeepEqual(finalPlan.Steps[0].References(), []string{"https://example.com/newref"}) {
		t.Errorf("Final Step 1 References mismatch: got %v", finalPlan.Steps[0].References())
	}
	if finalPlan.Steps[1].ID() != "step2" {
		t.Errorf("Final Step 2 ID mismatch (expected step2)")
	}
	if finalPlan.Steps[1].Status() != "DONE" {
		t.Errorf("Final Step 2 Status mismatch (expected DONE)")
	}
	if !reflect.DeepEqual(finalPlan.Steps[1].References(), []string{"https://example.com/ref"}) {
		t.Errorf("Final Step 2 References mismatch: got %v", finalPlan.Steps[1].References())
	}
	if finalPlan.Steps[2].ID() != "step3" {
		t.Errorf("Final Step 3 ID mismatch (expected step3)")
	}
	if finalPlan.Steps[2].Status() != "TODO" {
		t.Errorf("Final Step 3 Status mismatch (expected TODO)")
	}
	if !reflect.DeepEqual(finalPlan.Steps[2].References(), []string{}) {
		t.Errorf("Final Step 3 References mismatch (should be empty): got %v", finalPlan.Steps[2].References())
	}
	if finalPlan.isNew { // Should be false as it was retrieved from DB
		t.Errorf("finalPlan.isNew is true after Get, want false")
	}

}

// TestPlan_MarkStatus tests the in-memory status changes of steps in a Plan.
func TestPlan_MarkStatus(t *testing.T) {
	plan := &Plan{ID: "test-status-plan", Steps: []*Step{}}
	plan.AddStep("step1", "Step 1 desc", nil, nil)
	plan.AddStep("step2", "Step 2 desc", nil, nil)

	// Check initial status (should be TODO)
	if plan.Steps[0].Status() != "TODO" {
		t.Errorf("Initial status of step1 was %s, expected TODO", plan.Steps[0].Status())
	}

	// Mark step1 as completed
	err := plan.MarkAsCompleted("step1")
	if err != nil {
		t.Fatalf("MarkAsCompleted for step1 failed: %v", err)
	}
	if plan.Steps[0].Status() != "DONE" {
		t.Errorf("Status of step1 after MarkAsCompleted was %s, expected DONE", plan.Steps[0].Status())
	}
	// Verify step2 is still TODO
	if plan.Steps[1].Status() != "TODO" {
		t.Errorf("Status of step2 was %s, expected TODO", plan.Steps[1].Status())
	}

	// Mark step1 back to incomplete
	err = plan.MarkAsIncomplete("step1")
	if err != nil {
		t.Fatalf("MarkAsIncomplete for step1 failed: %v", err)
	}
	if plan.Steps[0].Status() != "TODO" {
		t.Errorf("Status of step1 after MarkAsIncomplete was %s, expected TODO", plan.Steps[0].Status())
	}

	// Mark non-existent step
	err = plan.MarkAsCompleted("non-existent-step")
	if err == nil {
		t.Error("Expected error when marking non-existent step as completed, got nil")
	}
	err = plan.MarkAsIncomplete("non-existent-step")
	if err == nil {
		t.Error("Expected error when marking non-existent step as incomplete, got nil")
	}
}

// TestPlanner_Save_NewAndExisting specifically tests the isNew logic with Save.
func TestPlanner_Save_NewAndExisting(t *testing.T) {
	planner, cleanup := setupTestDB(t)
	defer cleanup()

	planName := "test-plan-new-existing"

	// 1. Create a new plan
	plan1, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Create failed for plan1: %v", err)
	}
	if !plan1.isNew {
		t.Fatal("plan1.isNew should be true initially")
	}
	plan1.AddStep("s1", "Step 1", nil, nil)

	// 2. Save it (should be an INSERT)
	err = planner.Save(plan1)
	if err != nil {
		t.Fatalf("Save failed for new plan1: %v", err)
	}
	if plan1.isNew {
		t.Error("plan1.isNew should be false after first save")
	}

	// 3. Verify in DB
	var count int
	err = planner.db.QueryRow("SELECT COUNT(*) FROM plans WHERE id = ?", planName).Scan(&count)
	if err != nil {
		t.Fatalf("DB query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Plan count in DB should be 1 after first save, got %d", count)
	}

	// 4. Modify and save again (should be an UPDATE)
	plan1.AddStep("s2", "Step 2", nil, nil)
	err = planner.Save(plan1) // plan1.isNew is already false
	if err != nil {
		t.Fatalf("Second save of plan1 failed: %v", err)
	}
	if plan1.isNew { // Still should be false
		t.Error("plan1.isNew should remain false after second save")
	}

	// 5. Get it and verify
	retrievedPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Get failed after second save: %v", err)
	}
	if len(retrievedPlan.Steps) != 2 {
		t.Errorf("Expected 2 steps after second save, got %d", len(retrievedPlan.Steps))
	}
	if retrievedPlan.isNew { // isNew should be false for plans loaded from DB
		t.Error("Retrieved plan should have isNew = false")
	}

	// 6. Test saving a plan that was retrieved (so isNew is false)
	retrievedPlan.AddStep("s3", "Step 3", nil, nil)
	err = planner.Save(retrievedPlan)
	if err != nil {
		t.Fatalf("Save of retrieved plan failed: %v", err)
	}
	if retrievedPlan.isNew {
		t.Error("retrievedPlan.isNew should remain false after saving again")
	}

	finalPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Get failed after third save: %v", err)
	}
	if len(finalPlan.Steps) != 3 {
		t.Errorf("Expected 3 steps after third save, got %d", len(finalPlan.Steps))
	}

	// 7. Test saving a "new" plan when one with the same ID already exists in DB
	// This should fail because our Save now checks for UNIQUE constraint on INSERT
	plan2, err := planner.Create(planName) // Creates a new in-memory plan with isNew = true
	if err != nil {
		t.Fatalf("Create failed for plan2: %v", err)
	}
	plan2.AddStep("s4", "Step 4", nil, nil)
	err = planner.Save(plan2) // isNew is true, so Save will try to INSERT
	if err == nil {
		t.Error("Expected error when saving a new plan with an ID that already exists in DB, but got nil")
	}

	// 8. Test saving a plan that is NOT new but does not exist in DB (should fail)
	nonExistentPlan := &Plan{ID: "non-existent-plan", isNew: false}
	nonExistentPlan.AddStep("s1", "some step", nil, nil)
	err = planner.Save(nonExistentPlan)
	if err == nil {
		t.Error("Expected error when saving a non-new plan that does not exist in DB, got nil")
	}

}

// TestStep_References tests the References() method specifically.
func TestStep_References(t *testing.T) {
	plan := &Plan{ID: "test-references-plan", Steps: []*Step{}}

	// Test step with multiple references
	plan.AddStep("step1", "Step 1 desc", nil, []string{"https://example.com/doc1", "https://github.com/repo", "https://docs.example.com"})
	step1 := plan.Steps[0]
	refs := step1.References()
	expected := []string{"https://example.com/doc1", "https://github.com/repo", "https://docs.example.com"}
	if !reflect.DeepEqual(refs, expected) {
		t.Errorf("References() returned %v, want %v", refs, expected)
	}

	// Test step with single reference
	plan.AddStep("step2", "Step 2 desc", nil, []string{"https://single.com"})
	step2 := plan.Steps[1]
	refs2 := step2.References()
	expected2 := []string{"https://single.com"}
	if !reflect.DeepEqual(refs2, expected2) {
		t.Errorf("References() returned %v, want %v", refs2, expected2)
	}

	// Test step with no references
	plan.AddStep("step3", "Step 3 desc", nil, nil)
	step3 := plan.Steps[2]
	refs3 := step3.References()
	if len(refs3) != 0 {
		t.Errorf("References() for step with nil references returned %v, want empty slice", refs3)
	}

	// Test step with empty references slice
	plan.AddStep("step4", "Step 4 desc", nil, []string{})
	step4 := plan.Steps[3]
	refs4 := step4.References()
	if !reflect.DeepEqual(refs4, []string{}) {
		t.Errorf("References() for step with empty references returned %v, want []", refs4)
	}
}

// TestPlan_AddStepWithReferences tests the AddStep method specifically for references handling.
func TestPlan_AddStepWithReferences(t *testing.T) {
	plan := &Plan{ID: "test-addstep-references", Steps: []*Step{}}

	// Test adding step with multiple references
	refs1 := []string{"https://example.com/guide", "https://api.docs.com", "https://tutorial.com"}
	plan.AddStep("step1", "First step", []string{"AC1"}, refs1)

	if len(plan.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
	}

	step := plan.Steps[0]
	if step.ID() != "step1" {
		t.Errorf("Step ID = %s, want step1", step.ID())
	}
	if step.Description() != "First step" {
		t.Errorf("Step Description = %s, want 'First step'", step.Description())
	}
	if !reflect.DeepEqual(step.References(), refs1) {
		t.Errorf("Step References = %v, want %v", step.References(), refs1)
	}

	// Test adding step with nil references
	plan.AddStep("step2", "Second step", nil, nil)
	step2 := plan.Steps[1]
	if len(step2.References()) != 0 {
		t.Errorf("Step with nil references should return empty slice, got %v", step2.References())
	}

	// Test adding step with empty references
	plan.AddStep("step3", "Third step", nil, []string{})
	step3 := plan.Steps[2]
	if !reflect.DeepEqual(step3.References(), []string{}) {
		t.Errorf("Step with empty references should return empty slice, got %v", step3.References())
	}
}

// TestPlanner_ReferencesPersistence tests that references persist through Save/Get cycles.
func TestPlanner_ReferencesPersistence(t *testing.T) {
	planner, cleanup := setupTestDB(t)
	defer cleanup()

	planName := "test-references-persistence"

	// Create plan with steps that have different reference configurations
	plan, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Step with multiple references
	refs1 := []string{"https://example.com/doc1", "https://github.com/user/repo", "https://wiki.example.com/page"}
	plan.AddStep("step1", "Step with multiple refs", []string{"AC1"}, refs1)

	// Step with single reference
	refs2 := []string{"https://single-ref.com"}
	plan.AddStep("step2", "Step with single ref", []string{"AC2"}, refs2)

	// Step with no references
	plan.AddStep("step3", "Step with no refs", []string{"AC3"}, nil)

	// Step with empty references
	plan.AddStep("step4", "Step with empty refs", []string{"AC4"}, []string{})

	// Save the plan
	err = planner.Save(plan)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Retrieve the plan
	retrievedPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Verify all references persisted correctly
	if len(retrievedPlan.Steps) != 4 {
		t.Fatalf("Expected 4 steps, got %d", len(retrievedPlan.Steps))
	}

	// Check step1 references
	step1 := retrievedPlan.Steps[0]
	if !reflect.DeepEqual(step1.References(), refs1) {
		t.Errorf("Step1 references mismatch: got %v, want %v", step1.References(), refs1)
	}

	// Check step2 references
	step2 := retrievedPlan.Steps[1]
	if !reflect.DeepEqual(step2.References(), refs2) {
		t.Errorf("Step2 references mismatch: got %v, want %v", step2.References(), refs2)
	}

	// Check step3 references (should be empty)
	step3 := retrievedPlan.Steps[2]
	if !reflect.DeepEqual(step3.References(), []string{}) {
		t.Errorf("Step3 references should be empty: got %v", step3.References())
	}

	// Check step4 references (should be empty)
	step4 := retrievedPlan.Steps[3]
	if !reflect.DeepEqual(step4.References(), []string{}) {
		t.Errorf("Step4 references should be empty: got %v", step4.References())
	}
}

// TestPlanner_ReferencesOrder tests that reference order is maintained.
func TestPlanner_ReferencesOrder(t *testing.T) {
	planner, cleanup := setupTestDB(t)
	defer cleanup()

	planName := "test-references-order"

	// Create plan with step that has references in specific order
	plan, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// References in specific order
	orderedRefs := []string{
		"https://first.com",
		"https://second.com",
		"https://third.com",
		"https://fourth.com",
		"https://fifth.com",
	}
	plan.AddStep("step1", "Step with ordered refs", nil, orderedRefs)

	// Save and retrieve
	err = planner.Save(plan)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	retrievedPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Verify order is maintained
	step := retrievedPlan.Steps[0]
	retrievedRefs := step.References()
	if !reflect.DeepEqual(retrievedRefs, orderedRefs) {
		t.Errorf("Reference order not maintained.\nGot:  %v\nWant: %v", retrievedRefs, orderedRefs)
	}

	// Verify each reference is in the correct position
	for i, expectedRef := range orderedRefs {
		if i >= len(retrievedRefs) {
			t.Errorf("Missing reference at position %d: expected %s", i, expectedRef)
			continue
		}
		if retrievedRefs[i] != expectedRef {
			t.Errorf("Reference at position %d: got %s, want %s", i, retrievedRefs[i], expectedRef)
		}
	}
}

// TestPlanner_ReferencesWithPlanOperations tests that references work correctly with plan operations.
func TestPlanner_ReferencesWithPlanOperations(t *testing.T) {
	planner, cleanup := setupTestDB(t)
	defer cleanup()

	planName := "test-references-operations"

	plan, err := planner.Create(planName)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Add steps with references
	plan.AddStep("step1", "First step", nil, []string{"https://step1.com"})
	plan.AddStep("step2", "Second step", nil, []string{"https://step2-a.com", "https://step2-b.com"})
	plan.AddStep("step3", "Third step", nil, []string{"https://step3.com"})

	// Save initial state
	err = planner.Save(plan)
	if err != nil {
		t.Fatalf("Initial save failed: %v", err)
	}

	// Retrieve and perform operations
	retrievedPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Remove step2 (which has multiple references)
	retrievedPlan.RemoveSteps([]string{"step2"})

	// Add a new step with references
	retrievedPlan.AddStep("step4", "Fourth step", nil, []string{"https://step4.com", "https://step4-alt.com"})

	// Reorder steps
	retrievedPlan.Reorder([]string{"step4", "step1", "step3"})

	// Save the modified plan
	err = planner.Save(retrievedPlan)
	if err != nil {
		t.Fatalf("Modified save failed: %v", err)
	}

	// Get final state
	finalPlan, err := planner.Get(planName)
	if err != nil {
		t.Fatalf("Final get failed: %v", err)
	}

	// Verify final state
	if len(finalPlan.Steps) != 3 {
		t.Fatalf("Expected 3 steps in final plan, got %d", len(finalPlan.Steps))
	}

	// Check order and references
	expectedOrder := []struct {
		id   string
		refs []string
	}{
		{"step4", []string{"https://step4.com", "https://step4-alt.com"}},
		{"step1", []string{"https://step1.com"}},
		{"step3", []string{"https://step3.com"}},
	}

	for i, expected := range expectedOrder {
		step := finalPlan.Steps[i]
		if step.ID() != expected.id {
			t.Errorf("Step %d ID: got %s, want %s", i, step.ID(), expected.id)
		}
		if !reflect.DeepEqual(step.References(), expected.refs) {
			t.Errorf("Step %d references: got %v, want %v", i, step.References(), expected.refs)
		}
	}
}

// --- Add tests for List, Remove, Compact, MarkAsComplete/Incomplete etc. ---
