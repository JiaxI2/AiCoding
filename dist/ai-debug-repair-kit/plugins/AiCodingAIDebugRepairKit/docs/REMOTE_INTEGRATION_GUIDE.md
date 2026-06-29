# 远程 AiCoding 集成指导

## 目标

将 AI Debug Repair Kit 正式加入：

```text
https://github.com/JiaxI2/AiCoding
```

## 推荐流程

```powershell
git clone https://github.com/JiaxI2/AiCoding.git AiCoding-ai-debug-repair
cd AiCoding-ai-debug-repair
git checkout -b feature/ai-debug-repair-kit-v0.4.0
powershell -NoProfile -ExecutionPolicy Bypass -File "<package-root>\scripts\install-ai-debug-repair-kit.ps1" -PackageRoot "<package-root>" -Json -SkipPipInstall
powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\verify-ai-debug-repair-kit.ps1" -Json
git diff --check
git status --short
```

## 应提交

```text
dist/ai-debug-repair-kit/**
scripts/*ai-debug-repair-kit.ps1
.agents/plugins/marketplace.json
docs/AGENT_DEPLOYMENT_GUIDE.md
docs/REMOTE_INTEGRATION_CHECKLIST.md
```

## 不应提交

```text
.venv/
.ai-debug-repair/runs/
.ai-debug-repair/attempts.jsonl
.ai-debug-repair/install-state.json
真实硬件日志
本机私有路径
```

## 新电脑恢复

```powershell
git clone https://github.com/JiaxI2/AiCoding.git
cd AiCoding
powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\install-ai-debug-repair-kit.ps1" -PackageRoot "." -Json
powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\verify-ai-debug-repair-kit.ps1" -Json
```
