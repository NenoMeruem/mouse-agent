package prompt

import (
	"testing"
)

func TestInjectParamsEmpty(t *testing.T) {
	prompt := "Explain this code."
	result := InjectParams(prompt, map[string]string{})
	if result != prompt {
		t.Errorf("expected prompt unchanged, got %q", result)
	}
}

func TestInjectParamsTone(t *testing.T) {
	prompt := "Explain this code."
	result := InjectParams(prompt, map[string]string{"tone": "professional"})
	expected := "Explain this code.\n\nReply in a professional tone."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestInjectParamsMultiple(t *testing.T) {
	prompt := "Explain this code."
	result := InjectParams(prompt, map[string]string{
		"tone":   "casual",
		"length": "short",
	})
	expected := "Explain this code.\n\nReply in a casual, friendly tone. Keep the response under 3 sentences."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestInjectParamsUnknownValue(t *testing.T) {
	prompt := "Explain this code."
	result := InjectParams(prompt, map[string]string{"tone": "unknown"})
	if result != prompt {
		t.Errorf("expected prompt unchanged for unknown value, got %q", result)
	}
}

func TestInjectParamsNormalComplexity(t *testing.T) {
	prompt := "Explain this code."
	result := InjectParams(prompt, map[string]string{"complexity": "normal"})
	if result != prompt {
		t.Errorf("expected prompt unchanged for complexity=normal (empty suffix), got %q", result)
	}
}
