from __future__ import annotations
import argparse, datetime, json, re, shutil, subprocess
from pathlib import Path

REQUIRED_SPEC = ["spec/PRD.md","spec/APP_FLOW.md","spec/TECH_STACK.md","spec/CODING_GUIDELINES.md","spec/PROJECT_STRUCTURE.md","spec/IMPLEMENTATION_PLAN.md","spec/TEST_STRATEGY.md"]
REQUIRED_MEMORY = [".agent-memory/CURRENT.md",".agent-memory/DECISIONS.md"]
TDD_PHRASES = ["Write failing test","Run test and confirm failure","Implement minimal code","Run test and confirm pass","Refactor","Run test again"]

STAGES = {
    "L0": {"max": 5000, "files": [".agent-memory/CURRENT.md",".agent-memory/DECISIONS.md"], "changed": True},
    "L1": {"max": 12000, "files": [".agent-memory/CURRENT.md",".agent-memory/DECISIONS.md","spec/IMPLEMENTATION_PLAN.md","spec/TEST_STRATEGY.md"], "changed": True},
    "L2": {"max": 24000, "files": [".agent-memory/CURRENT.md",".agent-memory/DECISIONS.md","spec/PRD.md","spec/APP_FLOW.md","spec/TECH_STACK.md","spec/CODING_GUIDELINES.md","spec/PROJECT_STRUCTURE.md","spec/IMPLEMENTATION_PLAN.md","spec/TEST_STRATEGY.md"], "changed": True},
    "L3": {"max": 48000, "files": [".agent-memory/CURRENT.md",".agent-memory/DECISIONS.md","spec/PRD.md","spec/APP_FLOW.md","spec/TECH_STACK.md","spec/CODING_GUIDELINES.md","spec/PROJECT_STRUCTURE.md","spec/IMPLEMENTATION_PLAN.md","spec/TEST_STRATEGY.md","docs/traceability/TRACEABILITY_MATRIX.md"], "changed": True},
}

def repo_root(path: str) -> Path:
    return Path(path).resolve()

def ensure(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)

def write_file(path: Path, text: str, overwrite: bool = False) -> None:
    ensure(path.parent)
    if overwrite or not path.exists():
        path.write_text(text, encoding="utf-8")

def print_json(data) -> int:
    print(json.dumps(data, ensure_ascii=False, indent=2))
    return 0 if data.get("ok", True) else 1

def changed_files(root: Path, staged: bool = False) -> list[str]:
    cmd = ["git","-C",str(root),"diff","--name-only","HEAD"]
    if staged:
        cmd = ["git","-C",str(root),"diff","--cached","--name-only"]
    try:
        out = subprocess.check_output(cmd, text=True, stderr=subprocess.DEVNULL)
    except Exception:
        out = ""
    files = [x.strip() for x in out.splitlines() if x.strip()]
    if not files:
        try:
            out = subprocess.check_output(["git","-C",str(root),"status","--short"], text=True, stderr=subprocess.DEVNULL)
            files = [re.sub(r"^\s*\S+\s+","",x).strip() for x in out.splitlines() if x.strip()]
        except Exception:
            pass
    return sorted(set(files))

def auto_stage(files: list[str]) -> tuple[str, str]:
    if not files:
        return "L0", "no changed files"
    stage, reason = "L1", "default changed-file task context"
    for f in files:
        if re.match(r"^(\.github/|\.githooks/|scripts/|config/)", f):
            stage, reason = "L2", "workflow/hook/script/config change"
        if re.match(r"^(spec/|docs/adr/|specs/bdd/|specs/tdd/|docs/traceability/)", f):
            return "L3", "spec/ADR/BDD/TDD/traceability change"
    return stage, reason

def write_state(root: Path, owned: list[str]) -> None:
    ensure(root / ".agent-dev-kit")
    (root / ".agent-dev-kit/install-state.json").write_text(json.dumps({
        "schema": "aicoding-agent-dev-kit.install-state.v1",
        "version": "0.11.1",
        "ownedFiles": owned,
    }, indent=2), encoding="utf-8")

