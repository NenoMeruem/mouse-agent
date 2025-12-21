package storage

import "github.com/sl/prompt-builder-agent/pkg/models"

type PromptStore interface {
	Create(prompt *models.Prompt) error
	List() ([]models.Prompt, error)
	Get(id string) (*models.Prompt, error)
	Delete(id string) error
}
