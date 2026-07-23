package cli

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/JiaxI2/AiCoding/internal/registry"
	"github.com/JiaxI2/AiCoding/internal/report"
)

type CommandID string

type SubcommandID string

type LatencyClass string

const (
	LatencyFast     LatencyClass = "fast"
	LatencyStandard LatencyClass = "standard"
	LatencyWork     LatencyClass = "work"
)

func (c LatencyClass) BudgetMS() int64 {
	switch c {
	case LatencyFast:
		return 400
	case LatencyStandard:
		return 1200
	default:
		return 0
	}
}

const (
	CommandHelp       CommandID = "help"
	CommandVersion    CommandID = "version"
	CommandHook       CommandID = "hook"
	CommandBootstrap  CommandID = "bootstrap"
	CommandTest       CommandID = "test"
	CommandValidation CommandID = "validation"
	CommandDocSync    CommandID = "docsync"
	CommandSkill      CommandID = "skill"
	CommandLifecycle  CommandID = "lifecycle"
	CommandExport     CommandID = "export"
	CommandFreshClone CommandID = "fresh-clone"
	CommandCache      CommandID = "cache"
	CommandCapability CommandID = "capability"
	CommandCodex      CommandID = "codex"
	CommandMCP        CommandID = "mcp"
	CommandTag        CommandID = "tag"
	CommandRelease    CommandID = "release"
	CommandKit        CommandID = "kit"
	CommandDoctor     CommandID = "doctor"
	CommandVerify     CommandID = "verify"
	CommandGovernance CommandID = "governance"
	CommandPowerShell CommandID = "powershell"
	CommandTodolist   CommandID = "todolist"
	CommandWork       CommandID = "work"
	CommandPlan       CommandID = "plan"
	CommandProvision  CommandID = "provision"
	CommandChange     CommandID = "change"
)

type CommandDescriptor struct {
	ID                 CommandID              `json:"id"`
	Name               string                 `json:"name"`
	Aliases            []string               `json:"aliases,omitempty"`
	RequiresSubcommand bool                   `json:"requiresSubcommand,omitempty"`
	LatencyClass       LatencyClass           `json:"latencyClass"`
	LatencyProbe       []string               `json:"latencyProbe,omitempty"`
	Subcommands        []SubcommandDescriptor `json:"subcommands,omitempty"`
}

type SubcommandDescriptor struct {
	ID          SubcommandID           `json:"id"`
	Name        string                 `json:"name"`
	Aliases     []string               `json:"aliases,omitempty"`
	Subcommands []SubcommandDescriptor `json:"subcommands,omitempty"`
}

const (
	SubHookPreCommit       SubcommandID = "hook.pre-commit"
	SubHookCommitMsg       SubcommandID = "hook.commit-msg"
	SubHookPrePush         SubcommandID = "hook.pre-push"
	SubHookPostCommit      SubcommandID = "hook.post-commit"
	SubTestLatest          SubcommandID = "test.latest"
	SubValidationStatus    SubcommandID = "validation.status"
	SubValidationCheck     SubcommandID = "validation.check"
	SubValidationExplain   SubcommandID = "validation.explain"
	SubValidationList      SubcommandID = "validation.list"
	SubValidationClean     SubcommandID = "validation.clean"
	SubDocSyncStaged       SubcommandID = "docsync.staged"
	SubDocSyncAll          SubcommandID = "docsync.all"
	SubDocSyncCI           SubcommandID = "docsync.ci"
	SubDocSyncRelease      SubcommandID = "docsync.release"
	SubSkillInit           SubcommandID = "skill.init"
	SubSkillVerify         SubcommandID = "skill.verify"
	SubSkillC99            SubcommandID = "skill.c99-standard-c"
	SubSkillC99Status      SubcommandID = "skill.c99-standard-c.status"
	SubSkillC99Templates   SubcommandID = "skill.c99-standard-c.templates"
	SubSkillC99Fmt         SubcommandID = "skill.c99-standard-c.fmt"
	SubSkillC99Check       SubcommandID = "skill.c99-standard-c.check"
	SubSkillC99Verify      SubcommandID = "skill.c99-standard-c.verify"
	SubLifecyclePlan       SubcommandID = "lifecycle.plan"
	SubLifecycleInstall    SubcommandID = "lifecycle.install"
	SubLifecycleUpdate     SubcommandID = "lifecycle.update"
	SubLifecycleUninstall  SubcommandID = "lifecycle.uninstall"
	SubLifecycleStatus     SubcommandID = "lifecycle.status"
	SubLifecycleDoctor     SubcommandID = "lifecycle.doctor"
	SubLifecycleVerify     SubcommandID = "lifecycle.verify"
	SubLifecycleRollback   SubcommandID = "lifecycle.rollback"
	SubCacheStatus         SubcommandID = "cache.status"
	SubCacheClean          SubcommandID = "cache.clean"
	SubCapabilityList      SubcommandID = "capability.list"
	SubCapabilityDescribe  SubcommandID = "capability.describe"
	SubCapabilityIndex     SubcommandID = "capability.index"
	SubCodexUsage          SubcommandID = "codex.usage"
	SubCodexUsageParse     SubcommandID = "codex.usage.parse"
	SubCodexUsageRun       SubcommandID = "codex.usage.run"
	SubMCPInit             SubcommandID = "mcp.init"
	SubMCPList             SubcommandID = "mcp.list"
	SubMCPStatus           SubcommandID = "mcp.status"
	SubMCPDoctor           SubcommandID = "mcp.doctor"
	SubMCPVerify           SubcommandID = "mcp.verify"
	SubTagAudit            SubcommandID = "tag.audit"
	SubReleaseVerify       SubcommandID = "release.verify"
	SubReleaseGate         SubcommandID = "release.gate"
	SubKitList             SubcommandID = "kit.list"
	SubKitInit             SubcommandID = "kit.init"
	SubKitRegister         SubcommandID = "kit.register"
	SubKitPrefetch         SubcommandID = "kit.prefetch"
	SubKitDescribe         SubcommandID = "kit.describe"
	SubKitVerify           SubcommandID = "kit.verify"
	SubKitTest             SubcommandID = "kit.test"
	SubKitDoctor           SubcommandID = "kit.doctor"
	SubDoctorPerf          SubcommandID = "doctor.perf"
	SubDoctorPwsh          SubcommandID = "doctor.pwsh"
	SubDoctorPwshBudget    SubcommandID = "doctor.pwsh-budget"
	SubVerifyHooks         SubcommandID = "verify.hooks"
	SubVerifyRepoText      SubcommandID = "verify.repo-text"
	SubVerifyReleaseNotes  SubcommandID = "verify.release-notes"
	SubGovernanceLint      SubcommandID = "governance.lint"
	SubGovernanceDeps      SubcommandID = "governance.dependencies"
	SubGovernanceLayout    SubcommandID = "governance.layout"
	SubGovernanceReuse     SubcommandID = "governance.reuse"
	SubGovernanceCaps      SubcommandID = "governance.capabilities"
	SubPowerShellRegexLint SubcommandID = "powershell.regex-lint"
	SubWorkValidate        SubcommandID = "work.validate"
	SubWorkNext            SubcommandID = "work.next"
	SubWorkStatus          SubcommandID = "work.status"
	SubWorkRecord          SubcommandID = "work.record"
	SubPlanCheck           SubcommandID = "plan.check"
	SubPlanVerify          SubcommandID = "plan.verify"
	SubPlanStatus          SubcommandID = "plan.status"
	SubPlanApprove         SubcommandID = "plan.approve"
	SubChangeVerify        SubcommandID = "change.verify"
)

