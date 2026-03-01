package cli

import (
	"fmt"
	"os"

	"github.com/sl/prompt-builder-agent/internal/config"
	"github.com/sl/prompt-builder-agent/internal/llm"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(verifyConfigCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var verifyConfigCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify configuration and available LLM engines",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 Verifying Prompt Agent Configuration...\n")

		// Check config file location
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot get home directory: %w", err)
		}
		configPath := home + "/.prompt-agent/config.yaml"
		fmt.Printf("📁 Config file: %s\n", configPath)
		if _, err := os.Stat(configPath); err == nil {
			fmt.Println("   ✅ Config file exists")
		} else {
			fmt.Println("   ⚠️  Config file not found (using defaults)")
		}

		fmt.Println()

		// Check API keys from config file
		if config.AppConfig != nil {
			fmt.Println("📋 Configured Engines:")
			for engineName, engineConfig := range config.AppConfig.Engines {
				if engineConfig.APIKey != "" {
					fmt.Printf("   ✅ %s: API key configured (model: %s)\n", engineName, engineConfig.Model)
				} else {
					fmt.Printf("   ⚠️  %s: No API key in config\n", engineName)
				}
			}
			fmt.Println()
		}

		// Check environment variables
		fmt.Println("🔐 Environment Variables:")
		if os.Getenv("OPENAI_API_KEY") != "" {
			fmt.Println("   ✅ OPENAI_API_KEY is set")
		} else {
			fmt.Println("   ⚠️  OPENAI_API_KEY not set")
		}

		if os.Getenv("GEMINI_API_KEY") != "" {
			fmt.Println("   ✅ GEMINI_API_KEY is set")
		} else {
			fmt.Println("   ⚠️  GEMINI_API_KEY not set")
		}

		fmt.Println()

		// Test which engines are actually available
		fmt.Println("✨ Available Engines:")
		llmMgr := llm.NewManager()
		registerAvailableEngines(llmMgr)

		registered := llmMgr.List()
		if len(registered) == 0 {
			fmt.Println("   ❌ No engines available!")
			fmt.Println("\n💡 Tips to fix:")
			fmt.Println("   1. Add API keys to ~/.prompt-agent/config.yaml:")
			fmt.Println("      engines:")
			fmt.Println("        openai:")
			fmt.Println("          api_key: \"your-key-here\"")
			fmt.Println("          model: \"gpt-4-mini\"")
			fmt.Println("        gemini:")
			fmt.Println("          api_key: \"your-key-here\"")
			fmt.Println("          model: \"gemini-2.5-flash-lite\"")
			fmt.Println("   2. OR set environment variables:")
			fmt.Println("      export OPENAI_API_KEY=\"your-key\"")
			fmt.Println("      export GEMINI_API_KEY=\"your-key\"")
			return nil
		}

		for _, name := range registered {
			fmt.Printf("   ✅ %s\n", name)
		}

		fmt.Println("\n✅ Configuration verification complete!")
		return nil
	},
}
