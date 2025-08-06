package planner

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestDatabaseMigration tests the migration from old schema (without step_references table)
// to new schema (with step_references table) to ensure backward compatibility.
func TestDatabaseMigration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "migration_test.db")
	schemaPath := filepath.Join(tempDir, "schema.sql")

	// Copy schema.sql to temp directory for planner initialization
	schemaContent, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatalf("Failed to read schema.sql: %v", err)
	}
	err = os.WriteFile(schemaPath, schemaContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write schema.sql to temp dir: %v", err)
	}

	// Step 1: Create an "old" database without step_references table
	// This simulates a database from before the references feature was added
	t.Run("CreateOldDatabase", func(t *testing.T) {
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Enable foreign key constraints
		_, err = db.Exec("PRAGMA foreign_keys = ON;")
		if err != nil {
			t.Fatalf("Failed to enable foreign key constraints: %v", err)
		}

		// Create old schema (without step_references table)
		oldSchema := `
		-- Old schema without step_references table
		CREATE TABLE IF NOT EXISTS plans (
			id TEXT PRIMARY KEY NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TRIGGER IF NOT EXISTS plans_updated_at
		AFTER UPDATE ON plans
		FOR EACH ROW
		BEGIN
			UPDATE plans SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
		END;

		CREATE TABLE IF NOT EXISTS steps (
			id TEXT NOT NULL,
			plan_id TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL CHECK(status IN ('TODO', 'DONE')),
			step_order INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (plan_id, id),
			FOREIGN KEY (plan_id) REFERENCES plans(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_steps_plan_id ON steps(plan_id);

		CREATE TRIGGER IF NOT EXISTS steps_updated_at
		AFTER UPDATE ON steps
		FOR EACH ROW
		BEGIN
			UPDATE steps SET updated_at = CURRENT_TIMESTAMP WHERE plan_id = OLD.plan_id AND id = OLD.id;
			UPDATE plans SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.plan_id;
		END;

		CREATE TABLE IF NOT EXISTS step_acceptance_criteria (
			plan_id TEXT NOT NULL,
			step_id TEXT NOT NULL,
			criterion TEXT NOT NULL,
			criterion_order INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (plan_id, step_id, criterion_order),
			FOREIGN KEY (plan_id, step_id) REFERENCES steps(plan_id, id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_step_acceptance_criteria_plan_step ON step_acceptance_criteria(plan_id, step_id);

		CREATE TRIGGER IF NOT EXISTS step_acceptance_criteria_updated_at
		AFTER INSERT ON step_acceptance_criteria
		FOR EACH ROW
		BEGIN
			UPDATE steps SET updated_at = CURRENT_TIMESTAMP 
			WHERE plan_id = NEW.plan_id AND id = NEW.step_id;
			
			UPDATE plans SET updated_at = CURRENT_TIMESTAMP 
			WHERE id = NEW.plan_id;
		END;
		`

		_, err = db.Exec(oldSchema)
		if err != nil {
			t.Fatalf("Failed to create old schema: %v", err)
		}

		// Insert test data into old schema
		_, err = db.Exec("INSERT INTO plans (id) VALUES ('test-plan')")
		if err != nil {
			t.Fatalf("Failed to insert test plan: %v", err)
		}

		_, err = db.Exec(`INSERT INTO steps (id, plan_id, description, status, step_order) 
						  VALUES ('step1', 'test-plan', 'Test step 1', 'TODO', 0)`)
		if err != nil {
			t.Fatalf("Failed to insert test step: %v", err)
		}

		_, err = db.Exec(`INSERT INTO step_acceptance_criteria (plan_id, step_id, criterion, criterion_order)
						  VALUES ('test-plan', 'step1', 'Test criterion', 0)`)
		if err != nil {
			t.Fatalf("Failed to insert test acceptance criterion: %v", err)
		}

		// Verify step_references table does NOT exist yet
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='step_references'").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check for step_references table: %v", err)
		}
		if count != 0 {
			t.Fatalf("step_references table should not exist in old schema, but found %d", count)
		}

		// Verify existing data is present
		var planID string
		err = db.QueryRow("SELECT id FROM plans WHERE id = 'test-plan'").Scan(&planID)
		if err != nil {
			t.Fatalf("Failed to retrieve test plan: %v", err)
		}
		if planID != "test-plan" {
			t.Fatalf("Expected plan ID 'test-plan', got '%s'", planID)
		}

		var stepID, stepDescription string
		err = db.QueryRow("SELECT id, description FROM steps WHERE plan_id = 'test-plan'").Scan(&stepID, &stepDescription)
		if err != nil {
			t.Fatalf("Failed to retrieve test step: %v", err)
		}
		if stepID != "step1" || stepDescription != "Test step 1" {
			t.Fatalf("Expected step 'step1' with description 'Test step 1', got '%s' with '%s'", stepID, stepDescription)
		}
	})

	// Step 2: Test migration by opening the old database with new planner
	t.Run("TestMigration", func(t *testing.T) {
		// This should automatically execute the full schema.sql, creating step_references table
		planner, err := New(dbPath)
		if err != nil {
			t.Fatalf("Failed to create planner with old database: %v", err)
		}
		defer planner.Close()

		// Verify step_references table now exists
		var count int
		err = planner.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='step_references'").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check for step_references table after migration: %v", err)
		}
		if count != 1 {
			t.Fatalf("step_references table should exist after migration, but found %d tables", count)
		}

		// Verify all existing data is still intact
		var planID string
		err = planner.db.QueryRow("SELECT id FROM plans WHERE id = 'test-plan'").Scan(&planID)
		if err != nil {
			t.Fatalf("Failed to retrieve test plan after migration: %v", err)
		}
		if planID != "test-plan" {
			t.Fatalf("Expected plan ID 'test-plan' after migration, got '%s'", planID)
		}

		var stepID, stepDescription string
		err = planner.db.QueryRow("SELECT id, description FROM steps WHERE plan_id = 'test-plan'").Scan(&stepID, &stepDescription)
		if err != nil {
			t.Fatalf("Failed to retrieve test step after migration: %v", err)
		}
		if stepID != "step1" || stepDescription != "Test step 1" {
			t.Fatalf("Expected step 'step1' with description 'Test step 1' after migration, got '%s' with '%s'", stepID, stepDescription)
		}

		var criterion string
		err = planner.db.QueryRow("SELECT criterion FROM step_acceptance_criteria WHERE plan_id = 'test-plan' AND step_id = 'step1'").Scan(&criterion)
		if err != nil {
			t.Fatalf("Failed to retrieve test acceptance criterion after migration: %v", err)
		}
		if criterion != "Test criterion" {
			t.Fatalf("Expected criterion 'Test criterion' after migration, got '%s'", criterion)
		}
	})

	// Step 3: Test that existing functionality works after migration
	t.Run("TestExistingFunctionalityAfterMigration", func(t *testing.T) {
		planner, err := New(dbPath)
		if err != nil {
			t.Fatalf("Failed to create planner: %v", err)
		}
		defer planner.Close()

		// Load the existing plan
		plan, err := planner.Get("test-plan")
		if err != nil {
			t.Fatalf("Failed to get test plan: %v", err)
		}

		// Verify plan data is correct
		if plan.ID != "test-plan" {
			t.Fatalf("Expected plan ID 'test-plan', got '%s'", plan.ID)
		}
		if len(plan.Steps) != 1 {
			t.Fatalf("Expected 1 step, got %d", len(plan.Steps))
		}

		step := plan.Steps[0]
		if step.ID() != "step1" {
			t.Fatalf("Expected step ID 'step1', got '%s'", step.ID())
		}
		if step.Description() != "Test step 1" {
			t.Fatalf("Expected description 'Test step 1', got '%s'", step.Description())
		}
		if step.Status() != "TODO" {
			t.Fatalf("Expected status 'TODO', got '%s'", step.Status())
		}
		if len(step.AcceptanceCriteria()) != 1 {
			t.Fatalf("Expected 1 acceptance criterion, got %d", len(step.AcceptanceCriteria()))
		}
		if step.AcceptanceCriteria()[0] != "Test criterion" {
			t.Fatalf("Expected criterion 'Test criterion', got '%s'", step.AcceptanceCriteria()[0])
		}

		// References should be empty (old data had no references)
		if len(step.References()) != 0 {
			t.Fatalf("Expected 0 references for old data, got %d", len(step.References()))
		}

		// Test that we can modify the plan (existing functionality)
		err = plan.MarkAsCompleted("step1")
		if err != nil {
			t.Fatalf("Failed to mark step as completed: %v", err)
		}

		err = planner.Save(plan)
		if err != nil {
			t.Fatalf("Failed to save plan after modification: %v", err)
		}

		// Verify the change was saved
		reloadedPlan, err := planner.Get("test-plan")
		if err != nil {
			t.Fatalf("Failed to reload plan: %v", err)
		}
		if reloadedPlan.Steps[0].Status() != "DONE" {
			t.Fatalf("Expected status 'DONE' after save, got '%s'", reloadedPlan.Steps[0].Status())
		}
	})

	// Step 4: Test that new functionality (references) works with existing data
	t.Run("TestNewFunctionalityWithExistingData", func(t *testing.T) {
		planner, err := New(dbPath)
		if err != nil {
			t.Fatalf("Failed to create planner: %v", err)
		}
		defer planner.Close()

		// Load the existing plan
		plan, err := planner.Get("test-plan")
		if err != nil {
			t.Fatalf("Failed to get test plan: %v", err)
		}

		// Add a step with references to the existing plan
		plan.AddStep("step2", "Test step with references",
			[]string{"Reference criterion"},
			[]string{"https://example.com/ref1", "https://example.com/ref2"})

		err = planner.Save(plan)
		if err != nil {
			t.Fatalf("Failed to save plan with new step containing references: %v", err)
		}

		// Reload and verify the references were saved correctly
		reloadedPlan, err := planner.Get("test-plan")
		if err != nil {
			t.Fatalf("Failed to reload plan: %v", err)
		}

		if len(reloadedPlan.Steps) != 2 {
			t.Fatalf("Expected 2 steps after adding new step, got %d", len(reloadedPlan.Steps))
		}

		// Find the new step
		var newStep *Step
		for _, step := range reloadedPlan.Steps {
			if step.ID() == "step2" {
				newStep = step
				break
			}
		}
		if newStep == nil {
			t.Fatalf("Failed to find new step 'step2' in reloaded plan")
		}

		// Verify references were saved and loaded correctly
		if len(newStep.References()) != 2 {
			t.Fatalf("Expected 2 references, got %d", len(newStep.References()))
		}
		if newStep.References()[0] != "https://example.com/ref1" {
			t.Fatalf("Expected first reference 'https://example.com/ref1', got '%s'", newStep.References()[0])
		}
		if newStep.References()[1] != "https://example.com/ref2" {
			t.Fatalf("Expected second reference 'https://example.com/ref2', got '%s'", newStep.References()[1])
		}

		// Verify the old step still exists and has no references
		var oldStep *Step
		for _, step := range reloadedPlan.Steps {
			if step.ID() == "step1" {
				oldStep = step
				break
			}
		}
		if oldStep == nil {
			t.Fatalf("Failed to find old step 'step1' in reloaded plan")
		}
		if len(oldStep.References()) != 0 {
			t.Fatalf("Expected 0 references for old step, got %d", len(oldStep.References()))
		}
	})

	// Step 5: Test backward compatibility - old code without references still works
	t.Run("TestBackwardCompatibility", func(t *testing.T) {
		planner, err := New(dbPath)
		if err != nil {
			t.Fatalf("Failed to create planner: %v", err)
		}
		defer planner.Close()

		// Create a plan without references (simulating old code)
		plan, err := planner.Create("backward-compat-plan")
		if err != nil {
			t.Fatalf("Failed to create plan: %v", err)
		}

		// Add steps without references (old API usage)
		plan.AddStep("step1", "Step without references", []string{"Criterion 1"}, nil)
		plan.AddStep("step2", "Another step", []string{}, []string{}) // Empty slices

		err = planner.Save(plan)
		if err != nil {
			t.Fatalf("Failed to save plan without references: %v", err)
		}

		// Reload and verify
		reloadedPlan, err := planner.Get("backward-compat-plan")
		if err != nil {
			t.Fatalf("Failed to reload backward compatibility plan: %v", err)
		}

		if len(reloadedPlan.Steps) != 2 {
			t.Fatalf("Expected 2 steps, got %d", len(reloadedPlan.Steps))
		}

		for _, step := range reloadedPlan.Steps {
			if len(step.References()) != 0 {
				t.Fatalf("Expected 0 references for step %s, got %d", step.ID(), len(step.References()))
			}
		}

		// Test that List() still works
		plans, err := planner.List()
		if err != nil {
			t.Fatalf("Failed to list plans: %v", err)
		}

		// Should have at least our test plans
		if len(plans) < 2 {
			t.Fatalf("Expected at least 2 plans, got %d", len(plans))
		}

		// Test other operations still work
		nextStep := reloadedPlan.NextStep()
		if nextStep == nil {
			t.Fatalf("Expected to find next step")
		}
		if nextStep.ID() != "step1" {
			t.Fatalf("Expected next step 'step1', got '%s'", nextStep.ID())
		}

		if reloadedPlan.IsCompleted() {
			t.Fatalf("Expected plan to not be completed")
		}
	})
}

