# C 编程规范参考资料

## 华为 C 语言编程规范

- 首选检索文档：[huawei-c-language-programming-standard-dkba-2826-2011-5.md](huawei-c-language-programming-standard-dkba-2826-2011-5.md)
- 完整 PDF 副本：[huawei-c-language-programming-standard-dkba-2826-2011-5.pdf](huawei-c-language-programming-standard-dkba-2826-2011-5.pdf)
- MarkItDown 原始输出：[huawei-c-language-programming-standard-dkba-2826-2011-5.raw.md](huawei-c-language-programming-standard-dkba-2826-2011-5.raw.md)
- 文档编号：`DKBA 2826-2011.5`
- 文档日期：`2011-05-24`
- 文档页数：61 页
- SHA-256：`80D23AC9CACB83AEBAA1C28889271F744D5866CA45D09266533895F256262200`
- 转换工具：Microsoft MarkItDown `0.1.6`；Docker 构建入口为 [`Dockerfile.markitdown`](../tools/pdf-reference/Dockerfile.markitdown)。
- 规范化工具：[`normalize_reference.py`](../tools/pdf-reference/normalize_reference.py)。
- 完整性检查：[`verify_reference.py`](../tools/pdf-reference/verify_reference.py)。
- 转换与完整性报告：[`CONVERSION_REPORT.md`](CONVERSION_REPORT.md)。
- 用途：作为三个黄金模块、规则目录、JSON 配置、生成器、lint、编译门禁和测试的审核依据。

最终 Markdown 已与原 PDF 独立比较：61 页、0 至 16 章、139 条“原则/规则/建议”全部存在，缺失条款和额外条款均为 0；重复页眉页脚已移除，第 8 章条款可直接按标题检索。

该 PDF 是用户提供的完整、未修改参考副本。规则冲突仍按照正确性与安全性、用户明确要求、当前 C99 Skill、项目配置和通用格式偏好的优先级处理，不能仅以本 PDF 替代实际项目约束。

## 分发状态

PDF、规范化 Markdown 与 raw 转换件按用户授权作为 C Kit 的正式参考资产随受管发行包发布。
