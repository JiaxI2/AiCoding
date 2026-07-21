package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"
)

const SchemaVersion = 1

const (
	ErrorKindUsage      = "usage"
	ErrorKindExecution  = "execution"
	ErrorKindValidation = "validation"
)

type Category string

const (
	CategoryNone            Category = "none"
	CategoryUsage           Category = "usage"
	CategoryValidation      Category = "validation"
	CategoryTransient       Category = "transient"
	CategoryToolchain       Category = "toolchain"
	CategoryEvidenceMissing Category = "evidence-missing"
	CategoryConflict        Category = "conflict"
	CategoryInternal        Category = "internal"
)

type Result struct {
	SchemaVersion int         `json:"schemaVersion"`
	Command       string      `json:"command"`
	OK            bool        `json:"ok"`
	ErrorKind     string      `json:"errorKind,omitempty"`
	Category      Category    `json:"category"`
	Retryable     bool        `json:"retryable"`
	NextAction    string      `json:"nextAction,omitempty"`
	Message       string      `json:"message,omitempty"`
	RepoRoot      string      `json:"repoRoot,omitempty"`
	InputDigest   string      `json:"inputDigest,omitempty"`
	PlanDigest    string      `json:"planDigest,omitempty"`
	Checked       interface{} `json:"checked,omitempty"`
	Data          interface{} `json:"data,omitempty"`
	Warnings      []string    `json:"warnings,omitempty"`
	Errors        []string    `json:"errors,omitempty"`
	ElapsedMS     int64       `json:"elapsedMs"`
}

func ValidCategory(category Category) bool {
	switch category {
	case CategoryNone, CategoryUsage, CategoryValidation, CategoryTransient,
		CategoryToolchain, CategoryEvidenceMissing, CategoryConflict, CategoryInternal:
		return true
	default:
		return false
	}
}

func WithDecision(result Result, category Category, nextAction string) Result {
	result.Category = category
	result.NextAction = strings.TrimSpace(nextAction)
	result.Retryable = category == CategoryTransient || category == CategoryConflict
	return result
}

// FinalizeDecision makes every emitted Result self-contained for a machine
// consumer. Invalid or contradictory categories fail closed as internal errors.
func FinalizeDecision(result Result) Result {
	if result.OK {
		if result.Category != "" && result.Category != CategoryNone {
			return invalidDecision(result, "successful result used failure category "+string(result.Category))
		}
		result.Category = CategoryNone
		result.Retryable = false
		result.NextAction = ""
		return result
	}
	if result.Category == "" {
		switch result.ErrorKind {
		case ErrorKindUsage:
			result.Category = CategoryUsage
		case ErrorKindValidation:
			result.Category = CategoryValidation
		default:
			result.Category = CategoryInternal
		}
	}
	if !ValidCategory(result.Category) || result.Category == CategoryNone {
		return invalidDecision(result, "invalid failure category "+string(result.Category))
	}
	result.Retryable = result.Category == CategoryTransient || result.Category == CategoryConflict
	if strings.TrimSpace(result.NextAction) == "" {
		switch result.Category {
		case CategoryUsage:
			result.NextAction = "aicoding --help"
		default:
			result.NextAction = "aicoding doctor --all --json"
		}
	}
	return result
}

func invalidDecision(result Result, message string) Result {
	result.OK = false
	result.ErrorKind = ErrorKindExecution
	result.Category = CategoryInternal
	result.Retryable = false
	result.NextAction = "aicoding doctor --all --json"
	result.Errors = append(result.Errors, message)
	if result.Message == "" {
		result.Message = "invalid structured result decision"
	}
	return result
}

type ValidationError struct {
	messages []string
}

func (e ValidationError) Error() string {
	return strings.Join(e.messages, "; ")
}

func Elapsed(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

func Fail(command string, start time.Time, message string, data interface{}, errs ...string) Result {
	return Result{
		SchemaVersion: SchemaVersion,
		Command:       command,
		OK:            false,
		ErrorKind:     ErrorKindExecution,
		Message:       message,
		Data:          data,
		Errors:        errs,
		ElapsedMS:     Elapsed(start),
	}
}

func BoolErr(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return ValidationError{messages: append([]string{}, errs...)}
}

func IsValidationError(err error) bool {
	var target ValidationError
	return errors.As(err, &target)
}

func WriteJSON(v interface{}) {
	_ = WriteJSONTo(os.Stdout, v)
}

func WriteJSONTo(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func WriteText(res Result) {
	WriteTextTo(os.Stdout, res)
}

func WriteTextTo(w io.Writer, res Result) {
	status := "OK"
	if !res.OK {
		status = "FAIL"
	}
	if res.Message != "" {
		fmt.Fprintf(w, "[%s] %s (%d ms)\n", status, res.Message, res.ElapsedMS)
	} else {
		fmt.Fprintf(w, "[%s] %s (%d ms)\n", status, res.Command, res.ElapsedMS)
	}
	for _, e := range res.Errors {
		fmt.Fprintf(w, "  - %s\n", e)
	}
	for _, warning := range res.Warnings {
		fmt.Fprintf(w, "  ! %s\n", warning)
	}
	writeDataText(w, res.Data)
}

func writeDataText(w io.Writer, data interface{}) {
	value := reflect.ValueOf(data)
	if !value.IsValid() || value.Kind() != reflect.Slice {
		return
	}
	for i := 0; i < value.Len(); i++ {
		item := value.Index(i)
		if item.Kind() == reflect.Pointer {
			item = item.Elem()
		}
		if item.Kind() != reflect.Struct {
			continue
		}
		if hasFields(item, "Order", "ID", "Enabled", "Version", "Manifest") {
			fmt.Fprintf(w, "  %02d %-38s %-8t %-10s %s\n",
				fieldInt(item, "Order"),
				fieldString(item, "ID"),
				fieldBool(item, "Enabled"),
				fieldString(item, "Version"),
				fieldString(item, "Manifest"))
			continue
		}
		if hasFields(item, "OK", "ID", "Status", "Errors") {
			label := "OK"
			if !fieldBool(item, "OK") {
				label = "FAIL"
			}
			fmt.Fprintf(w, "  [%s] %-38s %s\n", label, fieldString(item, "ID"), fieldString(item, "Status"))
			for _, e := range fieldStringSlice(item, "Errors") {
				fmt.Fprintf(w, "      - %s\n", e)
			}
		}
	}
}

func hasFields(v reflect.Value, names ...string) bool {
	for _, name := range names {
		if !v.FieldByName(name).IsValid() {
			return false
		}
	}
	return true
}

func fieldString(v reflect.Value, name string) string {
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.String {
		return ""
	}
	return f.String()
}

func fieldBool(v reflect.Value, name string) bool {
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.Bool {
		return false
	}
	return f.Bool()
}

func fieldInt(v reflect.Value, name string) int64 {
	f := v.FieldByName(name)
	if !f.IsValid() {
		return 0
	}
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return f.Int()
	default:
		return 0
	}
}

func fieldStringSlice(v reflect.Value, name string) []string {
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.Slice || f.Type().Elem().Kind() != reflect.String {
		return nil
	}
	out := make([]string, 0, f.Len())
	for i := 0; i < f.Len(); i++ {
		out = append(out, f.Index(i).String())
	}
	return out
}
