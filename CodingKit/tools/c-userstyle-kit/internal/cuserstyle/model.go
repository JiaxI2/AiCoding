package cuserstyle

type Config struct {
	Schema      string      `json:"schema"`
	ID          string      `json:"id"`
	Standard    string      `json:"standard"`
	Reference   Reference   `json:"reference"`
	Scope       Scope       `json:"scope"`
	Style       Style       `json:"style"`
	Naming      Naming      `json:"naming"`
	Docs        Docs        `json:"documentation"`
	Safety      Safety      `json:"safety"`
	Flow        Flow        `json:"controlFlow"`
	Macro       Macro       `json:"macros"`
	Readability Readability `json:"readability"`
	Template    Template    `json:"template"`
	Gates       Gates       `json:"gates"`
	Hook        Hook        `json:"hook"`
	Agent       Agent       `json:"agent"`
}

type Scope struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type Style struct {
	IndentWidth       int    `json:"indentWidth"`
	Continuation      int    `json:"continuationIndentWidth"`
	ColumnLimit       int    `json:"columnLimit"`
	BraceStyle        string `json:"braceStyle"`
	PointerAlignment  string `json:"pointerAlignment"`
	Newline           string `json:"newline"`
	Encoding          string `json:"encoding"`
	ChangedLinesOnly  bool   `json:"changedLinesOnly"`
	RequireEOFNewline bool   `json:"requireEOFNewline"`
	ForbidLineComment bool   `json:"forbidLineComment"`
}

type Naming struct {
	ModulePrefix     string `json:"modulePrefix"`
	PublicFunction   string `json:"publicFunction"`
	PrivateFunction  string `json:"privateFunction"`
	Type             string `json:"type"`
	FunctionPointer  string `json:"functionPointer"`
	Macro            string `json:"macro"`
	EnumConstant     string `json:"enumConstant"`
	GlobalVariable   string `json:"globalVariable"`
	StaticVariable   string `json:"staticVariable"`
	LocalVariable    string `json:"localVariable"`
	Parameter        string `json:"parameter"`
	StructMember     string `json:"structMember"`
	ForbidReservedID bool   `json:"forbidReservedIdentifier"`
}

type Docs struct {
	FileHeader                  bool   `json:"fileHeader"`
	RequireFileMetadata         bool   `json:"requireFileMetadata"`
	AllFunctions                bool   `json:"allFunctions"`
	Types                       bool   `json:"types"`
	Macros                      bool   `json:"macros"`
	RequireBrief                bool   `json:"requireBrief"`
	RequireParamTags            bool   `json:"requireParamTags"`
	RequireParamDirection       bool   `json:"requireParamDirection"`
	RequireReturnTag            bool   `json:"requireReturnTag"`
	RequireDefinitionDetails    bool   `json:"requireDefinitionDetails"`
	RequirePublicPerformance    bool   `json:"requirePublicPerformance"`
	RequirePublicReentrancy     bool   `json:"requirePublicReentrancy"`
	RequireBarePrivatePrototype bool   `json:"requireBarePrivatePrototype"`
	RequireCaseIntentComment    bool   `json:"requireCaseIntentComment"`
	RequireGlobalVariableDetail bool   `json:"requireGlobalVariableDetail"`
	RequireExternC              bool   `json:"requireExternC"`
	EmployeeIDPolicy            string `json:"employeeIdPolicy"`
	ModificationHistoryPolicy   string `json:"modificationHistoryPolicy"`
}

type Reference struct {
	Title           string `json:"title"`
	PDF             string `json:"pdf"`
	Markdown        string `json:"markdown"`
	RuleCatalog     string `json:"ruleCatalog"`
	SHA256          string `json:"sha256"`
	Pages           int    `json:"pages"`
	ExpectedClauses int    `json:"expectedClauses"`
}

type GateProfile struct {
	LanguageStandard string   `json:"languageStandard"`
	WarningsAsErrors bool     `json:"warningsAsErrors"`
	Flags            []string `json:"flags"`
}

type Gates struct {
	GCC          GateProfile `json:"gcc"`
	Clang        GateProfile `json:"clang"`
	HeaderC      GateProfile `json:"headerC"`
	HeaderCXX    GateProfile `json:"headerCxx"`
	RunUnitTests bool        `json:"runUnitTests"`
	LintFixtures bool        `json:"lintFixtures"`
}

