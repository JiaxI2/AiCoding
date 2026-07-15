package cuserstyle

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func RunContract(args []string) error {
	fs := flag.NewFlagSet("contract", flag.ContinueOnError)
	configPath := fs.String("config", "", "configuration")
	file := fs.String("file", "", "target file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *configPath == "" || *file == "" {
		return fmt.Errorf("--config and --file are required")
	}
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		return err
	}
	result := map[string]any{
		"managed":       !isExcluded(*file, cfg),
		"file":          *file,
		"standard":      cfg.Standard,
		"reference":     cfg.Reference,
		"style":         cfg.Style,
		"naming":        cfg.Naming,
		"documentation": cfg.Docs,
		"readability":   cfg.Readability,
		"macros":        cfg.Macro,
		"gates":         cfg.Gates,
		"functionBodyContract": map[string]any{
			"modifyOnlyFunctionBody":   cfg.Agent.ModifyOnlyFunctionBody,
			"preservePrototype":        cfg.Agent.PreservePrototype,
			"preserveDocumentation":    cfg.Agent.PreserveDocumentation,
			"boundedExecutionRequired": cfg.Agent.BoundedExecution,
			"dynamicAllocationAllowed": !cfg.Safety.ForbidDynamicAllocation,
			"vlaAllowed":               !cfg.Safety.ForbidVLA,
			"unboundedLoopAllowed":     !cfg.Safety.ForbidUnboundedLoop,
			"forbiddenCalls":           cfg.Safety.ForbiddenCalls,
			"maxParameters":            cfg.Safety.MaxParameters,
		},
		"requiredValidation": []string{
			"python tools/rules/build_rule_catalog.py --check",
			"python tools/json/validate_json_contracts.py",
			"go test ./...",
			"scripts/verify.ps1 or scripts/verify.sh",
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
