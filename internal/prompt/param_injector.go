package prompt

import "strings"

var paramSuffixes = map[string]map[string]string{
	"tone": {
		"professional": "Reply in a professional tone.",
		"casual":       "Reply in a casual, friendly tone.",
		"concise":      "Be very concise and direct.",
	},
	"length": {
		"short":  "Keep the response under 3 sentences.",
		"medium": "Aim for 1-2 paragraphs.",
		"long":   "Provide a detailed, comprehensive response.",
	},
	"complexity": {
		"simple":    "Explain like I'm 5 years old. Use simple words.",
		"normal":    "",
		"technical": "Use technical terms. Assume expert-level knowledge.",
	},
}

// InjectParams appends tone/length/complexity instruction suffixes to the prompt.
// Only params present in paramSuffixes with non-empty values are appended.
// Params are iterated in deterministic order (tone → length → complexity).
func InjectParams(prompt string, paramValues map[string]string) string {
	var suffixes []string
	for _, param := range []string{"tone", "length", "complexity"} {
		value, ok := paramValues[param]
		if !ok || value == "" {
			continue
		}
		if m, ok := paramSuffixes[param]; ok {
			if s, ok := m[value]; ok && s != "" {
				suffixes = append(suffixes, s)
			}
		}
	}
	if len(suffixes) == 0 {
		return prompt
	}
	return prompt + "\n\n" + strings.Join(suffixes, " ")
}
