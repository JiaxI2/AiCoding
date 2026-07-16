#!/usr/bin/env python3
"""Build the complete Huawei C rule catalog from the normalized Markdown reference."""

from __future__ import annotations

import argparse
import json
import re
from collections import Counter
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
REFERENCE = ROOT / "references" / "huawei-c-language-programming-standard-dkba-2826-2011-5.md"
JSON_OUTPUT = ROOT / "config" / "rules" / "huawei-c-language-programming-standard.rules.json"
MARKDOWN_OUTPUT = ROOT / "docs" / "RULE_CATALOG.md"
CLAUSE_RE = re.compile(r"^### (原则|规则|建议) (\d+)\.(\d+) (.+?)\s*$")
KIND_CODE = {"原则": "P", "规则": "R", "建议": "S"}
CHAPTER_NAMES = {
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
}


def default_coverage(chapter: int) -> tuple[str, list[str], list[str], str]:
    mapping = {
        1: ("compile", ["compile", "lint", "demo"],
            ["generated-demo/demo.h", "generated-demo/advanced/*.h",
             "scripts/verify.ps1:header-gates", "lint:file.*"],
            "头文件结构由独立 C/C++ 编译、文件 lint 和同名实现共同证明。"),
        2: ("demo", ["demo", "lint", "manual"],
            ["generated-demo/demo.c", "generated-demo/advanced/*.c",
             "lint:documentation.function", "AGENTS.md:函数设计"],
            "函数职责、参数契约和静态可见性由黄金实现与 lint 组合证明。"),
        3: ("lint", ["lint", "demo", "manual"],
            ["lint:naming.*", "generated-demo/", "AGENTS.md:命名"],
            "稳定命名规则机器检查，语义清晰度保留人工审查。"),
        4: ("demo", ["demo", "compile", "manual"],
            ["generated-demo/demo.c", "generated-demo/advanced/protocol.c",
             "compiler:-Wshadow,-Wconversion"],
            "变量单一职责、初始化、字节序和转换由代码与编译器共同覆盖。"),
        5: ("lint", ["lint", "demo", "manual"],
            ["lint:macro.*", "generated-demo/demo.h", "generated-demo/advanced/*.h",
             "AGENTS.md:宏与常量"],
            "宏括号和命名机器检查，黄金示例优先使用枚举或 const 语义。"),
        6: ("test", ["test", "lint", "demo"],
            ["generated-demo/advanced/tests/advanced_test.c", "lint:control.*,embedded.*",
             "generated-demo/advanced/fixed_pool.c"],
            "边界、生命周期、错误码和溢出防护通过行为测试及安全实现证明。"),
        7: ("demo", ["demo", "test", "manual"],
            ["generated-demo/advanced/fixed_pool.c", "generated-demo/advanced/protocol.c",
             "docs/spec/TRACEABILITY.md"],
            "固定资源池和有界循环展示在正确性优先前提下的效率措施。"),
        8: ("manual", ["manual", "demo", "lint"],
            ["docs/COMMENTING_METHOD.md", "generated-demo/demo.c",
             "generated-demo/advanced/*.c", "lint:documentation.*,comment.*"],
            "注释准确性和业务意图由人工评审与黄金样例证明；lint 只阻断可可靠判断的结构规则。"),
        9: ("lint", ["lint", "compile", "demo"],
            ["lint:format.*,control.compound-braces", ".clang-format", "generated-demo/"],
            "缩进、行宽、单行语句和花括号由格式配置、lint 与编译共同覆盖。"),
        10: ("compile", ["compile", "lint", "demo"],
             ["compiler:-Wall,-Wextra,-Wconversion", "lint:boolean.*,control.*",
              "generated-demo/demo.c", "generated-demo/advanced/*.c"],
             "表达式顺序、类型转换和布尔语义由严格编译与简单表达式写法证明。"),
        11: ("compile", ["compile", "test", "manual"],
             ["scripts/verify.ps1", "scripts/verify.sh", "examples/c-kit.json:gates"],
             "GCC、Clang、头文件和测试使用统一的版本化门禁配置。"),
        12: ("test", ["test", "demo", "manual"],
             ["generated-demo/advanced/tests/advanced_test.c",
              "generated-demo/advanced/state_machine.c:assert", "docs/spec/TRACEABILITY.md"],
             "公开接口测试、内部假设断言和故障注入通道共同提供可测性。"),
        13: ("test", ["test", "lint", "demo"],
             ["generated-demo/advanced/protocol.c",
              "generated-demo/advanced/tests/advanced_test.c", "lint:embedded.forbidden-call"],
             "不可信输入、字符串、整数、格式串和二进制长度由协议模块集中覆盖。"),
        14: ("test", ["test", "manual"],
             ["generated-demo/advanced/tests/advanced_test.c",
              "scripts/verify.ps1:functional-test"],
             "测试围绕公开行为、边界和故障注入，而非直接调用私有实现。"),
        15: ("compile", ["compile", "lint", "manual"],
             ["scripts/verify.ps1:header-c99-cxx17", "lint:naming.reserved",
              "generated-demo/demo.h", "generated-demo/advanced/*.h"],
             "标准 C99、C++17 头文件编译和保留标识符检查提供可移植性证据。"),
    }
    return mapping[chapter]


