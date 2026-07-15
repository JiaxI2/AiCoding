package cuserstyle

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func RunInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	root := fs.String("root", ".", "target repository")
	force := fs.Bool("force", false, "overwrite existing configuration")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir := filepath.Join(*root, "UserCfg", "UserStyle")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	configPath := filepath.Join(dir, "c-kit.json")
	snippetsPath := filepath.Join(dir, "c-snippets.json")
	for _, path := range []string{configPath, snippetsPath} {
		if _, err := os.Stat(path); err == nil && !*force {
			return fmt.Errorf("%s already exists; use --force to replace it", path)
		}
	}
	if err := writeAtomic(configPath, []byte(defaultConfigJSON+"\n")); err != nil {
		return err
	}
	if err := writeAtomic(snippetsPath, []byte(defaultSnippetsJSON+"\n")); err != nil {
		return err
	}
	stylePath := filepath.Join(*root, ".clang-format")
	if _, err := os.Stat(stylePath); err != nil || *force {
		if err := writeAtomic(stylePath, []byte(defaultClangFormat)); err != nil {
			return err
		}
	}
	fmt.Println(configPath)
	fmt.Println(snippetsPath)
	fmt.Println(stylePath)
	return nil
}
