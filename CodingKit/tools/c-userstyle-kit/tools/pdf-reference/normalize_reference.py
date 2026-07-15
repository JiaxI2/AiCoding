#!/usr/bin/env python3
"""Normalize the MarkItDown extraction into a searchable, readable reference."""

from __future__ import annotations

import argparse
import hashlib
import re
from pathlib import Path


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

SUBSECTIONS = {
    "0.1": "前言",
    "0.2": "代码总体原则",
    "0.3": "规范实施、解释",
    "0.4": "术语定义",
    "3.1": "通用命名规则",
    "3.2": "文件命名规则",
    "3.3": "变量命名规则",
    "3.4": "函数命名规则",
    "3.5": "宏的命名规则",
    "13.1": "字符串操作安全",
    "13.2": "整数安全",
    "13.3": "格式化输出安全",
    "13.4": "文件I/O安全",
    "13.5": "其它",
}

HEADER_LINES = {
    "密级：confidentiality level",
    "DKBA 2826-2011.5",
}

STANDALONE_LINES = {
    "华为技术有限公司内部技术规范",
    "2011年5月9日发布 2011年5月9日实施",
    "华为技术有限公司",
    "Huawei Technologies Co., Ltd.",
    "版权所有 侵权必究",
    "All rights reserved",
    "本规范拟制与解释部门：",
    "本规范的相关系列规范或文件：",
    "相关国际规范或文件一致性：",
    "替代或作废的其它规范或文件：",
    "相关规范或文件的相互关系：",
}

SPECIAL_HEADINGS = {
    "修订声明Revision declaration": "## 修订声明",
    "目 录Table of Contents": "## 目录",
    "范 围:": "## 范围",
    "范 围：": "## 范围",
    "简 介:": "## 简介",
    "简 介：": "## 简介",
    "背景": "### 背景",
    "术语定义：": "### 术语定义",
}

FOOTER_RE = re.compile(
    r"^2011-05-24\s+华为机密，未经许可不得扩散\s+Huawei Confidential\s+"
    r"第\s*(\d+)页,共\s*(\d+)页Page\s*\d+\s*,\s*Total\s*\d+\s*$"
)
CLAUSE_RE = re.compile(r"^(原则|规则|建议)\s*(\d+\.\d+)\s*[：:]?\s*(.*)$")
LIST_RE = re.compile(r"^([0-9]+)[、﹑]\s*(.*)$")


def is_terminal(text: str) -> bool:
    return text.endswith(("。", "！", "？", "；", "：", ".", "!", "?", ";", ":"))


def join_text(left: str, right: str) -> str:
    if not left:
        return right
    if not right:
        return left
    if left[-1].isascii() and left[-1].isalnum() and right[0].isascii() and right[0].isalnum():
        return left + " " + right
    return left + right


def is_chapter(line: str) -> tuple[int, str] | None:
    for number, title in CHAPTERS.items():
        if line == f"{number} {title}":
            return number, title
    return None


def is_subsection(line: str) -> tuple[str, str] | None:
    for number, title in SUBSECTIONS.items():
        if line == f"{number} {title}":
            return number, title
    return None


def looks_like_code(line: str) -> bool:
    if not line:
        return False
    if line.startswith(("/*", "*/", "//", "#include", "#define", "#if", "#else", "#endif")):
        return True
    if line in {"{", "}", "};", "...", "……"}:
        return True
    patterns = (
        r"^(if|else|for|while|do|switch)\b",
        r"^(case\s+.+:|default\s*:)$",
        r"^(return\b.*;|break;|continue;|goto\b.*;)$",
        r"^(typedef|struct|enum|union|static|extern|const|volatile)\b",
        r"^(void|bool|char|short|int|long|float|double|size_t|BYTE|BOOL|UINT[0-9]*)\b",
        r"^[A-Za-z_][A-Za-z0-9_]*(?:->|\.)[A-Za-z_][A-Za-z0-9_]*\s*[=+\-*/|&^]",
        r"^[A-Za-z_][A-Za-z0-9_]*\s*=.*;$",
        r"^[A-Za-z_][A-Za-z0-9_]*\s*\([^。；：]*\)\s*[;{]?$",
        r"^(\+\+|--)?[A-Za-z_][A-Za-z0-9_]*(\+\+|--);?$",
        r"^[A-Za-z_][A-Za-z0-9_]*\s*,\s*/\*.*\*/$",
        r"^\*\s*@?[A-Za-z_].*$",
    )
    return any(re.match(pattern, line) for pattern in patterns)