type HelpSectionID string

const (
	HelpUsage  HelpSectionID = "usage"
	HelpFormal HelpSectionID = "formal"
	HelpDomain HelpSectionID = "domain"
)

type HelpSection struct {
	ID    HelpSectionID `json:"id"`
	Title string        `json:"title"`
}

type HelpForm struct {
	Command      CommandID      `json:"command"`
	Section      HelpSectionID  `json:"section"`
	Path         []SubcommandID `json:"path,omitempty"`
	Alternatives []SubcommandID `json:"alternatives,omitempty"`
	Usage        string         `json:"usage"`
	entryName    string
	suffix       string
}

type QuickstartForm struct {
	Operation string         `json:"operation"`
	Command   CommandID      `json:"command"`
	Path      []SubcommandID `json:"path,omitempty"`
	Args      []string       `json:"args,omitempty"`
}

type CatalogDescriptor struct {
	Commands    []CommandDescriptor `json:"commands"`
	Sections    []HelpSection       `json:"sections"`
	Help        []HelpForm          `json:"help"`
	Quickstarts []QuickstartForm    `json:"quickstarts,omitempty"`
}

type commandHandler func([]string, time.Time) (report.Result, error)

type directCommand uint8

const (
	directNone directCommand = iota
	directHelp
	directVersion
)

type commandRoute struct {
	descriptor CommandDescriptor
	handler    commandHandler
	direct     directCommand
}

type typedCommandCatalog struct {
	descriptor CatalogDescriptor
	routes     map[string]*commandRoute
	ordered    []commandRoute
}

var (
	catalogSnapshot              registry.Snapshot
	catalogHelpText              string
	commandCatalogEvidenceDigest string
)

func sub(id SubcommandID, name string, children ...SubcommandDescriptor) SubcommandDescriptor {
	return SubcommandDescriptor{ID: id, Name: name, Subcommands: children}
}

func help(command CommandID, section HelpSectionID, suffix string, path ...SubcommandID) HelpForm {
	return HelpForm{Command: command, Section: section, Path: path, suffix: suffix}
}

func helpAny(command CommandID, section HelpSectionID, suffix string, alternatives ...SubcommandID) HelpForm {
	return HelpForm{Command: command, Section: section, Alternatives: alternatives, suffix: suffix}
}

func helpAlias(command CommandID, section HelpSectionID, alias, suffix string) HelpForm {
	return HelpForm{Command: command, Section: section, entryName: alias, suffix: suffix}
}

var productProfileVocabulary = []string{"Smoke", "Full", "Release"}

func productProfileChoice() string {
	return strings.Join(productProfileVocabulary, "|")
}

func productProfileHelp() string {
	return strings.Join(productProfileVocabulary, ", ")
}

func productProfileOption() string {
	return "--profile " + productProfileChoice()
}

