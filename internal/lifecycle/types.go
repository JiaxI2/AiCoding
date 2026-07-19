package lifecycle

const (
	ScopeAll          = "all"
	ScopeKit          = "kit"
	ScopeMCP          = "mcp"
	ScopeRuntimeSkill = "runtime-skill"
	ScopeRepoContext  = "repo-context"
)

type Options struct {
	Action            string
	Scope             string
	All               bool
	KitID             string
	ComponentID       string
	CodexConfig       string
	VerifyProfile     string
	IncludeConfigured bool
	DryRun            bool
	RuntimeProfile    string
	RuntimeSkill      string
	SourceRepository  string
	StandaloneRoot    string
	MigrateUnmanaged  bool
}

type Report struct {
	SchemaVersion int             `json:"schemaVersion"`
	Action        string          `json:"action"`
	Mode          string          `json:"mode"`
	Scope         string          `json:"scope"`
	DryRun        bool            `json:"dryRun"`
	CatalogDigest string          `json:"catalogDigest"`
	PlanDigest    string          `json:"planDigest"`
	OK            bool            `json:"ok"`
	Summary       Summary         `json:"summary"`
	Adapters      []AdapterResult `json:"adapters"`
	Warnings      []string        `json:"warnings,omitempty"`
	Errors        []string        `json:"errors,omitempty"`
}

type Summary struct {
	Total    int `json:"total"`
	OK       int `json:"ok"`
	Failed   int `json:"failed"`
	Warnings int `json:"warnings"`
}

type AdapterResult struct {
	ID          string      `json:"id"`
	Action      string      `json:"action"`
	DryRun      bool        `json:"dryRun"`
	InputDigest string      `json:"inputDigest,omitempty"`
	OK          bool        `json:"ok"`
	Status      string      `json:"status"`
	Data        interface{} `json:"data,omitempty"`
	Warnings    []string    `json:"warnings,omitempty"`
	Errors      []string    `json:"errors,omitempty"`
}
