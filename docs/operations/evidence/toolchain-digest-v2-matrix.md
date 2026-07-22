# toolchainDigest.v2 真跑矩阵

日期：2026-07-22

批准计划：`toolchain-semantic-identity`，`approvedTree=f2778dface7d2c0fde1f01de7cb43ff981f51812`。
以下输出来自 Windows/amd64 当前仓库的真实测试进程与 CLI；Smoke 的完整原始 JSON 保留于
`test-results/0032-toolchain-v1-seed/` 和 `test-results/0032-toolchain-v2-audit/`。

## 1、2、4、5、6：包内可控 probe 真跑

命令：

```text
go test -run "TestToolchain|TestCorruptToolchainCache" -v ./internal/validationevidence
```

原始关键输出：

```text
=== RUN   TestToolchainSemanticDigestChangesWithToolVersions
=== RUN   TestToolchainSemanticDigestChangesWithToolVersions/git
    toolchain_test.go:42: version-change tool=git before=sha256:291f555cbe348d8d8ac5dda02b6c06c7cd9b70169eedb58a4b4ac4e6b123bd58 after=sha256:5c5b5ba2c92a37c390de680db83acbb4caedb0048f584888c5c61af3f6ea1f66
=== RUN   TestToolchainSemanticDigestChangesWithToolVersions/go
    toolchain_test.go:42: version-change tool=go before=sha256:291f555cbe348d8d8ac5dda02b6c06c7cd9b70169eedb58a4b4ac4e6b123bd58 after=sha256:714bb3149bd40ef436167ffd5695eaa77046e443ac1d3d73144e47b9f73d2507
--- PASS: TestToolchainSemanticDigestChangesWithToolVersions
=== RUN   TestToolchainPathAndMtimeReprobeWithoutSemanticDrift
    toolchain_test.go:93: path-move-and-touch digest=sha256:291f555cbe348d8d8ac5dda02b6c06c7cd9b70169eedb58a4b4ac4e6b123bd58 probeCalls=6 domain=toolchainDigest.v2
--- PASS: TestToolchainPathAndMtimeReprobeWithoutSemanticDrift
=== RUN   TestToolchainPlatformArchitectureInjectionChangesDigest
    toolchain_test.go:131: platform-architecture-injection windows/amd64=sha256:291f555cbe348d8d8ac5dda02b6c06c7cd9b70169eedb58a4b4ac4e6b123bd58 linux/amd64=sha256:d2390433b0415ab5f92db551f364270d91298d9ff235c986287ff59dc5773e75 linux/arm64=sha256:ade6c0d696721b5b48dfe5c1bb9e57c484a8ffcd2cdcc3bf5bb9d7df948a8a36 reuse=VALIDATION_RECEIPT_MISS
--- PASS: TestToolchainPlatformArchitectureInjectionChangesDigest
=== RUN   TestToolchainProbeFailuresAreFailClosed/locate
    toolchain_test.go:140: unreadable executable exit=fail code=VALIDATION_FINGERPRINT_INVALID message=executable is unreadable: go
=== RUN   TestToolchainProbeFailuresAreFailClosed/version-exit
    toolchain_test.go:149: version command failure exit=fail code=VALIDATION_FINGERPRINT_INVALID message=probe go version: go version exited 1
=== RUN   TestToolchainProbeFailuresAreFailClosed/unparseable
    toolchain_test.go:155: unparseable version output exit=fail code=VALIDATION_FINGERPRINT_INVALID message=go version output is not valid text
--- PASS: TestToolchainProbeFailuresAreFailClosed
=== RUN   TestCorruptToolchainCacheIsRejectedAndRebuiltFromProbe
    toolchain_test.go:185: corrupt-cache rejected=true rebuilt=true digest=sha256:291f555cbe348d8d8ac5dda02b6c06c7cd9b70169eedb58a4b4ac4e6b123bd58 probeCalls=4
--- PASS: TestCorruptToolchainCacheIsRejectedAndRebuiltFromProbe
PASS
```

这组输出分别证明：Git/Go 任一版本改变都会换 digest；等版本换路径并 touch mtime 共触发三轮、
六次实际 version probe 但 digest 不变；平台/架构注入改变 digest 且已有 Receipt 只产生 miss；
定位失败、version 非零和乱码都返回 `VALIDATION_FINGERPRINT_INVALID`；损坏 cache 未被使用，
而是由两工具各重探一次后重建为原语义 digest。

## 3：PowerShell 与 Git Bash 首次互认

PowerShell seed/audit 侧原始环境与结果：

```json
{
  "shell": "PowerShell",
  "gitPath": "C:\\Program Files\\Git\\cmd\\git.exe",
  "gitVersion": "git version 2.48.1.windows.1",
  "tree": "b394323976522d0ca926e6ae358038e207b0074e",
  "identity": "sha256:6ecbb43dc6742e1b2b43e93f2ab554ca951b4efb82051f39f75ee2aa405959cc",
  "toolchainDigest": "sha256:3f056e1a46bcf73e97fc52042f6b529fb64977141f5d10607a63522e000b6b3b",
  "cacheGitPath": "C:\\Program Files\\Git\\cmd\\git.exe",
  "cacheDomain": "toolchainDigest",
  "cacheVersion": 2,
  "platform": "windows",
  "architecture": "amd64",
  "decision": "hit"
}
```

