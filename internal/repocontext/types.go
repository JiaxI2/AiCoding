// Package repocontext scans a repository into deterministic facts and generates
// scoped context files as lifecycle-managed owned assets. See ADR 0003
// (docs/decisions/0003-repo-context-domain.md) and docs/architecture/07-roadmap.md
// §3 for the domain design; the kernel modules are not modified by this domain.
package repocontext

// ownedRoot is the managed directory that holds every generated artifact and the
// manifest. Nothing outside this root is ever written or deleted.
const ownedRoot = ".aicoding/repo-context"

// manifestRel is the manifest path relative to the repository root.
const manifestRel = ownedRoot + "/manifest.json"

// Facts is the normalized, deterministic snapshot of repository facts. All slices
// are sorted; no absolute paths or timestamps enter the value so the digest is
// stable across machines and runs.
type Facts struct {
	Repo       string         `json:"repo"`
	Languages  []LanguageStat `json:"languages"`
	Toolchains []string       `json:"toolchains"`
	Domains    []Domain       `json:"domains"`
}

// LanguageStat records how many tracked files share a source extension.
type LanguageStat struct {
	Language  string `json:"language"`
	Extension string `json:"extension"`
	Files     int    `json:"files"`
}

// Domain is a top-level directory treated as a scoped context unit.
type Domain struct {
	Path            string `json:"path"`
	Files           int    `json:"files"`
	PrimaryLanguage string `json:"primaryLanguage"`
}

// ManifestFile records one generated artifact and the digest of its content, so
// uninstall only deletes files it generated and doctor can detect tampering.
type ManifestFile struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// Manifest is the owned-state record written under ownedRoot. It carries the
// facts digest the artifacts were generated from and the per-file digests. It
// deliberately holds no timestamp so its own content stays deterministic.
type Manifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	FactsDigest   string         `json:"factsDigest"`
	Files         []ManifestFile `json:"files"`
}

// PlannedFile is a file that install/update would write, used by dry-run plans.
type PlannedFile struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Change string `json:"change"` // create, update, unchanged
}

// Report is the typed result every domain action returns. It is carried as the
// lifecycle AdapterResult.Data payload.
type Report struct {
	Action      string        `json:"action"`
	DryRun      bool          `json:"dryRun"`
	OK          bool          `json:"ok"`
	Status      string        `json:"status"`
	FactsDigest string        `json:"factsDigest"`
	Installed   bool          `json:"installed"`
	Fresh       bool          `json:"fresh"`
	Planned     []PlannedFile `json:"planned,omitempty"`
	Files       []string      `json:"files,omitempty"`
	Warnings    []string      `json:"warnings,omitempty"`
	Errors      []string      `json:"errors,omitempty"`
}
