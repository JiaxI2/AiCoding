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
	ExitSuccess = 0
	ExitFailure = 1
	ExitUsage   = 2
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

func validChoice(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

type deprecatedChoiceFlag struct {
	Old         string
	Replacement string
	Values      []string
	Drop        bool
	Warning     string
}

func rewriteDeprecatedChoiceFlag(args []string, spec deprecatedChoiceFlag) ([]string, []string, error) {
	oldName := strings.TrimLeft(spec.Old, "-")
	newName := strings.TrimLeft(spec.Replacement, "-")
	if oldName == "" || (!spec.Drop && newName == "") {
		return nil, nil, usageErrorf("invalid deprecated flag rewrite")
	}
	oldIndex, oldWidth, value, found, err := findChoiceFlag(args, oldName)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return append([]string(nil), args...), nil, nil
	}
	if _, _, _, duplicate, duplicateErr := findChoiceFlag(args[oldIndex+oldWidth:], oldName); duplicateErr != nil || duplicate {
		if duplicateErr != nil {
			return nil, nil, duplicateErr
		}
		return nil, nil, usageErrorf("--%s may be specified only once", oldName)
	}
	if !spec.Drop {
		if _, _, _, present, presentErr := findChoiceFlag(args, newName); presentErr != nil {
			return nil, nil, presentErr
		} else if present {
			return nil, nil, usageErrorf("deprecated --%s cannot be combined with --%s", oldName, newName)
		}
	}
	canonical := ""
	for _, candidate := range spec.Values {
		if strings.EqualFold(strings.TrimSpace(value), candidate) {
			canonical = candidate
			break
		}
	}
	if canonical == "" {
		return nil, nil, usageErrorf("deprecated --%s accepts %s", oldName, strings.Join(spec.Values, "|"))
	}
	rewritten := make([]string, 0, len(args)+1)
	rewritten = append(rewritten, args[:oldIndex]...)
	if !spec.Drop {
		rewritten = append(rewritten, "--"+newName, canonical)
	}
	rewritten = append(rewritten, args[oldIndex+oldWidth:]...)
	warnings := []string{}
	if strings.TrimSpace(spec.Warning) != "" {
		warnings = append(warnings, spec.Warning)
	}
	return rewritten, warnings, nil
}

func findChoiceFlag(args []string, name string) (index int, width int, value string, found bool, err error) {
	long := "--" + name
	short := "-" + name
	for index, arg := range args {
		if arg == "--" {
			break
		}
		if arg == long || arg == short {
			if index+1 >= len(args) {
				return 0, 0, "", false, usageErrorf("flag needs an argument: --%s", name)
			}
			return index, 2, args[index+1], true, nil
		}
		for _, prefix := range []string{long + "=", short + "="} {
			if strings.HasPrefix(arg, prefix) {
				value := strings.TrimPrefix(arg, prefix)
				if value == "" {
					return 0, 0, "", false, usageErrorf("flag needs an argument: --%s", name)
				}
				return index, 1, value, true, nil
			}
		}
	}
	return 0, 0, "", false, nil
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
