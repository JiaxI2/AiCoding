package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/report"
)

const (
	ExitSuccess   = 0
	ExitFailure   = 1
	ExitUsage     = 2
	deprecatedTag = "CLI_DEPRECATED"
)

type usageError struct {
	message string
}

func (e usageError) Error() string {
	return e.message
}

type helpRequest struct {
	text string
}

func (e helpRequest) Error() string {
	return "help requested"
}

func usageErrorf(format string, args ...interface{}) error {
	return usageError{message: fmt.Sprintf(format, args...)}
}

func isUsageError(err error) bool {
	var target usageError
	return errors.As(err, &target)
}

func requestedHelp(err error) (string, bool) {
	var target helpRequest
	if !errors.As(err, &target) {
		return "", false
	}
	return target.text, true
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	err := fs.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		return helpRequest{text: flagSetHelp(fs)}
	}
	if err != nil {
		return usageErrorf("%s arguments: %v", fs.Name(), err)
	}
	return nil
}

func parseNoPositionals(fs *flag.FlagSet, args []string) error {
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return usageErrorf("%s does not accept positional arguments: %s", fs.Name(), strings.Join(fs.Args(), " "))
	}
	return nil
}

func isHelpArg(value string) bool {
	switch value {
	case "help", "--help", "-h":
		return true
	default:
		return false
	}
}

func commandRequiresSubcommand(command string) bool {
	switch command {
	case "hook", "docsync", "skill", "lifecycle", "cache", "codex", "mcp", "tag", "release", "kit", "doctor", "verify", "governance", "powershell":
		return true
	default:
		return false
	}
}

func validChoice(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func flagSetHelp(fs *flag.FlagSet) string {
	var options bytes.Buffer
	fs.SetOutput(&options)
	fs.PrintDefaults()
	fs.SetOutput(io.Discard)

	var out strings.Builder
	fmt.Fprintf(&out, "Usage: aicoding %s [options]\n", fs.Name())
	if options.Len() != 0 {
		out.WriteString("\nOptions:\n")
		out.Write(options.Bytes())
	}
	return out.String()
}

func exitCodeFor(res report.Result, err error) int {
	if isUsageError(err) {
		return ExitUsage
	}
	if err != nil || !res.OK {
		return ExitFailure
	}
	return ExitSuccess
}

func addDeprecation(res report.Result, canonical string) report.Result {
	warning := deprecatedTag + ": use " + canonical
	for _, existing := range res.Warnings {
		if existing == warning {
			return res
		}
	}
	res.Warnings = append([]string{warning}, res.Warnings...)
	return res
}

func deprecatedCommand(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	switch strings.ToLower(args[0]) {
	case "smoke":
		return "aicoding test --profile Smoke", true
	case "ci":
		return "aicoding test --profile " + flagValue(args[1:], "profile", "Smoke"), true
	case "full":
		return "aicoding test --profile Full", true
	case "test":
		if len(args) < 2 {
			return "", false
		}
		switch strings.ToLower(args[1]) {
		case "full":
			return "aicoding test --profile Full", true
		case "release":
			return "aicoding test --profile Release", true
		}
	case "kit":
		if len(args) >= 2 && strings.EqualFold(args[1], "lifecycle") {
			action := strings.ToLower(flagValue(args[2:], "action", "status"))
			if action == "status" {
				return "aicoding lifecycle status --scope kit", true
			}
			return "aicoding lifecycle plan --action " + action + " --scope kit", true
		}
	case "mcp":
		if len(args) >= 2 {
			switch strings.ToLower(args[1]) {
			case "install", "update", "uninstall":
				return "aicoding lifecycle " + strings.ToLower(args[1]) + " --scope mcp", true
			}
		}
	case "status":
		return "aicoding lifecycle status --scope all", true
	}
	return "", false
}

func flagValue(args []string, name string, fallback string) string {
	long := "--" + name
	for index, arg := range args {
		if arg == long && index+1 < len(args) {
			return args[index+1]
		}
		if strings.HasPrefix(arg, long+"=") {
			return strings.TrimPrefix(arg, long+"=")
		}
	}
	return fallback
}
