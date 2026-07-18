package cli

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/JiaxI2/AiCoding/internal/registry"
	"github.com/JiaxI2/AiCoding/internal/report"
)

type CommandID string

const (
	CommandHelp       CommandID = "help"
	CommandVersion    CommandID = "version"
	CommandHook       CommandID = "hook"
	CommandBootstrap  CommandID = "bootstrap"
	CommandTest       CommandID = "test"
	CommandDocSync    CommandID = "docsync"
	CommandSkill      CommandID = "skill"
	CommandLifecycle  CommandID = "lifecycle"
	CommandExport     CommandID = "export"
	CommandFreshClone CommandID = "fresh-clone"
	CommandCache      CommandID = "cache"
	CommandCodex      CommandID = "codex"
	CommandMCP        CommandID = "mcp"
	CommandTag        CommandID = "tag"
	CommandRelease    CommandID = "release"
	CommandKit        CommandID = "kit"
	CommandDoctor     CommandID = "doctor"
	CommandVerify     CommandID = "verify"
	CommandGovernance CommandID = "governance"
	CommandPowerShell CommandID = "powershell"
)

type CommandDescriptor struct {
	ID                 CommandID `json:"id"`
	Name               string    `json:"name"`
	Aliases            []string  `json:"aliases,omitempty"`
	RequiresSubcommand bool      `json:"requiresSubcommand,omitempty"`
}

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
	Command CommandID     `json:"command"`
	Section HelpSectionID `json:"section"`
	Usage   string        `json:"usage"`
}

type CatalogDescriptor struct {
	Commands []CommandDescriptor `json:"commands"`
	Sections []HelpSection       `json:"sections"`
	Help     []HelpForm          `json:"help"`
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
	catalogSnapshotOnce sync.Once
	catalogSnapshot     registry.Snapshot
	catalogHelpOnce     sync.Once
	catalogHelpText     string
)