var commands = mustCommandCatalog(
	[]commandRoute{
		{descriptor: CommandDescriptor{ID: CommandHelp, Name: "help", Aliases: []string{"--help", "-h"}, LatencyClass: LatencyFast}, direct: directHelp},
		{descriptor: CommandDescriptor{ID: CommandVersion, Name: "version", Aliases: []string{"--version", "-v"}, LatencyClass: LatencyFast}, direct: directVersion},
		{descriptor: CommandDescriptor{ID: CommandHook, Name: "hook", RequiresSubcommand: true, LatencyClass: LatencyWork, Subcommands: []SubcommandDescriptor{
			sub(SubHookPreCommit, "pre-commit"), sub(SubHookCommitMsg, "commit-msg"), sub(SubHookPrePush, "pre-push"), sub(SubHookPostCommit, "post-commit"),
		}}, handler: runHook},
		{descriptor: CommandDescriptor{ID: CommandBootstrap, Name: "bootstrap", LatencyClass: LatencyWork}, handler: runBootstrap},
		{descriptor: CommandDescriptor{ID: CommandTest, Name: "test", LatencyClass: LatencyWork, Subcommands: []SubcommandDescriptor{sub(SubTestLatest, "latest")}}, handler: runTest},
		{descriptor: CommandDescriptor{ID: CommandValidation, Name: "validation", RequiresSubcommand: true, LatencyClass: LatencyFast, LatencyProbe: []string{"list"}, Subcommands: []SubcommandDescriptor{
			sub(SubValidationStatus, "status"), sub(SubValidationCheck, "check"), sub(SubValidationExplain, "explain"), sub(SubValidationList, "list"), sub(SubValidationClean, "clean"),
		}}, handler: runValidation},
		{descriptor: CommandDescriptor{ID: CommandDocSync, Name: "docsync", RequiresSubcommand: true, LatencyClass: LatencyWork, Subcommands: []SubcommandDescriptor{
			sub(SubDocSyncStaged, "staged"), sub(SubDocSyncAll, "all"), sub(SubDocSyncCI, "ci"), sub(SubDocSyncRelease, "release"),
		}}, handler: runDocSync},
		{descriptor: CommandDescriptor{ID: CommandSkill, Name: "skill", RequiresSubcommand: true, LatencyClass: LatencyWork, Subcommands: []SubcommandDescriptor{
			sub(SubSkillInit, "init"), sub(SubSkillVerify, "verify"),
			sub(SubSkillC99, "c99-standard-c",
				sub(SubSkillC99Status, "status"), sub(SubSkillC99Templates, "templates"), sub(SubSkillC99Fmt, "fmt"), sub(SubSkillC99Check, "check"), sub(SubSkillC99Verify, "verify")),
		}}, handler: runSkill},
		{descriptor: CommandDescriptor{ID: CommandLifecycle, Name: "lifecycle", RequiresSubcommand: true, LatencyClass: LatencyStandard, LatencyProbe: []string{"status", "--scope", "all"}, Subcommands: []SubcommandDescriptor{
			sub(SubLifecyclePlan, "plan"), sub(SubLifecycleInstall, "install"), sub(SubLifecycleUpdate, "update"), sub(SubLifecycleUninstall, "uninstall"),
			sub(SubLifecycleStatus, "status"), sub(SubLifecycleDoctor, "doctor"), sub(SubLifecycleVerify, "verify"), sub(SubLifecycleRollback, "rollback"),
		}}, handler: runLifecycle},
		{descriptor: CommandDescriptor{ID: CommandExport, Name: "export", LatencyClass: LatencyWork}, handler: runExport},
		{descriptor: CommandDescriptor{ID: CommandFreshClone, Name: "fresh-clone", LatencyClass: LatencyWork}, handler: runFreshClone},
		{descriptor: CommandDescriptor{ID: CommandCache, Name: "cache", RequiresSubcommand: true, LatencyClass: LatencyFast, LatencyProbe: []string{"status"}, Subcommands: []SubcommandDescriptor{sub(SubCacheStatus, "status"), sub(SubCacheClean, "clean")}}, handler: runCache},
		{descriptor: CommandDescriptor{ID: CommandCapability, Name: "capability", RequiresSubcommand: true, LatencyClass: LatencyFast, LatencyProbe: []string{"list"}, Subcommands: []SubcommandDescriptor{
			sub(SubCapabilityList, "list"), sub(SubCapabilityDescribe, "describe"), sub(SubCapabilityIndex, "index"),
		}}, handler: runCapability},
		{descriptor: CommandDescriptor{ID: CommandCodex, Name: "codex", RequiresSubcommand: true, LatencyClass: LatencyWork, Subcommands: []SubcommandDescriptor{
			sub(SubCodexUsage, "usage", sub(SubCodexUsageParse, "parse"), sub(SubCodexUsageRun, "run")),
		}}, handler: runCodexUsage},
		{descriptor: CommandDescriptor{ID: CommandMCP, Name: "mcp", RequiresSubcommand: true, LatencyClass: LatencyFast, LatencyProbe: []string{"list"}, Subcommands: []SubcommandDescriptor{
			sub(SubMCPInit, "init"), sub(SubMCPList, "list"), sub(SubMCPStatus, "status"), sub(SubMCPDoctor, "doctor"), sub(SubMCPVerify, "verify"),
		}}, handler: runMCP},
		{descriptor: CommandDescriptor{ID: CommandTag, Name: "tag", RequiresSubcommand: true, LatencyClass: LatencyFast, LatencyProbe: []string{"audit"}, Subcommands: []SubcommandDescriptor{sub(SubTagAudit, "audit")}}, handler: runTag},
		{descriptor: CommandDescriptor{ID: CommandRelease, Name: "release", RequiresSubcommand: true, LatencyClass: LatencyWork, Subcommands: []SubcommandDescriptor{sub(SubReleaseVerify, "verify"), sub(SubReleaseGate, "gate")}}, handler: runReleaseCommand},
		{descriptor: CommandDescriptor{ID: CommandKit, Name: "kit", RequiresSubcommand: true, LatencyClass: LatencyFast, LatencyProbe: []string{"list"}, Subcommands: []SubcommandDescriptor{
			sub(SubKitList, "list"), sub(SubKitInit, "init"), sub(SubKitRegister, "register"), sub(SubKitPrefetch, "prefetch"),
			sub(SubKitDescribe, "describe"), sub(SubKitVerify, "verify"), sub(SubKitTest, "test"), sub(SubKitDoctor, "doctor"),
		}}, handler: runKit},
		{descriptor: CommandDescriptor{ID: CommandDoctor, Name: "doctor", RequiresSubcommand: true, LatencyClass: LatencyStandard, LatencyProbe: []string{"--all"}, Subcommands: []SubcommandDescriptor{
			sub(SubDoctorPerf, "perf"), sub(SubDoctorPwsh, "pwsh"), sub(SubDoctorPwshBudget, "pwsh-budget"),
		}}, handler: runDoctor},
		{descriptor: CommandDescriptor{ID: CommandVerify, Name: "verify", RequiresSubcommand: true, LatencyClass: LatencyStandard, LatencyProbe: []string{"--profile", "Smoke"}, Subcommands: []SubcommandDescriptor{
			sub(SubVerifyHooks, "hooks"), sub(SubVerifyRepoText, "repo-text"), sub(SubVerifyReleaseNotes, "release-notes"),
		}}, handler: runVerify},
		{descriptor: CommandDescriptor{ID: CommandGovernance, Name: "governance", RequiresSubcommand: true, LatencyClass: LatencyStandard, LatencyProbe: []string{"dependencies"}, Subcommands: []SubcommandDescriptor{
			sub(SubGovernanceLint, "lint"), sub(SubGovernanceDeps, "dependencies"), sub(SubGovernanceLayout, "layout"), sub(SubGovernanceReuse, "reuse"), sub(SubGovernanceCaps, "capabilities"),
		}}, handler: runGovernance},
		{descriptor: CommandDescriptor{ID: CommandPowerShell, Name: "powershell", RequiresSubcommand: true, LatencyClass: LatencyFast, LatencyProbe: []string{"regex-lint", "--staged"}, Subcommands: []SubcommandDescriptor{sub(SubPowerShellRegexLint, "regex-lint")}}, handler: runPowerShell},
		{descriptor: CommandDescriptor{ID: CommandTodolist, Name: "todolist", LatencyClass: LatencyFast}, handler: runTodolist},
		{descriptor: CommandDescriptor{ID: CommandWork, Name: "work", RequiresSubcommand: true, LatencyClass: LatencyWork, Subcommands: []SubcommandDescriptor{
			sub(SubWorkValidate, "validate"), sub(SubWorkNext, "next"), sub(SubWorkStatus, "status"), sub(SubWorkRecord, "record"),
		}}, handler: runWork},
		{descriptor: CommandDescriptor{ID: CommandPlan, Name: "plan", RequiresSubcommand: true, LatencyClass: LatencyFast, LatencyProbe: []string{"check", "--staged"}, Subcommands: []SubcommandDescriptor{
			sub(SubPlanCheck, "check"), sub(SubPlanVerify, "verify"), sub(SubPlanStatus, "status"), sub(SubPlanApprove, "approve"),
		}}, handler: runPlan},
		{descriptor: CommandDescriptor{ID: CommandProvision, Name: "provision", LatencyClass: LatencyWork}, handler: runProvision},
		{descriptor: CommandDescriptor{ID: CommandChange, Name: "change", RequiresSubcommand: true, LatencyClass: LatencyWork, Subcommands: []SubcommandDescriptor{sub(SubChangeVerify, "verify")}}, handler: runChange},
	},
	[]HelpSection{
		{ID: HelpUsage, Title: "Usage:"},
		{ID: HelpFormal, Title: "Formal product workflow:"},
		{ID: HelpDomain, Title: "Domain and diagnostic commands:"},
	},
	[]HelpForm{
		helpAlias(CommandHelp, HelpUsage, "--help", ""),
		help(CommandVersion, HelpUsage, ""),
		help(CommandHook, HelpUsage, "[--repo-root PATH] [--json]", SubHookPreCommit),
		help(CommandHook, HelpUsage, "--file COMMIT_MSG [--repo-root PATH] [--json]", SubHookCommitMsg),
		help(CommandHook, HelpUsage, "[--repo-root PATH] [--json]", SubHookPrePush),
		help(CommandHook, HelpUsage, "[--repo-root PATH] [--json]", SubHookPostCommit),
		help(CommandBootstrap, HelpUsage, "[--repo-root PATH] [--json]"),

		help(CommandTest, HelpFormal, productProfileOption()+" [--reuse auto|off] [--force] [--allow-dirty] [--verify-reuse] [--repo-root PATH] [--timeout-sec N] [--long-timeout-sec N] [--concurrency N] [--json]"),
		help(CommandLifecycle, HelpFormal, "--action install|update|uninstall --scope kit --all [--repo-root PATH] [--json]", SubLifecyclePlan),
		helpAny(CommandLifecycle, HelpFormal, "--scope kit --all [--repo-root PATH] [--json]", SubLifecycleInstall, SubLifecycleUpdate, SubLifecycleUninstall),
		help(CommandLifecycle, HelpFormal, "--action install|update --scope all --runtime-profile runtime|full|skill-development [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--migrate-unmanaged] [--codex-config PATH] [--repo-root PATH] [--json]", SubLifecyclePlan),
		help(CommandLifecycle, HelpFormal, "--action uninstall --scope all [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--codex-config PATH] [--repo-root PATH] [--json]", SubLifecyclePlan),
		helpAny(CommandLifecycle, HelpFormal, "--scope all --runtime-profile runtime|full|skill-development [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--migrate-unmanaged] [--codex-config PATH] [--repo-root PATH] [--json]", SubLifecycleInstall, SubLifecycleUpdate),
		help(CommandLifecycle, HelpFormal, "--scope all [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--codex-config PATH] [--repo-root PATH] [--json]", SubLifecycleUninstall),
		helpAny(CommandLifecycle, HelpFormal, "--scope all [--runtime-profile runtime|full|skill-development] [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--codex-config PATH] [--repo-root PATH] [--json]", SubLifecycleStatus, SubLifecycleDoctor),
		help(CommandLifecycle, HelpFormal, "--scope all "+productProfileOption()+" [--runtime-profile runtime|full|skill-development] [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--configured] [--codex-config PATH] [--repo-root PATH] [--json]", SubLifecycleVerify),
		help(CommandLifecycle, HelpFormal, "--scope kit --last [--repo-root PATH] [--json]", SubLifecycleRollback),
		helpAny(CommandLifecycle, HelpDomain, "--scope repo-context [--repo-root PATH] [--json]", SubLifecyclePlan, SubLifecycleInstall, SubLifecycleUpdate, SubLifecycleUninstall, SubLifecycleStatus, SubLifecycleDoctor, SubLifecycleVerify),
		help(CommandTodolist, HelpDomain, "[--repo-root PATH] [--json]"),
		help(CommandWork, HelpDomain, "--file SPEC.json [--repo-root PATH] [--json]", SubWorkValidate),
		help(CommandWork, HelpDomain, "--file SPEC.json [--repo-root PATH] [--json]", SubWorkNext),
		help(CommandWork, HelpDomain, "--file SPEC.json [--repo-root PATH] [--json]", SubWorkStatus),
		help(CommandWork, HelpDomain, "--file SPEC.json --attempt ATTEMPT.json [--repo-root PATH] [--json]", SubWorkRecord),
		help(CommandPlan, HelpDomain, "(--staged | --paths PATH ...) [--repo-root PATH] [--json]", SubPlanCheck),
		help(CommandPlan, HelpDomain, "[--repo-root PATH] [--json]", SubPlanVerify),
		help(CommandPlan, HelpDomain, "[--id ID | --all] [--repo-root PATH] [--json]", SubPlanStatus),
		help(CommandPlan, HelpDomain, "--id ID [--repo-root PATH] [--json]", SubPlanApprove),
		help(CommandProvision, HelpDomain, "[--repo-root PATH] [--json]"),
		help(CommandChange, HelpFormal, "[--staged | --since REV] [--repo-root PATH] [--json]", SubChangeVerify),
		help(CommandDoctor, HelpFormal, "--all [--runtime-profile runtime|full|skill-development] [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--codex-config PATH] [--timeout-sec N] [--repo-root PATH] [--json]"),
		help(CommandVerify, HelpFormal, productProfileOption()+" [--runtime-profile runtime|full|skill-development] [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--configured] [--codex-config PATH] [--timeout-sec N] [--repo-root PATH] [--json]"),
		help(CommandRelease, HelpFormal, "[--repo-root PATH] [--json]", SubReleaseVerify),
		help(CommandRelease, HelpFormal, "[--repo-root PATH] [--json]", SubReleaseGate),

		help(CommandTest, HelpDomain, "[--repo-root PATH] [--json]", SubTestLatest),
		help(CommandValidation, HelpDomain, "[--repo-root PATH] [--json]", SubValidationStatus),
		help(CommandValidation, HelpDomain, productProfileOption()+" --target HEAD|INDEX [--bind-alias] [--repo-root PATH] [--json]", SubValidationCheck),
		help(CommandValidation, HelpDomain, productProfileOption()+" --target HEAD|INDEX [--repo-root PATH] [--json]", SubValidationExplain),
		help(CommandValidation, HelpDomain, "["+productProfileOption()+"] [--repo-root PATH] [--json]", SubValidationList),
		help(CommandValidation, HelpDomain, "["+productProfileOption()+"] [--repo-root PATH] [--json]", SubValidationClean),
		helpAny(CommandDocSync, HelpDomain, "[--repo-root PATH] [--json]", SubDocSyncStaged, SubDocSyncAll, SubDocSyncCI, SubDocSyncRelease),
		help(CommandSkill, HelpDomain, "--all "+productProfileOption()+" [--repo-root PATH] [--json]", SubSkillVerify),
		help(CommandSkill, HelpDomain, "[--repo-root PATH] [--json]", SubSkillC99, SubSkillC99Status),
		help(CommandSkill, HelpDomain, "[--repo-root PATH] [--json]", SubSkillC99, SubSkillC99Templates),
		help(CommandSkill, HelpDomain, "ID [--out PATH] [--dry-run] [--repo-root PATH] [--json]", SubSkillInit),
		help(CommandSkill, HelpDomain, "--scope changed|staged|all|paths [--path PATH ...] [--preview] [--repo-root PATH] [--json]", SubSkillC99, SubSkillC99Fmt),
		help(CommandSkill, HelpDomain, "--scope changed|staged|all|paths [--path PATH ...] [--repo-root PATH] [--json]", SubSkillC99, SubSkillC99Check),
		help(CommandSkill, HelpDomain, "--depth fast|full [--target PATH] [--overlay PATH ...] [--timings] [--repo-root PATH] [--json]", SubSkillC99, SubSkillC99Verify),
		help(CommandExport, HelpDomain, "--all --zip [--repo-root PATH] [--json]"),
		help(CommandFreshClone, HelpDomain, productProfileOption()+" [--repo-root PATH] [--json]"),
		help(CommandCache, HelpDomain, "[--repo-root PATH] [--json]", SubCacheStatus),
		help(CommandCache, HelpDomain, "[--scope fast-path|test-results|validation-reports|temp|work-state|pins] [--keep N] [--dry-run] [--adopt] [--all-repos] [--repo-root PATH] [--json]", SubCacheClean),
		help(CommandCapability, HelpDomain, "[--type TYPE] [--status STATUS] [--repo-root PATH] [--json]", SubCapabilityList),
		help(CommandCapability, HelpDomain, "--id ID [--repo-root PATH] [--json]", SubCapabilityDescribe),
		help(CommandCapability, HelpDomain, "--write [--repo-root PATH] [--json]", SubCapabilityIndex),
		help(CommandCodex, HelpDomain, "[--file FILE|-] [--json]", SubCodexUsage, SubCodexUsageParse),
		help(CommandCodex, HelpDomain, "[--json] -- codex exec --json \"PROMPT\"", SubCodexUsage, SubCodexUsageRun),
		help(CommandMCP, HelpDomain, "[--codex-config PATH] [--repo-root PATH] [--json]", SubMCPList),
		help(CommandMCP, HelpDomain, "ID [--out PATH] [--dry-run] [--repo-root PATH] [--json]", SubMCPInit),
		helpAny(CommandMCP, HelpDomain, "COMPONENT [--codex-config PATH] [--repo-root PATH] [--json]", SubMCPStatus, SubMCPDoctor),
		help(CommandMCP, HelpDomain, "COMPONENT|--all "+productProfileOption()+" [--configured] [--codex-config PATH] [--repo-root PATH] [--json]", SubMCPVerify),
		help(CommandTag, HelpDomain, "[--repo-root PATH] [--json]", SubTagAudit),
		help(CommandGovernance, HelpDomain, "[--repo-root PATH] [--json]", SubGovernanceLint),
		help(CommandGovernance, HelpDomain, "[--repo-root PATH] [--json]", SubGovernanceDeps),
		help(CommandGovernance, HelpDomain, "[--repo-root PATH] [--json]", SubGovernanceLayout),
		help(CommandGovernance, HelpDomain, "[--repo-root PATH] [--json]", SubGovernanceReuse),
		help(CommandGovernance, HelpDomain, "[--repo-root PATH] [--json]", SubGovernanceCaps),
		help(CommandKit, HelpDomain, "[--repo-root PATH] [--json]", SubKitList),
		help(CommandKit, HelpDomain, "ID [--external] [--dry-run] [--repo-root PATH] [--json]", SubKitInit),
		help(CommandKit, HelpDomain, "--manifest PATH [--prefetch] [--repo-root PATH] [--json]", SubKitRegister),
		help(CommandKit, HelpDomain, "--id ID [--repo-root PATH] [--json]", SubKitPrefetch),
		help(CommandKit, HelpDomain, "--kit ID|--all [--with-state] [--repo-root PATH] [--json]", SubKitDescribe),
		help(CommandKit, HelpDomain, "--all --level smoke|lifecycle [--repo-root PATH] [--json]", SubKitVerify),
		help(CommandKit, HelpDomain, "--kit ID|--all [--repo-root PATH] [--json]", SubKitTest),
		help(CommandKit, HelpDomain, "[--repo-root PATH] [--json]", SubKitDoctor),
		help(CommandDoctor, HelpDomain, "[--repo-root PATH] [--json]", SubDoctorPerf),
		help(CommandDoctor, HelpDomain, "[--repo-root PATH] [--json]", SubDoctorPwsh),
		help(CommandDoctor, HelpDomain, "[--repo-root PATH] [--json]", SubDoctorPwshBudget),
		help(CommandVerify, HelpDomain, "[--repo-root PATH] [--json]", SubVerifyHooks),
		help(CommandVerify, HelpDomain, "[--repo-root PATH] [--json]", SubVerifyRepoText),
		help(CommandVerify, HelpDomain, "[--repo-root PATH] [--json]", SubVerifyReleaseNotes),
		help(CommandPowerShell, HelpDomain, "--staged [--repo-root PATH] [--json]", SubPowerShellRegexLint),
		help(CommandPowerShell, HelpDomain, "--path PATH [--repo-root PATH] [--json]", SubPowerShellRegexLint),
	},
	[]QuickstartForm{
		{Operation: "status", Command: CommandLifecycle, Path: []SubcommandID{SubLifecycleStatus}, Args: []string{"--scope", "kit", "--kit", "{kit}", "--json"}},
		{Operation: "doctor", Command: CommandLifecycle, Path: []SubcommandID{SubLifecycleDoctor}, Args: []string{"--scope", "kit", "--kit", "{kit}", "--json"}},
		{Operation: "verify", Command: CommandLifecycle, Path: []SubcommandID{SubLifecycleVerify}, Args: []string{"--scope", "kit", "--kit", "{kit}", "--json"}},
		{Operation: "test", Command: CommandKit, Path: []SubcommandID{SubKitTest}, Args: []string{"--kit", "{kit}", "--json"}},
		{Operation: "skills", Command: CommandSkill, Path: []SubcommandID{SubSkillVerify}, Args: []string{"--kit", "{kit}", "--profile", productProfileVocabulary[0], "--json"}},
		{Operation: "verify-skills", Command: CommandSkill, Path: []SubcommandID{SubSkillVerify}, Args: []string{"--kit", "{kit}", "--profile", productProfileVocabulary[0], "--json"}},
	},
)

