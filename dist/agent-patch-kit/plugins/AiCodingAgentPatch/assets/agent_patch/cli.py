from __future__ import annotations

import argparse
import difflib
import fnmatch
import json
import os
import re
import shutil
import sys
import tarfile
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable, Any

from . import __version__
from .config import load_config, write_default_config, DEFAULT_CONFIG
from .runner import run

TEXT_EXTENSIONS = {
    ".md", ".txt", ".c", ".h", ".cpp", ".hpp", ".cc", ".hh", ".py", ".ps1", ".psm1",
    ".yml", ".yaml", ".json", ".toml", ".ini", ".cfg", ".cmake", ".xml", ".html", ".css", ".js", ".ts",
}

REQUIRED_TOOLS = ["git", "python", "rg"]
OPTIONAL_TOOLS = ["task", "sd", "ast-grep", "lychee"]
SKILL_NAME = "aicoding-agent-patch-kit"


def print_json(obj: object) -> None:
    print(json.dumps(obj, ensure_ascii=False, indent=2))


def now_id(prefix: str = "tx") -> str:
    return f"{prefix}-{datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')}-{os.getpid()}"


def tool_version(tool: str) -> str | None:
    exe = shutil.which(tool)
    if not exe:
        return None
    for args in ([tool, "--version"], [tool, "version"]):
        try:
            res = run(args)
            txt = (res.stdout or res.stderr).strip().splitlines()
            if txt:
                return txt[0]
        except Exception:
            pass
    return exe


def repo_root(path: Path) -> Path:
    res = run(["git", "rev-parse", "--show-toplevel"], cwd=path)
    if res.code == 0 and res.stdout.strip():
        return Path(res.stdout.strip()).resolve()
    return path.resolve()


def is_git_repo(path: Path) -> bool:
    return run(["git", "rev-parse", "--is-inside-work-tree"], cwd=path).code == 0


def cmd_doctor(args: argparse.Namespace) -> int:
    rows = []
    ok = True
    for name in REQUIRED_TOOLS:
        path = shutil.which(name)
        found = path is not None
        ok = ok and found
        rows.append({"name": name, "required": True, "found": found, "path": path, "version": tool_version(name) if found else None})
    for name in OPTIONAL_TOOLS:
        path = shutil.which(name)
        found = path is not None
        rows.append({"name": name, "required": False, "found": found, "path": path, "version": tool_version(name) if found else None})
    result = {"ok": ok, "version": __version__, "tools": rows}
    if args.json:
        print_json(result)
    else:
        print(f"Agent Patch Kit apatch {__version__}")
        print(f"environment: {'OK' if ok else 'MISSING REQUIRED TOOLS'}")
        for r in rows:
            marker = "required" if r["required"] else "optional"
            status = "OK" if r["found"] else "MISSING"
            print(f"{status:8} {marker:8} {r['name']:<10} {r['version'] or ''}")
        if not ok:
            print("\nInstall missing Windows tools with scripts/install-agent-patch-kit.ps1 -InstallMissing")
    return 0 if ok else 2


