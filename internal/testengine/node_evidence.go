package testengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/JiaxI2/AiCoding/internal/gitx"
	"github.com/JiaxI2/AiCoding/internal/registry"
	"github.com/JiaxI2/AiCoding/internal/validationevidence"
)

const (
	nodeRepo              = "repo"
	nodeGo                = "go"
	nodeDocSync           = "docsync"
	nodeGovernance        = "governance"
	nodeLifecycleReadonly = "lifecycle-readonly"
)

var listValidationTreeEntries = gitx.TreeEntries

type nodeEvidencePlan struct {
	groups          []nodeEvidenceGroup
	selectedCount   int
	hitCount        int
	checkDurationMS int64
	available       bool
}

type nodeEvidenceGroup struct {
	name          string
	tests         []TestCase
	fingerprint   validationevidence.Fingerprint
	decision      validationevidence.ReuseDecision
	cached        map[string]Status
	invalidReason string
}

type nodeResultStatus struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
}

type nodeEvidencePayload struct {
	SchemaVersion   int                `json:"schemaVersion"`
	Node            string             `json:"node"`
	Profile         Profile            `json:"profile"`
	NodeInputDigest string             `json:"nodeInputDigest"`
	Results         []nodeResultStatus `json:"results"`
}

type nodeEvidencePublishError struct {
	code    validationevidence.ErrorCode
	message string
}

func validationNode(configured string) (string, error) {
	node := strings.ToLower(strings.TrimSpace(configured))
	if node == "" {
		node = nodeRepo
	}
	switch node {
	case nodeRepo, nodeGo, nodeDocSync, nodeGovernance, nodeLifecycleReadonly:
		return node, nil
	default:
		return "", fmt.Errorf("unsupported validation node %q", configured)
	}
}

func buildNodeEvidencePlan(
	store validationevidence.Repository,
	subject validationevidence.Subject,
	whole validationevidence.Fingerprint,
	cfg Config,
	testCases []TestCase,
	check bool,
) (nodeEvidencePlan, error) {
	plan := nodeEvidencePlan{}
	groupIndex := map[string]int{}
	for _, testCase := range testCases {
		if !profileEnabled(testCase, cfg.Profile) {
			continue
		}
		node, err := validationNode(testCase.Node)
		if err != nil {
			return plan, fmt.Errorf("test case %s: %w", testCase.ID, err)
		}
		index, exists := groupIndex[node]
		if !exists {
			index = len(plan.groups)
			groupIndex[node] = index
			plan.groups = append(plan.groups, nodeEvidenceGroup{name: node})
		}
		plan.groups[index].tests = append(plan.groups[index].tests, testCase)
		plan.selectedCount++
	}
	if !subject.Reusable || plan.selectedCount == 0 {
		return plan, nil
	}
	entries, err := listValidationTreeEntries(cfg.Repo, subject.TreeOID)
	if err != nil {
		return plan, err
	}
	plan.available = true
	for index := range plan.groups {
		group := &plan.groups[index]
		inputDigest, err := validationNodeInputDigest(group.name, entries)
		if err != nil {
			return plan, err
		}
		group.fingerprint, err = store.DeriveNodeFingerprint(whole, group.name, inputDigest)
		if err != nil {
			return plan, err
		}
		if !check {
			continue
		}
		group.decision = store.CheckNode(subject, group.fingerprint)
		plan.checkDurationMS += group.decision.CheckDurationMS
		if !group.decision.Hit {
			continue
		}
		group.cached, err = decodeNodeEvidence(cfg.Profile, *group)
		if err != nil {
			group.invalidReason = string(validationevidence.CodeReceiptInvalid) + ": node " + group.name + ": " + err.Error()
			group.decision.Hit = false
			group.decision.Code = validationevidence.CodeReceiptInvalid
			group.decision.Reason = err.Error()
			continue
		}
		plan.hitCount += len(group.tests)
	}
	return plan, nil
}

