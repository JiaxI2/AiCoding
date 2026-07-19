package repocontext

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// generated is one rendered artifact: a repo-relative path and its content.
type generated struct {
	Path    string
	Content string
}

// render produces the full deterministic set of scoped context files for the
// given facts: one overview index plus one file per domain. Output ordering is
// stable (index first, then domains sorted by path) so digests do not churn.
func render(facts Facts) []generated {
	files := []generated{{Path: ownedRoot + "/index.md", Content: renderIndex(facts)}}
	for _, domain := range facts.Domains {
		files = append(files, generated{
			Path:    ownedRoot + "/domains/" + domain.Path + ".md",
			Content: renderDomain(facts, domain),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func renderIndex(facts Facts) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 仓库上下文总览：%s\n\n", facts.Repo)
	b.WriteString("> 由 `aicoding lifecycle --scope repo-context` 自动生成，随代码演进更新。\n")
	b.WriteString("> 请勿手工编辑本目录；改动会在下次 update 时被覆盖。\n\n")

	b.WriteString("## 工具链\n\n")
	if len(facts.Toolchains) == 0 {
		b.WriteString("- （未识别）\n")
	} else {
		for _, tool := range facts.Toolchains {
			fmt.Fprintf(&b, "- %s\n", tool)
		}
	}

	b.WriteString("\n## 语言构成\n\n")
	if len(facts.Languages) == 0 {
		b.WriteString("- （未识别源文件）\n")
	} else {
		for _, lang := range facts.Languages {
			fmt.Fprintf(&b, "- %s（%s）：%d 个文件\n", lang.Language, lang.Extension, lang.Files)
		}
	}

	b.WriteString("\n## 顶层域\n\n")
	if len(facts.Domains) == 0 {
		b.WriteString("- （无子目录）\n")
	} else {
		for _, domain := range facts.Domains {
			fmt.Fprintf(&b, "- `%s/`：%d 个文件，主语言 %s → 见 `domains/%s.md`\n",
				domain.Path, domain.Files, orNone(domain.PrimaryLanguage), domain.Path)
		}
	}
	return b.String()
}

func renderDomain(facts Facts, domain Domain) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 域：%s/\n\n", domain.Path)
	fmt.Fprintf(&b, "> 仓库 `%s` 的顶层域。自动生成，请勿手工编辑。\n\n", facts.Repo)
	fmt.Fprintf(&b, "- 路径：`%s/`\n", domain.Path)
	fmt.Fprintf(&b, "- 源文件数：%d\n", domain.Files)
	fmt.Fprintf(&b, "- 主语言：%s\n", orNone(domain.PrimaryLanguage))
	b.WriteString("\n代理进入本域前，先读本文件了解规模与主语言；")
	b.WriteString("详细约定见仓库根 `AGENTS.md` 与 `docs/` 权威文档。\n")
	return b.String()
}

func orNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "（未识别）"
	}
	return value
}

// contentDigest returns the stable digest of an artifact's content, matching the
// sha256 form used by the registry snapshot primitives.
func contentDigest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("sha256:%x", sum)
}
