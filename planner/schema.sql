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

-- Trigger to update parent plan's and step's updated_at timestamp when acceptance criteria change
CREATE TRIGGER IF NOT EXISTS step_acceptance_criteria_updated_at
AFTER INSERT ON step_acceptance_criteria
FOR EACH ROW
BEGIN
    -- Update the parent step's updated_at timestamp
    UPDATE steps SET updated_at = CURRENT_TIMESTAMP 
    WHERE plan_id = NEW.plan_id AND id = NEW.step_id;
    
    -- Update the parent plan's updated_at timestamp
    UPDATE plans SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.plan_id;
END;

-- step_references table: Stores reference URLs for each step
CREATE TABLE IF NOT EXISTS step_references (
    plan_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    reference_url TEXT NOT NULL,
    reference_order INTEGER NOT NULL, -- Order of references for a step
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (plan_id, step_id, reference_order),
    FOREIGN KEY (plan_id, step_id) REFERENCES steps(plan_id, id) ON DELETE CASCADE
);

-- Index for faster reference lookup
CREATE INDEX IF NOT EXISTS idx_step_references_plan_step ON step_references(plan_id, step_id);

-- Trigger to update parent plan's and step's updated_at timestamp when references change
CREATE TRIGGER IF NOT EXISTS step_references_updated_at
AFTER INSERT ON step_references
FOR EACH ROW
BEGIN
    -- Update the parent step's updated_at timestamp
    UPDATE steps SET updated_at = CURRENT_TIMESTAMP 
    WHERE plan_id = NEW.plan_id AND id = NEW.step_id;
    
    -- Update the parent plan's updated_at timestamp
    UPDATE plans SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.plan_id;
END;