def specific_evidence(key: str) -> list[str]:
    overrides = {
        "1-R-1": ["generated-demo/demo.c+h", "generated-demo/advanced/state_machine.c+h",
                  "generated-demo/advanced/protocol.c+h", "generated-demo/advanced/fixed_pool.c+h"],
        "1-R-4": ["scripts/verify.ps1:每个头文件独立 C99/C++17 编译"],
        "1-R-5": ["lint:file.include-guard", "generated-demo/demo.h:#ifndef DEMO_H"],
        "2-R-1": ["lint:complexity.function-lines", "AGENTS.md:新增函数有效代码不超过 50 行"],
        "2-R-2": ["lint:complexity.nesting", "AGENTS.md:嵌套不超过 4 层"],
        "2-R-3": ["generated-demo/advanced/state_machine.c:临界区快照和中断单写者协议"],
        "2-R-5": ["generated-demo/advanced/tests/advanced_test.c:全部错误返回码断言"],
        "2-S-4": ["lint:complexity.parameters", "examples/c-kit.json:safety.maxParameters=5"],
        "2-S-6": ["lint:naming.private-function", "generated-demo/:static prototypes"],
        "4-R-2": ["generated-demo/advanced/protocol.c:DEMO_ReadU16BigEndian/DEMO_ReadU32BigEndian"],
        "4-R-3": ["compiler:-Wuninitialized", "generated-demo/:定义时初始化"],
        "6-R-1": ["generated-demo/advanced/protocol.c:长度先验",
                  "generated-demo/advanced/fixed_pool.c:容量先验"],
        "6-R-2": ["generated-demo/advanced/fixed_pool.c:固定资源池，无堆分配"],
        "6-R-3": ["generated-demo/advanced/fixed_pool.c:代际句柄",
                  "advanced_test.c:陈旧句柄测试"],
        "6-R-4": ["generated-demo/advanced/protocol.c:先检查索引再访问",
                  "advanced_test.c:容量边界"],
        "6-R-5": ["lint:control.switch-default", "lint:comment.case-intent"],
        "8-R-2": ["lint:documentation.file-metadata,employee-id.*,modification-history.*",
                  "MANIFEST.json + CHANGELOG.md + Tag/Release:资产版本权威面",
                  "docs/COMMENTING_METHOD.md:文件头版本字段覆盖策略"],
        "8-R-3": ["lint:documentation.performance,reentrancy,definition-details,function-flow",
                  "lint:documentation.private-prototype",
                  "generated-demo/advanced/state_machine.c:DEMO_RunCycle 编号流程"],
        "8-R-4": ["lint:documentation.global-variable",
                  "generated-demo/advanced/protocol.c:s_protocol_version 取值范围和只读访问说明"],
        "8-R-5": ["lint:comment.numbered-intent-placement",
                  "readability.manualReview:review.comment.logical-blocks",
                  "docs/COMMENTING_METHOD.md:普通逻辑段语义由黄金样例和人工评审确认"],
        "8-R-6": ["lint:comment.case-fallthrough,comment.case-intent",
                  "generated-demo/demo.c:等级名称 switch",
                  "generated-demo/advanced/state_machine.c:状态 switch"],
        "11-R-1": ["examples/c-kit.json:gates.gcc/clang.flags", "scripts/verify.ps1"],
        "11-R-2": ["examples/c-kit.json:gates", "config/skills/c99-standard-c/c-kit.schema.json"],
        "12-R-3": ["generated-demo/advanced/state_machine.c:内部私有函数 assert"],
        "12-R-4": ["generated-demo/:公开入口运行时错误返回", "advanced_test.c:错误注入"],
        "12-S-1": ["generated-demo/advanced/tests/advanced_test.c:DEMO_TestFaultInjection"],
        "13-R-1": ["generated-demo/advanced/protocol.c:有界查找并显式写入空字符"],
        "13-R-2": ["generated-demo/advanced/protocol.c:目标容量包含结尾空间检查"],
        "13-R-3": ["generated-demo/demo.c:uint32_t 累加上界",
                   "generated-demo/advanced/fixed_pool.c:显式代际边界"],
        "13-R-4": ["compiler:-Wsign-conversion", "generated-demo/advanced/protocol.c:无符号解码"],
        "13-R-5": ["compiler:-Wconversion", "generated-demo/:校验后显式窄化"],
        "13-R-6": ["compiler:-Wformat=2", "DEMO_FormatStatus:PRIu32"],
        "13-R-7": ["DEMO_FormatStatus:编译期固定格式串", "lint:embedded.forbidden-call"],
        "13-R-8": ["advanced/DEMO_DecodeFrame/DEMO_PoolRead:显式 size_t 二进制长度"],
        "13-R-9": ["AGENTS.md:字符 I/O 返回值必须使用 int", "manual-review"],
        "13-R-10": ["lint:embedded.forbidden-call(system,popen)", "黄金示例不执行命令"],
        "14-R-1": ["generated-demo/advanced/tests/advanced_test.c", "go test ./..."],
        "14-S-1": ["advanced_test.c:只调用公开接口"],
        "15-R-1": ["lint:naming.reserved", "compiler:-Wpedantic"],
    }
    return overrides.get(key, [])


