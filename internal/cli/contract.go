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
