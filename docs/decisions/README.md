# Architecture Decisions

Repository maintainers record durable decisions and their rollback boundary here.

Use one numbered ADR per decision; keep implementation history in Git rather than
parallel versioned documents.

Authority: [AiCoding Primitive Constitution](https://github.com/JiaxI2/AiCoding/blob/main/docs/architecture/PRIMITIVE_CONSTITUTION.md).

Current numbered decisions:

- [ADR 0010: Kit 内容钉死引用注册](0010-pinned-reference-registration.md)
- [ADR 0011: pathpolicy 解析收敛与 policy schema 闭合](0011-pathpolicy-consolidation.md)
- [ADR 0012: `--profile` 词汇表正交化与子命令目录冻结](0012-profile-vocabulary-and-subcommand-catalog.md)
- [ADR 0013: Release 复用登记并发包 raceScope](0013-release-race-scope.md)
- [ADR 0014: `--reuse` 默认值晋级为 `auto`](0014-reuse-default-auto.md)