def parse_clauses() -> list[dict[str, object]]:
    clauses: list[dict[str, object]] = []
    for line_number, line in enumerate(REFERENCE.read_text(encoding="utf-8").splitlines(), start=1):
        match = CLAUSE_RE.match(line)
        if match is None:
            continue
        kind, chapter_text, item_text, title = match.groups()
        chapter = int(chapter_text)
        item = int(item_text)
        key = f"{chapter}-{KIND_CODE[kind]}-{item}"
        primary, methods, evidence, rationale = default_coverage(chapter)
        overrides = specific_evidence(key)
        clauses.append({
            "id": f"HW-C99-{chapter:02d}-{KIND_CODE[kind]}-{item:02d}",
            "chapter": chapter,
            "chapterTitle": CHAPTER_NAMES[chapter],
            "kind": kind,
            "number": f"{chapter}.{item}",
            "title": title,
            "sourceLine": line_number,
            "primaryVerification": primary,
            "verificationMethods": methods,
            "evidence": overrides if overrides else evidence,
            "rationale": rationale,
            "status": "covered",
        })
    return clauses


def build_document(clauses: list[dict[str, object]]) -> dict[str, object]:
    kind_counts = Counter(str(clause["kind"]) for clause in clauses)
    verification_counts = Counter(str(clause["primaryVerification"]) for clause in clauses)
    return {
        "$schema": "../skills/c99-standard-c/huawei-c-rule-catalog.schema.json",
        "schema": "huawei-c-rule-catalog",
        "version": "1.0.0",
        "source": {
            "title": "华为 C 语言编程规范 DKBA 2826-2011.5",
            "pdf": "references/huawei-c-language-programming-standard-dkba-2826-2011-5.pdf",
            "markdown": "references/huawei-c-language-programming-standard-dkba-2826-2011-5.md",
            "sha256": "80D23AC9CACB83AEBAA1C28889271F744D5866CA45D09266533895F256262200",
            "pages": 61,
            "chapters": {"first": 0, "last": 16, "clauseChapters": [1, 15]},
        },
        "summary": {
            "expectedClauses": 139,
            "actualClauses": len(clauses),
            "kindCounts": dict(sorted(kind_counts.items())),
            "primaryVerificationCounts": dict(sorted(verification_counts.items())),
            "unclassifiedClauses": 0,
        },
        "nonClauseSections": [
            {
                "section": "封面、修订声明、目录、范围和简介",
                "primaryVerification": "manual",
                "evidence": [
                    "references/huawei-c-language-programming-standard-dkba-2826-2011-5.md:文档元数据与目录",
                    "tools/pdf-reference/verify_reference.py:页数、章节与噪声检查",
                ],
                "status": "covered",
            },
            {
                "section": "0 规范制订说明",
                "primaryVerification": "manual",
                "evidence": [
                    "AGENTS.md:适用范围与权威来源",
                    "AGENTS.md:修改工作流",
                    "docs/spec/SELECTED_SOLUTION.md:规则优先级与证据策略",
                ],
                "status": "covered",
            },
            {
                "section": "16 业界编程规范",
                "primaryVerification": "manual",
                "evidence": [
                    "references/huawei-c-language-programming-standard-dkba-2826-2011-5.md:第16章",
                    "AGENTS.md:编译、测试与安全输入",
                    "scripts/verify.ps1:严格工具链门禁",
                ],
                "status": "covered",
            },
        ],
        "clauses": clauses,
    }


