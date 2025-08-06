-- Database schema for storing plans

-- Enforce foreign key constraints
PRAGMA foreign_keys = ON;

-- plans table: Stores information about each plan
CREATE TABLE IF NOT EXISTS plans (
    id TEXT PRIMARY KEY NOT NULL, -- Unique identifier for the plan (e.g., "active", "feature-x")
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trigger to update updated_at timestamp on plans table
CREATE TRIGGER IF NOT EXISTS plans_updated_at
AFTER UPDATE ON plans
FOR EACH ROW
BEGIN
    UPDATE plans SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

-- steps table: Stores information about each step within a plan
CREATE TABLE IF NOT EXISTS steps (
    id TEXT NOT NULL, -- Short identifier for the step (e.g., "add-tests")
    plan_id TEXT NOT NULL, -- Foreign key referencing plans.id
    description TEXT,
    status TEXT NOT NULL CHECK(status IN ('TODO', 'DONE')),
    step_order INTEGER NOT NULL, -- Order of steps within a plan
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (plan_id, id),
    FOREIGN KEY (plan_id) REFERENCES plans(id) ON DELETE CASCADE
);

-- Index for faster step lookup by plan_id
CREATE INDEX IF NOT EXISTS idx_steps_plan_id ON steps(plan_id);

-- Trigger to update updated_at timestamp on steps table
CREATE TRIGGER IF NOT EXISTS steps_updated_at
AFTER UPDATE ON steps
FOR EACH ROW
BEGIN
    UPDATE steps SET updated_at = CURRENT_TIMESTAMP WHERE plan_id = OLD.plan_id AND id = OLD.id;
    -- Also update the parent plan's updated_at timestamp
    UPDATE plans SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.plan_id;
END;

-- step_acceptance_criteria table: Stores acceptance criteria for each step
CREATE TABLE IF NOT EXISTS step_acceptance_criteria (
    plan_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    criterion TEXT NOT NULL,
    criterion_order INTEGER NOT NULL, -- Order of criteria for a step
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (plan_id, step_id, criterion_order),
    FOREIGN KEY (plan_id, step_id) REFERENCES steps(plan_id, id) ON DELETE CASCADE
);

-- Index for faster acceptance criteria lookup
CREATE INDEX IF NOT EXISTS idx_step_acceptance_criteria_plan_step ON step_acceptance_criteria(plan_id, step_id);

-- Trigger to update parent plan's and step's updated_at timestamp when criteria change
-- Note: SQLite does not directly support triggers on INSERT/DELETE/UPDATE for a table
-- that cause an update on a *grandparent* table (plans via steps) in a simple way
-- without more complex recursive triggers or application-level logic.
-- The steps_updated_at trigger will handle plan update when a step changes.
-- For criteria, we'll rely on application logic or direct step update if its criteria change.
