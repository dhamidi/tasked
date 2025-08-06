# Tasked

A simple task manager designed specifically for AI Agents. Tasked helps you organize and iterate quickly on plans by marking items as completed, reordering steps, and tracking progress.

## What Tasked Does

Tasked provides a structured way to manage plans and their execution steps. It's built with AI Agents in mind, offering:

- **Plan Management**: Create, modify, and track multi-step plans
- **Step Operations**: Mark steps as complete/incomplete, reorder, add, or remove steps
- **MCP Integration**: Expose plan management as Model Context Protocol (MCP) tools
- **Progress Tracking**: Check plan completion status and get next actionable steps
- **Local Storage**: All data stored locally in a SQLite database

## Installation

### Prerequisites
- Go 1.24.3 or later

### Build from Source
```bash
git clone https://github.com/dhamidi/tasked
cd tasked
go build -o tasked ./cmd/tasked
```

### Install Binary
```bash
go install github.com/dhamidi/tasked/cmd/tasked@latest
```

## Setting Up as an MCP Server

Tasked can run as an MCP server, making plan management tools available to AI Agents and MCP clients.

### Basic MCP Server Setup
```bash
# Start MCP server with default database location (~/.tasked/tasks.db)
tasked mcp

# Start MCP server with custom database file
tasked mcp --database-file /path/to/plans.db
```

### Example MCP Client Configuration
For Claude Desktop or other MCP clients, add this to your configuration:

```json
{
  "mcpServers": {
    "tasked": {
      "command": "tasked",
      "args": ["mcp", "--database-file", "plans.db"]
    }
  }
}
```

## How Plans and Storage Work

Tasked uses a simple but powerful workflow for managing plans:

### Architecture Overview

The system follows a "push IO to the edges" design pattern:

1. **Load** a plan from SQLite storage using `(*Planner).Get`
2. **Modify** the plan in memory with any operations you need
3. **Save** it back to storage using `(*Planner).Save`

### Available Plan Operations

- **Plan Management**: `new`, `remove`, `list`, `inspect`
- **Step Management**: `add-step`, `remove-steps`, `reorder-steps`
- **Progress Tracking**: `mark-as-completed`, `mark-as-incomplete`, `next-step`, `is-completed`

### Storage Details

- Plans are stored in a local SQLite database
- Default location: `~/.tasked/tasks.db`
- Custom location via `--database-file` flag
- Each plan contains multiple steps with IDs, descriptions, and acceptance criteria
- Steps can be marked as completed or incomplete
- Step order can be customized and reordered as needed

## Command Line Usage

### Plan Commands
```bash
# Create a new plan
tasked plan new "my-project"

# List all plans
tasked plan list

# Add a step to a plan
tasked plan add-step "my-project" "step-1" "Setup environment" "Environment is configured"

# Mark a step as completed
tasked plan mark-as-completed "my-project" "step-1"

# Get the next actionable step
tasked plan next-step "my-project"

# Check if plan is complete
tasked plan is-completed "my-project"

# Inspect plan details
tasked plan inspect "my-project"
```

### Testing

Tasked includes a self-test feature to verify it works in your environment:

```bash
# Run the default test scenario
tasked test default
```

The test creates an MCP client, connects to a tasked MCP server, and runs through all plan operations to ensure everything works correctly.

## Development

### Project Structure
```
tasked/
├── settings.go              # Global configuration settings
├── command_plan_*.go        # Plan subcommand implementations
├── command_mcp.go          # MCP server implementation
├── planner/                # Core planner module
└── cmd/tasked/main.go      # Main executable entry point
```

### Building
```bash
go build -o tasked ./cmd/tasked
```

### Running Tests
```bash
go test ./...
```
