package run

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestEveryGoldenHasAScenario(t *testing.T) {
	t.Helper()
	if *updateGoldens {
		t.Skip("golden orphan check skipped during -update")
	}
	goldenDir := filepath.Join("..", "..", "testdata", "golden")
	names, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("reading golden dir: %v", err)
	}
	var basenames []string
	for _, n := range names {
		name := n.Name()
		base := strings.TrimSuffix(name, ".text.golden")
		base = strings.TrimSuffix(base, ".json.golden")
		basenames = append(basenames, base)
	}
	sort.Strings(basenames)
	scenarioNames := scenarioNames()
	orphans := []string{}
	for _, base := range basenames {
		found := false
		for _, s := range scenarioNames {
			if base == s {
				found = true
				break
			}
		}
		if !found && base != "sample" {
			orphans = append(orphans, base)
		}
	}
	if len(orphans) > 0 {
		t.Fatalf("golden basenames with no matching scenario: %s", strings.Join(orphans, ", "))
	}
}

func scenarioNames() []string {
	seen := map[string]bool{}
	for _, sc := range scenarios() {
		seen[sc.name] = true
	}
	var names []string
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
