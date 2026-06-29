from __future__ import annotations

import json
from pathlib import Path
from typing import Any

DEFAULT_CONFIG: dict[str, Any] = {
    "schema": "agent-patch-kit.config.v2",
    "default_path": ".",
    "default_globs": [
        "*.md", "*.txt", "*.c", "*.h", "*.cpp", "*.hpp", "*.cc", "*.hh",
        "*.py", "*.ps1", "*.psm1", "*.yml", "*.yaml", "*.json", "*.toml",
        "*.ini", "*.cfg", "*.cmake", "*.xml", "*.html", "*.css", "*.js", "*.ts"
    ],
    "exclude_dirs": [
        ".git", ".agentpatch", "node_modules", "dist", "build", "coverage", ".venv", "__pycache__",
        ".pytest_cache", ".mypy_cache", ".ruff_cache"
    ],
    "transactions": {
        "auto_begin_for_apply": True,
        "root": ".agentpatch/transactions",
        "keep": 30
    },
    "verify": {
        "run_git_diff_check": True,
        "task_verify": False,
        "markdown_links": False
    },
    "links": {
        "mode": "offline",
        "include_fragments": "full",
        "inputs": ["**/*.md"],
        "config": "lychee.toml"
    },
    "deploy": {
        "default_agent": "both",
        "user_agents_root": "%USERPROFILE%/.agents/skills",
        "user_codex_root": "%USERPROFILE%/.codex/skills"
    },
    "state": {
        "default_enabled": True,
        "respect_system": True,
        "respect_user": True,
        "respect_project": True
    },
    "agent_brief": {
        "default_format": "md",
        "developer_only": True
    }
}


def deep_merge(a: dict[str, Any], b: dict[str, Any]) -> dict[str, Any]:
    out = dict(a)
    for k, v in b.items():
        if isinstance(v, dict) and isinstance(out.get(k), dict):
            out[k] = deep_merge(out[k], v)
        else:
            out[k] = v
    return out


def find_config(start: Path) -> Path | None:
    cur = start.resolve()
    for parent in [cur, *cur.parents]:
        p = parent / ".agentpatch.json"
        if p.exists():
            return p
    return None


def load_config(start: Path) -> dict[str, Any]:
    cfg = DEFAULT_CONFIG.copy()
    p = find_config(start)
    if not p:
        return cfg
    try:
        loaded = json.loads(p.read_text(encoding="utf-8"))
        return deep_merge(cfg, loaded)
    except Exception:
        return cfg


def write_default_config(path: Path) -> None:
    path.write_text(json.dumps(DEFAULT_CONFIG, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
