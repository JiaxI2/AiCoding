# PDF 转换与完整性报告

## 结论

最终可检索文档为
`huawei-c-language-programming-standard-dkba-2826-2011-5.md`。它与原 PDF 对照结果为：

- PDF 61 页，Markdown 保留 60 个正文分页标记；
- 0—16 章全部存在；
- 原 PDF 139 条“原则/规则/建议”全部存在；
- 缺失条款 0，额外条款 0；
- 240 个 C 示例代码块已经围栏化；
- 重复保密页眉页脚已移除；
- 第 8 章全部 14 条编号内容可按标题直接检索。

## 官方工具路径

转换版本固定为 Microsoft MarkItDown `0.1.6`。仓库保留 Docker 构建入口：

```powershell
docker build --tag cstylekit-markitdown:v0.1.6 `
    --file ./tools/pdf-reference/Dockerfile.markitdown `
    ./tools/pdf-reference
```

本次实际验证中 Docker Desktop 引擎成功启动；从官方 Git tag 直接构建及从本地 Dockerfile 构建都在
镜像/依赖下载阶段长时间没有进展，因此安全终止，未声称得到 Docker 镜像。随后使用同一个官方包版本
完成转换：

```powershell
uvx --from 'markitdown[pdf]==0.1.6' markitdown `
    ./references/huawei-c-language-programming-standard-dkba-2826-2011-5.pdf `
    -o ./references/huawei-c-language-programming-standard-dkba-2826-2011-5.raw.md
```

这保留了“官方转换器版本一致”和“Docker 入口可复现”两项证据，同时没有掩盖本次 Docker 下载阻塞。

## 规范化与验证

```powershell
python ./tools/pdf-reference/normalize_reference.py
uv run --with pdfplumber==0.11.7 python ./tools/pdf-reference/verify_reference.py `
    --pdf ./references/huawei-c-language-programming-standard-dkba-2826-2011-5.pdf `
    --markdown ./references/huawei-c-language-programming-standard-dkba-2826-2011-5.md
```

规范化只删除重复页眉页脚、合并断行、提升章节/条款标题和围栏化代码；原始 MarkItDown 输出仍作为
`*.raw.md` 保存，便于审计归一化前后的差异。

## 分发状态

PDF、原始 Markdown 和规范化 Markdown 按用户授权作为 C Kit 的正式参考资产随受管发行包发布。
