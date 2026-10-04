package migrations_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestMigrationVersionsAreUnique(t *testing.T) {
	const migrationsDir = "../../../internal/project/infrastructure/postgres/migrations"

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations directory: %v", err)
	}

	migrationName := regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)
	pairs := make(map[string]map[string]string)
	var problems []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		matches := migrationName.FindStringSubmatch(entry.Name())
		if matches == nil {
			problems = append(problems, "invalid migration filename: "+entry.Name())
			continue
		}
		version, direction := matches[1], matches[3]
		if pairs[version] == nil {
			pairs[version] = make(map[string]string)
		}
		if previous, exists := pairs[version][direction]; exists {
			problems = append(problems, "duplicate "+direction+" migration for version "+version+": "+previous+" and "+entry.Name())
		}
		pairs[version][direction] = entry.Name()
	}

	for _, version := range []string{"000001", "000002", "000003", "000004", "000005", "000006", "000007", "000008", "000009", "000010"} {
		if pairs[version] == nil {
			problems = append(problems, "missing canonical migration version "+version)
		}
	}
	if len(pairs) != 10 {
		problems = append(problems, "canonical sequence must contain exactly ten migration pairs")
	}

	for version, files := range pairs {
		if files["up"] == "" || files["down"] == "" {
			problems = append(problems, "version "+version+" must have exactly one matching .up.sql and .down.sql migration")
			continue
		}
		upBase := strings.TrimSuffix(filepath.Base(files["up"]), ".up.sql")
		downBase := strings.TrimSuffix(filepath.Base(files["down"]), ".down.sql")
		if upBase != downBase {
			problems = append(problems, "version "+version+" has mismatched migration pair: "+files["up"]+" and "+files["down"])
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		t.Fatalf("migration files must form unique versioned up/down pairs:\n- %s", strings.Join(problems, "\n- "))
	}
}
