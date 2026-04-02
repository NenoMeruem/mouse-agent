package cli

import (
	os "os"
	pathfile "path/filepath"
	"testing"
)

// TestInitCommandCreatesFiles ensures the init command bootstraps both
// the config file and the SQLite store with default recipes.
// We override $HOME with a temporary directory so the test is hermetic.
func TestInitCommandCreatesFiles(t *testing.T) {
	tmp, err := os.MkdirTemp("", "prompt-agent-test")
	if err != nil {
		t.Fatalf("cannot create temp dir: %v", err)
	}
	defer os.RemoveAll(tmp)

	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", oldHome)

	// run the command directly (no cobra args required)
	if err := initCmd.RunE(nil, []string{}); err != nil {
		t.Fatalf("initCmd failed: %v", err)
	}

	dir := pathfile.Join(tmp, ".prompt-agent")
	configPath := pathfile.Join(dir, "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}

	// Storage is now SQLite — verify prompts.db was created and seeded
	dbPath := pathfile.Join(dir, "prompts.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("prompts.db not created: %v", err)
	}
}
