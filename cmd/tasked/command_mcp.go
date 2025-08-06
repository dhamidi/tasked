package main

import (
	"fmt"
	"log"

	"github.com/dhamidi/tasked"
	"github.com/dhamidi/tasked/planner"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start an MCP server providing planner tools",
	Long: `Start a Model Context Protocol (MCP) server that provides access to the planner
functionality. The server runs on standard input/output and can be used by MCP clients
to interact with the task planner.`,
	RunE: runMCPServer,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	// Get the database file path from settings
	dbPath := tasked.GlobalSettings.GetDatabaseFile()
	
	// Initialize the planner tool
	toolInfo, err := planner.MakePlannerToolHandler(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize planner tool: %w", err)
	}

	// Create a new MCP server
	srv := server.NewMCPServer(
		"tasked-planner",
		"1.0.0",
		server.WithLogging(),
	)

	// Register the planner tool
	srv.AddTool(toolInfo.Tool, toolInfo.Handler)

	// Start the server on stdio
	log.Printf("Starting MCP server with database: %s", dbPath)
	if err := server.ServeStdio(srv); err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	return nil
}
