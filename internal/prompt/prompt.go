// Package prompt implements the interactive wizard that collects a
// config.Project from the user, one small huh.Form at a time so that
// later questions can depend on earlier answers (framework -> which
// ORMs are offered, auth on -> which auth type, etc).
package prompt

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/tobibamidele/goforge/internal/config"
)

var moduleSafe = regexp.MustCompile(`^[a-zA-Z0-9._~/-]+$`)

func Run() (config.Project, error) {
	p := config.Project{GoVersion: "1.23"}

	// --- Stage 1: identity -------------------------------------------------
	nameForm := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Project name").
			Placeholder("orders-api").
			Value(&p.Name).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("project name can't be empty")
				}
				return nil
			}),
		huh.NewInput().
			Title("Go module path").
			DescriptionFunc(func() string {
				return "e.g. github.com/you/" + p.Name
			}, &p.Name).
			Value(&p.Module).
			Validate(func(s string) error {
				if s == "" || !moduleSafe.MatchString(s) {
					return fmt.Errorf("not a valid module path")
				}
				return nil
			}),
	))
	if err := nameForm.Run(); err != nil {
		return p, err
	}

	// --- Stage 2: framework + db --------------------------------------------
	coreForm := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("HTTP framework").
			Options(
				huh.NewOption("Gin", "gin"),
				huh.NewOption("Fiber", "fiber"),
				huh.NewOption("Echo", "echo"),
				huh.NewOption("net/http (stdlib 1.22+ ServeMux)", "nethttp"),
			).
			Value(&p.Framework),
		huh.NewSelect[string]().
			Title("Database").
			Options(
				huh.NewOption("PostgreSQL", "postgres"),
				huh.NewOption("MySQL", "mysql"),
				huh.NewOption("SQLite", "sqlite"),
				huh.NewOption("MongoDB", "mongodb"),
				huh.NewOption("None", "none"),
			).
			Value(&p.Database),
	))
	if err := coreForm.Run(); err != nil {
		return p, err
	}

	// --- Stage 3: ORM (depends on db) --------------------------------------
	if p.Database != "none" {
		if p.Database == "mongodb" {
			p.ORM = "mongo-driver"
		} else {
			ormForm := huh.NewForm(huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("ORM / data-access layer for %s", p.Database)).
					Options(
						huh.NewOption("Bun (lightweight SQL-first ORM)", "bun"),
						huh.NewOption("GORM", "gorm"),
						huh.NewOption("sqlx (raw SQL, typed scanning)", "sqlx"),
						huh.NewOption("database/sql only (no helper library)", "database-sql"),
					).
					Value(&p.ORM),
			))
			if err := ormForm.Run(); err != nil {
				return p, err
			}
		}
	} else {
		p.ORM = "none"
	}

	// --- Stage 4: infra toggles ---------------------------------------------
	infraForm := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title("Use Redis?").Value(&p.UseRedis),
		huh.NewConfirm().Title("Generate a Dockerfile + docker-compose.yml?").Value(&p.UseDocker),
		huh.NewConfirm().Title("Set up authentication?").Value(&p.UseAuth),
	))
	if err := infraForm.Run(); err != nil {
		return p, err
	}

	// --- Stage 5: auth details (depends on UseAuth) -------------------------
	if p.UseAuth {
		authForm := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Auth strategy").
				Options(
					huh.NewOption("JWT (stateless, access+refresh tokens)", "jwt"),
					huh.NewOption("Server-side sessions (cookie + store)", "session"),
				).
				Value(&p.AuthType),
			huh.NewConfirm().
				Title("Add OAuth / social login too? (wires up markbates/goth)").
				Value(&p.UseOAuth),
		))
		if err := authForm.Run(); err != nil {
			return p, err
		}

		if p.UseOAuth {
			var providers []string
			oauthForm := huh.NewForm(huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("OAuth providers to wire up").
					Options(
						huh.NewOption("Google", "google"),
						huh.NewOption("GitHub", "github"),
						huh.NewOption("GitLab", "gitlab"),
						huh.NewOption("Discord", "discord"),
					).
					Value(&providers),
			))
			if err := oauthForm.Run(); err != nil {
				return p, err
			}
			if len(providers) == 0 {
				providers = []string{"google"}
			}
			p.OAuthProviders = providers
		}
	}

	// --- Stage 6: everything else -------------------------------------------
	miscForm := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Logger").
			Options(
				huh.NewOption("slog (stdlib)", "slog"),
				huh.NewOption("zerolog", "zerolog"),
				huh.NewOption("zap", "zap"),
			).
			Value(&p.Logger),
		huh.NewSelect[string]().
			Title("Env / config loading").
			Options(
				huh.NewOption("godotenv + os.Getenv", "godotenv"),
				huh.NewOption("viper", "viper"),
			).
			Value(&p.EnvLoader),
		huh.NewConfirm().Title("Generate a Makefile?").Value(&p.UseMakefile),
		huh.NewConfirm().Title("Generate a GitHub Actions CI workflow (build + test)?").Value(&p.UseCI),
		huh.NewConfirm().Title("Run `git init` when done?").Value(&p.GitInit),
	))
	if err := miscForm.Run(); err != nil {
		return p, err
	}

	return p, nil
}