func init() {
	resolveCatalogSubcommandID = commands.resolveSubcommandID
	var err error
	catalogSnapshot, err = registry.NewSnapshot("command-catalog", commands.descriptor)
	if err != nil {
		panic(err)
	}
	catalogHelpText = renderCatalogHelp()
	commandCatalogEvidenceDigest = catalogSnapshot.Digest()
	commandPublicEntryExists = catalogHasPublicEntry
}

var resolveCatalogSubcommandID func(CommandID, ...string) (SubcommandID, error)

func mustCommandCatalog(routes []commandRoute, sections []HelpSection, help []HelpForm, quickstartSets ...[]QuickstartForm) typedCommandCatalog {
	catalog, err := newCommandCatalog(routes, sections, help, quickstartSets...)
	if err != nil {
		panic(err)
	}
	return catalog
}

func newCommandCatalog(routes []commandRoute, sections []HelpSection, help []HelpForm, quickstartSets ...[]QuickstartForm) (typedCommandCatalog, error) {
	var quickstarts []QuickstartForm
	if len(quickstartSets) > 1 {
		return typedCommandCatalog{}, fmt.Errorf("command catalog accepts one quickstart set")
	}
	if len(quickstartSets) == 1 {
		quickstarts = quickstartSets[0]
	}
	catalog := typedCommandCatalog{
		descriptor: CatalogDescriptor{
			Commands:    make([]CommandDescriptor, 0, len(routes)),
			Sections:    append([]HelpSection(nil), sections...),
			Help:        append([]HelpForm(nil), help...),
			Quickstarts: append([]QuickstartForm(nil), quickstarts...),
		},
		routes:  make(map[string]*commandRoute, len(routes)),
		ordered: append([]commandRoute(nil), routes...),
	}
	ids := make(map[CommandID]struct{}, len(routes))
	subcommandIDs := make(map[SubcommandID]struct{})
	for index := range catalog.ordered {
		route := &catalog.ordered[index]
		descriptor := route.descriptor
		if descriptor.ID == "" || descriptor.Name == "" {
			return typedCommandCatalog{}, fmt.Errorf("command id and name are required")
		}
		if descriptor.LatencyClass != LatencyFast && descriptor.LatencyClass != LatencyStandard && descriptor.LatencyClass != LatencyWork {
			return typedCommandCatalog{}, fmt.Errorf("command %s has invalid latency class %q", descriptor.ID, descriptor.LatencyClass)
		}
		if _, exists := ids[descriptor.ID]; exists {
			return typedCommandCatalog{}, fmt.Errorf("duplicate command id: %s", descriptor.ID)
		}
		ids[descriptor.ID] = struct{}{}
		if route.handler == nil && route.direct == directNone {
			return typedCommandCatalog{}, fmt.Errorf("command %s has no route", descriptor.ID)
		}
		if route.handler != nil && route.direct != directNone {
			return typedCommandCatalog{}, fmt.Errorf("command %s has multiple routes", descriptor.ID)
		}
		if descriptor.RequiresSubcommand && len(descriptor.Subcommands) == 0 {
			return typedCommandCatalog{}, fmt.Errorf("command %s requires subcommands but registers none", descriptor.ID)
		}
		if err := validateSubcommands(descriptor.Name, descriptor.Subcommands, subcommandIDs); err != nil {
			return typedCommandCatalog{}, err
		}
		names := append([]string{descriptor.Name}, descriptor.Aliases...)
		for _, name := range names {
			if name == "" {
				return typedCommandCatalog{}, fmt.Errorf("command %s has an empty name", descriptor.ID)
			}
			if _, exists := catalog.routes[name]; exists {
				return typedCommandCatalog{}, fmt.Errorf("duplicate command name: %s", name)
			}
			catalog.routes[name] = route
		}
		descriptor.Aliases = append([]string(nil), descriptor.Aliases...)
		descriptor.LatencyProbe = append([]string(nil), descriptor.LatencyProbe...)
		descriptor.Subcommands = cloneSubcommands(descriptor.Subcommands)
		catalog.descriptor.Commands = append(catalog.descriptor.Commands, descriptor)
	}
	sectionIDs := make(map[HelpSectionID]struct{}, len(sections))
	for _, section := range sections {
		if section.ID == "" || section.Title == "" {
			return typedCommandCatalog{}, fmt.Errorf("help section id and title are required")
		}
		if _, exists := sectionIDs[section.ID]; exists {
			return typedCommandCatalog{}, fmt.Errorf("duplicate help section: %s", section.ID)
		}
		sectionIDs[section.ID] = struct{}{}
	}
	commandsWithHelp := make(map[CommandID]struct{}, len(routes))
	for index, form := range help {
		if _, exists := ids[form.Command]; !exists {
			return typedCommandCatalog{}, fmt.Errorf("help references unknown command: %s", form.Command)
		}
		if _, exists := sectionIDs[form.Section]; !exists {
			return typedCommandCatalog{}, fmt.Errorf("help references unknown section: %s", form.Section)
		}
		projected, err := projectHelpForm(catalog.descriptor.Commands, form)
		if err != nil {
			return typedCommandCatalog{}, err
		}
		if strings.TrimSpace(projected.Usage) == "" {
			return typedCommandCatalog{}, fmt.Errorf("help usage is empty for command: %s", form.Command)
		}
		if err := validateProfileHelp(projected); err != nil {
			return typedCommandCatalog{}, err
		}
		catalog.descriptor.Help[index] = projected
		commandsWithHelp[form.Command] = struct{}{}
	}
	for id := range ids {
		if _, exists := commandsWithHelp[id]; !exists {
			return typedCommandCatalog{}, fmt.Errorf("command %s has no help form", id)
		}
	}
	operations := make(map[string]struct{}, len(quickstarts))
	for index, form := range quickstarts {
		if strings.TrimSpace(form.Operation) == "" {
			return typedCommandCatalog{}, fmt.Errorf("quickstart operation is required")
		}
		if _, exists := operations[form.Operation]; exists {
			return typedCommandCatalog{}, fmt.Errorf("duplicate quickstart operation: %s", form.Operation)
		}
		operations[form.Operation] = struct{}{}
		if _, err := catalog.invocationTokens(form.Command, form.Path, form.Args); err != nil {
			return typedCommandCatalog{}, fmt.Errorf("quickstart %s: %w", form.Operation, err)
		}
		form.Path = append([]SubcommandID(nil), form.Path...)
		form.Args = append([]string(nil), form.Args...)
		catalog.descriptor.Quickstarts[index] = form
	}
	return catalog, nil
}

