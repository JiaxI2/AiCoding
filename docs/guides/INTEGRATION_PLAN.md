# 集成实施计划

## 1. 创建分支

```powershell
git switch -c feature/loop-engineering-kit
```
## 2. 复制文件

把压缩包内容合入仓库根目录。不要直接启用 Kit。

## 3. 运行 Phase 1 测试

```powershell
go test ./internal/loopkit/...
```

## 4. 增加 registry 条目

仅在 ADR 评审通过后，把以下对象加入 `config/kit-registry.json`：

```json
{
  "id": "loop-engineering-kit",
  "manifest": "config/kits/loop-engineering-kit.json",
  "enabled": false
}
```

实际字段必须以当前 registry schema 为准，不要机械复制示例。

## 5. 加入依赖治理

将 `internal/loopkit/*` 设置为 platform 层。禁止它们被 `registry`、`runner`、`report`、`gitx` 反向依赖。

## 6. CLI 只加最小入口

Phase 1 推荐：

```text
work validate
work inspect
work evaluate
```

这些命令只调用纯函数，不创建任务数据库。

## 7. 全局验证

```powershell
bin\aicoding.exe governance dependencies --json
bin\aicoding.exe governance layout --json
bin\aicoding.exe docsync all --json
bin\aicoding.exe verify --profile Smoke --json
bin\aicoding.exe test --profile Full --json
git diff --check
```
