package prompt

import (
	"testing"
)

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name:     "single variable",
			template: "Hello {{name}}",
			expected: []string{"name"},
		},
		{
			name:     "multiple variables",
			template: "{{greeting}} {{name}}, you are {{age}} years old",
			expected: []string{"greeting", "name", "age"},
		},
		{
			name:     "duplicate variables",
			template: "{{var}} and {{var}}",
			expected: []string{"var"},
		},
		{
			name:     "no variables",
			template: "just plain text",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractVariables(tt.template)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSimpleBuilder(t *testing.T) {
	builder := NewSimpleBuilder()

	template := "Hello {{name}}, you are {{age}} years old"
	data := map[string]string{
		"name": "Alice",
		"age":  "30",
	}

	result, err := builder.Build(template, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello Alice, you are 30 years old"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSimpleBuilderPartialData(t *testing.T) {
	builder := NewSimpleBuilder()

	template := "Hello {{name}}, you are {{age}} years old"
	data := map[string]string{
		"name": "Bob",
	}

	result, err := builder.Build(template, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should leave unmatched variables as-is
	expected := "Hello Bob, you are {{age}} years old"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
