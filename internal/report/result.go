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

type Result struct {
	SchemaVersion int         `json:"schemaVersion"`
	Command       string      `json:"command"`
	OK            bool        `json:"ok"`
	Message       string      `json:"message,omitempty"`
	RepoRoot      string      `json:"repoRoot,omitempty"`
	Checked       interface{} `json:"checked,omitempty"`
	Data          interface{} `json:"data,omitempty"`
	Warnings      []string    `json:"warnings,omitempty"`
	Errors        []string    `json:"errors,omitempty"`
	ElapsedMS     int64       `json:"elapsedMs"`
}

func Elapsed(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

func Fail(command string, start time.Time, message string, data interface{}, errs ...string) Result {
	return Result{SchemaVersion: 1, Command: command, OK: false, Message: message, Data: data, Errors: errs, ElapsedMS: Elapsed(start)}
}

func BoolErr(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
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
