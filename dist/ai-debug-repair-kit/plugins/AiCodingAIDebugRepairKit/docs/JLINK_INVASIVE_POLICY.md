# J-Link Invasive Operation Policy

## 1. 结论

J-Link backend 也应有 reset/halt/flash/write-memory 的功能入口，但默认不支持执行。

本 Kit v0.4.1 提供以下 CLI 入口：

```powershell
airepair jlink profile-template --profile .ai-debug-repair\profiles\jlink.json --output json
airepair jlink validate-profile --profile .ai-debug-repair\profiles\jlink.json --output json
airepair jlink capabilities --profile .ai-debug-repair\profiles\jlink.json --output json
airepair jlink reset --profile .ai-debug-repair\profiles\jlink.json --output json
airepair jlink halt --profile .ai-debug-repair\profiles\jlink.json --output json
airepair jlink flash --profile .ai-debug-repair\profiles\jlink.json --output json
airepair jlink write-memory --profile .ai-debug-repair\profiles\jlink.json --output json
```

## 2. 默认行为

默认 profile：

```json
{
  "mode": "readonly",
  "allow_reset": false,
  "allow_halt": false,
  "allow_flash": false,
  "allow_write_memory": false
}
```

默认调用会返回：

```text
POLICY_DENIED
```

## 3. 维护模式

如果 profile 设置：

```json
{
  "mode": "maintenance",
  "allow_reset": true
}
```

并且命令带 `--approve`，当前包会返回：

```text
CAPABILITY_DISABLED
```

原因：Repair Kit 包只提供 policy-gated interface stub，不直接实现真实 J-Link reset/halt/flash/write。真实硬件实现应放在主 `ai-debug-kit` 的 `JLinkBackend` 中，通过 pylink 实现，并继续遵守 profile/policy/session 约束。

## 4. 设计理由

- Agent 可以统一调用 reset/halt/flash/write-memory 入口。
- 默认 profile 不会产生侵入式副作用。
- 后续实现硬件功能时不需要改 Skill 语义。
- CI 可以验证“接口存在但默认拒绝”。
