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
