package prompt

import "testing"

func TestDefaultRecipes(t *testing.T) {
	recipes := DefaultRecipes()
	if len(recipes) == 0 {
		t.Fatalf("expected at least 1 recipe, got 0")
	}
	ids := map[string]bool{}
	for _, r := range recipes {
		// All required fields set
		if r.ID == "" {
			t.Errorf("recipe has empty ID")
		}
		if r.Name == "" {
			t.Errorf("recipe %s has empty Name", r.ID)
		}
		if r.Engine == "" {
			t.Errorf("recipe %s has empty Engine", r.ID)
		}
		if r.Template == "" {
			t.Errorf("recipe %s has empty Template", r.ID)
		}
		// No duplicate IDs
		if ids[r.ID] {
			t.Errorf("duplicate recipe ID: %s", r.ID)
		}
		ids[r.ID] = true
		// selection variable present
		hasSelection := false
		for _, v := range r.Variables {
			if v == "selection" {
				hasSelection = true
			}
		}
		if !hasSelection {
			t.Errorf("recipe %s missing selection variable", r.ID)
		}
	}
}