var commands = mustCommandCatalog(
	[]commandRoute{
		{descriptor: CommandDescriptor{ID: CommandHelp, Name: "help", Aliases: []string{"--help", "-h"}}, direct: directHelp},
		{descriptor: CommandDescriptor{ID: CommandVersion, Name: "version", Aliases: []string{"--version", "-v"}}, direct: directVersion},
		{descriptor: CommandDescriptor{ID: CommandHook, Name: "hook", RequiresSubcommand: true}, handler: runHook},
		{descriptor: CommandDescriptor{ID: CommandBootstrap, Name: "bootstrap"}, handler: runBootstrap},
		{descriptor: CommandDescriptor{ID: CommandTest, Name: "test"}, handler: runTest},
		{descriptor: CommandDescriptor{ID: CommandDocSync, Name: "docsync", RequiresSubcommand: true}, handler: runDocSync},
		{descriptor: CommandDescriptor{ID: CommandSkill, Name: "skill", RequiresSubcommand: true}, handler: runSkill},
		{descriptor: CommandDescriptor{ID: CommandLifecycle, Name: "lifecycle", RequiresSubcommand: true}, handler: runLifecycle},
		{descriptor: CommandDescriptor{ID: CommandExport, Name: "export"}, handler: runExport},
		{descriptor: CommandDescriptor{ID: CommandFreshClone, Name: "fresh-clone"}, handler: runFreshClone},
		{descriptor: CommandDescriptor{ID: CommandCache, Name: "cache", RequiresSubcommand: true}, handler: runCache},
		{descriptor: CommandDescriptor{ID: CommandCodex, Name: "codex", RequiresSubcommand: true}, handler: runCodexUsage},
		{descriptor: CommandDescriptor{ID: CommandMCP, Name: "mcp", RequiresSubcommand: true}, handler: runMCP},
		{descriptor: CommandDescriptor{ID: CommandTag, Name: "tag", RequiresSubcommand: true}, handler: runTag},
		{descriptor: CommandDescriptor{ID: CommandRelease, Name: "release", RequiresSubcommand: true}, handler: runReleaseCommand},
		{descriptor: CommandDescriptor{ID: CommandKit, Name: "kit", RequiresSubcommand: true}, handler: runKit},
		{descriptor: CommandDescriptor{ID: CommandDoctor, Name: "doctor", RequiresSubcommand: true}, handler: runDoctor},
		{descriptor: CommandDescriptor{ID: CommandVerify, Name: "verify", RequiresSubcommand: true}, handler: runVerify},
		{descriptor: CommandDescriptor{ID: CommandGovernance, Name: "governance", RequiresSubcommand: true}, handler: runGovernance},
		{descriptor: CommandDescriptor{ID: CommandPowerShell, Name: "powershell", RequiresSubcommand: true}, handler: runPowerShell},
	},
	[]HelpSection{
		{ID: HelpUsage, Title: "Usage:"},
		{ID: HelpFormal, Title: "Formal product workflow:"},
		{ID: HelpDomain, Title: "Domain and diagnostic commands:"},
	},
	[]HelpForm{
		{Command: CommandHelp, Section: HelpUsage, Usage: "aicoding --help"},
		{Command: CommandVersion, Section: HelpUsage, Usage: "aicoding version"},
		{Command: CommandHook, Section: HelpUsage, Usage: "aicoding hook pre-commit [--repo-root PATH] [--json]"},
		{Command: CommandHook, Section: HelpUsage, Usage: "aicoding hook commit-msg --file COMMIT_MSG [--repo-root PATH] [--json]"},
		{Command: CommandBootstrap, Section: HelpUsage, Usage: "aicoding bootstrap [--repo-root PATH] [--json]"},

		{Command: CommandTest, Section: HelpFormal, Usage: "aicoding test --profile Smoke|Full|Release [--repo-root PATH] [--timeout-sec N] [--long-timeout-sec N] [--concurrency N] [--json]"},
		{Command: CommandLifecycle, Section: HelpFormal, Usage: "aicoding lifecycle plan --action install|update|uninstall --scope kit --all [--repo-root PATH] [--json]"},
		{Command: CommandLifecycle, Section: HelpFormal, Usage: "aicoding lifecycle install|update|uninstall --scope kit --all [--repo-root PATH] [--json]"},
		{Command: CommandLifecycle, Section: HelpFormal, Usage: "aicoding lifecycle plan --action install|update --scope all --runtime-profile runtime|full|skill-development [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--migrate-unmanaged] [--codex-config PATH] [--repo-root PATH] [--json]"},
		{Command: CommandLifecycle, Section: HelpFormal, Usage: "aicoding lifecycle plan --action uninstall --scope all [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--codex-config PATH] [--repo-root PATH] [--json]"},
		{Command: CommandLifecycle, Section: HelpFormal, Usage: "aicoding lifecycle install|update --scope all --runtime-profile runtime|full|skill-development [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--migrate-unmanaged] [--codex-config PATH] [--repo-root PATH] [--json]"},
		{Command: CommandLifecycle, Section: HelpFormal, Usage: "aicoding lifecycle uninstall --scope all [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--codex-config PATH] [--repo-root PATH] [--json]"},
		{Command: CommandLifecycle, Section: HelpFormal, Usage: "aicoding lifecycle status|doctor --scope all [--runtime-profile runtime|full|skill-development] [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--codex-config PATH] [--repo-root PATH] [--json]"},
		{Command: CommandLifecycle, Section: HelpFormal, Usage: "aicoding lifecycle verify --scope all --profile Smoke|Full|Release [--runtime-profile runtime|full|skill-development] [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--configured] [--codex-config PATH] [--repo-root PATH] [--json]"},
		{Command: CommandLifecycle, Section: HelpFormal, Usage: "aicoding lifecycle rollback --scope kit --last [--repo-root PATH] [--json]"},
		{Command: CommandDoctor, Section: HelpFormal, Usage: "aicoding doctor --all [--runtime-profile runtime|full|skill-development] [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--codex-config PATH] [--timeout-sec N] [--repo-root PATH] [--json]"},
		{Command: CommandVerify, Section: HelpFormal, Usage: "aicoding verify --profile Smoke|Full|Release [--runtime-profile runtime|full|skill-development] [--runtime-skill NAME] [--source-repository PATH] [--standalone-root agents|codex] [--configured] [--codex-config PATH] [--timeout-sec N] [--repo-root PATH] [--json]"},
		{Command: CommandRelease, Section: HelpFormal, Usage: "aicoding release verify [--repo-root PATH] [--json]"},
		{Command: CommandRelease, Section: HelpFormal, Usage: "aicoding release gate [--repo-root PATH] [--json]"},

		{Command: CommandTest, Section: HelpDomain, Usage: "aicoding test latest [--repo-root PATH] [--json]"},
		{Command: CommandDocSync, Section: HelpDomain, Usage: "aicoding docsync staged|all|ci|release [--repo-root PATH] [--json]"},
		{Command: CommandSkill, Section: HelpDomain, Usage: "aicoding skill verify --all --profile Smoke|Full|Release [--repo-root PATH] [--json]"},
		{Command: CommandSkill, Section: HelpDomain, Usage: "aicoding skill c99-standard-c status [--repo-root PATH] [--json]"},
		{Command: CommandSkill, Section: HelpDomain, Usage: "aicoding skill c99-standard-c templates [--repo-root PATH] [--json]"},
		{Command: CommandSkill, Section: HelpDomain, Usage: "aicoding skill c99-standard-c fmt --scope changed|staged|all|paths [--path PATH ...] [--preview] [--repo-root PATH] [--json]"},
		{Command: CommandSkill, Section: HelpDomain, Usage: "aicoding skill c99-standard-c check --scope changed|staged|all|paths [--path PATH ...] [--repo-root PATH] [--json]"},
		{Command: CommandSkill, Section: HelpDomain, Usage: "aicoding skill c99-standard-c verify --profile fast|full [--target PATH] [--overlay PATH ...] [--timings] [--repo-root PATH] [--json]"},
		{Command: CommandExport, Section: HelpDomain, Usage: "aicoding export --all --zip [--repo-root PATH] [--json]"},
		{Command: CommandFreshClone, Section: HelpDomain, Usage: "aicoding fresh-clone --profile Smoke|Full|Release [--repo-root PATH] [--json]"},
		{Command: CommandCache, Section: HelpDomain, Usage: "aicoding cache status [--repo-root PATH] [--json]"},
		{Command: CommandCache, Section: HelpDomain, Usage: "aicoding cache clean [--repo-root PATH] [--json]"},
		{Command: CommandCodex, Section: HelpDomain, Usage: "aicoding codex usage parse [--file FILE|-] [--json]"},
		{Command: CommandCodex, Section: HelpDomain, Usage: "aicoding codex usage run [--json] -- codex exec --json \"PROMPT\""},
		{Command: CommandMCP, Section: HelpDomain, Usage: "aicoding mcp list [--codex-config PATH] [--repo-root PATH] [--json]"},
		{Command: CommandMCP, Section: HelpDomain, Usage: "aicoding mcp status|doctor COMPONENT [--codex-config PATH] [--repo-root PATH] [--json]"},
		{Command: CommandMCP, Section: HelpDomain, Usage: "aicoding mcp verify COMPONENT|--all --profile Smoke|Full|Release [--configured] [--codex-config PATH] [--repo-root PATH] [--json]"},
		{Command: CommandTag, Section: HelpDomain, Usage: "aicoding tag audit [--repo-root PATH] [--json]"},
		{Command: CommandGovernance, Section: HelpDomain, Usage: "aicoding governance lint [--repo-root PATH] [--json]"},
		{Command: CommandGovernance, Section: HelpDomain, Usage: "aicoding governance dependencies [--repo-root PATH] [--json]"},
		{Command: CommandGovernance, Section: HelpDomain, Usage: "aicoding governance layout [--repo-root PATH] [--json]"},
		{Command: CommandGovernance, Section: HelpDomain, Usage: "aicoding governance reuse [--repo-root PATH] [--json]"},
		{Command: CommandKit, Section: HelpDomain, Usage: "aicoding kit list [--repo-root PATH] [--json]"},
		{Command: CommandKit, Section: HelpDomain, Usage: "aicoding kit verify --all --profile Smoke|Lifecycle [--repo-root PATH] [--json]"},
		{Command: CommandKit, Section: HelpDomain, Usage: "aicoding kit doctor [--repo-root PATH] [--json]"},
		{Command: CommandDoctor, Section: HelpDomain, Usage: "aicoding doctor perf [--repo-root PATH] [--json]"},
		{Command: CommandDoctor, Section: HelpDomain, Usage: "aicoding doctor pwsh [--repo-root PATH] [--json]"},
		{Command: CommandDoctor, Section: HelpDomain, Usage: "aicoding doctor pwsh-budget [--repo-root PATH] [--json]"},
		{Command: CommandVerify, Section: HelpDomain, Usage: "aicoding verify hooks [--repo-root PATH] [--json]"},
		{Command: CommandVerify, Section: HelpDomain, Usage: "aicoding verify repo-text [--repo-root PATH] [--json]"},
		{Command: CommandVerify, Section: HelpDomain, Usage: "aicoding verify release-notes [--repo-root PATH] [--json]"},
		{Command: CommandPowerShell, Section: HelpDomain, Usage: "aicoding powershell regex-lint --staged [--repo-root PATH] [--json]"},
		{Command: CommandPowerShell, Section: HelpDomain, Usage: "aicoding powershell regex-lint --path PATH [--repo-root PATH] [--json]"},
	},
)