func projectHelpForm(commands []CommandDescriptor, form HelpForm) (HelpForm, error) {
	if form.Usage != "" && form.entryName == "" && form.suffix == "" && len(form.Path) == 0 && len(form.Alternatives) == 0 {
		return form, nil
	}
	command, ok := findCommandByID(commands, form.Command)
	if !ok {
		return HelpForm{}, fmt.Errorf("help references unknown command: %s", form.Command)
	}
	entry := command.Name
	if form.entryName != "" {
		entry = ""
		for _, candidate := range append([]string{command.Name}, command.Aliases...) {
			if candidate == form.entryName {
				entry = candidate
				break
			}
		}
		if entry == "" {
			return HelpForm{}, fmt.Errorf("help for %s references unknown command alias: %s", command.Name, form.entryName)
		}
	}
	parts := []string{"aicoding", entry}
	if len(form.Path) > 0 && len(form.Alternatives) > 0 {
		return HelpForm{}, fmt.Errorf("help for %s cannot combine path and alternatives", command.Name)
	}
	descriptors := command.Subcommands
	for _, id := range form.Path {
		descriptor, exists := findDirectSubcommandByID(descriptors, id)
		if !exists {
			return HelpForm{}, fmt.Errorf("help for %s references unregistered subcommand: %s", strings.Join(parts[1:], " "), id)
		}
		parts = append(parts, descriptor.Name)
		descriptors = descriptor.Subcommands
	}
	if len(form.Alternatives) > 0 {
		names := make([]string, 0, len(form.Alternatives))
		for _, id := range form.Alternatives {
			descriptor, exists := findDirectSubcommandByID(command.Subcommands, id)
			if !exists {
				return HelpForm{}, fmt.Errorf("help for %s references unregistered subcommand: %s", command.Name, id)
			}
			names = append(names, descriptor.Name)
		}
		parts = append(parts, strings.Join(names, "|"))
	}
	if strings.TrimSpace(form.suffix) != "" {
		parts = append(parts, strings.TrimSpace(form.suffix))
	}
	form.Usage = strings.Join(parts, " ")
	form.entryName = ""
	form.suffix = ""
	return form, nil
}