func validationNodeInputDigest(node string, entries []gitx.TreeEntry) (string, error) {
	selected := make([]gitx.TreeEntry, 0, len(entries))
	for _, entry := range entries {
		if validationNodeOwnsPath(node, entry.Path) {
			selected = append(selected, entry)
		}
	}
	sort.Slice(selected, func(i, j int) bool {
		if selected[i].Path == selected[j].Path {
			if selected[i].Mode == selected[j].Mode {
				if selected[i].Type == selected[j].Type {
					return selected[i].OID < selected[j].OID
				}
				return selected[i].Type < selected[j].Type
			}
			return selected[i].Mode < selected[j].Mode
		}
		return selected[i].Path < selected[j].Path
	})
	snapshot, err := registry.NewSnapshot("validation-node-input-"+node, struct {
		Node    string           `json:"node"`
		Entries []gitx.TreeEntry `json:"entries"`
	}{Node: node, Entries: selected})
	if err != nil {
		return "", err
	}
	return snapshot.Digest(), nil
}

func validationNodeOwnsPath(node, path string) bool {
	path = strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	isGoInput := strings.HasSuffix(path, ".go") || path == "go.mod" || path == "go.sum" || path == "testdata" || strings.HasPrefix(path, "testdata/") || strings.Contains(path, "/testdata/")
	isRootDocs := path == "changelog.md" || strings.HasPrefix(path, "readme") && !strings.Contains(path, "/")
	switch node {
	case nodeRepo:
		return true
	case nodeGo:
		return isGoInput
	case nodeDocSync:
		return isRootDocs || strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, "internal/docsync/") || strings.HasPrefix(path, "internal/testengine/") || strings.HasPrefix(path, "internal/registry/") || strings.HasPrefix(path, "internal/report/") || strings.HasPrefix(path, "internal/cli/") || path == "config/kits/docsync-plus.json" || path == "go.mod" || path == "go.sum"
	case nodeGovernance:
		return isGoInput || isRootDocs || strings.HasPrefix(path, ".github/") || strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, "config/") || path == ".gitattributes" || path == ".gitignore" || path == "taskfile.yml"
	case nodeLifecycleReadonly:
		return isGoInput || strings.HasPrefix(path, "config/") || strings.HasPrefix(path, "codingkit/") || path == "taskfile.yml"
	default:
		return false
	}
}

func decodeNodeEvidence(profile Profile, group nodeEvidenceGroup) (map[string]Status, error) {
	if group.decision.Receipt == nil || group.decision.ReportBundle == nil {
		return nil, fmt.Errorf("matching node Receipt has no retained report")
	}
	var payload nodeEvidencePayload
	if err := json.Unmarshal(group.decision.ReportBundle.ResultsJSON, &payload); err != nil {
		return nil, fmt.Errorf("decode retained node report: %w", err)
	}
	if payload.SchemaVersion != 1 || payload.Node != group.name || payload.Profile != profile || payload.NodeInputDigest != group.fingerprint.NodeInputDigest {
		return nil, fmt.Errorf("retained node report identity does not match")
	}
	expected := make(map[string]struct{}, len(group.tests))
	for _, testCase := range group.tests {
		expected[testCase.ID] = struct{}{}
	}
	cached := make(map[string]Status, len(payload.Results))
	for _, result := range payload.Results {
		if _, ok := expected[result.ID]; !ok {
			return nil, fmt.Errorf("retained node report contains unexpected case %s", result.ID)
		}
		if _, duplicate := cached[result.ID]; duplicate {
			return nil, fmt.Errorf("retained node report duplicates case %s", result.ID)
		}
		if result.Status != Pass && result.Status != Warn && result.Status != Skip {
			return nil, fmt.Errorf("retained node report contains non-reusable status for %s", result.ID)
		}
		cached[result.ID] = result.Status
	}
	if len(cached) != len(expected) {
		return nil, fmt.Errorf("retained node report is missing selected cases")
	}
	digest, err := nodeResultStatusDigest(profile, group.name, payload.Results)
	if err != nil {
		return nil, err
	}
	if digest != group.decision.Receipt.ResultsDigest {
		return nil, fmt.Errorf("retained node statuses do not match Receipt")
	}
	return cached, nil
}