func mustCommandCatalog(routes []commandRoute, sections []HelpSection, help []HelpForm) typedCommandCatalog {
	catalog, err := newCommandCatalog(routes, sections, help)
	if err != nil {
		panic(err)
	}
	return catalog
}

func newCommandCatalog(routes []commandRoute, sections []HelpSection, help []HelpForm) (typedCommandCatalog, error) {
	catalog := typedCommandCatalog{
		descriptor: CatalogDescriptor{
			Commands: make([]CommandDescriptor, 0, len(routes)),
			Sections: append([]HelpSection(nil), sections...),
			Help:     append([]HelpForm(nil), help...),
		},
		routes:  make(map[string]*commandRoute, len(routes)),
		ordered: append([]commandRoute(nil), routes...),
	}
	ids := make(map[CommandID]struct{}, len(routes))
	for index := range catalog.ordered {
		route := &catalog.ordered[index]
		descriptor := route.descriptor
		if descriptor.ID == "" || descriptor.Name == "" {
			return typedCommandCatalog{}, fmt.Errorf("command id and name are required")
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
	for _, form := range help {
		if _, exists := ids[form.Command]; !exists {
			return typedCommandCatalog{}, fmt.Errorf("help references unknown command: %s", form.Command)
		}
		if _, exists := sectionIDs[form.Section]; !exists {
			return typedCommandCatalog{}, fmt.Errorf("help references unknown section: %s", form.Section)
		}
		if strings.TrimSpace(form.Usage) == "" {
			return typedCommandCatalog{}, fmt.Errorf("help usage is empty for command: %s", form.Command)
		}
		commandsWithHelp[form.Command] = struct{}{}
	}
	for id := range ids {
		if _, exists := commandsWithHelp[id]; !exists {
			return typedCommandCatalog{}, fmt.Errorf("command %s has no help form", id)
		}
	}
	return catalog, nil
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
		descriptor.Commands[index] = command
	}
	descriptor.Sections = append([]HelpSection(nil), commands.descriptor.Sections...)
	descriptor.Help = append([]HelpForm(nil), commands.descriptor.Help...)
	return descriptor
}

func CatalogSnapshot() registry.Snapshot {
	catalogSnapshotOnce.Do(func() {
		var err error
		catalogSnapshot, err = registry.NewSnapshot("command-catalog", commands.descriptor)
		if err != nil {
			panic(err)
		}
	})
	return catalogSnapshot
}

func commandRequiresSubcommand(command string) bool {
	route, ok := commands.lookup(command)
	return ok && route.descriptor.RequiresSubcommand
}

func writeCatalogHelp(w io.Writer) {
	catalogHelpOnce.Do(func() {
		catalogHelpText = renderCatalogHelp()
	})
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