var profileHelpPattern = regexp.MustCompile(`(?:^|[\s\[])--profile\s+([A-Za-z]+(?:\|[A-Za-z]+)*)`)

func validateProfileHelp(form HelpForm) error {
	if !strings.Contains(form.Usage, "--profile") {
		return nil
	}
	matches := profileHelpPattern.FindAllStringSubmatch(form.Usage, -1)
	if len(matches) == 0 {
		return fmt.Errorf("%s: --profile help has no accepted values", helpCommandPath(form.Usage))
	}
	for _, match := range matches {
		if match[1] != productProfileChoice() {
			return fmt.Errorf("%s: --profile help accepts %s; only %s is allowed", helpCommandPath(form.Usage), match[1], productProfileChoice())
		}
	}
	return nil
}

func helpCommandPath(usage string) string {
	fields := strings.Fields(usage)
	path := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.HasPrefix(field, "-") || strings.HasPrefix(field, "[-") || strings.Contains(field, "|") {
			break
		}
		path = append(path, field)
	}
	return strings.Join(path, " ")
}

func findCommandByID(commands []CommandDescriptor, id CommandID) (CommandDescriptor, bool) {
	for _, command := range commands {
		if command.ID == id {
			return command, true
		}
	}
	return CommandDescriptor{}, false
}

