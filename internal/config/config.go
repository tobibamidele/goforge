package config

// Project holds every answer collected from the interactive wizard.
// It is the single source of truth passed into every template.
type Project struct {
	Name      string // directory + display name, e.g. "orders-api"
	Module    string // go module path, e.g. "github.com/tobi/orders-api"
	GoVersion string // e.g. "1.23"

	Framework string // gin | fiber | echo | nethttp

	Database string // postgres | mysql | sqlite | mongodb | none
	ORM      string // bun | gorm | sqlx | database-sql | mongo-driver | none

	UseRedis bool

	UseDocker bool

	UseAuth        bool
	AuthType       string // jwt | session
	UseOAuth       bool
	OAuthProviders []string // google, github, etc. (only when UseOAuth)

	Logger    string // slog | zerolog | zap
	EnvLoader string // godotenv | viper

	UseMakefile bool
	UseCI       bool // GitHub Actions
	GitInit     bool
}

// SQLDatabase reports whether the chosen database is a SQL one that goes
// through database/sql (as opposed to mongodb, which uses its own driver).
func (p Project) SQLDatabase() bool {
	switch p.Database {
	case "postgres", "mysql", "sqlite":
		return true
	default:
		return false
	}
}

func (p Project) HasDatabase() bool { return p.Database != "" && p.Database != "none" }

// TemplateKey returns the db+orm combination key used to look up the
// right connection-setup template, e.g. "postgres_bun", "mongodb_mongo-driver".
func (p Project) DBTemplateKey() string {
	if !p.HasDatabase() {
		return ""
	}
	if p.Database == "mongodb" {
		return "mongodb"
	}
	return p.Database + "_" + p.ORM
}