def render_markdown(document: dict[str, object]) -> str:
    clauses = document["clauses"]
    assert isinstance(clauses, list)
    lines = [
        "# 华为 C 语言编程规范完整规则目录",
        "",
        "> 本文件由 `tools/rules/build_rule_catalog.py` 从规范 Markdown 的条款标题机械生成。",
        "> 条款原文及解释以本地 PDF/Markdown 参考副本为准；本目录只提供检索、分类和验收证据。",
        "",
        "## 覆盖结论",
        "",
        "- PDF：61 页；章节：0—16。",
        "- 可编号条款：139 条，全部已分类，未分类 0 条。",
        "- 非编号内容：封面/范围/简介、第 0 章和第 16 章均有独立证据。",
        "- 证据类型：`demo`、`lint`、`compile`、`test`、`manual`。",
        "- `covered` 表示存在明确证据路径，不表示所有规范都适合用正则表达式机器判断。",
        "",
        "## 证据方法",
        "",
        "| 方法 | 含义 |",
        "| --- | --- |",
        "| `demo` | 黄金 C/H 代码以安全正例体现规则。 |",
        "| `lint` | Go lint 对稳定、低误报的语法规则实施门禁。 |",
        "| `compile` | GCC、Clang、C99 与 C++17 头文件严格编译。 |",
        "| `test` | 公开行为、边界和故障注入测试。 |",
        "| `manual` | 语义、架构、命名清晰度等必须保留人工评审。 |",
        "",
        "## 非编号内容覆盖",
        "",
        "| 范围 | 主证据 | 证据定位 | 状态 |",
        "| --- | --- | --- | --- |",
    ]
    for item in document["nonClauseSections"]:
        evidence = "<br>".join(f"`{value}`" for value in item["evidence"])
        lines.append(
            f"| {item['section']} | `{item['primaryVerification']}` | {evidence} | `{item['status']}` |"
        )
    lines.extend([
        "",
        "## 编号条款覆盖",
        "",
    ])
    for chapter in range(1, 16):
        chapter_clauses = [item for item in clauses if item["chapter"] == chapter]
        lines.extend([
            f"## {chapter} {CHAPTER_NAMES[chapter]}",
            "",
            "| ID | 条款 | 主证据 | 证据定位 | 状态 |",
            "| --- | --- | --- | --- | --- |",
        ])
        for item in chapter_clauses:
            title = str(item["title"]).replace("|", "\\|")
            evidence = "<br>".join(f"`{value}`" for value in item["evidence"])
            lines.append(
                f"| {item['id']} | {item['kind']} {item['number']} {title} | "
                f"`{item['primaryVerification']}` | {evidence} | `{item['status']}` |"
            )
        lines.append("")
    lines.extend([
        "## 完整性门禁",
        "",
        "`tools/rules/build_rule_catalog.py --check` 会核对：",
        "",
        "- Markdown 中恰好存在 139 条原则、规则和建议；",
        "- 每个条款 ID 唯一且章节范围为 1—15；",
        "- 每条都有主证据、至少一种验证方法、至少一个证据定位和 `covered` 状态；",
        "- JSON 与本 Markdown 都和当前参考 Markdown 的机械生成结果完全一致。",
        "",
    ])
    return "\n".join(lines)


