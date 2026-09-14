// Package generator renders the embedded template tree into a real
// project on disk, based on a config.Project.
package generator

import (
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/tobibamidele/goforge/internal/config"
)

//go:embed all:templates
var templateFS embed.FS

const templatesRoot = "templates"

// view is what gets passed into every template: the project config
// plus a couple of computed extras that are awkward to express with
// template conditionals alone (like the go.mod require list).
type view struct {
	config.Project
	Requires []string
}

var funcMap = template.FuncMap{
	"upper": strings.ToUpper,
	"title": func(s string) string {
		if s == "" {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	},
}

// Generate renders the whole template tree into outDir, which must
// not already exist (or must be empty).
func Generate(p config.Project, outDir string) ([]string, error) {
	v := view{Project: p, Requires: goModRequires(p)}

	var written []string

	err := fs.WalkDir(templateFS, templatesRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(templatesRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		// db/*.go.tmpl is the one category where every file in the
		// directory is unconditional Go source (there's no per-file
		// {{if}} guard) because the *selection* itself is the
		// condition: only the one file matching the chosen db+orm
		// combo should ever be read at all.
		if strings.HasPrefix(rel, "db/") {
			key := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(rel), ".tmpl"), ".go")
			if key != p.DBTemplateKey() {
				return nil
			}
		}

		destRel, ok := mapDest(rel)
		if !ok {
			return nil
		}
		destPath := filepath.Join(outDir, destRel)

		raw, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(d.Name()).Funcs(funcMap).Parse(string(raw))
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", path, err)
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, v); err != nil {
			return fmt.Errorf("rendering template %s: %w", path, err)
		}

		out := buf.String()
		if strings.TrimSpace(out) == "" {
			// This file didn't apply to the chosen options (e.g. a
			// router template for a framework that wasn't picked).
			return nil
		}

		// Tidy up whitespace left behind by template conditionals,
		// and gofmt actual Go source so generated code looks hand
		// written.
		out = collapseBlankLines(out)
		if strings.HasSuffix(destPath, ".go") {
			if formatted, ferr := format.Source([]byte(out)); ferr == nil {
				out = string(formatted)
			}
			// If formatting fails we still write the raw file so the
			// user can see (and fix) exactly what went wrong, instead
			// of silently dropping a broken template combination.
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destPath, []byte(out), 0o644); err != nil {
			return err
		}
		written = append(written, destPath)
		return nil
	})
	if err != nil {
		return written, err
	}

	return written, nil
}

// mapDest translates a path inside templates/ (still ending in
// .tmpl, using forward slashes) into the project-relative path it
// should be written to. The template tree is organized by concern
// (router/, db/, auth/jwt/, ...) for maintainability, which doesn't
// match the layout a generated project actually wants on disk, so
// this is the one place that reconciles the two.
func mapDest(rel string) (string, bool) {
	trimmed := strings.TrimSuffix(rel, ".tmpl")
	base := path.Base(trimmed)

	switch {
	case rel == "common" || strings.HasPrefix(rel, "common/"):
		return strings.TrimPrefix(trimmed, "common/"), true

	case strings.HasPrefix(rel, "db/"):
		// Only one db/*.go.tmpl file ever reaches here (see the
		// filter in Generate), so it always becomes internal/db/db.go
		// regardless of which db+orm combo produced it.
		return "internal/db/db.go", true

	case strings.HasPrefix(rel, "redis/"):
		return "internal/cache/" + base, true

	case strings.HasPrefix(rel, "router/"):
		// Every router/*.go.tmpl is self-guarded by framework and
		// renders blank except for the chosen one, so this is safe:
		// at most one file ever actually gets written here.
		return "internal/router/router.go", true

	case strings.HasPrefix(rel, "docker/"):
		return strings.TrimPrefix(trimmed, "docker/"), true

	case strings.HasPrefix(rel, "auth/oauth/") && base == "core.go":
		return "internal/auth/oauth.go", true

	case strings.HasPrefix(rel, "auth/jwt/") || strings.HasPrefix(rel, "auth/session/") || strings.HasPrefix(rel, "auth/oauth/"):
		return "internal/auth/" + base, true
	}

	return "", false
}