def normalize(raw_text: str, source_pdf: Path) -> str:
    output: list[str] = []
    pending_type: str | None = None
    pending_text = ""
    pending_prefix = ""
    deferred_markers: list[str] = []
    code_lines: list[str] = []
    in_c_comment = False
    seen_title_metadata = False

    def append_line(line: str = "") -> None:
        if line == "":
            if output and output[-1] != "":
                output.append("")
            return
        output.append(line)

    def flush_pending() -> None:
        nonlocal pending_type, pending_text, pending_prefix, deferred_markers
        if pending_type is None:
            return
        if pending_type == "clause":
            append_line(f"### {pending_prefix} {pending_text}")
        elif pending_type == "label":
            append_line(f"**{pending_prefix}** {pending_text}")
        else:
            append_line(pending_text)
        append_line()
        for marker in deferred_markers:
            append_line(marker)
            append_line()
        pending_type = None
        pending_text = ""
        pending_prefix = ""
        deferred_markers = []

    def flush_code() -> None:
        nonlocal code_lines, in_c_comment
        if not code_lines:
            return
        append_line("```c")
        output.extend(code_lines)
        append_line("```")
        append_line()
        code_lines = []
        in_c_comment = False

    cleaned_lines: list[str] = []
    for original in raw_text.replace("\r\n", "\n").replace("\r", "\n").split("\n"):
        line = original.strip()
        footer = FOOTER_RE.match(line)
        if footer:
            cleaned_lines.append(f"<!-- PDF page {footer.group(1)} of {footer.group(2)} -->")
            continue
        if line in HEADER_LINES:
            continue
        cleaned_lines.append(line)

    index = 0
    while index < len(cleaned_lines):
        line = cleaned_lines[index]
        index += 1

        if line.startswith("<!-- PDF page "):
            if code_lines:
                flush_code()
            if pending_type is not None:
                deferred_markers.append(line)
            else:
                append_line(line)
                append_line()
            continue

        if not line:
            if code_lines:
                flush_code()
            elif pending_type is not None and is_terminal(pending_text):
                flush_pending()
            continue

        if line == "|     |     |     |     |     |" and index < len(cleaned_lines):
            if cleaned_lines[index].startswith("| --- | --- | --- | --- | --- |"):
                index += 1
                continue

        chapter = is_chapter(line)
        subsection = is_subsection(line)
        clause = CLAUSE_RE.match(line)

        if chapter or subsection or clause or line.startswith("|") or line in SPECIAL_HEADINGS:
            flush_code()
            flush_pending()

        if line == "C语言编程规范":
            continue

        if line in SPECIAL_HEADINGS:
            append_line(SPECIAL_HEADINGS[line])
            append_line()
            continue

        if line in STANDALONE_LINES:
            flush_code()
            flush_pending()
            append_line(line)
            append_line()
            continue

        if re.match(r"^[0-9]+(?:\.[0-9]+)?\s+.+\.{5,}\s*[0-9]+$", line):
            flush_code()
            flush_pending()
            append_line(f"- {line}")
            continue

        if chapter:
            append_line(f"## {chapter[0]} {chapter[1]}")
            append_line()
            continue

        if subsection:
            append_line(f"### {subsection[0]} {subsection[1]}")
            append_line()
            continue

        if clause:
            pending_type = "clause"
            pending_prefix = f"{clause.group(1)} {clause.group(2)}"
            pending_text = clause.group(3).strip()
            if is_terminal(pending_text):
                flush_pending()
            continue

        if line.startswith("|"):
            append_line(line)
            if index >= len(cleaned_lines) or not cleaned_lines[index].startswith("|"):
                append_line()
            continue

        label = next((item for item in ("说明：", "示例：", "延伸阅读材料：") if line.startswith(item)), None)
        if label:
            flush_code()
            flush_pending()
            pending_type = "label"
            pending_prefix = label[:-1]
            pending_text = line[len(label) :].strip()
            if is_terminal(pending_text):
                flush_pending()
            continue

        list_item = LIST_RE.match(line)
        if list_item:
            flush_code()
            flush_pending()
            append_line(f"{list_item.group(1)}. {list_item.group(2)}")
            append_line()
            continue

        if line.startswith("“") and "”——" in line:
            flush_code()
            flush_pending()
            append_line(f"> {line}")
            append_line()
            continue

        code_like = looks_like_code(line) or in_c_comment
        if code_like:
            flush_pending()
            code_lines.append(line)
            if "/*" in line and "*/" not in line:
                in_c_comment = True
            if in_c_comment and "*/" in line:
                in_c_comment = False
            continue

        if code_lines:
            flush_code()

        if not seen_title_metadata and line == "DKBA":
            seen_title_metadata = True
            continue

        if pending_type is None:
            pending_type = "prose"
            pending_text = line
        else:
            pending_text = join_text(pending_text, line)

        if is_terminal(pending_text):
            flush_pending()

    flush_code()
    flush_pending()

    digest = hashlib.sha256(source_pdf.read_bytes()).hexdigest().upper()
    header = [
        "# 华为 C 语言编程规范（DKBA 2826-2011.5）",
        "",
        "> [!IMPORTANT]",
        "> 本文件是 C Kit 随 AiCoding 发布的可检索参考副本；原始页眉信息保留在 PDF 与 raw 转换件中。",
        "",
        "- 来源：`huawei-c-language-programming-standard-dkba-2826-2011-5.pdf`",
        "- PDF SHA-256：`{}`".format(digest),
        "- 初始转换：Microsoft MarkItDown `0.1.6`",
        "- 规范化：移除重复页眉、保留页码标记、合并跨页断句、提升章节与规则标题、标记 C 示例",
        "- 权威性：发生歧义时以原 PDF 页面为准",
        "",
    ]

    compact: list[str] = []
    for line in header + output:
        if line == "" and compact and compact[-1] == "":
            continue
        compact.append(line.rstrip())
    return "\n".join(compact).strip() + "\n"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--raw", required=True, type=Path)
    parser.add_argument("--pdf", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    normalized = normalize(args.raw.read_text(encoding="utf-8"), args.pdf)
    args.output.write_text(normalized, encoding="utf-8", newline="\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
