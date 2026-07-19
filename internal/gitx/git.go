package gitx

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func Run(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if repo != "" {
		cmd.Dir = repo
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func StagedFiles(repo string) ([]string, error) {
	out, err := Run(repo, "diff", "--cached", "--name-only", "--diff-filter=ACMR")
	if err != nil {
		return nil, err
	}
	return splitFileList(out), nil
}

// CommitFiles returns the paths changed by a single commit reference (default
// HEAD when ref is empty). Paths use forward slashes.
func CommitFiles(repo, ref string) ([]string, error) {
	if strings.TrimSpace(ref) == "" {
		ref = "HEAD"
	}
	out, err := Run(repo, "diff-tree", "--no-commit-id", "--name-only", "-r", ref)
	if err != nil {
		return nil, err
	}
	return splitFileList(out), nil
}

func splitFileList(out string) []string {
	var files []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(strings.ReplaceAll(line, "\\", "/"))
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}