def install(args) -> int:
    root = repo_root(args.repo)
    owned = []
    if args.spec_pack:
        for rel in REQUIRED_SPEC:
            write_file(root / rel, f"# {Path(rel).stem}\n\nTODO: Fill this spec file.\n")
        write_file(root / "spec/IMPLEMENTATION_PLAN.md", "# IMPLEMENTATION_PLAN\n\n## TDD Loop\n\n- Write failing test\n- Run test and confirm failure\n- Implement minimal code\n- Run test and confirm pass\n- Refactor\n- Run test again\n", overwrite=True)
        owned.append("spec")
    if args.memory:
        write_file(root / ".agent-memory/README.md", "# Agent Decision Memory\n\nOnly record important decisions and current state.\n")
        write_file(root / ".agent-memory/CURRENT.md", "# Current State\n\n## Current Goal\n\n\n## Active Task\n\n\n## Next Step\n\n\n## Blockers\n\nNone.\n")
        write_file(root / ".agent-memory/DECISIONS.md", "# Decision Memory\n\nUse this file for important decisions only.\n")
        owned += [".agent-memory/CURRENT.md", ".agent-memory/DECISIONS.md"]
    if args.hooks:
        write_file(root / ".githooks/pre-commit", "#!/usr/bin/env bash\npwsh -NoProfile -ExecutionPolicy Bypass -File scripts/invoke-agent-quality-gate.ps1 -Mode pre-commit\n")
        write_file(root / ".githooks/commit-msg", "#!/usr/bin/env bash\nexit 0\n")
        owned += [".githooks/pre-commit", ".githooks/commit-msg"]
        try:
            subprocess.run(["git","-C",str(root),"config","core.hooksPath",".githooks"], check=False)
        except Exception:
            pass
    if args.workflow:
        write_file(root / ".github/workflows/agent-dev-kit-ci.yml", "name: Agent Dev Kit CI\non: [push, pull_request]\njobs:\n  quality-gate:\n    runs-on: windows-latest\n    steps:\n      - uses: actions/checkout@v4\n      - shell: pwsh\n        run: pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/invoke-agent-quality-gate.ps1 -Mode ci -Json\n")
        owned.append(".github/workflows/agent-dev-kit-ci.yml")
    if args.thin_skill:
        write_file(root / ".agents/skills/aicoding-agent-dev-kit/SKILL.md", "---\nname: aicoding-agent-dev-kit\n---\n\nThin Skill. Executable truth is the platform Kit.\n")
        owned.append(".agents/skills/aicoding-agent-dev-kit/SKILL.md")
    if args.subagents:
        for name in ["spec-reviewer","implementation-planner","tdd-enforcer","worktree-coordinator","systematic-debugger"]:
            write_file(root / f".agents/subagents/{name}.md", f"# {name}\n\nGeneric subagent template.\n")
        owned.append(".agents/subagents")
    write_state(root, owned)
    return print_json({"ok": True, "action": "install", "repo": str(root), "ownedFiles": owned})

def status(args) -> int:
    root = repo_root(args.repo)
    return print_json({
        "ok": True,
        "installed": (root / ".agent-dev-kit/install-state.json").exists(),
        "hasSpecPack": all((root / p).exists() for p in REQUIRED_SPEC),
        "hasDecisionMemory": all((root / p).exists() for p in REQUIRED_MEMORY),
        "hasHook": (root / ".githooks/pre-commit").exists(),
        "hasWorkflow": (root / ".github/workflows/agent-dev-kit-ci.yml").exists(),
        "hasManifest": (root / ".agent-dev-kit/context/context-manifest.json").exists()
    })

def changed(args) -> int:
    files = changed_files(repo_root(args.repo), args.staged)
    return print_json({"ok": True, "count": len(files), "files": files})

def scope(args) -> int:
    root = repo_root(args.repo)
    files = changed_files(root)
    st, reason = auto_stage(files)
    return print_json({"ok": True, "stage": st, "reason": reason, "changedCount": len(files), "changedFiles": files})

def verify(args) -> int:
    root = repo_root(args.repo)
    missing = [p for p in REQUIRED_SPEC + REQUIRED_MEMORY if not (root / p).exists()]
    plan = root / "spec/IMPLEMENTATION_PLAN.md"
    if plan.exists():
        text = plan.read_text(encoding="utf-8", errors="ignore")
        missing += [f"IMPLEMENTATION_PLAN phrase: {p}" for p in TDD_PHRASES if p not in text]
    return print_json({"ok": not missing, "missing": missing})