func findDirectSubcommandByID(descriptors []SubcommandDescriptor, id SubcommandID) (SubcommandDescriptor, bool) {
	for _, descriptor := range descriptors {
		if descriptor.ID == id {
			return descriptor, true
		}
	}
	return SubcommandDescriptor{}, false
}

func (c typedCommandCatalog) invocationTokens(commandID CommandID, path []SubcommandID, args []string) ([]string, error) {
	command, ok := findCommandByID(c.descriptor.Commands, commandID)
	if !ok {
		return nil, fmt.Errorf("unregistered command: %s", commandID)
	}
	tokens := []string{"aicoding", command.Name}
	descriptors := command.Subcommands
	for _, id := range path {
		descriptor, exists := findDirectSubcommandByID(descriptors, id)
		if !exists {
			return nil, fmt.Errorf("%s references unregistered subcommand: %s", strings.Join(tokens, " "), id)
		}
		tokens = append(tokens, descriptor.Name)
		descriptors = descriptor.Subcommands
	}
	tokens = append(tokens, args...)
	return tokens, nil
}

func validateSubcommands(parent string, descriptors []SubcommandDescriptor, ids map[SubcommandID]struct{}) error {
	names := make(map[string]struct{}, len(descriptors))
	for _, descriptor := range descriptors {
		path := strings.TrimSpace(parent + " " + descriptor.Name)
		if descriptor.ID == "" || descriptor.Name == "" {
			return fmt.Errorf("subcommand id and name are required under %s", parent)
		}
		if _, exists := ids[descriptor.ID]; exists {
			return fmt.Errorf("duplicate subcommand id: %s", descriptor.ID)
		}
		ids[descriptor.ID] = struct{}{}
		for _, name := range append([]string{descriptor.Name}, descriptor.Aliases...) {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("subcommand %s has an empty name or alias", path)
			}
			if _, exists := names[name]; exists {
				return fmt.Errorf("duplicate subcommand name under %s: %s", parent, name)
			}
			names[name] = struct{}{}
		}
		if err := validateSubcommands(path, descriptor.Subcommands, ids); err != nil {
			return err
		}
	}
	return nil
}

