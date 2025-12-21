package prompt

import (
	"fmt"
)

// ValidateVariables checks that all required variables are present in data
func ValidateVariables(required []string, data map[string]string) error {
	missing := make([]string, 0)

	for _, varName := range required {
		if _, ok := data[varName]; !ok {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		if len(missing) == 1 {
			return fmt.Errorf("missing variable: %s", missing[0])
		}
		return fmt.Errorf("missing variables: %v", missing)
	}

	return nil
}
