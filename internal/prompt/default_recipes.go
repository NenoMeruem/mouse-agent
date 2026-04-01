package prompt

import (
	"time"

	"github.com/sl/prompt-builder-agent/pkg/models"
)

// DefaultRecipes returns the 6 built-in recipe definitions.
// These are seeded into the store on first init.
// CreatedAt is fixed to a past date so they sort consistently at the bottom
// when the user creates new recipes.
func DefaultRecipes() []models.Prompt {
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return []models.Prompt{
		{
			ID:          "explain",
			Name:        "Explain",
			Description: "Explain the selected text.",
			Engine:      "gemini",
			Template:    "Explain the following:\n\n{{selection}}",
			Variables:   []string{"selection"},
			Params:      []string{"tone", "length"},
			Icon:        "💡",
			CreatedAt:   seed,
			UpdatedAt:   seed,
		},
		{
			ID:          "summarize",
			Name:        "Summarize",
			Description: "Summarize the selected text concisely.",
			Engine:      "gemini",
			Template:    "Summarize the following text concisely:\n\n{{selection}}",
			Variables:   []string{"selection"},
			Params:      []string{"length"},
			Icon:        "📝",
			CreatedAt:   seed,
			UpdatedAt:   seed,
		},
		{
			ID:          "rephrase",
			Name:        "Rephrase",
			Description: "Rephrase the selected text.",
			Engine:      "gemini",
			Template:    "Rephrase the following text:\n\n{{selection}}",
			Variables:   []string{"selection"},
			Params:      []string{"tone", "complexity"},
			Icon:        "✏️",
			CreatedAt:   seed,
			UpdatedAt:   seed,
		},
		{
			ID:          "fix-code",
			Name:        "Fix Code",
			Description: "Fix bugs or issues in the selected code.",
			Engine:      "gemini",
			Template:    "Fix any bugs or issues in the following code and explain what you changed:\n\n{{selection}}",
			Variables:   []string{"selection"},
			Params:      []string{},
			Icon:        "🔧",
			CreatedAt:   seed,
			UpdatedAt:   seed,
		},
		{
			ID:          "translate-vi",
			Name:        "Translate → VI",
			Description: "Translate the selected text to Vietnamese.",
			Engine:      "gemini",
			Template:    "Translate the following text to Vietnamese:\n\n{{selection}}",
			Variables:   []string{"selection"},
			Params:      []string{},
			Icon:        "🌐",
			CreatedAt:   seed,
			UpdatedAt:   seed,
		},
		{
			ID:          "review-pr",
			Name:        "Review PR",
			Description: "Review the selected code diff as a pull request.",
			Engine:      "gemini",
			Template:    "Review the following code diff as if it were a pull request. Point out bugs, style issues, and improvements:\n\n{{selection}}",
			Variables:   []string{"selection"},
			Params:      []string{"complexity"},
			Icon:        "👀",
			CreatedAt:   seed,
			UpdatedAt:   seed,
		},
	}
}