def load(args) -> int:
    root = repo_root(args.repo)
    files = changed_files(root)
    reason = "manual stage"
    stage = args.stage
    if args.auto:
        stage, reason = auto_stage(files)
    policy = STAGES[stage]
    max_chars = args.max_chars or policy["max"]
    paths = list(policy["files"])
    if policy.get("changed"):
        paths += files
    if stage == "L3":
        for glob in ["docs/adr/*.md","specs/bdd/*.feature","specs/tdd/*.md"]:
            paths += [str(p.relative_to(root)).replace("\\","/") for p in root.glob(glob)]
    ensure(root / ".agent-dev-kit/context")
    out_path = root / ".agent-dev-kit/context/context-pack.md"
    manifest_path = root / ".agent-dev-kit/context/context-manifest.json"
    chunks = [f"# Agent Context Pack\n\nStage: {stage}\nReason: {reason}\n\n"]
    included, skipped, truncated = [], [], []
    seen, chars = set(), 0
    for rel in paths:
        if not rel or rel in seen:
            continue
        seen.add(rel)
        p = root / rel
        if not p.exists() or p.is_dir():
            skipped.append({"path": rel, "reason": "not found or directory"})
            continue
        if p.stat().st_size > 1024 * 1024:
            skipped.append({"path": rel, "reason": "too large", "bytes": p.stat().st_size})
            continue
        text = p.read_text(encoding="utf-8", errors="ignore")
        remaining = max_chars - chars
        if remaining <= 0:
            skipped.append({"path": rel, "reason": "context budget exhausted"})
            continue
        original = len(text)
        if len(text) > remaining:
            text = text[:remaining] + "\n...[truncated]..."
            truncated.append({"path": rel, "originalChars": original, "includedChars": len(text)})
        chunks.append(f"## {rel}\n```text\n{text}\n```\n\n")
        chars += len(text)
        included.append({"path": rel, "chars": len(text)})
    out_path.write_text("".join(chunks), encoding="utf-8")
    manifest = {
        "schema": "aicoding-agent-dev-kit.context-manifest.v1",
        "version": "0.11.1",
        "stage": stage,
        "auto": args.auto,
        "reason": reason,
        "maxChars": max_chars,
        "chars": chars,
        "roughTokens": chars // 4,
        "changedFiles": files,
        "includedFiles": included,
        "skippedFiles": skipped,
        "truncatedFiles": truncated,
        "contextPack": str(out_path)
    }
    manifest_path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2), encoding="utf-8")
    return print_json({"ok": True, "stage": stage, "reason": reason, "contextPack": str(out_path), "manifest": str(manifest_path), "chars": chars, "roughTokens": chars//4})

def manifest(args) -> int:
    root = repo_root(args.repo)
    p = root / ".agent-dev-kit/context/context-manifest.json"
    if not p.exists():
        return print_json({"ok": False, "error": "context manifest not found", "path": str(p)})
    print(p.read_text(encoding="utf-8"))
    return 0

def fast_start(args) -> int:
    root = repo_root(args.repo)
    files = changed_files(root)
    def short(rel, n=2000):
        p = root / rel
        if not p.exists():
            return ""
        t = p.read_text(encoding="utf-8", errors="ignore")
        return t[:n] + ("\n...[truncated]..." if len(t) > n else "")
    st, reason = auto_stage(files)
    return print_json({"ok": True, "repo": str(root), "recommendedStage": st, "reason": reason, "changedCount": len(files), "changedFiles": files[:30], "current": short(".agent-memory/CURRENT.md", 1600), "decisions": short(".agent-memory/DECISIONS.md", 2500), "next": ["load --auto", "manifest", "gate --mode pre-commit"]})

def context(args) -> int:
    # Backward compatible alias.
    stage = "L1"
    if args.mode == "memory":
        stage = "L0"
    if args.mode in ("spec","all"):
        stage = "L3"
    args2 = argparse.Namespace(repo=args.repo, stage=stage, auto=False, max_chars=args.max_chars)
    return load(args2)

def token_audit(args) -> int:
    root = repo_root(args.repo)
    large, total = [], 0
    for p in root.rglob("*"):
        if not p.is_file():
            continue
        s = str(p)
        if any(x in s for x in [".git", "node_modules", ".next", "build", "dist"]):
            continue
        if p.stat().st_size > 5*1024*1024:
            continue
        try:
            t = p.read_text(encoding="utf-8", errors="ignore")
        except Exception:
            continue
        total += len(t)
        if len(t) > args.max_file_chars:
            large.append({"path": str(p.relative_to(root)).replace("\\","/"), "chars": len(t), "roughTokens": len(t)//4})
    return print_json({"ok": True, "totalChars": total, "roughTokens": total//4, "largeFiles": large[:50]})

def shard(args) -> int:
    root = repo_root(args.repo)
    plan = root / "spec/IMPLEMENTATION_PLAN.md"
    shard_dir = root / ".agent-dev-kit/shards"
    ensure(shard_dir)
    created = []
    if plan.exists():
        text = plan.read_text(encoding="utf-8", errors="ignore")
        matches = re.findall(r"(?m)^### Task\s+\d+:\s+(.+)$", text)
        for i, title in enumerate(matches, 1):
            name = re.sub(r"[^\w\-]+", "-", title).strip("-").lower() or f"task-{i}"
            p = shard_dir / f"{i:02d}-{name}.md"
            p.write_text(f"# Task Shard {i}\n\nTask: {title}\n\nLoop: Red -> Green -> Refactor -> Gate -> Commit\n", encoding="utf-8")
            created.append(str(p))
    return print_json({"ok": True, "shardCount": len(created), "shards": created})

def index(args) -> int:
    root = repo_root(args.repo)
    cache = root / ".agent-dev-kit/cache"
    ensure(cache)
    files = []
    for p in root.rglob("*"):
        if p.is_file() and ".git" not in str(p) and "node_modules" not in str(p):
            files.append({"path": str(p.relative_to(root)).replace("\\","/"), "bytes": p.stat().st_size})
    out = cache / "repo-index.json"
    out.write_text(json.dumps({"ok": True, "fileCount": len(files), "files": files}, ensure_ascii=False, indent=2), encoding="utf-8")
    return print_json({"ok": True, "indexPath": str(out), "fileCount": len(files)})

def current_show(args) -> int:
    root = repo_root(args.repo)
    p = root / ".agent-memory/CURRENT.md"
    return print_json({"ok": True, "path": str(p), "content": p.read_text(encoding="utf-8", errors="ignore") if p.exists() else ""})

def current_set(args) -> int:
    root = repo_root(args.repo)
    p = root / ".agent-memory/CURRENT.md"
    write_file(p, f"# Current State\n\n## Current Goal\n\n{args.goal}\n\n## Active Task\n\n{args.task}\n\n## Next Step\n\n{args.next}\n\n## Blockers\n\n{args.blockers}\n", overwrite=True)
    return print_json({"ok": True, "path": str(p)})

def decision_add(args) -> int:
    root = repo_root(args.repo)
    p = root / ".agent-memory/DECISIONS.md"
    write_file(p, "# Decision Memory\n\nUse this file for important decisions only.\n")
    text = p.read_text(encoding="utf-8", errors="ignore")
    ids = [int(x) for x in re.findall(r"D-(\d{4})", text)]
    did = f"D-{(max(ids)+1 if ids else 1):04d}"
    type_map = {"human": "Human Decision", "agent-accepted": "Agent Proposal, Human Accepted", "rejected": "Rejected"}
    entry = f"\n\n## {did}: {args.title}\n\n- Type: {type_map[args.type]}\n- Status: Accepted\n- Date: {datetime.date.today().isoformat()}\n- Context: {args.context}\n- Decision: {args.decision}\n- Impact: {args.impact}\n- Link: {args.link}\n"
    with p.open("a", encoding="utf-8") as f:
        f.write(entry)
    return print_json({"ok": True, "id": did, "path": str(p)})

def decision_list(args) -> int:
    root = repo_root(args.repo)
    p = root / ".agent-memory/DECISIONS.md"
    items = []
    if p.exists():
        for did, title in re.findall(r"(?m)^##\s+(D-\d{4}):\s+(.+)$", p.read_text(encoding="utf-8", errors="ignore")):
            items.append({"id": did, "title": title})
    return print_json({"ok": True, "count": len(items), "decisions": items})

def decision_promote_adr(args) -> int:
    root = repo_root(args.repo)
    dec = root / ".agent-memory/DECISIONS.md"
    if not dec.exists():
        return print_json({"ok": False, "error": "DECISIONS.md not found"})
    text = dec.read_text(encoding="utf-8", errors="ignore")
    m = re.search(rf"(?ms)^##\s+{re.escape(args.id)}:\s+(.+?)(?=^##\s+D-\d{{4}}:|\Z)", text)
    if not m:
        return print_json({"ok": False, "error": "decision not found", "id": args.id})
    title = m.group(1).splitlines()[0].strip()
    slug = re.sub(r"[^a-zA-Z0-9]+", "-", title).strip("-").lower() or args.id.lower()
    adr = root / "docs/adr" / f"adr-{args.id.lower().replace('d-','')}-{slug}.md"
    write_file(adr, f"# ADR: {title}\n\n## Status\n\nProposed\n\n## Source Decision\n\n{args.id}\n\n## Context\n\nPromoted from `.agent-memory/DECISIONS.md`.\n\n## Decision\n\n{m.group(0)}\n\n## Consequences\n\nTBD.\n\n## Enforcement\n\nTBD.\n", overwrite=True)
    return print_json({"ok": True, "id": args.id, "adr": str(adr)})

def compact(args) -> int:
    return print_json({"ok": True, "deprecated": True, "message": "v0.11.1 uses lightweight decision memory and sequential loader. Use current set, decision add, and load instead."})



def hook_detect(args) -> int:
    root = repo_root(args.repo)
    candidates = [
        root / ".githooks/pre-commit",
        root / ".githooks/pre-commit.ps1",
        root / ".git/hooks/pre-commit",
        root / ".git/hooks/pre-commit.ps1",
    ]
    existing = [str(p) for p in candidates if p.exists()]
    has_bridge = False
    for p in candidates:
        if p.exists() and "BEGIN AICODING_AGENT_DEV_KIT_BRIDGE" in p.read_text(encoding="utf-8", errors="ignore"):
            has_bridge = True
    return print_json({"ok": True, "existingPreCommitHooks": existing, "hasExistingHook": bool(existing), "hasAgentDevKitBridge": has_bridge})

def hook_install_bridge(args) -> int:
    root = repo_root(args.repo)
    target = Path(args.hook_file).resolve() if args.hook_file else None
    if not target:
        candidates = [root / ".githooks/pre-commit", root / ".git/hooks/pre-commit"]
        target = next((p for p in candidates if p.exists()), root / ".githooks/pre-commit")
    exists = target.exists()
    if exists and not args.merge_existing_hook:
        return print_json({"ok": False, "error": "Existing hook detected. Use --merge-existing-hook.", "hookFile": str(target)})
    if not exists and not args.create_if_missing:
        return print_json({"ok": False, "error": "No existing hook. Use --create-if-missing.", "hookFile": str(target)})
    snippet = """\n# BEGIN AICODING_AGENT_DEV_KIT_BRIDGE\nrepo_root=\"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\"\nif [ -f \"$repo_root/scripts/invoke-agent-quality-gate.ps1\" ]; then\n  pwsh -NoProfile -ExecutionPolicy Bypass -File \"$repo_root/scripts/invoke-agent-quality-gate.ps1\" -Mode pre-commit -Json\n  code=$?\n  if [ \"$code\" -ne 0 ]; then\n    exit \"$code\"\n  fi\nfi\n# END AICODING_AGENT_DEV_KIT_BRIDGE\n"""
    ensure(target.parent)
    current = target.read_text(encoding="utf-8", errors="ignore") if target.exists() else "#!/usr/bin/env bash\n"
    if "BEGIN AICODING_AGENT_DEV_KIT_BRIDGE" not in current:
        target.write_text(current.rstrip() + "\n" + snippet + "\n", encoding="utf-8")
    return print_json({"ok": True, "hookFile": str(target), "created": not exists, "merged": exists})

def hook_uninstall_bridge(args) -> int:
    root = repo_root(args.repo)
    targets = [root / ".githooks/pre-commit", root / ".git/hooks/pre-commit"]
    changed = []
    for p in targets:
        if not p.exists():
            continue
        txt = p.read_text(encoding="utf-8", errors="ignore")
        new = re.sub(r"(?ms)\n?# BEGIN AICODING_AGENT_DEV_KIT_BRIDGE.*?# END AICODING_AGENT_DEV_KIT_BRIDGE\n?", "\n", txt)
        if new != txt:
            p.write_text(new, encoding="utf-8")
            changed.append(str(p))
    return print_json({"ok": True, "changed": changed})


def codex_native_status(args) -> int:
    root = repo_root(args.repo)
    required = [
        ".agents/plugins/marketplace.json",
        "plugins/aicoding-agent-dev-kit/.codex-plugin/plugin.json",
        "plugins/aicoding-agent-dev-kit/skills/aicoding-agent-dev-kit/SKILL.md",
        "plugins/aicoding-agent-dev-kit/hooks/hooks.json",
        ".codex/config.toml",
        ".codex/hooks.json",
        ".codex/agents/spec-reviewer.toml",
        ".codex/agents/implementation-planner.toml",
        ".codex/agents/tdd-enforcer.toml",
        ".codex/agents/worktree-coordinator.toml",
        ".codex/agents/systematic-debugger.toml",
    ]
    missing = [p for p in required if not (root / p).exists()]
    return print_json({"ok": not missing, "adapter": "codex-native", "version": "0.11.1", "missing": missing})



def clarify_init(args) -> int:
    root = repo_root(args.repo)
    path = root / "spec/PRD_OPTIONS.md"
    write_file(path, f"""# PRD Options and Solution Matrix\n\n## Requirement Question\n\n{args.requirement}\n\n## Options\n\n| Option ID | Name | Architecture | Flow | Pros | Cons | Risks | Validation | Effort | Recommended When |\n|---|---|---|---|---|---|---|---|---|---|\n| OPT-001 | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |\n| OPT-002 | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |\n| OPT-003 | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |\n""", overwrite=False)
    return print_json({"ok": True, "path": str(path), "requirement": args.requirement})

def clarify_choose(args) -> int:
    root = repo_root(args.repo)
    selected = root / "spec/SELECTED_SOLUTION.md"
    write_file(selected, f"""# Selected Solution\n\n## Selected Option\n\n- Option ID: {args.option_id}\n- Option Name: {args.name}\n- Status: Accepted\n\n## Human Decision\n\n- Decision owner: human\n- Date: {datetime.date.today().isoformat()}\n- Reason: {args.reason}\n\n## Required Document Sync\n\n- spec/PRD.md\n- spec/APP_FLOW.md\n- spec/IMPLEMENTATION_PLAN.md\n- docs/adr/*.md\n- docs/traceability/TRACEABILITY_MATRIX.md\n""", overwrite=True)
    args2 = argparse.Namespace(repo=args.repo, type="human", title=f"Select solution option {args.option_id}", decision=f"Selected {args.option_id} {args.name}. {args.reason}", context="Requirement clarification option selection", impact="PRD, APP_FLOW, plan, ADR and tests must align.", link="spec/SELECTED_SOLUTION.md")
    decision_add(args2)
    return print_json({"ok": True, "selectedSolution": str(selected), "optionId": args.option_id, "name": args.name})

def progress_init(args) -> int:
    root = repo_root(args.repo); ensure(root / ".agent-dev-kit/progress")
    board = root / ".agent-dev-kit/progress/progress-board.json"
    features = [{"featureId":"F-001","title":"Example small feature","mvp":True,"status":"todo","owner":"agent","linkedSpec":"spec/IMPLEMENTATION_PLAN.md","linkedTests":[],"currentStep":"Not started","nextStep":"Write failing test","evidence":[]}]
    plan=root/"spec/IMPLEMENTATION_PLAN.md"
    if args.from_plan and plan.exists():
        matches=re.findall(r"(?m)^### Task\s+(\d+):\s+(.+)$", plan.read_text(encoding="utf-8", errors="ignore"))
        if matches:
            features=[{"featureId":f"F-{int(n):03d}","title":t.strip(),"mvp":True,"status":"todo","owner":"agent","linkedSpec":"spec/IMPLEMENTATION_PLAN.md","linkedTests":[],"currentStep":"","nextStep":"Write failing test","evidence":[]} for n,t in matches]
    data={"schema":"aicoding-agent-dev-kit.progress-board.v1","version":"0.11.1","updatedAt":datetime.datetime.now().isoformat(),"activeFeature":"","features":features}
    board.write_text(json.dumps(data,ensure_ascii=False,indent=2),encoding="utf-8")
    return print_json({"ok":True,"board":str(board),"count":len(features)})

def progress_status(args) -> int:
    root=repo_root(args.repo); board=root/".agent-dev-kit/progress/progress-board.json"
    if not board.exists(): return print_json({"ok":False,"error":"progress board not found; run progress init"})
    print(board.read_text(encoding="utf-8")); return 0

def progress_update(args) -> int:
    root=repo_root(args.repo); board=root/".agent-dev-kit/progress/progress-board.json"
    if not board.exists(): progress_init(argparse.Namespace(repo=args.repo, from_plan=False))
    data=json.loads(board.read_text(encoding="utf-8")); found=False
    for f in data["features"]:
        if f["featureId"]==args.id:
            f["status"]=args.status
            if args.current: f["currentStep"]=args.current
            if args.next: f["nextStep"]=args.next
            if args.evidence: f.setdefault("evidence",[]).append(args.evidence)
            found=True
    if not found: data["features"].append({"featureId":args.id,"title":f"New feature {args.id}","mvp":True,"status":args.status,"owner":"agent","linkedSpec":"spec/IMPLEMENTATION_PLAN.md","linkedTests":[],"currentStep":args.current,"nextStep":args.next,"evidence":[args.evidence] if args.evidence else []})
    data["updatedAt"]=datetime.datetime.now().isoformat()
    if args.status=="doing": data["activeFeature"]=args.id
    board.write_text(json.dumps(data,ensure_ascii=False,indent=2),encoding="utf-8")
    return print_json({"ok":True,"board":str(board),"featureId":args.id,"status":args.status})

def uninstall(args) -> int:
    root = repo_root(args.repo)
    removed = []
    state = root / ".agent-dev-kit/install-state.json"
    if state.exists():
        data = json.loads(state.read_text(encoding="utf-8"))
        for rel in data.get("ownedFiles", []):
            p = root / rel
            if p.exists():
                shutil.rmtree(p) if p.is_dir() else p.unlink()
                removed.append(rel)
        state.unlink()
    if args.purge and args.force:
        for rel in ["spec",".agent-memory","docs/adr","specs/bdd","specs/tdd","docs/traceability"]:
            p = root / rel
            if p.exists():
                shutil.rmtree(p)
                removed.append(rel)
    return print_json({"ok": True, "action": "uninstall", "removed": removed, "purge": args.purge})

def main(argv=None) -> int:
    parser = argparse.ArgumentParser(prog="aicoding-agent-kit")
    sub = parser.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("install")
    p.add_argument("--repo", default=".")
    p.add_argument("--spec-pack", action="store_true")
    p.add_argument("--memory", action="store_true")
    p.add_argument("--hooks", action="store_true")
    p.add_argument("--workflow", action="store_true")
    p.add_argument("--thin-skill", action="store_true")
    p.add_argument("--subagents", action="store_true")
    p.set_defaults(func=install)

    for name, fn in [("status", status), ("verify", verify), ("test", verify), ("gate", verify)]:
        sp = sub.add_parser(name); sp.add_argument("--repo", default="."); sp.add_argument("--mode", default="all"); sp.set_defaults(func=fn)

    sp = sub.add_parser("changed"); sp.add_argument("--repo", default="."); sp.add_argument("--staged", action="store_true"); sp.set_defaults(func=changed)
    sp = sub.add_parser("scope"); sp.add_argument("--repo", default="."); sp.set_defaults(func=scope)
    sp = sub.add_parser("fast-start"); sp.add_argument("--repo", default="."); sp.set_defaults(func=fast_start)
    sp = sub.add_parser("load"); sp.add_argument("--repo", default="."); sp.add_argument("--stage", default="L0", choices=["L0","L1","L2","L3"]); sp.add_argument("--auto", action="store_true"); sp.add_argument("--max-chars", type=int, default=0); sp.set_defaults(func=load)
    sp = sub.add_parser("manifest"); sp.add_argument("--repo", default="."); sp.set_defaults(func=manifest)
    sp = sub.add_parser("context"); sp.add_argument("--repo", default="."); sp.add_argument("--mode", default="changed", choices=["changed","staged","spec","memory","all"]); sp.add_argument("--max-chars", type=int, default=12000); sp.set_defaults(func=context)
    sp = sub.add_parser("token-audit"); sp.add_argument("--repo", default="."); sp.add_argument("--max-file-chars", type=int, default=20000); sp.set_defaults(func=token_audit)
    sp = sub.add_parser("compact"); sp.add_argument("--repo", default="."); sp.set_defaults(func=compact)
    sp = sub.add_parser("shard"); sp.add_argument("--repo", default="."); sp.set_defaults(func=shard)
    sp = sub.add_parser("index"); sp.add_argument("--repo", default="."); sp.set_defaults(func=index)

    cur = sub.add_parser("current")
    cur_sub = cur.add_subparsers(dest="current_cmd", required=True)
    cs = cur_sub.add_parser("show"); cs.add_argument("--repo", default="."); cs.set_defaults(func=current_show)
    cset = cur_sub.add_parser("set")
    cset.add_argument("--repo", default="."); cset.add_argument("--goal", default=""); cset.add_argument("--task", default=""); cset.add_argument("--next", default=""); cset.add_argument("--blockers", default="None."); cset.set_defaults(func=current_set)

    dec = sub.add_parser("decision")
    dec_sub = dec.add_subparsers(dest="decision_cmd", required=True)
    da = dec_sub.add_parser("add")
    da.add_argument("--repo", default="."); da.add_argument("--type", choices=["human","agent-accepted","rejected"], default="human"); da.add_argument("--title", required=True); da.add_argument("--decision", required=True); da.add_argument("--context", default=""); da.add_argument("--impact", default=""); da.add_argument("--link", default=""); da.set_defaults(func=decision_add)
    dl = dec_sub.add_parser("list"); dl.add_argument("--repo", default="."); dl.set_defaults(func=decision_list)
    dp = dec_sub.add_parser("promote-adr"); dp.add_argument("--repo", default="."); dp.add_argument("--id", required=True); dp.set_defaults(func=decision_promote_adr)



    hook = sub.add_parser("hook")
    hook_sub = hook.add_subparsers(dest="hook_cmd", required=True)
    hd = hook_sub.add_parser("detect"); hd.add_argument("--repo", default="."); hd.set_defaults(func=hook_detect)
    hi = hook_sub.add_parser("install-bridge"); hi.add_argument("--repo", default="."); hi.add_argument("--hook-file", default=""); hi.add_argument("--merge-existing-hook", action="store_true"); hi.add_argument("--create-if-missing", action="store_true"); hi.set_defaults(func=hook_install_bridge)
    hu = hook_sub.add_parser("uninstall-bridge"); hu.add_argument("--repo", default="."); hu.set_defaults(func=hook_uninstall_bridge)

    cn = sub.add_parser("codex-native")
    cn_sub = cn.add_subparsers(dest="codex_native_cmd", required=True)
    cns = cn_sub.add_parser("status"); cns.add_argument("--repo", default="."); cns.set_defaults(func=codex_native_status)
    cnv = cn_sub.add_parser("verify"); cnv.add_argument("--repo", default="."); cnv.set_defaults(func=codex_native_status)


    clarify = sub.add_parser("clarify")
    clarify_sub = clarify.add_subparsers(dest="clarify_cmd", required=True)
    ci = clarify_sub.add_parser("init"); ci.add_argument("--repo", default="."); ci.add_argument("--requirement", default=""); ci.set_defaults(func=clarify_init)
    cc = clarify_sub.add_parser("choose"); cc.add_argument("--repo", default="."); cc.add_argument("--option-id", required=True); cc.add_argument("--name", default=""); cc.add_argument("--reason", default=""); cc.set_defaults(func=clarify_choose)

    progress = sub.add_parser("progress")
    progress_sub = progress.add_subparsers(dest="progress_cmd", required=True)
    pi = progress_sub.add_parser("init"); pi.add_argument("--repo", default="."); pi.add_argument("--from-plan", action="store_true"); pi.set_defaults(func=progress_init)
    ps = progress_sub.add_parser("status"); ps.add_argument("--repo", default="."); ps.set_defaults(func=progress_status)
    pu = progress_sub.add_parser("update"); pu.add_argument("--repo", default="."); pu.add_argument("--id", required=True); pu.add_argument("--status", choices=["todo","doing","blocked","review","done","dropped"], required=True); pu.add_argument("--current", default=""); pu.add_argument("--next", default=""); pu.add_argument("--evidence", default=""); pu.set_defaults(func=progress_update)

    u = sub.add_parser("uninstall")
    u.add_argument("--repo", default="."); u.add_argument("--purge", action="store_true"); u.add_argument("--force", action="store_true"); u.set_defaults(func=uninstall)

    args = parser.parse_args(argv)
    return args.func(args)

if __name__ == "__main__":
    raise SystemExit(main())