type Safety struct {
	ForbidDynamicAllocation bool     `json:"forbidDynamicAllocation"`
	ForbidVLA               bool     `json:"forbidVLA"`
	RequireStdintTypes      bool     `json:"requireStdintTypes"`
	RequireExplicitVoid     bool     `json:"requireExplicitVoid"`
	ForbidUnboundedLoop     bool     `json:"forbidUnboundedLoop"`
	ForbidGoto              bool     `json:"forbidGoto"`
	ForbiddenCalls          []string `json:"forbiddenCalls"`
	RequirePointerNullCheck bool     `json:"requirePointerNullComparison"`
	ForbidBooleanComparison bool     `json:"forbidBooleanLiteralComparison"`
	PreferPreIncrement      bool     `json:"preferPreIncrement"`
	RequireSizeT            bool     `json:"requireSizeTForSize"`
	MaxParameters           int      `json:"maxParameters"`
}

type Flow struct {
	RequireCompoundBraces bool `json:"requireCompoundBraces"`
	RequireSwitchDefault  bool `json:"requireSwitchDefault"`
	RequireCaseBreak      bool `json:"requireCaseBreak"`
	ForbidTernaryCall     bool `json:"forbidStandaloneTernaryCall"`
}

type Macro struct {
	RequireUppercase         bool   `json:"requireUppercase"`
	ProtectParameters        bool   `json:"protectParameters"`
	ProtectFinalExpression   bool   `json:"protectFinalExpression"`
	MultiStatementDoWhile    bool   `json:"multiStatementDoWhile"`
	RequireDocumentation     bool   `json:"requireDocumentation"`
	SimpleObjectCommentStyle string `json:"simpleObjectCommentStyle"`
}

type Readability struct {
	ComplexFunction                       ComplexFunctionPolicy `json:"complexFunction"`
	RequireNumberedIntentCommentPlacement bool                  `json:"requireNumberedIntentCommentPlacement"`
	ReportNonObviousBranches              bool                  `json:"reportNonObviousBranches"`
	ReportSingleCallStaticHelpers         bool                  `json:"reportSingleCallStaticHelpers"`
}

type ComplexFunctionPolicy struct {
	MinEffectiveLines   int  `json:"minEffectiveLines"`
	MinBranches         int  `json:"minBranches"`
	MinNesting          int  `json:"minNesting"`
	RequireNumberedFlow bool `json:"requireNumberedFlow"`
}

type Template struct {
	ModuleName  string `json:"moduleName"`
	FileStem    string `json:"fileStem"`
	Brief       string `json:"brief"`
	Details     string `json:"details"`
	ContextType string `json:"contextType"`
	HeaderGuard string `json:"headerGuard"`
}

type Hook struct {
	Enabled           bool   `json:"enabled"`
	Scope             string `json:"scope"`
	FailLevel         string `json:"failLevel"`
	MaxDiagnostics    int    `json:"maxDiagnostics"`
	NoCChangeFastExit bool   `json:"noCChangeFastExit"`
}

type Agent struct {
	ModifyOnlyFunctionBody bool `json:"modifyOnlyFunctionBody"`
	PreservePrototype      bool `json:"preservePrototype"`
	PreserveDocumentation  bool `json:"preserveDocumentation"`
	BoundedExecution       bool `json:"boundedExecutionRequired"`
	RequireBuild           bool `json:"requireBuild"`
	RequireTests           bool `json:"requireTests"`
}

type Diagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type FileReadabilitySummary struct {
	File         string                `json:"file"`
	Analyzer     string                `json:"analyzer"`
	Functions    []FunctionReadability `json:"functions"`
	ManualReview []ReviewItem          `json:"manualReview"`
}

type FunctionReadability struct {
	Name                   string   `json:"name"`
	Line                   int      `json:"line"`
	Static                 bool     `json:"static"`
	EffectiveLines         int      `json:"effectiveLines"`
	BranchCount            int      `json:"branchCount"`
	MaxNesting             int      `json:"maxNesting"`
	FanOut                 int      `json:"fanOut"`
	Callees                []string `json:"callees"`
	IncomingDirectCalls    int      `json:"incomingDirectCalls"`
	Complex                bool     `json:"complex"`
	NumberedFlowDocumented bool     `json:"numberedFlowDocumented"`
	SingleCallStaticHelper bool     `json:"singleCallStaticHelper"`
}

type ReviewItem struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function,omitempty"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
}

type LineRange struct {
	Start int
	End   int
}
