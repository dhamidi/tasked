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
tasked plan add-step [--after step-id] <plan-name> <step-id> <description> <acceptance-criteria> ...
```

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
