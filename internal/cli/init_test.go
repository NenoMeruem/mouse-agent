package cli

import (
	os "os"
	pathfile "path/filepath"
	"testing"
)

// TestInitCommandCreatesFiles ensures the init command bootstraps both
// configuration and prompts files in the user's home directory.  We override
// $HOME with a temporary directory so the test is hermetic.
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

	promptsPath := pathfile.Join(dir, "prompts.json")
	if _, err := os.Stat(promptsPath); err != nil {
		t.Fatalf("prompts.json not created: %v", err)
	}

	data, err := os.ReadFile(promptsPath)
	if err != nil {
		t.Fatalf("unable to read prompts.json: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("prompts.json should contain sample content")
	}
}