def cmd_init(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    p = root / ".agentpatch.json"
    if p.exists() and not args.force:
        print(f"exists: {p}")
    else:
        write_default_config(p)
        print(f"created: {p}")
    if args.write_lychee:
        ly = root / "lychee.toml"
        if not ly.exists() or args.force:
            ly.write_text(DEFAULT_LYCHEE_TOML, encoding="utf-8")
            print(f"created: {ly}")
    return 0


def cmd_status(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    res = run(["git", "status", "--short"], cwd=root)
    lines = [ln for ln in res.stdout.splitlines() if ln.strip()]
    out = {"clean": len(lines) == 0, "changed": len(lines), "lines": lines}
    if args.json:
        print_json(out)
    else:
        if not lines:
            print("git status: clean")
        else:
            print(f"git status: {len(lines)} changed/untracked entries")
            for line in lines[: args.limit]:
                print(line)
            if len(lines) > args.limit:
                print(f"... {len(lines) - args.limit} more")
    return 0 if res.code == 0 else res.code


def should_skip(path: Path, exclude_dirs: list[str]) -> bool:
    parts = set(path.parts)
    return any(x in parts for x in exclude_dirs)


def is_text_candidate(path: Path) -> bool:
    if path.suffix.lower() in TEXT_EXTENSIONS:
        return True
    return path.name in {"Makefile", "Dockerfile", "Taskfile.yml", "AGENTS.md", "SKILL.md", "CHANGELOG", "README"}


def iter_files(root: Path, globs: list[str], exclude_dirs: list[str]) -> Iterable[Path]:
    root = root.resolve()
    for p in root.rglob("*"):
        if not p.is_file():
            continue
        try:
            relp = p.relative_to(root)
        except ValueError:
            continue
        if should_skip(relp, exclude_dirs):
            continue
        rel = relp.as_posix()
        if globs and not any(fnmatch.fnmatch(rel, g) or fnmatch.fnmatch(p.name, g) for g in globs):
            continue
        if is_text_candidate(p):
            yield p


def read_text(path: Path) -> str | None:
    try:
        return path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        try:
            return path.read_text(encoding="utf-8-sig")
        except Exception:
            return None
    except Exception:
        return None


def rg_scan(pattern: str, root: Path, globs: list[str], fixed: bool, max_count: int | None) -> tuple[int, str, str]:
    cmd = ["rg", "-n"]
    if fixed:
        cmd.append("-F")
    for g in globs:
        cmd.extend(["-g", g])
    if max_count:
        cmd.extend(["--max-count", str(max_count)])
    cmd.extend([pattern, str(root)])
    res = run(cmd)
    return res.code, res.stdout, res.stderr


def python_scan(pattern: str, root: Path, globs: list[str], fixed: bool, max_count: int | None) -> list[dict[str, object]]:
    cfg = load_config(root)
    exclude = cfg.get("exclude_dirs", [])
    out = []
    regex = None if fixed else re.compile(pattern)
    for p in iter_files(root, globs, exclude):
        text = read_text(p)
        if text is None:
            continue
        count_for_file = 0
        for i, line in enumerate(text.splitlines(), start=1):
            matched = pattern in line if fixed else bool(regex.search(line))
            if matched:
                out.append({"path": str(p.relative_to(root)), "line": i, "text": line})
                count_for_file += 1
                if max_count and count_for_file >= max_count:
                    break
    return out


def cmd_scan(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    globs = args.glob or load_config(root).get("default_globs", [])
    fixed = bool(args.fixed)
    if shutil.which("rg"):
        code, stdout, stderr = rg_scan(args.pattern, root, globs, fixed, args.max_count)
        lines = [ln for ln in stdout.splitlines() if ln.strip()]
        if args.json:
            print_json({"tool": "rg", "matches": len(lines), "lines": lines, "code": code})
        else:
            print(stdout, end="")
            if stderr:
                print(stderr, file=sys.stderr, end="")
            print(f"matches: {len(lines)}", file=sys.stderr)
        return 0 if code in (0, 1) else code
    matches = python_scan(args.pattern, root, globs, fixed, args.max_count)
    if args.json:
        print_json({"tool": "python", "matches": len(matches), "lines": matches})
    else:
        for m in matches:
            print(f"{m['path']}:{m['line']}:{m['text']}")
        print(f"matches: {len(matches)}", file=sys.stderr)
    return 0


def replace_text(text: str, old: str, new: str, fixed: bool) -> tuple[str, int]:
    if fixed:
        return text.replace(old, new), text.count(old)
    pattern = re.compile(old)
    return pattern.subn(new, text)


def unified_diff(rel: str, old_text: str, new_text: str) -> str:
    return "".join(difflib.unified_diff(
        old_text.splitlines(True),
        new_text.splitlines(True),
        fromfile=f"a/{rel}",
        tofile=f"b/{rel}",
    ))


def affected_files(root: Path, old: str, globs: list[str], fixed: bool) -> list[Path]:
    cfg = load_config(root)
    exclude = cfg.get("exclude_dirs", [])
    hits = []
    regex = None if fixed else re.compile(old)
    for p in iter_files(root, globs, exclude):
        text = read_text(p)
        if text is None:
            continue
        has = old in text if fixed else bool(regex.search(text))
        if has:
            hits.append(p)
    return hits


def tx_root(root: Path) -> Path:
    cfg = load_config(root)
    return root / cfg.get("transactions", {}).get("root", ".agentpatch/transactions")


def porcelain_paths(root: Path, z: bool = True) -> tuple[list[str], list[str]]:
    res = run(["git", "status", "--porcelain=v1", "-z"], cwd=root)
    if res.code != 0:
        return [], []
    data = res.stdout
    parts = data.split("\0")
    tracked: list[str] = []
    untracked: list[str] = []
    i = 0
    while i < len(parts):
        item = parts[i]
        if not item:
            i += 1
            continue
        status = item[:2]
        path = item[3:]
        if status == "??":
            untracked.append(path)
        else:
            # rename/copy porcelain has an extra path item; keep both conservatively
            tracked.append(path)
            if status[0] in "RC" or status[1] in "RC":
                i += 1
                if i < len(parts) and parts[i]:
                    tracked.append(parts[i])
        i += 1
    return sorted(set(tracked)), sorted(set(untracked))


def copy_if_exists(src: Path, dest: Path) -> None:
    if src.exists() and src.is_file():
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dest)


def create_transaction(root: Path, reason: str = "manual") -> str:
    root = repo_root(root) if is_git_repo(root) else root.resolve()
    tid = now_id("tx")
    tdir = tx_root(root) / tid
    tdir.mkdir(parents=True, exist_ok=True)
    git = is_git_repo(root)
    meta: dict[str, Any] = {
        "schema": "agent-patch-kit.transaction.v1",
        "id": tid,
        "created_at": datetime.now(timezone.utc).isoformat(),
        "root": str(root),
        "reason": reason,
        "git": git,
    }
    if git:
        head = run(["git", "rev-parse", "HEAD"], cwd=root)
        meta["head"] = head.stdout.strip() if head.code == 0 else None
        status = run(["git", "status", "--short"], cwd=root)
        (tdir / "status-before.txt").write_text(status.stdout, encoding="utf-8")
        diff = run(["git", "diff", "--binary"], cwd=root)
        (tdir / "worktree.diff").write_text(diff.stdout, encoding="utf-8")
        staged = run(["git", "diff", "--cached", "--binary"], cwd=root)
        (tdir / "staged.diff").write_text(staged.stdout, encoding="utf-8")
        tracked, untracked = porcelain_paths(root)
        meta["tracked_changed_paths"] = tracked
        meta["untracked_paths"] = untracked
        for rel in untracked:
            copy_if_exists(root / rel, tdir / "untracked" / rel)
    else:
        cfg = load_config(root)
        files = list(iter_files(root, cfg.get("default_globs", []), cfg.get("exclude_dirs", [])))
        with tarfile.open(tdir / "snapshot.tar.gz", "w:gz") as tf:
            for f in files:
                tf.add(f, arcname=f.relative_to(root).as_posix())
        meta["snapshot_file_count"] = len(files)
    (tdir / "meta.json").write_text(json.dumps(meta, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return tid


def read_transaction(root: Path, tid: str) -> tuple[Path, dict[str, Any]]:
    tdir = tx_root(repo_root(root) if is_git_repo(root) else root.resolve()) / tid
    meta = json.loads((tdir / "meta.json").read_text(encoding="utf-8"))
    return tdir, meta


def rollback_transaction(root: Path, tid: str, clean_created: bool) -> None:
    root = repo_root(root) if is_git_repo(root) else root.resolve()
    tdir, meta = read_transaction(root, tid)
    if meta.get("git"):
        # Reset tracked state, then restore the exact dirty state captured at begin.
        run(["git", "reset", "--hard", str(meta.get("head") or "HEAD")], cwd=root, check=True)
        if clean_created:
            current_untracked = porcelain_paths(root)[1]
            saved_untracked = set(meta.get("untracked_paths", []))
            for rel in current_untracked:
                if rel not in saved_untracked:
                    p = root / rel
                    if p.is_file():
                        p.unlink()
                    elif p.is_dir():
                        shutil.rmtree(p)
        worktree_diff = tdir / "worktree.diff"
        if worktree_diff.exists() and worktree_diff.stat().st_size > 0:
            run(["git", "apply", "--binary", str(worktree_diff)], cwd=root, check=True)
        staged_diff = tdir / "staged.diff"
        if staged_diff.exists() and staged_diff.stat().st_size > 0:
            run(["git", "apply", "--cached", "--binary", str(staged_diff)], cwd=root, check=True)
        for rel in meta.get("untracked_paths", []):
            src = tdir / "untracked" / rel
            dst = root / rel
            if src.exists():
                dst.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(src, dst)
    else:
        snap = tdir / "snapshot.tar.gz"
        if snap.exists():
            with tarfile.open(snap, "r:gz") as tf:
                tf.extractall(root)


def cmd_tx(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    if args.txcmd == "begin":
        tid = create_transaction(root, args.name or "manual")
        if args.json:
            print_json({"id": tid, "path": str(tx_root(repo_root(root) if is_git_repo(root) else root) / tid)})
        else:
            print(f"transaction: {tid}")
        return 0
    if args.txcmd == "list":
        base = tx_root(repo_root(root) if is_git_repo(root) else root)
        items = []
        if base.exists():
            for p in sorted(base.iterdir(), reverse=True):
                m = p / "meta.json"
                if m.exists():
                    try:
                        items.append(json.loads(m.read_text(encoding="utf-8")))
                    except Exception:
                        pass
        if args.json:
            print_json({"transactions": items})
        else:
            for m in items[: args.limit]:
                print(f"{m.get('id')}  {m.get('created_at')}  {m.get('reason')}")
            if not items:
                print("no transactions")
        return 0
    if args.txcmd == "rollback":
        tdir, meta = read_transaction(root, args.id)
        if args.preview:
            print(f"transaction: {meta.get('id')}")
            print(f"created_at: {meta.get('created_at')}")
            print(f"reason: {meta.get('reason')}")
            print(f"root: {meta.get('root')}")
            print(f"git: {meta.get('git')}")
            print("rollback preview only; use --apply --force to restore snapshot")
            if (tdir / "status-before.txt").exists():
                print("\nstatus at begin:")
                print((tdir / "status-before.txt").read_text(encoding="utf-8"))
            return 0
        if not (args.apply and args.force):
            print("rollback requires --apply --force", file=sys.stderr)
            return 2
        rollback_transaction(root, args.id, args.clean_created)
        print(f"rolled back: {args.id}")
        return 0
    print("unknown tx command", file=sys.stderr)
    return 2


def cmd_replace(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    globs = args.glob or load_config(root).get("default_globs", [])
    fixed = bool(args.fixed or not args.regex)
    if args.apply:
        code = ensure_enabled(root, "replace apply", args.allow_disabled)
        if code:
            return code
    if args.preview == args.apply:
        print("choose exactly one: --preview or --apply", file=sys.stderr)
        return 2
    tx_id = None
    if args.apply and not args.no_tx and load_config(root).get("transactions", {}).get("auto_begin_for_apply", True):
        tx_id = create_transaction(root, "replace")
    files = affected_files(root, args.old, globs, fixed)
    total = 0
    changed = []
    for p in files:
        old_text = read_text(p)
        if old_text is None:
            continue
        new_text, count = replace_text(old_text, args.old, args.new, fixed)
        if count == 0 or new_text == old_text:
            continue
        rel = p.relative_to(root).as_posix()
        total += count
        changed.append({"path": rel, "replacements": count})
        if args.preview:
            print(unified_diff(rel, old_text, new_text), end="")
        else:
            p.write_text(new_text, encoding="utf-8", newline="")
    if args.json:
        print_json({"mode": "preview" if args.preview else "apply", "transaction": tx_id, "files": len(changed), "replacements": total, "changed": changed})
    else:
        print(f"\nmode: {'preview' if args.preview else 'apply'}")
        if tx_id:
            print(f"transaction: {tx_id}")
        print(f"files: {len(changed)}")
        print(f"replacements: {total}")
    return 0


def find_ast_grep() -> str | None:
    exe = shutil.which("ast-grep")
    if exe:
        return exe
    sg = shutil.which("sg")
    if sg:
        res = run([sg, "--version"])
        txt = (res.stdout + res.stderr).lower()
        if "ast-grep" in txt:
            return sg
    return None


def cmd_ast(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    if args.apply:
        code = ensure_enabled(root, "ast apply", args.allow_disabled)
        if code:
            return code
    exe = find_ast_grep()
    if not exe:
        print("ast-grep not found. Install with: npm install --global @ast-grep/cli  OR  pip install ast-grep-cli  OR  scoop install main/ast-grep", file=sys.stderr)
        return 2
    if args.apply and not args.rewrite:
        print("--apply requires --rewrite", file=sys.stderr)
        return 2
    if args.preview and args.apply:
        print("choose only one: --preview or --apply", file=sys.stderr)
        return 2
    cmd = [exe, "--pattern", args.pattern]
    if args.lang:
        cmd.extend(["--lang", args.lang])
    if args.rewrite:
        cmd.extend(["--rewrite", args.rewrite])
    for g in args.glob or []:
        cmd.extend(["--globs", g])
    if args.context:
        cmd.extend(["--context", str(args.context)])
    if args.json:
        cmd.extend(["--json", "compact"])
    if args.apply:
        cmd.append("--update-all")
    # When previewing rewrite, ast-grep prints a rewrite session/output without updating files.
    cmd.append(str(root))
    tx_id = None
    if args.apply and not args.no_tx and load_config(root).get("transactions", {}).get("auto_begin_for_apply", True):
        tx_id = create_transaction(root, "ast-grep")
    res = run(cmd, cwd=root)
    if args.json:
        # ast-grep may already emit JSON; wrap only metadata if requested with passthrough disabled.
        if res.stdout.strip().startswith("{") or res.stdout.strip().startswith("["):
            print(res.stdout, end="")
        else:
            print_json({"ok": res.code == 0, "transaction": tx_id, "cmd": cmd, "stdout": res.stdout, "stderr": res.stderr})
    else:
        if tx_id:
            print(f"transaction: {tx_id}")
        print(res.stdout, end="")
        if res.stderr:
            print(res.stderr, file=sys.stderr, end="")
    return 0 if res.code in (0, 1) else res.code


def count_matches(root: Path, pattern: str, globs: list[str], fixed: bool) -> int:
    cfg = load_config(root)
    exclude = cfg.get("exclude_dirs", [])
    total = 0
    regex = None if fixed else re.compile(pattern)
    for p in iter_files(root, globs, exclude):
        text = read_text(p)
        if text is None:
            continue
        if fixed:
            total += text.count(pattern)
        else:
            total += len(regex.findall(text))
    return total


def cmd_links(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    exe = shutil.which("lychee")
    if not exe:
        print("lychee not found. Install with winget install lycheeverse.lychee or use scripts/install-agent-patch-kit.ps1 -InstallMissing", file=sys.stderr)
        return 2
    cfg = load_config(root).get("links", {})
    mode = args.mode or cfg.get("mode", "offline")
    include_fragments = args.include_fragments or cfg.get("include_fragments", "full")
    inputs = args.input or cfg.get("inputs", ["**/*.md"])
    lychee_cfg = args.config or cfg.get("config", "lychee.toml")
    cmd = [exe]
    if mode == "offline":
        cmd.append("--offline")
    if include_fragments != "none":
        cmd.append(f"--include-fragments={include_fragments}")
    if args.no_progress:
        cmd.append("--no-progress")
    config_path = root / lychee_cfg
    if config_path.exists():
        cmd.extend(["--config", str(config_path)])
    cmd.extend(inputs)
    res = run(cmd, cwd=root)
    if args.json:
        print_json({"ok": res.code == 0, "cmd": cmd, "stdout": res.stdout, "stderr": res.stderr})
    else:
        print(res.stdout, end="")
        if res.stderr:
            print(res.stderr, file=sys.stderr, end="")
    return res.code


def cmd_verify(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    globs = args.glob or load_config(root).get("default_globs", [])
    fixed = bool(args.fixed or not args.regex)
    checks = []
    diff = run(["git", "diff", "--check"], cwd=root)
    checks.append({"name": "git diff --check", "ok": diff.code == 0, "stdout": diff.stdout, "stderr": diff.stderr})
    old_count = None
    new_count = None
    if args.old is not None:
        old_count = count_matches(root, args.old, globs, fixed)
        checks.append({"name": "old pattern count", "ok": old_count == 0 if args.expect_old_zero else True, "count": old_count})
    if args.new is not None:
        new_count = count_matches(root, args.new, globs, fixed)
        checks.append({"name": "new pattern count", "ok": new_count > 0 if args.expect_new_nonzero else True, "count": new_count})
    if args.links:
        code = cmd_links(argparse.Namespace(path=str(root), mode=args.links_mode, include_fragments="full", input=["**/*.md"], config=None, no_progress=True, json=True))
        checks.append({"name": "markdown links", "ok": code == 0})
    if args.task:
        if shutil.which("task"):
            t = run(["task", "verify"], cwd=root)
            checks.append({"name": "task verify", "ok": t.code == 0, "stdout": t.stdout[-4000:], "stderr": t.stderr[-4000:]})
        else:
            checks.append({"name": "task verify", "ok": False, "stderr": "task not found"})
    ok = all(c.get("ok", False) for c in checks)
    if args.json:
        print_json({"ok": ok, "old_count": old_count, "new_count": new_count, "checks": checks})
    else:
        for c in checks:
            print(f"{'OK' if c.get('ok') else 'FAIL'} {c['name']}")
            if "count" in c:
                print(f"  count: {c['count']}")
            if c.get("stdout"):
                print(c["stdout"].rstrip())
            if c.get("stderr"):
                print(c["stderr"].rstrip(), file=sys.stderr)
    return 0 if ok else 1


def cmd_summary(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    stat = run(["git", "diff", "--stat"], cwd=root)
    names = run(["git", "diff", "--name-only"], cwd=root)
    status = run(["git", "status", "--short"], cwd=root)
    obj = {
        "changed_files": [x for x in names.stdout.splitlines() if x.strip()],
        "status_lines": [x for x in status.stdout.splitlines() if x.strip()],
        "stat": stat.stdout.strip(),
    }
    if args.json:
        print_json(obj)
    else:
        print("diff stat:")
        print(stat.stdout.rstrip() or "(no diff)")
        print("\nchanged files:")
        for f in obj["changed_files"]:
            print(f"- {f}")
        if not obj["changed_files"]:
            print("(none)")
    return 0


def package_root() -> Path:
    return Path(__file__).resolve().parents[1]


def copytree_merge(src: Path, dst: Path) -> None:
    if dst.exists():
        shutil.rmtree(dst)
    shutil.copytree(src, dst, ignore=shutil.ignore_patterns("__pycache__", "*.pyc", ".agentpatch", "dist"))


def skill_source_dir() -> Path:
    return package_root() / "skills" / SKILL_NAME


DEVELOPER_BRIEF_MD = """# Agent Patch Kit developer brief

This file is for AI agents and kit developers, not end users.

## Purpose
Agent Patch Kit standardizes safe repository edits. It prevents broad unpreviewed PowerShell regex edits by forcing a short CLI workflow:

1. `apatch state status`
2. `apatch status`
3. `apatch scan <target> --fixed`
4. `apatch replace ... --preview` or `apatch ast ... --preview`
5. `apatch replace ... --apply` or `apatch ast ... --apply`
6. `apatch verify ...`
7. `apatch summary`

## Tool map
- Literal search: `apatch scan --fixed`, internally prefers `rg -F`.
- Literal replacement: `apatch replace --fixed`, Python diff/apply with transaction snapshot.
- Structural code replacement: `apatch ast`, wrapper around `ast-grep` / `sg`.
- Markdown link check: `apatch links`, wrapper around `lychee`.
- Rollback: `apatch tx rollback <id> --apply --force`.
- Enable/disable: `apatch state enable|disable --scope system|user|project`.

## Safety rules
- Never apply when `apatch state status` reports disabled unless the user explicitly asks to override.
- Prefer `--fixed` for paths, URLs, Markdown links, version strings, and Windows paths.
- Regex is allowed only when fixed string mode cannot express the target.
- Apply commands create transaction snapshots by default.
- End every file-modification task with `apatch verify` and `apatch summary`.

## Scope semantics
Agent Patch Kit is enabled only when all enabled scopes allow it. A system disable blocks all users and projects. A user disable blocks the current user. A project disable blocks the current repository only. Missing state files default to enabled.
"""


def system_state_dir() -> Path:
    if os.name == "nt":
        return Path(os.environ.get("ProgramData", r"C:\ProgramData")) / "AgentPatchKit"
    return Path("/etc/agent-patch-kit")


def user_state_dir_path() -> Path:
    return Path.home() / ".agentpatch"


def project_state_dir_path(project: Path) -> Path:
    return project.resolve() / ".agentpatch"


def state_file_for(scope: str, project: Path | None = None) -> Path:
    if scope == "system":
        return system_state_dir() / "state.json"
    if scope == "user":
        return user_state_dir_path() / "state.json"
    if scope == "project":
        if project is None:
            raise ValueError("project path is required for project scope")
        root = repo_root(project) if is_git_repo(project) else project.resolve()
        return project_state_dir_path(root) / "state.json"
    raise ValueError(scope)


def default_state(scope: str) -> dict[str, Any]:
    return {
        "schema": "agent-patch-kit.state.v1",
        "scope": scope,
        "enabled": True,
        "updated_at": None,
        "updated_by": None,
        "reason": "default-enabled"
    }


def read_state(scope: str, project: Path | None = None) -> dict[str, Any]:
    p = state_file_for(scope, project)
    if not p.exists():
        st = default_state(scope)
        st["path"] = str(p)
        st["exists"] = False
        return st
    try:
        st = json.loads(p.read_text(encoding="utf-8"))
        if "enabled" not in st:
            st["enabled"] = True
        st["path"] = str(p)
        st["exists"] = True
        return st
    except Exception as e:
        st = default_state(scope)
        st["enabled"] = False
        st["reason"] = f"invalid-state-file: {e}"
        st["path"] = str(p)
        st["exists"] = True
        return st


def write_state(scope: str, enabled: bool, reason: str | None, project: Path | None = None) -> Path:
    p = state_file_for(scope, project)
    p.parent.mkdir(parents=True, exist_ok=True)
    st = {
        "schema": "agent-patch-kit.state.v1",
        "scope": scope,
        "enabled": enabled,
        "updated_at": datetime.now(timezone.utc).isoformat(),
        "updated_by": os.environ.get("USERNAME") or os.environ.get("USER") or "unknown",
        "reason": reason or ("enabled" if enabled else "disabled"),
    }
    p.write_text(json.dumps(st, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return p


def effective_state(project: Path) -> dict[str, Any]:
    root = repo_root(project) if is_git_repo(project) else project.resolve()
    scopes = [read_state("system"), read_state("user"), read_state("project", root)]
    blockers = [s for s in scopes if not bool(s.get("enabled", True))]
    return {
        "enabled": len(blockers) == 0,
        "root": str(root),
        "scopes": scopes,
        "blockers": blockers,
    }


def ensure_enabled(project: Path, action: str, allow_disabled: bool = False) -> int:
    st = effective_state(project)
    if st["enabled"] or allow_disabled:
        return 0
    print(f"Agent Patch Kit is disabled; refusing action: {action}", file=sys.stderr)
    for b in st["blockers"]:
        print(f"disabled scope={b.get('scope')} path={b.get('path')} reason={b.get('reason')}", file=sys.stderr)
    print("Use `apatch state enable --scope <system|user|project>` to re-enable, or pass --allow-disabled only when explicitly authorized.", file=sys.stderr)
    return 3


def cmd_brief(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    data = {
        "schema": "agent-patch-kit.agent-brief.v1",
        "version": __version__,
        "developer_only": True,
        "skill_name": SKILL_NAME,
        "effective_state": effective_state(root),
        "commands": {
            "state": "apatch state status",
            "status": "apatch status",
            "scan_fixed": "apatch scan <target> --fixed",
            "replace_preview": "apatch replace --old <old> --new <new> --fixed --preview",
            "replace_apply": "apatch replace --old <old> --new <new> --fixed --apply",
            "ast_preview": "apatch ast --lang <lang> --pattern <pattern> --rewrite <rewrite> --preview",
            "ast_apply": "apatch ast --lang <lang> --pattern <pattern> --rewrite <rewrite> --apply",
            "links": "apatch links --mode offline --include-fragments full",
            "verify": "apatch verify --old <old> --new <new> --fixed",
            "summary": "apatch summary",
            "rollback": "apatch tx rollback <transaction-id> --apply --force"
        },
        "rules": [
            "Check state before using the kit.",
            "Prefer --fixed for literals, paths, URLs, Markdown links, and Windows paths.",
            "Preview before apply.",
            "Apply commands create transactions by default.",
            "Run verify and summary after modifications."
        ],
    }
    if args.format == "json":
        print_json(data)
    else:
        print(DEVELOPER_BRIEF_MD)
        print("\n## Current effective state")
        print("enabled: " + str(data["effective_state"]["enabled"]).lower())
        for scope in data["effective_state"]["scopes"]:
            print(f"- {scope.get('scope')}: enabled={scope.get('enabled')} exists={scope.get('exists')} path={scope.get('path')}")
    return 0


def cmd_state(args: argparse.Namespace) -> int:
    root = Path(args.path).resolve()
    if args.statecmd == "status":
        st = effective_state(root)
        if args.json:
            print_json(st)
        else:
            print(f"effective: {'enabled' if st['enabled'] else 'disabled'}")
            print(f"root: {st['root']}")
            for s in st["scopes"]:
                print(f"{s.get('scope'):<7} enabled={str(s.get('enabled')).lower():<5} exists={str(s.get('exists')).lower():<5} path={s.get('path')} reason={s.get('reason')}")
        return 0 if st["enabled"] else 3
    if args.statecmd in {"enable", "disable"}:
        enabled = args.statecmd == "enable"
        target_project = root if args.scope == "project" else None
        try:
            p = write_state(args.scope, enabled, args.reason, target_project)
        except PermissionError:
            print(f"permission denied writing {args.scope} state; run as Administrator or choose --scope user/project", file=sys.stderr)
            return 13
        if args.json:
            print_json({"scope": args.scope, "enabled": enabled, "path": str(p)})
        else:
            print(f"{args.scope}: {'enabled' if enabled else 'disabled'}")
            print(f"state_file: {p}")
        return 0
    if args.statecmd == "where":
        obj = {
            "system": str(state_file_for("system")),
            "user": str(state_file_for("user")),
            "project": str(state_file_for("project", root)),
        }
        if args.json:
            print_json(obj)
        else:
            for k, v in obj.items():
                print(f"{k}: {v}")
        return 0
    print("unknown state command", file=sys.stderr)
    return 2


def user_root(kind: str) -> Path:
    home = Path.home()
    if kind == "codex":
        return home / ".codex" / "skills"
    if kind == "agents":
        return home / ".agents" / "skills"
    raise ValueError(kind)


def system_root(kind: str) -> Path:
    # System-level deployment is a managed staging path. Agent launchers can add it
    # to their discovery list; user/project deployment remains the safest default.
    base = system_state_dir()
    if kind == "codex":
        return base / "codex" / "skills"
    if kind == "agents":
        return base / "agents" / "skills"
    raise ValueError(kind)


def project_root(kind: str, project: Path) -> Path:
    if kind == "codex":
        return project / ".codex" / "skills"
    if kind == "agents":
        return project / ".agents" / "skills"
    raise ValueError(kind)


def deploy_skill_to(target_root: Path, dry_run: bool) -> dict[str, str]:
    src = skill_source_dir()
    dst = target_root / SKILL_NAME
    if not src.exists():
        raise RuntimeError(f"skill source not found: {src}")
    if not dry_run:
        target_root.mkdir(parents=True, exist_ok=True)
        copytree_merge(src, dst)
    return {"source": str(src), "target": str(dst)}


def cmd_deploy(args: argparse.Namespace) -> int:
    kinds = ["agents", "codex"] if args.agent == "both" else [args.agent]
    results = []
    if args.scope == "system":
        for k in kinds:
            results.append(deploy_skill_to(system_root(k), args.dry_run) | {"scope": "system", "agent": k})
    elif args.scope == "user":
        for k in kinds:
            results.append(deploy_skill_to(user_root(k), args.dry_run) | {"scope": "user", "agent": k})
    elif args.scope == "project":
        if not args.project:
            print("--project is required for --scope project", file=sys.stderr)
            return 2
        project = Path(args.project).resolve()
        for k in kinds:
            results.append(deploy_skill_to(project_root(k, project), args.dry_run) | {"scope": "project", "agent": k})
        if args.write_agents_snippet:
            snippet_src = package_root() / "docs" / "AGENTS.agentpatch.snippet.md"
            snippet_dst = project / "docs" / "agent-patch-kit-agents-snippet.md"
            if not args.dry_run and snippet_src.exists():
                snippet_dst.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(snippet_src, snippet_dst)
    else:
        print("unsupported scope", file=sys.stderr)
        return 2
    if args.json:
        print_json({"dry_run": args.dry_run, "results": results})
    else:
        for r in results:
            action = "would deploy" if args.dry_run else "deployed"
            print(f"{action}: [{r['scope']}/{r['agent']}] {r['target']}")
    return 0


def plugin_template_dir() -> Path:
    return package_root() / "plugins" / "AiCodingAgentPatch"


def make_plugin(out: Path, version: str, include_cli: bool = True) -> Path:
    dst = out / "plugins" / "AiCodingAgentPatch"
    copytree_merge(plugin_template_dir(), dst)
    # ensure skill copy is current
    copytree_merge(skill_source_dir(), dst / "skills" / SKILL_NAME)
    if include_cli:
        copytree_merge(package_root() / "agent_patch", dst / "assets" / "agent_patch")
        scripts_src = package_root() / "scripts"
        if scripts_src.exists():
            copytree_merge(scripts_src, dst / "assets" / "scripts")
        config_src = package_root() / "config"
        if config_src.exists():
            copytree_merge(config_src, dst / "assets" / "config")
        docs_src = package_root() / "docs"
        if docs_src.exists():
            copytree_merge(docs_src, dst / "assets" / "docs")
        for file_name in ["pyproject.toml", "README_CN.md", "README.md", "Taskfile.yml", "lychee.toml", "agent-patch-kit.manifest.json"]:
            src = package_root() / file_name
            if src.exists():
                shutil.copy2(src, dst / "assets" / file_name)
    manifest = dst / ".codex-plugin" / "plugin.json"
    if manifest.exists():
        data = json.loads(manifest.read_text(encoding="utf-8"))
        data["version"] = version
        manifest.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return dst


def cmd_package(args: argparse.Namespace) -> int:
    out = Path(args.out).resolve()
    if args.kind != "aicoding-plugin":
        print("only kind supported now: aicoding-plugin", file=sys.stderr)
        return 2
    out.mkdir(parents=True, exist_ok=True)
    plugin_dir = make_plugin(out, args.plugin_version or __version__, include_cli=not args.no_cli_assets)
    marketplace = {
        "name": "aicoding-agent-patch-platform",
        "interface": {"displayName": "AiCoding Agent Patch Platform"},
        "plugins": [
            {
                "name": "aicoding-agent-patch-kit",
                "source": {"source": "local", "path": "./dist/agent-patch-kit/plugins/AiCodingAgentPatch"},
                "policy": {"installation": "AVAILABLE", "authentication": "ON_INSTALL"},
                "category": "Developer Tools"
            }
        ]
    }
    market_path = out / "marketplace.agent-patch.json"
    market_path.write_text(json.dumps(marketplace, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    if args.zip:
        archive = shutil.make_archive(str(out / "agent-patch-kit-plugin"), "zip", out)
    else:
        archive = None
    if args.json:
        print_json({"plugin_dir": str(plugin_dir), "marketplace": str(market_path), "archive": archive})
    else:
        print(f"plugin_dir: {plugin_dir}")
        print(f"marketplace: {market_path}")
        if archive:
            print(f"archive: {archive}")
    return 0


DEFAULT_LYCHEE_TOML = """# Agent Patch Kit link validation config\nverbose = \"info\"\nno_progress = true\ntimeout = 20\nmax_retries = 2\naccept = [\"200..=204\", \"429\"]\ninclude_fragments = \"full\"\nfallback_extensions = [\"md\", \"html\"]\nexclude_path = [\n  \"(^|/)\\\\.git/\",\n  \"(^|/)node_modules/\",\n  \"(^|/)dist/\",\n  \"(^|/)build/\",\n  \"(^|/)coverage/\"\n]\nexclude = [\n  \"^https://example\\\\.com\",\n  \"^http://localhost\",\n  \"^https://localhost\"\n]\n"""


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(prog="apatch", description="Agent Patch Kit safe patch workflow CLI")
    p.add_argument("--version", action="version", version=f"apatch {__version__}")
    sub = p.add_subparsers(dest="cmd", required=True)

    d = sub.add_parser("doctor", help="check local environment")
    d.add_argument("--json", action="store_true")
    d.set_defaults(func=cmd_doctor)

    brief = sub.add_parser("brief", help="print developer-only quick brief for agents")
    brief.add_argument("--path", default=".")
    brief.add_argument("--format", choices=["md", "json"], default="md")
    brief.set_defaults(func=cmd_brief)

    state = sub.add_parser("state", help="enable, disable, or inspect Agent Patch Kit state by scope")
    statesub = state.add_subparsers(dest="statecmd", required=True)
    st_status = statesub.add_parser("status", help="show effective system/user/project state")
    st_status.add_argument("--path", default=".")
    st_status.add_argument("--json", action="store_true")
    st_status.set_defaults(func=cmd_state)
    st_enable = statesub.add_parser("enable", help="enable kit at a scope")
    st_enable.add_argument("--scope", choices=["system", "user", "project"], required=True)
    st_enable.add_argument("--path", default=".")
    st_enable.add_argument("--reason")
    st_enable.add_argument("--json", action="store_true")
    st_enable.set_defaults(func=cmd_state)
    st_disable = statesub.add_parser("disable", help="disable kit at a scope")
    st_disable.add_argument("--scope", choices=["system", "user", "project"], required=True)
    st_disable.add_argument("--path", default=".")
    st_disable.add_argument("--reason")
    st_disable.add_argument("--json", action="store_true")
    st_disable.set_defaults(func=cmd_state)
    st_where = statesub.add_parser("where", help="show state file locations")
    st_where.add_argument("--path", default=".")
    st_where.add_argument("--json", action="store_true")
    st_where.set_defaults(func=cmd_state)

    init = sub.add_parser("init", help="write default .agentpatch.json and optional lychee.toml")
    init.add_argument("--path", default=".")
    init.add_argument("--force", action="store_true")
    init.add_argument("--write-lychee", action="store_true")
    init.set_defaults(func=cmd_init)

    s = sub.add_parser("status", help="show git status --short")
    s.add_argument("--path", default=".")
    s.add_argument("--json", action="store_true")
    s.add_argument("--limit", type=int, default=80)
    s.set_defaults(func=cmd_status)

    sc = sub.add_parser("scan", help="scan text before editing")
    sc.add_argument("pattern")
    sc.add_argument("--path", default=".")
    sc.add_argument("--glob", action="append", help="file glob; can repeat")
    sc.add_argument("--fixed", action="store_true", help="treat pattern as fixed string")
    sc.add_argument("--max-count", type=int)
    sc.add_argument("--json", action="store_true")
    sc.set_defaults(func=cmd_scan)

    r = sub.add_parser("replace", help="preview or apply safe replacement")
    r.add_argument("--old", required=True)
    r.add_argument("--new", required=True)
    r.add_argument("--path", default=".")
    r.add_argument("--glob", action="append")
    r.add_argument("--fixed", action="store_true", help="fixed string mode")
    r.add_argument("--regex", action="store_true", help="regex mode")
    r.add_argument("--preview", action="store_true")
    r.add_argument("--apply", action="store_true")
    r.add_argument("--no-tx", action="store_true", help="do not create transaction before applying")
    r.add_argument("--allow-disabled", action="store_true", help="allow apply when kit is disabled; use only with explicit authorization")
    r.add_argument("--json", action="store_true")
    r.set_defaults(func=cmd_replace)

    ast = sub.add_parser("ast", help="wrap ast-grep structural search/rewrite")
    ast.add_argument("--pattern", "-p", required=True)
    ast.add_argument("--rewrite", "-r")
    ast.add_argument("--lang", "-l")
    ast.add_argument("--path", default=".")
    ast.add_argument("--glob", action="append")
    ast.add_argument("--context", type=int)
    ast.add_argument("--preview", action="store_true")
    ast.add_argument("--apply", action="store_true")
    ast.add_argument("--no-tx", action="store_true")
    ast.add_argument("--allow-disabled", action="store_true", help="allow apply when kit is disabled; use only with explicit authorization")
    ast.add_argument("--json", action="store_true")
    ast.set_defaults(func=cmd_ast)

    tx = sub.add_parser("tx", help="transaction snapshot and rollback")
    txsub = tx.add_subparsers(dest="txcmd", required=True)
    txb = txsub.add_parser("begin", help="create transaction snapshot")
    txb.add_argument("--path", default=".")
    txb.add_argument("--name")
    txb.add_argument("--json", action="store_true")
    txb.set_defaults(func=cmd_tx)
    txl = txsub.add_parser("list", help="list transactions")
    txl.add_argument("--path", default=".")
    txl.add_argument("--limit", type=int, default=20)
    txl.add_argument("--json", action="store_true")
    txl.set_defaults(func=cmd_tx)
    txr = txsub.add_parser("rollback", help="rollback to transaction snapshot")
    txr.add_argument("id")
    txr.add_argument("--path", default=".")
    txr.add_argument("--preview", action="store_true")
    txr.add_argument("--apply", action="store_true")
    txr.add_argument("--force", action="store_true")
    txr.add_argument("--clean-created", action="store_true", help="delete untracked files created after transaction begin")
    txr.add_argument("--json", action="store_true")
    txr.set_defaults(func=cmd_tx)

    links = sub.add_parser("links", help="validate Markdown links through lychee")
    links.add_argument("--path", default=".")
    links.add_argument("--mode", choices=["offline", "online"], default=None)
    links.add_argument("--include-fragments", choices=["none", "anchor-only", "text-only", "full"], default=None)
    links.add_argument("--input", action="append", help="lychee input glob; can repeat")
    links.add_argument("--config")
    links.add_argument("--no-progress", action="store_true", default=True)
    links.add_argument("--json", action="store_true")
    links.set_defaults(func=cmd_links)

    v = sub.add_parser("verify", help="verify patch after editing")
    v.add_argument("--path", default=".")
    v.add_argument("--old")
    v.add_argument("--new")
    v.add_argument("--glob", action="append")
    v.add_argument("--fixed", action="store_true")
    v.add_argument("--regex", action="store_true")
    v.add_argument("--task", action="store_true", help="run task verify if task is installed")
    v.add_argument("--links", action="store_true", help="run Markdown link validation")
    v.add_argument("--links-mode", choices=["offline", "online"], default="offline")
    v.add_argument("--expect-old-zero", action="store_true", default=True)
    v.add_argument("--expect-new-nonzero", action="store_true", default=False)
    v.add_argument("--json", action="store_true")
    v.set_defaults(func=cmd_verify)

    sm = sub.add_parser("summary", help="summarize current diff")
    sm.add_argument("--path", default=".")
    sm.add_argument("--json", action="store_true")
    sm.set_defaults(func=cmd_summary)

    dep = sub.add_parser("deploy", help="deploy Agent Patch Kit skill to user or project agent roots")
    dep.add_argument("--scope", choices=["system", "user", "project"], required=True)
    dep.add_argument("--agent", choices=["agents", "codex", "both"], default="both")
    dep.add_argument("--project", help="target project root for --scope project")
    dep.add_argument("--dry-run", action="store_true")
    dep.add_argument("--write-agents-snippet", action="store_true")
    dep.add_argument("--json", action="store_true")
    dep.set_defaults(func=cmd_deploy)

    pkg = sub.add_parser("package", help="package Agent Patch Kit for marketplaces/plugins")
    pkg.add_argument("kind", choices=["aicoding-plugin"])
    pkg.add_argument("--out", default="dist/agent-patch-kit")
    pkg.add_argument("--plugin-version")
    pkg.add_argument("--no-cli-assets", action="store_true")
    pkg.add_argument("--zip", action="store_true")
    pkg.add_argument("--json", action="store_true")
    pkg.set_defaults(func=cmd_package)
    return p


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    try:
        return args.func(args)
    except KeyboardInterrupt:
        return 130
    except Exception as e:
        print(f"apatch error: {e}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
