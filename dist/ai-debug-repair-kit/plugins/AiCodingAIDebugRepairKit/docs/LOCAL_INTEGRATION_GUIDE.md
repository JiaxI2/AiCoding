# 本地部署与本地 AiCoding 集成速查

## 单独安装

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File "<package-root>\scripts\install-airepair-standalone.ps1" -PackageRoot "<package-root>" -Json
airepair doctor --output json
```

## 集成到本地 AiCoding

```powershell
cd <AiCoding-root>
powershell -NoProfile -ExecutionPolicy Bypass -File "<package-root>\scripts\install-ai-debug-repair-kit.ps1" -PackageRoot "<package-root>" -Json
powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\verify-ai-debug-repair-kit.ps1" -Json
```

## 初始化 Repair Profile

```powershell
airepair init --workspace . --output json
airepair profile validate --profile .ai-debug-repair\profiles\loop.safe.json --output json
```

## 卸载

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\uninstall-ai-debug-repair-kit.ps1" -Json
```
