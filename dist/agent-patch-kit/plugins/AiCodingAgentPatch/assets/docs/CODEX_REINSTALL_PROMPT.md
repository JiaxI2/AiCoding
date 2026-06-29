# Codex reinstall prompt for Agent Patch Kit v0.2.2

Use this prompt when v0.2.1 was installed with `pip install -e` and the original kit source directory was deleted.

```text
请使用我已经提供给你的 Agent Patch Kit v0.2.2 文件，修复当前 Agent Patch Kit 安装并重新部署到当前 AiCoding 仓库。

背景：
- 当前 v0.2.1 可能是 editable pip 安装，指向已删除的源码目录，导致 apatch 报 ModuleNotFoundError: No module named 'agent_patch'。
- v0.2.2 默认必须使用非 editable 用户安装，安装后允许删除原始 zip 和解压目录。
- 不要再使用 pip install -e，除非我明确要求开发模式。

执行要求：
1. 使用 Windows-native / PowerShell 7，不引入 WSL。
2. 修改前运行：git status --short。
3. 新建或切换分支：fix/agent-patch-kit-v2.2-install。
4. 解压我提供的 Agent Patch Kit v0.2.2 到临时目录；临时目录不要提交。
5. 在 v0.2.2 Kit 根目录执行修复安装：
   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\repair-agent-patch-kit.ps1
6. 如果 repair 脚本不可用，则手动执行：
   python -m pip uninstall -y agent-patch-kit
   python -m pip install --force-reinstall .
7. 重新部署到当前 AiCoding 仓库，优先 repo-scoped skill：
   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1 -DeployScope project -ProjectRoot <当前仓库根目录> -Agent both -WriteAgentsSnippet
8. 如果存在 integrations\aicoding\install-to-aicoding.ps1，执行 repo-skill 集成。
9. 如果存在 integrations\aicoding\package-marketplace.ps1，生成 marketplace sidecar；只有确认 .agents/plugins/marketplace.json 是当前仓库管理文件后才 merge。
10. 验证：
    apatch --version
    apatch install doctor
    apatch doctor
    apatch brief --format md
    apatch state status
    git diff --check
    git diff --stat
11. 确认 apatch install doctor 显示：
    - install_mode: non-editable / user mode
    - bundle_assets: OK
    - 原始 zip / 解压目录可以删除
12. 提交：
    git add 相关文件
    git commit -m "fix: install Agent Patch Kit v2.2 as self-contained CLI"
13. 推送：
    git push -u origin fix/agent-patch-kit-v2.2-install
14. 最后报告：
    - 当前 apatch 是否完全生效
    - 是否仍是 editable install
    - 部署了哪些路径
    - 修改了哪些文件
    - marketplace 是否修改
    - 验证结果
    - 远程分支名称
```
