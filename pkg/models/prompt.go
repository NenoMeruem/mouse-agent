package models

import "time"

type Prompt struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Engine      string    `json:"engine"`
	Template    string    `json:"template"`
	Variables   []string  `json:"variables"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
