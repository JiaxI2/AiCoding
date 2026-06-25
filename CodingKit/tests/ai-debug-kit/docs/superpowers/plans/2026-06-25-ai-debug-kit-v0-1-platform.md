# AI Debug Kit v0.1 Platform Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a reusable, platform-oriented AI debug code kit with a host-only CLI/Core, simulator backend, replay-ready session bundle, dual Codex skills, and a minimal non-blocking C target shim template.

**Architecture:** The kit follows the local requirements documents: Agent/Skill calls a deterministic `ai-debug` CLI, CLI delegates to application services, services use Core models and backend ports, and real probe support remains a future backend. The Mklink-AI-Probe repository is used as a template for Skill + references + Python CLI packaging, but its hardware-specific commands are not copied into Core.

**Tech Stack:** Python 3.11+ standard library for the initial executable skeleton, pytest tests, C99 target shim files, JSON/TOML/YAML-compatible text artifacts.

---

### Task 1: Project Skeleton and Tests

**Files:**
- Create: `pyproject.toml`
- Create: `src/ai_debug/__init__.py`
- Create: `src/ai_debug/__main__.py`
- Create: `src/ai_debug/cli.py`
- Create: `tests/test_cli.py`

- [ ] **Step 1: Write failing CLI tests**

Create tests that run `python -m ai_debug version --output json` and `python -m ai_debug doctor --output json`, then assert the process exits `0` and returns a JSON envelope with `schema_version`, `ok`, `code`, and `data`.

- [ ] **Step 2: Verify RED**

Run: `python -m pytest tests/test_cli.py -q`

Expected: FAIL because `ai_debug` does not exist.

- [ ] **Step 3: Implement minimal CLI**

Create package files and an argparse CLI with `version`, `doctor`, and shared JSON envelope output.

- [ ] **Step 4: Verify GREEN**

Run: `python -m pytest tests/test_cli.py -q`

Expected: PASS.

### Task 2: Core Models and Simulator Backend

**Files:**
- Create: `src/ai_debug/core/result.py`
- Create: `src/ai_debug/core/address.py`
- Create: `src/ai_debug/core/capability.py`
- Create: `src/ai_debug/core/policy.py`
- Create: `src/ai_debug/backends/base.py`
- Create: `src/ai_debug/backends/simulator.py`
- Create: `tests/test_simulator_backend.py`

- [ ] **Step 1: Write failing simulator tests**

Test that simulator capabilities expose read/write/variable/telemetry, memory read returns the requested length, invalid ranges fail deterministically, and write requires approval policy unless configured for simulator test mode.

- [ ] **Step 2: Verify RED**

Run: `python -m pytest tests/test_simulator_backend.py -q`

Expected: FAIL because core/backend modules do not exist.

- [ ] **Step 3: Implement minimal models and simulator**

Add explicit address unit fields, capability flags, risk levels, policy checks, and an in-memory simulator backend.

- [ ] **Step 4: Verify GREEN**

Run: `python -m pytest tests/test_simulator_backend.py -q`

Expected: PASS.

### Task 3: Sessions, Reports, and Smoke Test

**Files:**
- Create: `src/ai_debug/core/session.py`
- Create: `src/ai_debug/app/services.py`
- Create: `src/ai_debug/reports/markdown.py`
- Modify: `src/ai_debug/cli.py`
- Create: `tests/test_smoke_flow.py`

- [ ] **Step 1: Write failing smoke-flow tests**

Test `ai-debug smoke-test --workspace <tmp> --output json` creates `.ai-debug/deployment/active-profile.json`, a session bundle, and a markdown report. Assert the envelope points to those files.

- [ ] **Step 2: Verify RED**

Run: `python -m pytest tests/test_smoke_flow.py -q`

Expected: FAIL because smoke-test is not implemented.

- [ ] **Step 3: Implement services**

Add deployment/profile generation, simulator read/write/readback sample flow, session action recording, and report generation.

- [ ] **Step 4: Verify GREEN**

Run: `python -m pytest tests/test_smoke_flow.py -q`

Expected: PASS.

### Task 4: Dual Skills and Documentation

**Files:**
- Create: `.agents/skills/ai-debug-kit-deploy/SKILL.md`
- Create: `.agents/skills/ai-debug-kit-deploy/references/installation.md`
- Create: `.agents/skills/ai-debug-operations/SKILL.md`
- Create: `.agents/skills/ai-debug-operations/references/operation-lifecycle.md`
- Modify: `README.md`

- [ ] **Step 1: Add Skill docs**

Skill A covers deployment and capability validation. Skill B covers generic debug operation lifecycle and explicitly refuses business root-cause analysis.

- [ ] **Step 2: Verify docs**

Run: `rg -n "root cause|business|smoke-test|active-profile|Policy" .agents README.md`

Expected: references are present and Skill B states the platform boundary.

### Task 5: C99 Target Shim Template

**Files:**
- Create: `src/ai_debug/target_shim/include/ai_debug_target_shim.h`
- Create: `src/ai_debug/target_shim/src/ai_debug_target_shim.c`
- Create: `tests/test_target_shim_static.py`

- [ ] **Step 1: Write failing static tests**

Test the C files contain no dynamic allocation, no printf/sprintf, no blocking waits, and expose `AiDebug_Init`, `AiDebug_PushSampleU32`, `AiDebug_PublishSnapshot`, and `AiDebug_Service`.

- [ ] **Step 2: Verify RED**

Run: `python -m pytest tests/test_target_shim_static.py -q`

Expected: FAIL because target shim files do not exist.

- [ ] **Step 3: Implement minimal C99 shim**

Add a fixed-size SPSC ring buffer, drop-newest overflow counter, compile-time disable macro, no transport in ISR path, and no dynamic memory.

- [ ] **Step 4: Verify GREEN**

Run: `python -m pytest tests/test_target_shim_static.py -q`

Expected: PASS.

### Task 6: Final Verification

**Files:**
- All created files

- [ ] **Step 1: Run complete test suite**

Run: `python -m pytest -q`

Expected: PASS.

- [ ] **Step 2: Run CLI smoke commands**

Run: `python -m ai_debug --help`

Expected: help text exits `0`.

Run: `python -m ai_debug smoke-test --workspace . --output json`

Expected: JSON envelope with `ok=true`, generated active profile, generated session, and report path.

- [ ] **Step 3: Check platform boundary**

Run: `rg -n "root cause|Kp|FOC|EtherCAT|Flash" src .agents README.md`

Expected: no business root-cause logic in `src`; docs only mention boundaries and future extensions.
