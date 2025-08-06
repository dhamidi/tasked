# MCP Tool Usage

The `manage_plan` MCP tool provides comprehensive plan management functionality through a single unified interface.

## Tool Overview

**Tool Name**: `manage_plan`

**Description**: Manage plans and their steps with various operations. Steps can include references to relevant files, URLs, or documentation.

## Parameters

### Required Parameters
- `plan_name` (string): Name of the plan to operate on
- `action` (string): Action to perform (see Available Actions below)

### Conditional Parameters
- `step_id` (string): ID of the step (required for set_status, single step operations)
- `description` (string): Description of the step (required for add_steps when adding single step)
- `acceptance_criteria` (array): Acceptance criteria for the step (for add_steps)
- `references` (array): References for the step (for add_steps) - URLs, file paths, or other resource identifiers (1-5 items)
- `step_ids` (array): IDs of steps (required for remove_steps)
- `step_order` (array): New order of step IDs (required for reorder_steps)
- `plan_names` (array): Names of plans to remove (required for remove_plans)
- `status` (string): Status to set for step - "completed" or "incomplete" (required for set_status)

## Available Actions

1. **add_steps**: Add a new step to a plan (creates plan if it doesn't exist)
2. **inspect**: Get detailed information about a plan and its steps
3. **list_plans**: List all available plans
4. **remove_plans**: Remove one or more plans
5. **compact_plans**: Remove all completed plans from storage
6. **remove_steps**: Remove specific steps from a plan
7. **reorder_steps**: Change the order of steps in a plan
8. **set_status**: Mark a step as completed or incomplete
9. **get_next_step**: Get the next incomplete step in a plan
10. **is_completed**: Check if all steps in a plan are completed

## Examples

### Adding Steps with References

```json
{
  "plan_name": "web-app-development",
  "action": "add_steps",
  "step_id": "setup-database",
  "description": "Configure PostgreSQL database",
  "acceptance_criteria": ["Database is accessible", "Migrations run successfully"],
  "references": [
    "/config/database.yml",
    "https://postgresql.org/docs/current/tutorial.html",
    "/scripts/setup-db.sh"
  ]
}
```

### Adding Steps with File and URL References

```json
{
  "plan_name": "api-integration",
  "action": "add_steps", 
  "step_id": "implement-auth",
  "description": "Implement OAuth 2.0 authentication",
  "acceptance_criteria": ["Token validation works", "Refresh tokens implemented"],
  "references": [
    "https://tools.ietf.org/rfc/rfc6749.txt",
    "/src/auth/oauth.js",
    "https://oauth.net/2/grant-types/"
  ]
}
```

### Getting Next Step (includes references)

```json
{
  "plan_name": "deployment-pipeline",
  "action": "get_next_step"
}
```

Returns:
```json
{
  "id": "setup-ci",
  "description": "Configure continuous integration",
  "status": "incomplete", 
  "acceptance_criteria": ["Tests run on every commit", "Build artifacts are stored"],
  "references": [
    "/ci/github-actions.yml",
    "https://docs.github.com/en/actions"
  ]
}
```

## Reference Guidelines

When using the `references` parameter:

### Supported Reference Types
- **File paths**: Absolute paths to local files (e.g., `/src/components/Auth.js`)
- **URLs**: Web documentation, APIs, specifications (e.g., `https://example.com/docs`)
- **Relative identifiers**: Project-specific references (e.g., `config/prod.env`)

### Best Practices
- **Limit**: Use 1-5 references per step for clarity
- **Relevance**: Only include references directly needed for step implementation
- **Absolute paths**: Use absolute file paths when referencing local files
- **Documentation**: Include links to relevant API docs, tutorials, or specifications
- **Consistency**: Use consistent reference formats within your project

### Common Use Cases
- Point to configuration files that need to be modified
- Link to API documentation for integrations
- Reference existing code files that need updating
- Include deployment scripts or infrastructure configs
- Point to design specifications or requirements documents

## Response Format

All tool responses return JSON formatted results. When inspecting plans or getting next steps, the response includes the references array for each step, making it easy for AI agents to access the relevant resources.
