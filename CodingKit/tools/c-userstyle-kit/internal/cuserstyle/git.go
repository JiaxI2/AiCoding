package cuserstyle

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var hunkRE = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

func stagedChangedLines() (map[string][]LineRange, error) {
	cmd := exec.Command("git", "diff", "--cached", "--unified=0", "--no-color", "--diff-filter=ACMR", "--", "*.c", "*.h")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached failed: %w", err)
	}
	result := map[string][]LineRange{}
	var current string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "+++ b/") {
			current = strings.TrimPrefix(line, "+++ b/")
			continue
		}
		m := hunkRE.FindStringSubmatch(line)
		if current == "" || m == nil {
			continue
		}
		start, _ := strconv.Atoi(m[1])
		count := 1
		if m[2] != "" {
			count, _ = strconv.Atoi(m[2])
		}
		if count > 0 {
			result[current] = append(result[current], LineRange{Start: start, End: start + count - 1})
		}
	}
	return result, sc.Err()
}

func stagedContent(path string) ([]byte, error) {
	out, err := exec.Command("git", "show", ":"+path).Output()
	if err != nil {
		return nil, fmt.Errorf("read staged file %s: %w", path, err)
	}
	return out, nil
}

func lineSelected(line int, ranges []LineRange) bool {
	for _, r := range ranges {
		if line >= r.Start && line <= r.End {
			return true
		}
	}
	return false
}
