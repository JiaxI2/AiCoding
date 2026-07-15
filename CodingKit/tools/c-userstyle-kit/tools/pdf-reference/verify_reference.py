#!/usr/bin/env python3
"""Verify that the normalized Markdown preserves the complete PDF rule set."""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path

import pdfplumber


CHAPTERS = {
    0: "规范制订说明",
    1: "头文件",
    2: "函数",
    3: "标识符命名与定义",
    4: "变量",
    5: "宏、常量",
    6: "质量保证",
    7: "程序效率",
    8: "注释",
    9: "排版与格式",
    10: "表达式",
    11: "代码编辑、编译",
    12: "可测性",
    13: "安全性",
    14: "单元测试",
    15: "可移植性",
    16: "业界编程规范",
}

CLAUSE_RE = re.compile(r"(原则|规则|建议)\s*(\d+\.\d+)")
PAGE_RE = re.compile(r"<!-- PDF page (\d+) of (\d+) -->")


def clauses(text: str) -> set[str]:
    return {f"{kind}-{number}" for kind, number in CLAUSE_RE.findall(text)}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--pdf", required=True, type=Path)
    parser.add_argument("--markdown", required=True, type=Path)
    args = parser.parse_args()

    with pdfplumber.open(args.pdf) as document:
        page_count = len(document.pages)
        pdf_text = "\n".join(page.extract_text() or "" for page in document.pages)

    markdown = args.markdown.read_text(encoding="utf-8")
    pdf_clauses = clauses(pdf_text)
    markdown_clauses = clauses(markdown)
    missing_clauses = sorted(pdf_clauses - markdown_clauses)
    extra_clauses = sorted(markdown_clauses - pdf_clauses)
    missing_chapters = [
        f"{number} {title}"
        for number, title in CHAPTERS.items()
        if f"## {number} {title}" not in markdown
    ]
    page_markers = PAGE_RE.findall(markdown)
    code_fence_count = markdown.count("```c")
    noise = [
        token
        for token in ("密级：confidentiality level", "Huawei Confidential 第")
        if token in markdown
    ]

    checks = {
        "pdf_pages_61": page_count == 61,
        "all_chapters_present": not missing_chapters,
        "all_pdf_clauses_present": not missing_clauses,
        "no_unknown_clauses": not extra_clauses,
        "page_markers_present": len(page_markers) >= 59,
        "c_examples_fenced": code_fence_count >= 10,
        "code_fences_balanced": markdown.count("```") == (code_fence_count * 2),
        "repeated_header_footer_removed": not noise,
        "markdown_substantial": len(markdown) >= 40000,
        "comment_chapter_searchable": all(
            token in markdown
            for token in (
                "## 8 注释",
                "### 原则 8.1",
                "### 规则 8.2",
                "### 规则 8.6",
                "### 建议 8.3",
            )
        ),
    }
    ok = all(checks.values())
    result = {
        "ok": ok,
        "pdf_pages": page_count,
        "pdf_clause_count": len(pdf_clauses),
        "markdown_clause_count": len(markdown_clauses),
        "page_marker_count": len(page_markers),
        "c_code_block_count": code_fence_count,
        "missing_chapters": missing_chapters,
        "missing_clauses": missing_clauses,
        "extra_clauses": extra_clauses,
        "noise": noise,
        "checks": checks,
    }
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
