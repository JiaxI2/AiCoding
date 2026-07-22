# 复利知识架构（方向草稿）

Status: Draft

> 本文是**方向锚点**，不定义契约、不承诺实现。它固定"复利机制该长什么样"的
> 设计边界，供后续 ADR 落骨架。与任何 Accepted/Frozen 文档冲突时以后者为准。
> 落地前必须由一份 ADR 回答第 6 节的开放问题。

## 1. 问题

四象限演进观（[00-vision](00-vision.md) §3）已经定义了"已知/未知"四格与
"地基只进不出"的单向沉淀。缺的是**让这套观念产生复利的机制**：
每一次使用如何沉淀出可被下次复用的事实，使仓库"越用越值钱"。

## 2. 第一性：复利 = 不可变事实沿四象限单向晋升

复利不是"存更多东西"，是**让每次使用产生的不可变、内容寻址事实，
沿四象限从低象限晋升到高象限，高象限只进不出**。

仓库已经有三条复利流在跑，只是未被命名：

```text
证据复利   每次验证 → Receipt（内容寻址·不可变）  → 同内容下次零成本复用
方法复利   每次决策 → attempts.jsonl / plan tree   → 可审计·可回放
能力复利   每次扩展 → kit 与既有 Primitive 组合    → 组合数增长
```

四象限是这三条流的**晋升阶梯**：

```text
未知的未知  →  未知的已知  →  已知的未知  →  已知的已知（地基）
(不预测)      (生态已验证)    (有触发条件)     (冻结·门禁·只进不出)
              吸收进来        规划落地          晋升冻结
```

**复利发生在"晋升"这个动作上**：一个事实每被验证一次、每被复用一次，
就更靠近地基；到达地基后被冻结，成为下一层能力的免费基础。

## 3. 知识载体：外部 Obsidian vault，单独管理（不嵌入 AiCoding）

**核心决定（owner）：知识库基于 Obsidian，像 Codex skill 一样单独管理，
不整体集成进 AiCoding。**

这与全仓库的单一权威纪律精确一致：

```text
Obsidian vault    知识的权威源。独立仓库/目录，自己演进。
                  用双链（wiki）组织，不用 RAG。
AiCoding          用 pinned reference 引用它（见 KIT_PLUGIN_VIEW / 0026 pinned 注册）。
                  不拥有、不嵌入、不复制正文。
```

**为什么不 RAG**：RAG 是*检索*（从文本捞相关段落），复利是*累积 + 晋升*。
双链天然是知识图谱——一条 note 被多少链引用、被多少 Receipt 佐证，
本身就是它该不该晋升的信号，比向量相似度更贴合"晋升"这个动作。

**为什么单独管理**：知识一旦嵌进 AiCoding 就成第二事实源，要维护一致性——
正是本仓库一路消灭的东西。外部 vault + pin 引用后，
Obsidian 随便怎么重组，AiCoding 的复利机制不受影响（它引用的是 pinned 版本 + Receipt，
不是易变正文）。

## 4. AiCoding 侧只加一样：promotion-ledger（晋升账本）

**不建知识库，建晋升账本。** 追加不可变（与 attempts.jsonl 同纪律），
只记**引用**与**晋升事件**，永不复制知识正文：

```text
<git-common-dir>/aicoding/promotion-ledger.jsonl
{ "noteRef": { "vaultPin": "sha256:...", "notePath": "..." },
  "from": "known-unknown", "to": "foundation",
  "evidence": ["receipt:sha256:...", "reuse-count:7", "profile:Full"],
  "promotedAt": "...", "promotedBy": "human" }
```

- `noteRef` 是**引用**（vault 的 pin + note 路径），不是内容拷贝；
- `evidence` 只引用既有事实（Receipt ID、复用次数、验证 profile）；
- ledger 是复利的**可审计轨迹**，不是第二数据源。

## 5. 晋升必须有门禁（复利纪律的第三次应用）

晋升不是随手标记，是过门禁的动作——沿用已验证的两条纪律：

```text
validation evidence:  三次远端绿灯才晋级默认复用（ADR 0007 §5）
loop engineering:      stop-satisfied 才算完成（requiredGates 有 Receipt）
compounding（本文）:    满足晋升条件才从"已知的未知"进入"地基"
```

**从 foundation 象限出来是禁止的**（地基只进不出）——与冻结面同一约束：
改地基走 ADR + 三条件。

## 6. 开放问题（落地 ADR 必须先回答）

本文**不预答**以下问题，它们决定实现骨架：

1. **晋升到地基的充要条件是什么？** 引用数阈值？Receipt 佐证数？必须过 Full？
   人工批准是否必需？（倾向：人工批准 + 至少一份 Receipt 佐证，防自动晋升失控）
2. **降级/失效如何处理？** 地基只进不出，但如果佐证它的 Receipt 因内容变化失效了呢？
   （倾向：ledger 记失效事件但不删晋升史；晋升的是"当时凭这些证据成立"的事实）
3. **vault pin 前移时，ledger 里的 noteRef 如何跟随？** 内容变了还算同一条知识吗？
4. **promotion-ledger 是否进 capability / usage view 投影？** 让复利可见。
5. **与 0026 pinned 注册的关系**：vault 是不是就是一个 `kind: content` 的 pinned kit？
   （倾向：是。vault = 一个外部 pinned 能力，复用 0026 的注册机制，不新建引用类型）

## 7. 明确不做

- **不 RAG / 不向量库 / 不 embedding 存储**。
- **不把 Obsidian 集成进 AiCoding**（外部单独管理，pin 引用）。
- **不在 AiCoding 存知识正文**（ledger 只记引用与晋升事件）。
- **不做自动晋升**（自动改自身认知 = hill-climbing，已三次拒绝）。
- **不建第二事实源**（vault 是权威，ledger 是轨迹，Receipt 是佐证——各有其一）。
- 本文落地前不写任何实现——先 ADR 回答第 6 节。

## 8. 一句话

> 复利 = 让每次使用产生的不可变事实，沿四象限单向晋升、门禁保证只进不出、
> 账本保证可审计；知识正文活在外部 Obsidian vault（单独管理、pin 引用），
> AiCoding 只记"哪条知识凭哪份证据晋升到了哪个象限"。全程零嵌入、零第二数据源。
