# Tasked

Tasked is a simple task manager intended for use by AI Agents.

It's main purpose is to iterate quickly on plans: marking items as completed, reordering steps etc.

## Usage

```
# starts an MCP server storing plans in the given database file 
tasked mcp --database-file plans.db

# all plan functions are exposed under the plan subcommand
tasked plan new <plan-name>
tasked plan remove <plan-name> ...
tasked plan list 
tasked plan inspect <plan-name>
tasked plan next-step <plan-name>
tasked plan mark-as-completed <plan-name> <step-id>
tasked plan inspect <plan-name>
tasked plan is-completed <plan-name>
tasked plan mark-as-incomplete <plan-name> <step-id>
tasked plan remove-steps <plan-name> <step-id> ...
tasked plan reorder-steps <plan-name> <step-id> ...
tasked plan add-step [--after step-id] [--references ref1,ref2] <plan-name> <step-id> <description> <acceptance-criteria> ...

# the test subcommand performs a self-test in the current environment
tasked test <test-name>
```

## References Feature

Steps can include references to relevant resources using the `--references` flag when adding steps. References help point to information needed for implementing the step.

### Using References

```bash
# Add a step with references to documentation and files
tasked plan add-step my-plan setup-auth "Configure authentication" "Auth is working" \
  --references "https://example.com/auth-docs,/path/to/config.yaml"

# Add multiple references (comma-separated)
tasked plan add-step my-plan deploy "Deploy application" "App is deployed" \
  --references "deploy-script.sh,https://deploy-docs.com,/config/prod.env"
```

### Reference Format

- **URLs**: Web links to documentation, APIs, or specifications
- **File paths**: Absolute paths to local files (recommended for code files)
- **Format**: Comma-separated list, 1-5 references per step
- **Purpose**: Point to information needed for step implementation

### Best Practices

- Use absolute file paths for local files: `/home/user/project/config.yml`
- Include relevant documentation URLs for external resources
- Keep references focused and directly related to the step
- Limit to 5 references per step for clarity

## Project structure

```
tasked/
  settings.go             # configuration settings to be made available in all commands
  command_plan_new.go     # the plan new subcommand
  command_plan_list.go    # the plan list subcommand
  ...
  command_mcp.go          # the mcp subcommand
  planner/ # the planner module
  cmd/ 
    tasked/
      main.go  # the main executable entrypoint - sets up the cobra root command
```

## Working with the planner

The planner module pushes IO to the edges:

1. Load a plan from storage using (*Planner).Get
2. Modify the plan to your heart's content
3. Save it back to storage using (*Planner).Save

## Testing

Testing is implemented through the `test` subcommand, allowing you to make sure that tasked works in your environment.

The test subcommand creates an MCP client using github.com/mark3labs/go-mcp and then invokes tools through the MCP client according to the test scenario.

The default test scenario creates and modifies a plan using all available operations on a plan, and then checks whether the plan is in the expected state.

Each tool call is written to stdout as it is performed.

Only failing assertions are reported on stdout.

If any one assertion fails, the test subcommand exits immediately with exit status 1.