func cloneSubcommands(descriptors []SubcommandDescriptor) []SubcommandDescriptor {
	if descriptors == nil {
		return nil
	}
	cloned := make([]SubcommandDescriptor, len(descriptors))
	for index, descriptor := range descriptors {
		descriptor.Aliases = append([]string(nil), descriptor.Aliases...)
		descriptor.Subcommands = cloneSubcommands(descriptor.Subcommands)
		cloned[index] = descriptor
	}
	return cloned
}

func (c typedCommandCatalog) lookup(name string) (*commandRoute, bool) {
	route, ok := c.routes[name]
	return route, ok
}

func Catalog() CatalogDescriptor {
	descriptor := commands.descriptor
	descriptor.Commands = make([]CommandDescriptor, len(commands.descriptor.Commands))
	for index, command := range commands.descriptor.Commands {
		command.Aliases = append([]string(nil), command.Aliases...)
		command.LatencyProbe = append([]string(nil), command.LatencyProbe...)
		command.Subcommands = cloneSubcommands(command.Subcommands)
		descriptor.Commands[index] = command
	}
	descriptor.Sections = append([]HelpSection(nil), commands.descriptor.Sections...)
	descriptor.Help = make([]HelpForm, len(commands.descriptor.Help))
	for index, form := range commands.descriptor.Help {
		form.Path = append([]SubcommandID(nil), form.Path...)
		form.Alternatives = append([]SubcommandID(nil), form.Alternatives...)
		descriptor.Help[index] = form
	}
	descriptor.Quickstarts = make([]QuickstartForm, len(commands.descriptor.Quickstarts))
	for index, form := range commands.descriptor.Quickstarts {
		form.Path = append([]SubcommandID(nil), form.Path...)
		form.Args = append([]string(nil), form.Args...)
		descriptor.Quickstarts[index] = form
	}
	return descriptor
}

func CatalogSnapshot() registry.Snapshot {
	return catalogSnapshot
}

func commandRequiresSubcommand(command string) bool {
	route, ok := commands.lookup(command)
	return ok && route.descriptor.RequiresSubcommand
}

func (c typedCommandCatalog) prepareInvocation(route *commandRoute, args []string) ([]string, error) {
	if route == nil || len(route.descriptor.Subcommands) == 0 || len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return args, nil
	}
	normalized := append([]string(nil), args...)
	descriptors := route.descriptor.Subcommands
	path := route.descriptor.Name
	for index := 0; index < len(normalized) && len(descriptors) > 0; index++ {
		if strings.HasPrefix(normalized[index], "-") {
			break
		}
		descriptor, ok := findSubcommandByName(descriptors, normalized[index])
		if !ok {
			return args, usageErrorf("unsupported %s subcommand: %s", path, normalized[index])
		}
		normalized[index] = descriptor.Name
		path += " " + descriptor.Name
		descriptors = descriptor.Subcommands
		if len(descriptors) == 0 {
			break
		}
	}
	return normalized, nil
}

func findSubcommandByName(descriptors []SubcommandDescriptor, name string) (SubcommandDescriptor, bool) {
	for _, descriptor := range descriptors {
		for _, candidate := range append([]string{descriptor.Name}, descriptor.Aliases...) {
			if name == candidate {
				return descriptor, true
			}
		}
	}
	return SubcommandDescriptor{}, false
}

func (c typedCommandCatalog) resolveSubcommandID(commandID CommandID, path ...string) (SubcommandID, error) {
	command, ok := findCommandByID(c.descriptor.Commands, commandID)
	if !ok {
		return "", usageErrorf("unregistered command: %s", commandID)
	}
	if len(path) == 0 {
		return "", usageErrorf("%s requires a subcommand", command.Name)
	}
	descriptors := command.Subcommands
	parent := command.Name
	var resolved SubcommandDescriptor
	for _, name := range path {
		var exists bool
		resolved, exists = findSubcommandByName(descriptors, name)
		if !exists {
			return "", usageErrorf("unsupported %s subcommand: %s", parent, name)
		}
		parent += " " + resolved.Name
		descriptors = resolved.Subcommands
	}
	return resolved.ID, nil
}

func findSubcommandByID(descriptors []SubcommandDescriptor, id SubcommandID) (SubcommandDescriptor, bool) {
	for _, descriptor := range descriptors {
		if descriptor.ID == id {
			return descriptor, true
		}
		if nested, ok := findSubcommandByID(descriptor.Subcommands, id); ok {
			return nested, true
		}
	}
	return SubcommandDescriptor{}, false
}

func writeCatalogHelp(w io.Writer) {
	_, _ = io.WriteString(w, catalogHelpText)
}

func renderCatalogHelp() string {
	var out strings.Builder
	out.WriteString("AiCoding CLI\n")
	for _, section := range commands.descriptor.Sections {
		out.WriteByte('\n')
		out.WriteString(section.Title)
		out.WriteByte('\n')
		for _, form := range commands.descriptor.Help {
			if form.Section == section.ID {
				out.WriteString("  ")
				out.WriteString(form.Usage)
				out.WriteByte('\n')
			}
		}
	}
	out.WriteString("\nThis CLI owns Go-native fast, lifecycle, export, DocSync, fresh-clone, Full, and Release control paths.\n")
	return out.String()
}
