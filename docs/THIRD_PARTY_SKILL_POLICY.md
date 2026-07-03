# Third-Party Skill Policy

第三方 Skill 不是 AiCoding canonical source。MVP 阶段只处理三件事：下载过程可观察、安装状态可查询、完整仓库型 Skill 保留 repo 根目录依赖文件。

## Source Registry

Sources are tracked in `config/skill-sources.json`:

```json
{
  "schemaVersion": 1,
  "sources": [
    {
      "name": "ppt-master",
      "skill": "ppt-master",
      "type": "git",
      "url": "https://github.com/hugohe3/ppt-master.git",
      "branch": "main",
      "trust": "third-party",
      "updatePolicy": "manual",
      "layout": "external-repo",
      "skillPath": "skills/ppt-master",
      "repoRequired": true,
      "dependencyFile": "requirements.txt",
      "targets": ["CodexUser", "RepoLocal"]
    }
  ]
}
```

## Commands

旧 action 保持兼容：

```powershell
pwsh scripts/aicoding-skill.ps1 sources -Json
pwsh scripts/aicoding-skill.ps1 add-source -Name example -Url https://github.com/example/skills.git
pwsh scripts/aicoding-skill.ps1 download -Source example -Skill skill-id
pwsh scripts/aicoding-skill.ps1 verify -Skill skill-id -Json
pwsh scripts/aicoding-skill.ps1 install -Skill skill-id -Json
pwsh scripts/aicoding-skill.ps1 update -Skill skill-id -Pin tag-or-commit -Json
pwsh scripts/aicoding-skill.ps1 remove -Skill skill-id -Force -Json
```

External MVP 只暴露三个主流程 action：

```powershell
pwsh scripts/aicoding-skill.ps1 install-external -Source ppt-master -Target CodexUser -Json
pwsh scripts/aicoding-skill.ps1 status-external -Source ppt-master -Json
pwsh scripts/aicoding-skill.ps1 verify-external -Source ppt-master -Target CodexUser -Json
```

如果 builtin quick audit 返回 `warn`，默认停止并提示：

```text
audit returned warn; re-run with -AllowWarn to install
```

用户确认后可继续：

```powershell
pwsh scripts/aicoding-skill.ps1 install-external -Source ppt-master -Target CodexUser -AllowWarn -Json
```

## Directory Protocol

```text
.aicoding/skill-cache/external/<source>/repo/
.aicoding/skill-cache/external/<source>/install-state.json
.aicoding/skill-cache/external/<source>/install-log.ndjson
.aicoding/skill-cache/external/<source>/audit-report.json
```

`install-external` 下载完整 repo 到 external cache，状态先记为 `trust=pending`。只有 audit 为 `pass`，或 audit 为 `warn` 且用户传入 `-AllowWarn`，才会把 `skillPath` 复制到目标运行入口。

`status-external` 输出 `stage`、`trust`、`auditStatus`、`auditScore`、`elapsedSec`、`lastCommand`、`lastExitCode`、cache/target 路径和最近 20 行 `install-log.ndjson`。

## Rules

- Do not download directly into `.agents/skills`.
- Do not enable an unverified third-party skill.
- Do not auto-update third-party skills.
- Updates must be pinned to a commit or tag.
- Unknown or untrusted sources remain in external cache with `trust=pending`.
- A third-party skill can become AiCoding-owned only through fork or copy-as-new, new id if needed, origin record, manifest update, verification, docs, and changelog.
