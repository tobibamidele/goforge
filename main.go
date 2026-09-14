package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tobibamidele/goforge/internal/generator"
	"github.com/tobibamidele/goforge/internal/prompt"
)

func main() {
	fmt.Println("goforge — scaffold a new Go HTTP project")
	fmt.Println()

	p, err := prompt.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cancelled:", err)
		os.Exit(1)
	}

	outDir, err := filepath.Abs(p.Name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if info, err := os.Stat(outDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(outDir)
		if len(entries) > 0 {
			fmt.Fprintf(os.Stderr, "refusing to write into non-empty directory %s\n", outDir)
			os.Exit(1)
		}
	}

	files, err := generator.Generate(p, outDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generation failed:", err)
		os.Exit(1)
	}

	if p.GitInit {
		cmd := exec.Command("git", "init")
		cmd.Dir = outDir
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "git init failed (continuing anyway):", err)
		}
	}

	fmt.Println()
	fmt.Printf("✔ scaffolded %d files in %s\n\n", len(files), outDir)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", p.Name)
	fmt.Println("  cp .env.example .env")
	fmt.Println("  go mod tidy")
	if p.UseDocker {
		fmt.Println("  docker compose up -d")
	}
	fmt.Println("  go run ./cmd/server")
}
