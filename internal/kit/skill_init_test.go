package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillInitPreviewDryRunAndExternalWrite(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	preview, err := InitSkill(repo, "demo-skill", SkillInitOptions{DryRun: true})
	if err != nil || !preview.OK || preview.OutputMode != "preview" || !strings.Contains(preview.Content, "## Gate Rules") {
		t.Fatalf("Skill preview failed: report=%#v err=%v", preview, err)
	}
	if len(preview.Files) != 1 || preview.Files[0].Action != "preview" || preview.Files[0].Digest == "" {
		t.Fatalf("Skill preview evidence is incomplete: %#v", preview.Files)
	}

	out := t.TempDir()
	dryRun, err := InitSkill(repo, "demo-skill", SkillInitOptions{Out: out, DryRun: true})
	target := filepath.Join(out, "SKILL.md")
	if err != nil || !dryRun.OK || len(dryRun.Files) != 1 || dryRun.Files[0].Action != "planned-create" {
		t.Fatalf("Skill output dry-run failed: report=%#v err=%v", dryRun, err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("Skill dry-run wrote a file: %v", err)
	}

	created, err := InitSkill(repo, "demo-skill", SkillInitOptions{Out: out})
	if err != nil || !created.OK || created.Files[0].Action != "created" {
		t.Fatalf("Skill output failed: report=%#v err=%v", created, err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if errs := validateSkillInitContent(content, "demo-skill"); len(errs) != 0 {
		t.Fatalf("generated Skill is not structurally valid: %v", errs)
	}
	for _, section := range []string{"## Skill Type", "## When to use", "## Workflow", "## Workflow Contract", "## Verification", "## Constraints", "## Gate Rules", "## Human Confirmation"} {
		if !strings.Contains(string(content), section) {
			t.Fatalf("generated Skill is missing %q", section)
		}
	}
	before := string(content)
	duplicate, duplicateErr := InitSkill(repo, "demo-skill", SkillInitOptions{Out: out})
	if duplicateErr == nil || duplicate.OK || !strings.Contains(duplicateErr.Error(), "will not be overwritten") {
		t.Fatalf("duplicate Skill init was not fail-closed: report=%#v err=%v", duplicate, duplicateErr)
	}
	after, err := os.ReadFile(target)
	if err != nil || string(after) != before {
		t.Fatalf("duplicate Skill init changed the target: err=%v", err)
	}
}

func TestSkillInitRejectsAiCodingAndReadOnlySubmoduleOutputs(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	submodule := filepath.Join(repo, "CodingKit", "agents", "skills")
	if err := os.MkdirAll(submodule, 0o755); err != nil {
		t.Fatal(err)
	}
	insideRepo, repoErr := InitSkill(repo, "demo-skill", SkillInitOptions{Out: filepath.Join(repo, "tmp-skill")})
	if repoErr == nil || insideRepo.OK || !strings.Contains(repoErr.Error(), "does not own Skill source") {
		t.Fatalf("AiCoding-owned output was not rejected: report=%#v err=%v", insideRepo, repoErr)
	}
	insideSubmodule, submoduleErr := InitSkill(repo, "demo-skill", SkillInitOptions{Out: filepath.Join(submodule, "demo-skill")})
	if submoduleErr == nil || insideSubmodule.OK || !strings.Contains(submoduleErr.Error(), "read-only CodingKit/agents/skills") {
		t.Fatalf("read-only Skill submodule output was not rejected: report=%#v err=%v", insideSubmodule, submoduleErr)
	}
	if invalid, err := InitSkill(repo, "Demo_Skill", SkillInitOptions{}); err == nil || invalid.OK {
		t.Fatalf("invalid Skill id was accepted: %#v", invalid)
	}
}