Git Bash 侧直接运行同一 Windows 二进制；原始输出中的路径、版本与判定为：

```text
/mingw64/bin/git
git version 2.48.1.windows.1
/c/Program Files/Go/bin/go
go version go1.26.5 windows/amd64
"ok": true,
"inputDigest": "sha256:6ecbb43dc6742e1b2b43e93f2ab554ca951b4efb82051f39f75ee2aa405959cc",
"toolchainDigest": "sha256:3f056e1a46bcf73e97fc52042f6b529fb64977141f5d10607a63522e000b6b3b",
"hit": true,
"code": "VALIDATION_RECEIPT_HIT",
"receiptID": "sha256:334ec5e420e0d9e81114184102794eb1f4ac2a3ea65de522ea76006c4f824c9c"
```

Git Bash 运行后再由 PowerShell 读取并回查，原始汇总为：

```json
{
  "bashCacheGitPath": "C:\\Program Files\\Git\\mingw64\\bin\\git.exe",
  "bashCacheDigest": "sha256:3f056e1a46bcf73e97fc52042f6b529fb64977141f5d10607a63522e000b6b3b",
  "powerShellGitPath": "C:\\Program Files\\Git\\cmd\\git.exe",
  "powerShellCacheGitPath": "C:\\Program Files\\Git\\cmd\\git.exe",
  "powerShellCacheDigest": "sha256:3f056e1a46bcf73e97fc52042f6b529fb64977141f5d10607a63522e000b6b3b",
  "identity": "sha256:6ecbb43dc6742e1b2b43e93f2ab554ca951b4efb82051f39f75ee2aa405959cc",
  "decisionHit": true,
  "decisionCode": "VALIDATION_RECEIPT_HIT",
  "exit": 0
}
```

两侧路径确实不同并各自使 cache 重探；版本语义一致，因此 digest/identity 不变并命中同一
Receipt。这是本限制修复前无法成立的核心正例。

## 7：存量 v1 Receipt 在 v2 下是普通 miss

开工前保留的 v1 二进制：

```text
C:\Users\24322\AppData\Local\Temp\aicoding-0032-v1.exe
SHA256 E5DCDEEFBC93BAE0271BE874D4954ECFE63A6C4FD2DA47FC1416F500A0D6FBAD
```

它先对同一 staged tree 冷跑：

```json
{
  "status": "PASS",
  "profile": "smoke",
  "execution_mode": "executed",
  "subject_mode": "index",
  "subject_tree_oid": "b394323976522d0ca926e6ae358038e207b0074e",
  "validation_identity": "sha256:730cf8f3e2bab6c913301d536f340581e9439a4e1c682d0e5a4dfefc7597b7fb",
  "toolchain_digest": "sha256:69698975d987cdcdf2eafcc822fcb40dacc605c5147e9d526e7f5b63f0196f8c",
  "receipt_id": "sha256:b35a0bcc7b8d118738fa2b0799183def905c089999c1d121f01c1067cbe4518c",
  "total": 71,
  "pass": 49,
  "fail": 0,
  "warn": 0,
  "skip": 22
}
```

随后 v2 二进制在完全相同 tree 上执行 `test --profile Smoke --verify-reuse`：

```json
{
  "status": "PASS",
  "execution_mode": "executed",
  "subject_tree_oid": "b394323976522d0ca926e6ae358038e207b0074e",
  "validation_identity": "sha256:6ecbb43dc6742e1b2b43e93f2ab554ca951b4efb82051f39f75ee2aa405959cc",
  "receipt_invalid_reason": "VALIDATION_RECEIPT_MISS: no reusable Receipt exists",
  "validation_code": "",
  "receipt_id": "sha256:334ec5e420e0d9e81114184102794eb1f4ac2a3ea65de522ea76006c4f824c9c",
  "total": 71,
  "pass": 49,
  "fail": 0,
  "warn": 0,
  "skip": 22
}
```

结果没有 `VALIDATION_RECEIPT_INVALID`、`VALIDATION_REUSE_AUDIT_MISMATCH` 或伪 corruption；
v2 全量执行通过并发布自己的 Receipt。

同一语义也由定向 testengine 回归锁定：

```text
=== RUN   TestVerifyReuseTreatsV1ToolchainReceiptAsOrdinaryMiss
    evidence_test.go:215: v1-receipt=sha256:618f38bddd23a022e4fd1aeff29da5737bd012af3acf483c6892bf7ecdb21167 v2-identity=sha256:ee2bc95ea8c0575ec4d981106aaeb0ae38bafb000decf5857e07dd68c860a365 verify-reuse=PASS miss=VALIDATION_RECEIPT_MISS: no reusable Receipt exists
--- PASS: TestVerifyReuseTreatsV1ToolchainReceiptAsOrdinaryMiss
PASS
```