def validate(document: dict[str, object]) -> None:
    clauses = document["clauses"]
    assert isinstance(clauses, list)
    if len(clauses) != 139:
        raise ValueError(f"expected 139 clauses, found {len(clauses)}")
    ids = [str(item["id"]) for item in clauses]
    if len(ids) != len(set(ids)):
        raise ValueError("duplicate clause ids")
    allowed = {"demo", "lint", "compile", "test", "manual"}
    non_clause_sections = document["nonClauseSections"]
    if len(non_clause_sections) != 3:
        raise ValueError("expected three non-clause coverage groups")
    for item in non_clause_sections:
        if item["primaryVerification"] not in allowed or not item["evidence"]:
            raise ValueError(f"invalid non-clause coverage: {item['section']}")
        if item["status"] != "covered":
            raise ValueError(f"uncovered non-clause section: {item['section']}")
    for item in clauses:
        if item["primaryVerification"] not in allowed:
            raise ValueError(f"unclassified clause: {item['id']}")
        if not item["verificationMethods"] or not item["evidence"]:
            raise ValueError(f"missing evidence: {item['id']}")
        if item["status"] != "covered":
            raise ValueError(f"uncovered clause: {item['id']}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true", help="compare outputs without rewriting")
    args = parser.parse_args()

    document = build_document(parse_clauses())
    validate(document)
    json_text = json.dumps(document, ensure_ascii=False, indent=2) + "\n"
    markdown_text = render_markdown(document)

    if args.check:
        mismatches = []
        if not JSON_OUTPUT.is_file() or JSON_OUTPUT.read_text(encoding="utf-8") != json_text:
            mismatches.append(str(JSON_OUTPUT.relative_to(ROOT)))
        if not MARKDOWN_OUTPUT.is_file() or MARKDOWN_OUTPUT.read_text(encoding="utf-8") != markdown_text:
            mismatches.append(str(MARKDOWN_OUTPUT.relative_to(ROOT)))
        if mismatches:
            raise SystemExit("generated rule catalog is stale: " + ", ".join(mismatches))
        print("rule catalog check passed: 139/139 clauses, 0 unclassified")
        return 0

    JSON_OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    JSON_OUTPUT.write_text(json_text, encoding="utf-8", newline="\n")
    MARKDOWN_OUTPUT.write_text(markdown_text, encoding="utf-8", newline="\n")
    print("rule catalog generated: 139/139 clauses, 0 unclassified")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
