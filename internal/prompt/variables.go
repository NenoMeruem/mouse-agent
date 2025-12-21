package prompt

import (
	"regexp"
)

// ExtractVariables finds all {{variable}} placeholders in the template
func ExtractVariables(tmpl string) []string {
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(tmpl, -1)

	varMap := make(map[string]bool)
	var vars []string

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !varMap[varName] {
				varMap[varName] = true
				vars = append(vars, varName)
			}
		}
	}

	return vars
}