func executeWithNodeReuse(ctx context.Context, cfg Config, testCases []TestCase, plan nodeEvidencePlan) []Result {
	if plan.hitCount == 0 {
		return executeTestCases(ctx, cfg, testCases)
	}
	type cachedCase struct {
		node   string
		status Status
	}
	cached := make(map[string]cachedCase, plan.hitCount)
	for _, group := range plan.groups {
		for id, status := range group.cached {
			cached[id] = cachedCase{node: group.name, status: status}
		}
	}
	misses := make([]TestCase, 0, plan.selectedCount-plan.hitCount)
	for _, testCase := range testCases {
		if !profileEnabled(testCase, cfg.Profile) {
			continue
		}
		if _, hit := cached[testCase.ID]; !hit {
			misses = append(misses, testCase)
		}
	}
	executed := []Result{}
	if len(misses) > 0 {
		executed = executeTestCases(ctx, cfg, misses)
	}
	executedByID := make(map[string]Result, len(executed))
	for _, result := range executed {
		executedByID[result.ID] = result
	}
	merged := make([]Result, 0, len(testCases)+len(executed))
	consumed := make(map[string]bool, len(executedByID))
	for _, testCase := range testCases {
		if !profileEnabled(testCase, cfg.Profile) {
			merged = append(merged, Result{ID: testCase.ID, Category: testCase.Category, Title: testCase.Title, Severity: testCase.Severity, Status: Skip, Reason: "not selected by profile", Profile: cfg.Profile})
			continue
		}
		if hit, ok := cached[testCase.ID]; ok {
			merged = append(merged, Result{
				ID: testCase.ID, Category: testCase.Category, Title: testCase.Title, Severity: testCase.Severity,
				Status: hit.status, ExitCode: 0, JSONValid: testCase.ExpectJSON && hit.status == Pass,
				Command: strings.Join(testCase.Command, " "), Reason: "reused-from-node:" + hit.node, Profile: cfg.Profile,
			})
			continue
		}
		if result, ok := executedByID[testCase.ID]; ok {
			merged = append(merged, result)
			consumed[testCase.ID] = true
		}
	}
	for _, result := range executed {
		if !consumed[result.ID] {
			merged = append(merged, result)
		}
	}
	return merged
}

func (plan nodeEvidencePlan) auditFailures(cfg Config, results []Result) []Result {
	failures := []Result{}
	for _, group := range plan.groups {
		reason := ""
		if group.invalidReason != "" {
			reason = group.invalidReason
		} else if group.decision.Hit {
			statuses, err := nodeStatuses(group.tests, results)
			if err != nil {
				reason = err.Error()
			} else {
				digest, digestErr := nodeResultStatusDigest(cfg.Profile, group.name, statuses)
				if digestErr != nil {
					reason = digestErr.Error()
				} else if group.decision.Receipt == nil || digest != group.decision.Receipt.ResultsDigest {
					reason = "executed node statuses do not match the reusable node Receipt"
				}
			}
		} else if group.decision.Code != "" && group.decision.Code != validationevidence.CodeReceiptMiss && group.decision.Code != validationevidence.CodeSubjectNotReusable {
			reason = string(group.decision.Code) + ": " + group.decision.Reason
		}
		if reason == "" {
			continue
		}
		idNode := strings.ToUpper(strings.ReplaceAll(group.name, "-", "_"))
		failures = append(failures, Result{
			ID: "EVIDENCE-NODE-" + idNode, Category: "VALIDATION_EVIDENCE", Title: "节点 Receipt 复用审计",
			Severity: Required, Status: Fail, ExitCode: 1, Reason: group.name + ": " + reason, Profile: cfg.Profile,
		})
	}
	return failures
}

func (plan nodeEvidencePlan) publish(store validationevidence.Repository, subject validationevidence.Subject, results []Result) []nodeEvidencePublishError {
	if !plan.available || !subject.Reusable {
		return nil
	}
	errorsFound := []nodeEvidencePublishError{}
	for _, group := range plan.groups {
		statuses, err := nodeStatuses(group.tests, results)
		if err != nil || !nodeStatusesEligible(group.tests, statuses) {
			continue
		}
		digest, err := nodeResultStatusDigest(group.fingerprint.Profile, group.name, statuses)
		if err != nil {
			errorsFound = append(errorsFound, newNodeEvidencePublishError(group.name, err))
			continue
		}
		bundle, err := nodeReportBundle(group, statuses, digest)
		if err != nil {
			errorsFound = append(errorsFound, newNodeEvidencePublishError(group.name, err))
			continue
		}
		_, err = store.PutNode(validationevidence.Receipt{
			ValidationIdentity: group.fingerprint.Identity,
			Fingerprint:        group.fingerprint,
			Conclusion:         "PASS",
			ResultsDigest:      digest,
			Reusable:           true,
			Scope:              subject.Scope,
		}, bundle)
		if err != nil {
			errorsFound = append(errorsFound, newNodeEvidencePublishError(group.name, err))
		}
	}
	return errorsFound
}

