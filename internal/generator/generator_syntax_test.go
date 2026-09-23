package generator

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tobibamidele/goforge/internal/config"
)

// TestGeneratedGoSyntax is stricter than TestAllCombinations: it fails
// loudly if any generated .go file doesn't parse, instead of silently
// falling back to unformatted output like Generate does at runtime.
func TestGeneratedGoSyntax(t *testing.T) {
	frameworks := []string{"gin", "fiber", "echo", "nethttp"}
	dbs := map[string][]string{
		"postgres": {"bun", "gorm", "sqlx", "database-sql", "lathe"},
		"mysql":    {"bun", "lathe"},
		"sqlite":   {"sqlx", "lathe"},
		"mongodb":  {"mongo-driver"},
		"none":     {"none"},
	}
	authTypes := []string{"jwt", "session"}

	bad := 0
	for _, fw := range frameworks {
		for db, orms := range dbs {
			for _, orm := range orms {
				for _, useAuth := range []bool{true, false} {
					authOpts := []string{""}
					if useAuth {
						authOpts = authTypes
					}
					for _, at := range authOpts {
						for _, oauth := range []bool{true, false} {
							if !useAuth && oauth {
								continue
							}
							for _, redis := range []bool{true, false} {
								p := config.Project{
									Name:           "testapp",
									Module:         "github.com/tobi/testapp",
									GoVersion:      "1.23",
									Framework:      fw,
									Database:       db,
									ORM:            orm,
									UseRedis:       redis,
									UseDocker:      true,
									UseAuth:        useAuth,
									AuthType:       at,
									UseOAuth:       oauth,
									OAuthProviders: []string{"google", "github"},
									Logger:         "slog",
									EnvLoader:      "godotenv",
									UseMakefile:    true,
									UseCI:          true,
								}
								out := t.TempDir() + "/proj"
								files, err := Generate(p, out)
								if err != nil {
									t.Fatalf("generate failed for %+v: %v", p, err)
								}
								for _, f := range files {
									if !strings.HasSuffix(f, ".go") {
										continue
									}
									src, rerr := os.ReadFile(f)
									if rerr != nil {
										t.Fatal(rerr)
									}
									if _, ferr := format.Source(src); ferr != nil {
										bad++
										t.Errorf("SYNTAX ERROR in %s (framework=%s db=%s orm=%s auth=%v type=%s oauth=%v redis=%v): %v\n---\n%s\n---",
											filepath.Base(f), fw, db, orm, useAuth, at, oauth, redis, ferr, src)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	if bad > 0 {
		t.Fatalf("%d files with syntax errors", bad)
	}
}
