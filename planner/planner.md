# Planner Module Documentation

The Planner module provides functionality for creating, managing, and persisting plans. Plans consist of a series of steps, each with a description, status, and acceptance criteria.

## Overview

The module is designed around a `Planner` type, which acts as the main interface for interacting with plans. Plans are stored in a SQLite database, specified by a database file path. Each plan has a unique ID and contains a list of steps.

## Core Concepts

### Planner

The `Planner` struct is the entry point for all plan management operations. It manages the connection to the SQLite database where plan data is stored.

- `New(databasePath string) (*Planner, error)`: Creates a new `Planner` instance, connecting to or creating a SQLite database at the given `databasePath`. It initializes the database schema (defined in `schema.sql`) if it's not already present.

### Plan

The `Plan` struct represents a collection of steps.

- `ID`: A unique identifier for the plan (e.g., "active"). This ID corresponds to the primary key in the `plans` database table.
- `Steps`: A slice of `*Step` pointers, representing the ordered tasks within the plan.

#### Plan Methods

- `Create(name string) (*Plan, error)`: (Associated with `Planner`) Creates a new **in-memory** `Plan` object with the given name (which will serve as its ID upon saving). This method **does not** interact with the database; the plan is only persisted when `Save` is called.
- `Get(name string) (*Plan, error)`: (Associated with `Planner`) Retrieves a plan and its associated steps and acceptance criteria by its name (ID) from the database.
- `Save(plan *Plan) error`: (Associated with `Planner`) Persists the state of the given `Plan` object (including its steps and acceptance criteria) to the database. If the plan's internal `isNew` flag is true (set by `Create`), it will first attempt to insert the plan record into the `plans` table. If `isNew` is false (e.g., for a plan retrieved via `Get` or already saved), or if the plan record already exists, this method synchronizes the plan's steps and acceptance criteria. This involves inserting new steps/criteria, updating existing ones, and deleting any that are no longer present in the in-memory `Plan` object. After a new plan is successfully inserted, its `isNew` flag is set to false in memory.
- `Remove(planNames []string) map[string]error`: (Associated with `Planner`) Attempts to delete plans (and their associated steps/criteria due to cascading deletes) by their names (IDs) from the database. Returns a map of plan names to errors (nil on success).
- `List() ([]PlanInfo, error)`: (Associated with `Planner`) Returns summary information (name, status, task counts) for all plans stored in the database.
- `Compact() error`: (Associated with `Planner`) Removes all completed plans (where all steps are "DONE" or the plan has no steps) from the database.

- `Inspect() string`: (Method of `Plan`) Returns a string representation of the plan, formatted for display, showing each step's number, status, ID, description, and acceptance criteria.
- `NextStep() *Step`: (Method of `Plan`) Returns the first step in the plan that is not marked as "DONE". Returns `nil` if all steps are completed.
- `MarkAsCompleted(stepID string) error`: (Method of `Plan`) Finds a step by its ID within the plan's `Steps` slice and sets its status to "DONE" **in-memory**. Returns an error if the step is not found. Changes are persisted to the database when `Planner.Save(plan)` is called.
- `MarkAsIncomplete(stepID string) error`: (Method of `Plan`) Finds a step by its ID within the plan's `Steps` slice and sets its status to "TODO" **in-memory**. Returns an error if the step is not found. Changes are persisted to the database when `Planner.Save(plan)` is called.
- `AddStep(id, description string, acceptanceCriteria []string)`: (Method of `Plan`) Appends a new step to the plan. The new step is initialized with status "TODO".
- `RemoveSteps(stepIDs []string) int`: (Method of `Plan`) Removes steps from the plan based on a slice of step IDs. Returns the count of removed steps.
- `Reorder(newStepOrder []string)`: (Method of `Plan`) Rearranges the steps in the plan according to the `newStepOrder`. Steps in `newStepOrder` come first, followed by remaining steps in their original relative order.
- `IsCompleted() bool`: (Method of `Plan`) Checks if all steps in the plan are marked as "DONE".

### Step

The `Step` struct represents a single task within a plan.

- `id`: A short identifier for the step (e.g., "add-tests").
- `description`: A textual description of the step.
- `status`: The current status of the step, either "DONE" or "TODO".
- `acceptance`: A slice of strings representing the acceptance criteria for the step.

#### Step Methods

- `ID() string`: Returns the step's ID.
- `Status() string`: Returns the step's status (always uppercase).
- `Description() string`: Returns the step's description.
- `AcceptanceCriteria() []string`: Returns the step's acceptance criteria.

### PlanInfo

The `PlanInfo` struct holds summary information about a plan, used by the `Planner.List()` method.

- `Name`: The name of the plan.
- `Status`: Overall status of the plan ("DONE" or "TODO").
- `TotalTasks`: The total number of steps in the plan.
- `CompletedTasks`: The number of completed steps in the plan.

## Internal Storage

Plans are stored in a SQLite database. The database schema defines how plans, steps, and their acceptance criteria are organized.

-   **Database File**: The planner uses a single SQLite database file, the path to which is provided when a `Planner` is instantiated.
-   **Schema**: The database schema consists of three main tables:
    -   `plans`: Stores high-level information about each plan, primarily its unique `id`.
    -   `steps`: Stores details for each step within a plan, including its `id`, `plan_id` (linking to the `plans` table), `description`, `status`, and `step_order`.
    -   `step_acceptance_criteria`: Stores each acceptance criterion for a step, linking to the `steps` table via `plan_id` and `step_id`, and includes the `criterion` text and its `criterion_order`.
-   **Relationships**: Foreign key constraints are used to maintain integrity between these tables (e.g., deleting a plan cascades to delete its steps and their criteria).
-   **Schema Definition**: The complete schema is defined in `schema.sql` within the planner module directory. This file is used to initialize the database tables if they do not already exist.

Data is read from and written to these tables using SQL queries executed by the planner methods.
