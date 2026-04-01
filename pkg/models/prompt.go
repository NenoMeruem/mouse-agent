package models

import "time"

type Prompt struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Engine      string    `json:"engine"`
	Template    string    `json:"template"`
	Variables   []string  `json:"variables"`
	Params      []string  `json:"params,omitempty"`  // e.g. ["tone","length","complexity"]
	Icon        string    `json:"icon,omitempty"`    // emoji or icon name
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RunRecord struct {
	ID          string    `json:"id"`
	PromptID    string    `json:"prompt_id"`
	Engine      string    `json:"engine"`
	InputText   string    `json:"input_text"`
	FinalPrompt string    `json:"final_prompt"`
	Response    string    `json:"response"`
	DurationMs  int64     `json:"duration_ms"`
	Error       string    `json:"error"`
	CreatedAt   time.Time `json:"created_at"`
}
