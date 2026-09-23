package generator

import (
	"os"
	"testing"

	"github.com/tobibamidele/goforge/internal/config"
)

func TestAllCombinations(t *testing.T) {
	frameworks := []string{"gin", "fiber", "echo", "nethttp"}
	dbs := map[string][]string{
		"postgres": {"bun", "gorm", "sqlx", "database-sql", "lathe"},
		"mysql":    {"bun", "gorm", "sqlx", "database-sql", "lathe"},
		"sqlite":   {"bun", "gorm", "sqlx", "database-sql", "lathe"},
		"mongodb":  {"mongo-driver"},
		"none":     {"none"},
	}
	authTypes := []string{"jwt", "session"}

	i := 0
	for _, fw := range frameworks {
		for db, orms := range dbs {
			for _, orm := range orms {
				for _, redis := range []bool{true, false} {
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
								i++
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
								_ = os.RemoveAll(out)
								if _, err := Generate(p, out); err != nil {
									t.Fatalf("combo %d (%s/%s/%s/redis=%v/auth=%v/%s/oauth=%v): %v",
										i, fw, db, orm, redis, useAuth, at, oauth, err)
								}
							}
						}
					}
				}
			}
		}
	}
	t.Logf("generated %d combinations OK", i)
}
