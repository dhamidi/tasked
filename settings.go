package tasked

import (
	"os"
	"path/filepath"
)

type Settings struct {
	DatabaseFile string
}

var GlobalSettings = &Settings{}

func (s *Settings) GetDatabaseFile() string {
	if s.DatabaseFile != "" {
		return s.DatabaseFile
	}

	// Default to ~/.tasked/tasks.db
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "tasks.db"
	}

	taskedDir := filepath.Join(homeDir, ".tasked")
	if err := os.MkdirAll(taskedDir, 0755); err != nil {
		return "tasks.db"
	}

	return filepath.Join(taskedDir, "tasks.db")
}
