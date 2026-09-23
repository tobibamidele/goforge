package generator

import (
	"io"
	"strings"
	"testing"

	"github.com/tobibamidele/goforge/internal/config"
)

func TestGoModRequiresLathe(t *testing.T) {
	want := map[string][]string{
		"postgres_lathe": {"github.com/tobibamidele/lathe v0.1.0", "github.com/lib/pq v1.10.9"},
		"mysql_lathe":    {"github.com/tobibamidele/lathe v0.1.0", "github.com/go-sql-driver/mysql v1.8.1"},
		"sqlite_lathe":   {"github.com/tobibamidele/lathe v0.1.0", "modernc.org/sqlite v1.38.0"},
	}
	for key, wantMods := range want {
		db, orm, ok := strings.Cut(key, "_")
		if !ok {
			t.Fatalf("bad test key %q", key)
		}
		p := config.Project{Database: db, ORM: orm}
		got := goModRequires(p)
		for _, m := range wantMods {
			if !contains(got, m) {
				t.Errorf("%s: goModRequires missing %q (got %v)", key, m, got)
			}
		}
	}
}

func TestRunLatheNoopForOtherORMs(t *testing.T) {
	// RunLathe must not try to touch the network or the CLI for the other
	// options — it should return nil immediately.
	for db, orms := range map[string][]string{
		"postgres": {"bun", "gorm"},
		"mongodb":  {"mongo-driver"},
		"none":     {"none"},
	} {
		for _, orm := range orms {
			p := config.Project{Database: db, ORM: orm}
			if err := RunLathe(t.TempDir(), p, io.Discard); err != nil {
				t.Errorf("%s/%s: RunLathe should be a no-op, got %v", db, orm, err)
			}
		}
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