// TestSchemaValidation ensures that schema.sql uses proper IF NOT EXISTS clauses
func TestSchemaValidation(t *testing.T) {
	schemaContent, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatalf("Failed to read schema.sql: %v", err)
	}

	schema := string(schemaContent)

	// Check that all CREATE TABLE statements use IF NOT EXISTS
	tables := []string{"plans", "steps", "step_acceptance_criteria", "step_references"}
	for _, table := range tables {
		expectedPattern := "CREATE TABLE IF NOT EXISTS " + table
		if !contains(schema, expectedPattern) {
			t.Errorf("Schema missing 'IF NOT EXISTS' for table '%s'", table)
		}
	}

	// Check that all CREATE INDEX statements use IF NOT EXISTS
	indexes := []string{
		"idx_steps_plan_id",
		"idx_step_acceptance_criteria_plan_step",
		"idx_step_references_plan_step",
	}
	for _, index := range indexes {
		expectedPattern := "CREATE INDEX IF NOT EXISTS " + index
		if !contains(schema, expectedPattern) {
			t.Errorf("Schema missing 'IF NOT EXISTS' for index '%s'", index)
		}
	}

	// Check that all CREATE TRIGGER statements use IF NOT EXISTS
	triggers := []string{
		"plans_updated_at",
		"steps_updated_at",
		"step_acceptance_criteria_updated_at",
		"step_references_updated_at",
	}
	for _, trigger := range triggers {
		expectedPattern := "CREATE TRIGGER IF NOT EXISTS " + trigger
		if !contains(schema, expectedPattern) {
			t.Errorf("Schema missing 'IF NOT EXISTS' for trigger '%s'", trigger)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAtIndex(s, substr)))
}

func containsAtIndex(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
