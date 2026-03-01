// Package prompt provides template building and variable validation for AI prompts.
// It handles substituting template variables with actual values and ensures
// all required variables are provided.
package prompt

import (
	"strings"
)

// Builder handles template variable substitution.
type Builder interface {
	// Build takes a template string and data map, replacing {{key}} placeholders
	// with corresponding values from the data map.
	Build(template string, data map[string]string) (string, error)
}

// SimpleBuilder is a basic implementation of Builder using string replacement.
type SimpleBuilder struct{}

// NewSimpleBuilder creates a new SimpleBuilder instance.
func NewSimpleBuilder() *SimpleBuilder {
	return &SimpleBuilder{}
}

// Build replaces template placeholders with provided data values.
// Example: "Hello {{name}}" with {"name": "World"} returns "Hello World".
func (b *SimpleBuilder) Build(
	template string,
	data map[string]string,
) (string, error) {
	result := template

	for k, v := range data {
		placeholder := "{{" + k + "}}"
		result = strings.ReplaceAll(result, placeholder, v)
	}

	return result, nil
}
