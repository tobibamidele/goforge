package generator

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tobibamidele/goforge/internal/config"
)

// latheCLIVersion pins the lathe CLI that goforge installs when `lathe` is
// not already on PATH. Keep it in lock-step with the module version listed in
// goModRequires.
const latheCLIVersion = "v0.1.0"

// RunLathe runs the lathe toolchain inside a freshly scaffolded project that
// chose lathe as its ORM. It makes sure the CLI is available, resolves the
// module's dependencies, then generates the Go package (internal/db) and the
// initial migration (migrations/).
//
// It is best-effort: the scaffold is complete without it — the schema and the
// connect code are already on disk — so a failure here returns a descriptive
// error and the caller prints the equivalent manual commands instead of
// aborting the scaffold.
func RunLathe(outDir string, p config.Project, out io.Writer) error {
	if !p.HasDatabase() || p.ORM != "lathe" {
		return nil
	}

	latheBin, err := ensureLatheCLI()
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "┌─ running lathe toolchain")
	if err := runIn(outDir, out, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy (to resolve the lathe + driver deps): %w", err)
	}
	if err := runIn(outDir, out, latheBin, "generate", "--schema", "internal/schema"); err != nil {
		return fmt.Errorf("lathe generate --schema internal/schema: %w", err)
	}
	if err := runIn(outDir, out, latheBin, "migrate", "diff", "init", "--schema", "internal/schema"); err != nil {
		return fmt.Errorf("lathe migrate diff init: %w", err)
	}
	fmt.Fprintln(out, "└─ done: internal/db generated, initial migration written to migrations/")
	return nil
}

// ensureLatheCLI returns a path to a working `lathe` binary, installing it if
// it is not on PATH. The returned path is used directly so it does not depend
// on the caller's PATH including $GOBIN or $GOPATH/bin.
func ensureLatheCLI() (string, error) {
	if bin, err := exec.LookPath("lathe"); err == nil {
		return bin, nil
	}

	fmt.Fprintln(os.Stderr, "┌─ lathe not found on PATH, installing it")
	cmd := exec.Command("go", "install", "github.com/tobibamidele/lathe/cmd/lathe@"+latheCLIVersion)
	if body, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("lathe is not on PATH and `go install github.com/tobibamidele/lathe/cmd/lathe@%s` failed: %v\n%s",
			latheCLIVersion, err, body)
	}

	for _, bin := range goBinCandidates() {
		if st, err := os.Stat(bin); err == nil && !st.IsDir() {
			return bin, nil
		}
	}
	return "", fmt.Errorf("installed lathe %s but could not find the binary (looked for %v)",
		latheCLIVersion, goBinCandidates())
}

func goBinCandidates() []string {
	var out []string
	for key, sub := range map[string]string{"GOBIN": "", "GOPATH": "bin"} {
		if raw, err := exec.Command("go", "env", key).Output(); err == nil {
			if s := strings.TrimSpace(string(raw)); s != "" {
				out = append(out, filepath.Join(s, sub, "lathe"))
			}
		}
	}
	return out
}

// runIn runs name in dir, streaming its standard output and error to out.
func runIn(dir string, out io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}
