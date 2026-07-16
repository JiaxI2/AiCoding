# Agent Decisions

## AiCoding Agent Dev Kit Plan Mode Overlay

Decision Status: Selected

Selected option: integrate the v0.4 Plan Mode overlay through AiCoding-owned repository files and the existing hook registry.

Constraints:

- Do not create a new branch.
- Do not edit `CodingKit/agents/skills`.
- Do not modify Codex plugin cache files.
- Do not replace `scripts/aicoding-kit.ps1`.
- Keep one hook bridge through AiCoding-owned scripts and registry metadata.

## C UserStyle Kit 1.2.0 Integration

Decision Status: Selected

Selected option: keep C UserStyle Kit as a self-contained Go module under
`CodingKit/tools/c-userstyle-kit`, register it as an external CLI Kit, and expose fast/full
verification only through the existing `c99-standard-c` Skill route.

Constraints:

- Do not edit `CodingKit/agents/skills`, generated plugins, Marketplace, or plugin caches.
- Do not create a second top-level C formatting or lint command.
- Exclude local build/state and obsolete direct-integration drafts from the snapshot.
- The user explicitly authorized the PDF and Markdown reference copies for public release.
- Release the user-visible platform integration as SemVer minor `v0.8.0`.

## Dependency Direction And Stable Identity Governance

Decision Status: Selected

Selected rule: dependencies may point only from a higher layer to the same or a lower layer:

```text
platform -> integration -> capability -> runtime
```

Constraints:

- A lower layer must not depend on or observe an upper-layer product namespace.
- `aicoding-*` is reserved for genuine platform/integration assets, not capabilities merely distributed by AiCoding.
- Generic MCP capability servers do not own workflow prompts; workflow orchestration belongs to Skills.
- Stable asset names, paths, packages, symbols, models and runtime code do not encode the asset version.
- Versions are controlled by manifest metadata, asset documentation, changelog, Tag/Release URLs and exact-authority README badges.
- Existing reverse names and self-version code are corrected immediately rather than registered as deferred debt.