func newNodeEvidencePublishError(node string, err error) nodeEvidencePublishError {
	code := validationevidence.CodeStoreError
	var evidenceError *validationevidence.Error
	if errors.As(err, &evidenceError) {
		code = evidenceError.Code
	}
	return nodeEvidencePublishError{code: code, message: "node " + node + ": " + err.Error()}
}

func nodeStatuses(testCases []TestCase, results []Result) ([]nodeResultStatus, error) {
	byID := make(map[string]Result, len(results))
	for _, result := range results {
		byID[result.ID] = result
	}
	statuses := make([]nodeResultStatus, 0, len(testCases))
	for _, testCase := range testCases {
		result, ok := byID[testCase.ID]
		if !ok {
			return nil, fmt.Errorf("selected node case has no result: %s", testCase.ID)
		}
		statuses = append(statuses, nodeResultStatus{ID: testCase.ID, Status: result.Status})
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].ID < statuses[j].ID })
	return statuses, nil
}

func nodeStatusesEligible(testCases []TestCase, statuses []nodeResultStatus) bool {
	statusByID := make(map[string]Status, len(statuses))
	for _, result := range statuses {
		if result.Status != Pass && result.Status != Warn && result.Status != Skip {
			return false
		}
		statusByID[result.ID] = result.Status
	}
	for _, testCase := range testCases {
		status, ok := statusByID[testCase.ID]
		if !ok || testCase.Severity == Required && status != Pass || status == Skip && testCase.OptionalPath == "" {
			return false
		}
	}
	return true
}

func nodeResultStatusDigest(profile Profile, node string, statuses []nodeResultStatus) (string, error) {
	ordered := append([]nodeResultStatus(nil), statuses...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].ID == ordered[j].ID {
			return ordered[i].Status < ordered[j].Status
		}
		return ordered[i].ID < ordered[j].ID
	})
	snapshot, err := registry.NewSnapshot("validation-node-result-statuses", struct {
		Profile  Profile            `json:"profile"`
		Node     string             `json:"node"`
		Statuses []nodeResultStatus `json:"statuses"`
	}{Profile: profile, Node: node, Statuses: ordered})
	if err != nil {
		return "", err
	}
	return snapshot.Digest(), nil
}

func nodeReportBundle(group nodeEvidenceGroup, statuses []nodeResultStatus, resultsDigest string) (validationevidence.ReportBundle, error) {
	payload := nodeEvidencePayload{
		SchemaVersion: 1, Node: group.name, Profile: group.fingerprint.Profile,
		NodeInputDigest: group.fingerprint.NodeInputDigest, Results: statuses,
	}
	resultsJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return validationevidence.ReportBundle{}, err
	}
	summaryJSON, err := json.MarshalIndent(struct {
		SchemaVersion int     `json:"schemaVersion"`
		Node          string  `json:"node"`
		Profile       Profile `json:"profile"`
		Conclusion    string  `json:"conclusion"`
		ResultsDigest string  `json:"resultsDigest"`
	}{SchemaVersion: 1, Node: group.name, Profile: group.fingerprint.Profile, Conclusion: "PASS", ResultsDigest: resultsDigest}, "", "  ")
	if err != nil {
		return validationevidence.ReportBundle{}, err
	}
	markdown := []byte(fmt.Sprintf("# Validation Node Evidence\n\n- Node: `%s`\n- Profile: `%s`\n- Results digest: `%s`\n", group.name, group.fingerprint.Profile, resultsDigest))
	return validationevidence.ReportBundle{ResultsJSON: resultsJSON, SummaryJSON: summaryJSON, ReportMarkdown: markdown}, nil
}

func (plan nodeEvidencePlan) invalidReason() string {
	for _, group := range plan.groups {
		if group.invalidReason != "" {
			return group.invalidReason
		}
		if group.decision.Code == validationevidence.CodeReceiptInvalid || group.decision.Code == validationevidence.CodeStoreError {
			return string(group.decision.Code) + ": node " + group.name + ": " + group.decision.Reason
		}
	}
	return ""
}

func (plan nodeEvidencePlan) failClosedReuseError() error {
	for _, group := range plan.groups {
		if err := failClosedReuseError(group.decision); err != nil {
			return fmt.Errorf("node %s: %w", group.name, err)
		}
	}
	return nil
}

func appendReceiptInvalidReason(current, addition string) string {
	if current == "" {
		return addition
	}
	if addition == "" {
		return current
	}
	return current + "; " + addition
}