// collapseBlankLines turns runs of 3+ blank lines (a common side
// effect of chained {{if}}/{{end}} blocks) into a single blank line.
func collapseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blank := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			blank++
			if blank > 1 {
				continue
			}
		} else {
			blank = 0
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

// goModRequires builds the require-block contents for go.mod based on
// what was picked. Kept as plain Go (rather than deep template
// conditionals) so it's easy to bump versions or add a new
// framework/db/etc. in one place.
func goModRequires(p config.Project) []string {
	var r []string
	add := func(mods ...string) { r = append(r, mods...) }

	switch p.Framework {
	case "gin":
		add("github.com/gin-gonic/gin v1.10.0")
	case "fiber":
		add("github.com/gofiber/fiber/v2 v2.52.5")
	case "echo":
		add("github.com/labstack/echo/v4 v4.12.0")
	}

	switch p.DBTemplateKey() {
	case "postgres_bun":
		add(
			"github.com/uptrace/bun v1.2.1",
			"github.com/uptrace/bun/dialect/pgdialect v1.2.1",
			"github.com/uptrace/bun/driver/pgdriver v1.2.1",
		)
	case "mysql_bun":
		add(
			"github.com/uptrace/bun v1.2.1",
			"github.com/uptrace/bun/dialect/mysqldialect v1.2.1",
			"github.com/go-sql-driver/mysql v1.8.1",
		)
	case "sqlite_bun":
		add(
			"github.com/uptrace/bun v1.2.1",
			"github.com/uptrace/bun/dialect/sqlitedialect v1.2.1",
			"github.com/mattn/go-sqlite3 v1.14.22",
		)
	case "postgres_gorm":
		add("gorm.io/gorm v1.25.11", "gorm.io/driver/postgres v1.5.9")
	case "mysql_gorm":
		add("gorm.io/gorm v1.25.11", "gorm.io/driver/mysql v1.5.7")
	case "sqlite_gorm":
		add("gorm.io/gorm v1.25.11", "gorm.io/driver/sqlite v1.5.6")
	case "postgres_sqlx":
		add("github.com/jmoiron/sqlx v1.4.0", "github.com/lib/pq v1.10.9")
	case "mysql_sqlx":
		add("github.com/jmoiron/sqlx v1.4.0", "github.com/go-sql-driver/mysql v1.8.1")
	case "sqlite_sqlx":
		add("github.com/jmoiron/sqlx v1.4.0", "github.com/mattn/go-sqlite3 v1.14.22")
	case "postgres_database-sql":
		add("github.com/lib/pq v1.10.9")
	case "mysql_database-sql":
		add("github.com/go-sql-driver/mysql v1.8.1")
	case "sqlite_database-sql":
		add("github.com/mattn/go-sqlite3 v1.14.22")
	case "mongodb":
		add("go.mongodb.org/mongo-driver v1.16.1")
	}

	if p.UseRedis {
		add("github.com/redis/go-redis/v9 v9.6.1")
	}

	if p.UseAuth {
		add("golang.org/x/crypto v0.27.0", "github.com/google/uuid v1.6.0")
		if p.AuthType == "jwt" {
			add("github.com/golang-jwt/jwt/v5 v5.2.1")
		}
	}

	if p.UseOAuth {
		add(
			"github.com/markbates/goth v1.80.0",
			"github.com/gorilla/sessions v1.3.0",
		)
		if p.Framework == "fiber" {
			// adaptor lives inside gofiber/fiber/v2 itself, no extra module.
		}
	}

	switch p.Logger {
	case "zerolog":
		add("github.com/rs/zerolog v1.33.0")
	case "zap":
		add("go.uber.org/zap v1.27.0")
	}

	switch p.EnvLoader {
	case "godotenv":
		add("github.com/joho/godotenv v1.5.1")
	case "viper":
		add("github.com/spf13/viper v1.19.0")
	}

	return r
}
