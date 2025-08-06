package main

import (
	"fmt"
	"os"

	"github.com/dhamidi/tasked"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tasked",
	Short: "A simple task management tool",
	Long: `Tasked is a command-line task management tool that helps you organize
and track your tasks efficiently. Store tasks in a local SQLite database
and manage them through simple CLI commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&tasked.GlobalSettings.DatabaseFile, "database-file", "", "Path to the SQLite database file (default: ~/.tasked/tasks.db)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
