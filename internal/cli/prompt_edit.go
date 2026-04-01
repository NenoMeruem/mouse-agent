package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/sl/prompt-builder-agent/pkg/models"
	"github.com/spf13/cobra"
)

func init() {
	promptCmd.AddCommand(promptEditCmd)
}

var promptEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a prompt in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE:  promptEditCommand,
}

func promptEditCommand(cmd *cobra.Command, args []string) error {
	id := args[0]

	prompt, err := app.GlobalContext.PromptStore.Get(id)
	if err != nil {
		return fmt.Errorf("cannot get prompt: %w", err)
	}

	data, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal prompt: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "prompt-agent-edit-*.json")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("cannot write temp file: %w", err)
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	editorCmd := exec.Command(editor, tmpFile.Name())
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("cannot read temp file: %w", err)
	}

	var updated models.Prompt
	if err := json.Unmarshal(edited, &updated); err != nil {
		return fmt.Errorf("cannot parse edited JSON: %w", err)
	}

	if err := app.GlobalContext.PromptStore.Update(&updated); err != nil {
		return fmt.Errorf("cannot update prompt: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Prompt %q updated successfully.\n", updated.ID)
	return nil
}
