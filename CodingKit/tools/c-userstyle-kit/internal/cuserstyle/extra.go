package cuserstyle

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func RunDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	configPath := fs.String("config", "", "configuration")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		return err
	}
	_, gitErr := exec.LookPath("git")
	_, formatErr := exec.LookPath("clang-format")
	_, gccErr := exec.LookPath("gcc")
	_, clangErr := exec.LookPath("clang")
	result := map[string]any{
		"ok":               gitErr == nil && gccErr == nil && clangErr == nil,
		"config":           cfg.ID,
		"git":              gitErr == nil,
		"formatter":        formatErr == nil,
		"gcc":              gccErr == nil,
		"clang":            clangErr == nil,
		"changedLinesOnly": cfg.Style.ChangedLinesOnly,
	}
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}
	fmt.Printf("config: %s\ngit: %v\nformatter: %v\ngcc: %v\nclang: %v\nchanged-lines-only: %v\n",
		cfg.ID, gitErr == nil, formatErr == nil, gccErr == nil, clangErr == nil,
		cfg.Style.ChangedLinesOnly)
	return nil
}

func RunBench(args []string) error {
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	configPath := fs.String("config", "", "configuration")
	file := fs.String("file", "", "file")
	n := fs.Int("n", 1000, "iterations")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	start := time.Now()
	total := 0
	for i := 0; i < *n; i++ {
		total += len(lintContent(*file, data, nil, cfg, false))
	}
	elapsed := time.Since(start)
	fmt.Printf("iterations=%d total=%s average=%s diagnostics=%d\n",
		*n, elapsed, elapsed/time.Duration(*n), total)
	return nil
}
