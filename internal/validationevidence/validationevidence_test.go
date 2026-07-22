package validationevidence

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JiaxI2/AiCoding/internal/gitx"
)

func TestPackageBoundaryAndPublicAPIRemainSmall(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	exported := make([]string, 0, 14)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range parsed.Imports {
			name := strings.Trim(imported.Path.Value, "\"")
			if strings.Contains(name, "/internal/") && name != "github.com/JiaxI2/AiCoding/internal/gitx" && name != "github.com/JiaxI2/AiCoding/internal/pathpolicy" {
				t.Fatalf("validationevidence imports business package %s", name)
			}
		}
		parsed, err = parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Name.IsExported() {
				exported = append(exported, function.Name.Name)
			}
		}
	}
	sort.Strings(exported)
	if len(exported) > 14 {
		t.Fatalf("validationevidence public API grew beyond 13 operations plus Error(): %v", exported)
	}
	for _, required := range []string{"BindCommit", "Capture", "Check", "CheckNode", "Clean", "DeriveNodeFingerprint", "Fingerprint", "GatePush", "List", "LoadPolicy", "Open", "Put", "PutNode"} {
		index := sort.SearchStrings(exported, required)
		if index == len(exported) || exported[index] != required {
			t.Fatalf("validationevidence public API is missing %s: %v", required, exported)
		}
	}
	subjectSource, err := os.ReadFile("subject.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, leakedGitLayout := range []string{"gitdir:", `".git"`, `"commondir"`} {
		if strings.Contains(string(subjectSource), leakedGitLayout) {
			t.Fatalf("subject.go owns Git layout knowledge %q instead of gitx", leakedGitLayout)
		}
	}
}

func TestPolicyLoadIsStrict(t *testing.T) {
	repo := t.TempDir()
	valid := `{
  "schemaVersion": 1,
  "unmatchedAction": "allow",
  "contexts": [{
    "id": "stable",
    "remoteRef": "refs/heads/main",
    "requiredProfile": "release",
    "requireFastForward": true,
    "allowDelete": false
  }]
}`
	writeEvidenceFile(t, repo, validationPolicyPath, valid)
	policy, err := LoadPolicy(repo)
	if err != nil || len(policy.Contexts) != 1 || policy.Contexts[0].RequiredProfile != "release" {
		t.Fatalf("LoadPolicy(valid) = %#v, %v", policy, err)
	}
	writeEvidenceFile(t, repo, validationPolicyPath, strings.Replace(valid, `"allowDelete": false`, `"allowDelete": false, "unknown": true`, 1))
	if _, err := LoadPolicy(repo); err == nil {
		t.Fatal("LoadPolicy accepted an unknown field")
	}
	writeEvidenceFile(t, repo, validationPolicyPath, strings.Replace(valid, `"release"`, `"manual"`, 1))
	if _, err := LoadPolicy(repo); err == nil {
		t.Fatal("LoadPolicy accepted an unsupported profile")
	}
}

func TestPushContextPrefixUsesSharedPathPolicyBoundary(t *testing.T) {
	policy := Policy{SchemaVersion: 1, UnmatchedAction: "allow", Contexts: []PushContext{
		{ID: "heads", RemoteRefPrefix: "refs/heads/", RequiredProfile: "smoke"},
	}}
	if err := validatePolicy(policy); err != nil {
		t.Fatal(err)
	}
	context, ok := matchPushContext(policy, "refs/heads/feature/x")
	if !ok || context.ID != "heads" {
		t.Fatalf("nested ref did not match prefix: context=%#v matched=%v", context, ok)
	}
	if _, ok := matchPushContext(policy, "refs/headship/x"); ok {
		t.Fatal("prefix crossed a path segment boundary")
	}
}

func TestContextGateUsesPushedCommitTreeAndProfileAlias(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "first")
	firstCommit := mustEvidenceGit(t, repo, "rev-parse", "HEAD")
	store, _, fingerprint := evidenceFixture(t, repo, TargetHead)
	receipt := putFixture(t, store, fingerprint)
	if err := store.BindCommit(firstCommit, receipt); err != nil {
		t.Fatal(err)
	}

	writeEvidenceFile(t, repo, "tracked.txt", "two\n")
	mustEvidenceGit(t, repo, "commit", "-am", "second")
	secondCommit := mustEvidenceGit(t, repo, "rev-parse", "HEAD")
	zero := strings.Repeat("0", len(firstCommit))
	policy := Policy{SchemaVersion: 1, UnmatchedAction: "allow", Contexts: []PushContext{{
		ID: "stable-main", RemoteRef: "refs/heads/main", RequiredProfile: "smoke", RequireFastForward: true,
	}}}

	gate := store.GatePush(policy, []gitx.PushUpdate{{
		LocalRef: "refs/heads/old", LocalOID: firstCommit, RemoteRef: "refs/heads/main", RemoteOID: zero,
	}})
	if !gate.OK || gate.Required != 1 || len(gate.Updates) != 1 || gate.Updates[0].Code != CodeReceiptHit || gate.Updates[0].SubjectTreeOID != fingerprint.SubjectTreeOID {
		t.Fatalf("exact pushed commit did not hit its alias: %#v", gate)
	}

	bypassed := store.GatePush(policy, []gitx.PushUpdate{{
		LocalRef: "refs/heads/feature", LocalOID: secondCommit, RemoteRef: "refs/heads/feature", RemoteOID: zero,
	}})
	if !bypassed.OK || bypassed.Bypassed != 1 || !bypassed.Updates[0].Allowed || bypassed.Updates[0].ContextID != "" {
		t.Fatalf("unmatched feature ref was not bypassed: %#v", bypassed)
	}

	missing := store.GatePush(policy, []gitx.PushUpdate{{
		LocalRef: "refs/heads/main", LocalOID: secondCommit, RemoteRef: "refs/heads/main", RemoteOID: zero,
	}})
	if missing.OK || missing.Updates[0].Code != CodeReceiptMiss || !strings.Contains(missing.Updates[0].RequiredAction, "--bind-alias") {
		t.Fatalf("protected ref without an alias passed: %#v", missing)
	}

	nonFastForward := store.GatePush(policy, []gitx.PushUpdate{{
		LocalRef: "refs/heads/old", LocalOID: firstCommit, RemoteRef: "refs/heads/main", RemoteOID: secondCommit,
	}})
	if nonFastForward.OK || nonFastForward.Updates[0].Code != CodePushContextRejected || !strings.Contains(nonFastForward.Updates[0].Reason, "fast-forward") {
		t.Fatalf("non-fast-forward update passed: %#v", nonFastForward)
	}

	deletion := store.GatePush(policy, []gitx.PushUpdate{{
		LocalRef: "(delete)", LocalOID: zero, RemoteRef: "refs/heads/main", RemoteOID: secondCommit,
	}})
	if deletion.OK || deletion.Updates[0].Code != CodePushContextRejected || !strings.Contains(deletion.Updates[0].Reason, "deletion") {
		t.Fatalf("protected ref deletion passed: %#v", deletion)
	}

	if err := store.BindCommit(secondCommit, receipt); err == nil {
		t.Fatal("BindCommit accepted a Receipt for a different tree")
	}
	if removed, err := store.Clean("smoke"); err != nil || removed != 1 {
		t.Fatalf("Clean(smoke) = %d, %v", removed, err)
	}
	cleaned := store.GatePush(policy, []gitx.PushUpdate{{
		LocalRef: "refs/heads/old", LocalOID: firstCommit, RemoteRef: "refs/heads/main", RemoteOID: zero,
	}})
	if cleaned.OK || cleaned.Updates[0].Code != CodeReceiptMiss {
		t.Fatalf("profile clean left a usable commit alias: %#v", cleaned)
	}
}

func TestExactCheckPathDoesNotWalkRepositoryFiles(t *testing.T) {
	for _, path := range []string{"checker.go", "subject.go", "fingerprint.go"} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		for _, forbidden := range []string{"filepath.Walk(", "filepath.WalkDir(", "fs.WalkDir(", "os.ReadDir(", "git ls-files"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s introduces repository enumeration through %q", path, forbidden)
			}
		}
	}
	checker, err := os.ReadFile("checker.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(checker), "r.receiptPath(") || !strings.Contains(string(checker), "r.readReceipt(") || strings.Contains(string(checker), "r.List(") {
		t.Fatal("Check no longer uses the exact Receipt path")
	}
}

func TestReceiptSurvivesCommitMessageAmendAndLinkedWorktree(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, _, fingerprint := evidenceFixture(t, repo, TargetHead)
	receipt := putFixture(t, store, fingerprint)

	mustEvidenceGit(t, repo, "commit", "--amend", "-m", "message only")
	afterAmend, err := store.Capture(TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	afterFingerprint, err := store.Fingerprint(afterAmend, fixtureSpec())
	if err != nil {
		t.Fatal(err)
	}
	if afterFingerprint.Identity != fingerprint.Identity || !store.Check(afterAmend, afterFingerprint).Hit {
		t.Fatalf("amend changed reusable identity: before=%s after=%s", fingerprint.Identity, afterFingerprint.Identity)
	}

	linked := filepath.Join(t.TempDir(), "linked")
	mustEvidenceGit(t, repo, "worktree", "add", "--detach", linked, "HEAD")
	linkedStore, err := Open(linked)
	if err != nil {
		t.Fatal(err)
	}
	linkedSubject, err := linkedStore.Capture(TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	linkedFingerprint, err := linkedStore.Fingerprint(linkedSubject, fixtureSpec())
	if err != nil {
		t.Fatal(err)
	}
	decision := linkedStore.Check(linkedSubject, linkedFingerprint)
	if !decision.Hit || decision.Receipt == nil || decision.Receipt.ReceiptID != receipt.ReceiptID {
		t.Fatalf("linked worktree did not reuse Receipt: %#v", decision)
	}
}

func TestDirtyAndChangedContentInvalidateReuse(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, _, fingerprint := evidenceFixture(t, repo, TargetHead)
	putFixture(t, store, fingerprint)

	writeEvidenceFile(t, repo, "untracked.txt", "new\n")
	dirty, err := store.Capture(TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	dirtyFingerprint, err := store.Fingerprint(dirty, fixtureSpec())
	if err != nil {
		t.Fatal(err)
	}
	if decision := store.Check(dirty, dirtyFingerprint); decision.Hit || decision.Code != CodeSubjectNotReusable {
		t.Fatalf("untracked file reused evidence: %#v", decision)
	}
	if err := os.Remove(filepath.Join(repo, "untracked.txt")); err != nil {
		t.Fatal(err)
	}
	writeEvidenceFile(t, repo, "tracked.txt", "two\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	indexSubject, err := store.Capture(TargetIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !indexSubject.Reusable || indexSubject.Mode != SubjectIndex {
		t.Fatalf("index-only subject = %#v", indexSubject)
	}
	changedFingerprint, err := store.Fingerprint(indexSubject, fixtureSpec())
	if err != nil {
		t.Fatal(err)
	}
	if changedFingerprint.Identity == fingerprint.Identity || store.Check(indexSubject, changedFingerprint).Hit {
		t.Fatal("changed tracked content reused the previous Receipt")
	}
}

func TestNodeReceiptReusesAcrossWholeTreesButNotDirtySubjects(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, firstSubject, firstWhole := evidenceFixture(t, repo, TargetHead)
	inputDigest := fixtureDigest("go-node-input")
	firstNode, err := store.DeriveNodeFingerprint(firstWhole, "go", inputDigest)
	if err != nil {
		t.Fatal(err)
	}
	putNodeFixture(t, store, firstNode)
	if decision := store.CheckNode(firstSubject, firstNode); !decision.Hit || decision.Receipt == nil || decision.Receipt.Fingerprint.SubjectTreeOID != "" {
		t.Fatalf("fresh node Receipt did not hit: %#v", decision)
	}

	listed, err := store.List("smoke")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("private node Receipt leaked through List: %#v", listed)
	}
	if _, err := os.Stat(store.nodeReportDir(firstNode.Identity)); err != nil {
		t.Fatalf("private node report was not retained separately: %v", err)
	}
	if err := store.BindCommit("HEAD", Receipt{ValidationIdentity: firstNode.Identity, Fingerprint: firstNode}); err == nil {
		t.Fatal("BindCommit accepted private node evidence")
	}

	writeEvidenceFile(t, repo, "tracked.txt", "two\n")
	mustEvidenceGit(t, repo, "commit", "-am", "unrelated tree change")
	_, secondSubject, secondWhole := evidenceFixture(t, repo, TargetHead)
	secondNode, err := store.DeriveNodeFingerprint(secondWhole, "go", inputDigest)
	if err != nil {
		t.Fatal(err)
	}
	if secondWhole.Identity == firstWhole.Identity || secondNode.Identity != firstNode.Identity {
		t.Fatalf("node identity did not isolate its input: whole=%s/%s node=%s/%s", firstWhole.Identity, secondWhole.Identity, firstNode.Identity, secondNode.Identity)
	}
	if decision := store.CheckNode(secondSubject, secondNode); !decision.Hit {
		t.Fatalf("unchanged node input did not reuse across whole trees: %#v", decision)
	}

	writeEvidenceFile(t, repo, "untracked.txt", "dirty\n")
	dirtySubject, err := store.Capture(TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	if decision := store.CheckNode(dirtySubject, secondNode); decision.Hit || decision.Code != CodeSubjectNotReusable {
		t.Fatalf("dirty subject reused node evidence: %#v", decision)
	}
	changedNode, err := store.DeriveNodeFingerprint(secondWhole, "go", fixtureDigest("changed-go-input"))
	if err != nil {
		t.Fatal(err)
	}
	if changedNode.Identity == secondNode.Identity {
		t.Fatal("changed node input did not change node identity")
	}
}

func TestNodeReceiptTamperFailsClosedAndCleanRepairsStore(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, subject, whole := evidenceFixture(t, repo, TargetHead)
	node, err := store.DeriveNodeFingerprint(whole, "docsync", fixtureDigest("docs-node-input"))
	if err != nil {
		t.Fatal(err)
	}
	putNodeFixture(t, store, node)
	if err := os.WriteFile(filepath.Join(store.nodeReportDir(node.Identity), "report.md"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if decision := store.CheckNode(subject, node); decision.Hit || decision.Code != CodeReceiptInvalid {
		t.Fatalf("tampered node report did not fail closed: %#v", decision)
	}
	removed, err := store.Clean("smoke")
	if err != nil || removed != 1 {
		t.Fatalf("Clean(smoke) removed %d node Receipts: %v", removed, err)
	}
	if _, err := os.Stat(store.nodeReportDir(node.Identity)); !os.IsNotExist(err) {
		t.Fatalf("node report survived profile clean: %v", err)
	}
	if decision := store.CheckNode(subject, node); decision.Hit || decision.Code != CodeReceiptMiss {
		t.Fatalf("cleaned node evidence remained reusable: %#v", decision)
	}
	putNodeFixture(t, store, node)
	path := store.nodeReceiptPath(node.Profile, node.Node, node.Identity)
	if err := os.WriteFile(path, []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if decision := store.CheckNode(subject, node); decision.Hit || decision.Code != CodeReceiptInvalid {
		t.Fatalf("corrupt node Receipt did not fail closed: %#v", decision)
	}
	if removed, err := store.Clean("smoke"); err != nil || removed != 1 {
		t.Fatalf("clean did not repair corrupt node Receipt: removed=%d err=%v", removed, err)
	}
}

func TestWholeAndNodeStoreOperationsRejectWrongFingerprintKind(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, _, whole := evidenceFixture(t, repo, TargetHead)
	node, err := store.DeriveNodeFingerprint(whole, "repo", fixtureDigest("repo-node-input"))
	if err != nil {
		t.Fatal(err)
	}
	wholeReceipt := Receipt{ValidationIdentity: whole.Identity, Fingerprint: whole, Conclusion: "PASS", ResultsDigest: fixtureDigest("results"), Reusable: true, Scope: Scope{IgnoredFilesOutOfScope: true}}
	nodeReceipt := Receipt{ValidationIdentity: node.Identity, Fingerprint: node, Conclusion: "PASS", ResultsDigest: fixtureDigest("results"), Reusable: true, Scope: Scope{IgnoredFilesOutOfScope: true}}
	if _, err := store.Put(nodeReceipt, fixtureReports()); err == nil {
		t.Fatal("Put accepted a node fingerprint")
	}
	if _, err := store.PutNode(wholeReceipt, fixtureReports()); err == nil {
		t.Fatal("PutNode accepted a whole-tree fingerprint")
	}
	nodeReceipt.Conclusion = "FAIL"
	if _, err := store.PutNode(nodeReceipt, fixtureReports()); err == nil {
		t.Fatal("PutNode cached a failing result")
	}
}

func TestDifferentRepositoriesCannotReuseReceipt(t *testing.T) {
	first := newEvidenceRepo(t)
	second := newEvidenceRepo(t)
	for _, repo := range []string{first, second} {
		writeEvidenceFile(t, repo, "same.txt", "same\n")
		mustEvidenceGit(t, repo, "add", "same.txt")
		mustEvidenceGit(t, repo, "commit", "-m", "same")
	}
	firstStore, _, firstFingerprint := evidenceFixture(t, first, TargetHead)
	secondStore, secondSubject, secondFingerprint := evidenceFixture(t, second, TargetHead)
	if firstFingerprint.SubjectTreeOID != secondFingerprint.SubjectTreeOID || firstFingerprint.Identity == secondFingerprint.Identity {
		t.Fatalf("repository identity isolation failed: %#v %#v", firstFingerprint, secondFingerprint)
	}
	putFixture(t, firstStore, firstFingerprint)
	if err := os.MkdirAll(secondStore.reportDir(firstFingerprint.Identity), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range reportNames {
		content, err := os.ReadFile(filepath.Join(firstStore.reportDir(firstFingerprint.Identity), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(secondStore.reportDir(firstFingerprint.Identity), name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	foreignReceipt, err := os.ReadFile(firstStore.receiptPath(firstFingerprint.Profile, firstFingerprint.Identity))
	if err != nil {
		t.Fatal(err)
	}
	foreignPath := secondStore.receiptPath(firstFingerprint.Profile, firstFingerprint.Identity)
	if err := os.MkdirAll(filepath.Dir(foreignPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreignPath, foreignReceipt, 0o644); err != nil {
		t.Fatal(err)
	}
	if decision := secondStore.Check(secondSubject, firstFingerprint); decision.Hit || decision.Code != CodeFingerprintInvalid {
		t.Fatalf("foreign repository fingerprint was accepted: %#v", decision)
	}
}

func TestSubmoduleGitlinkAndDirtyWorktreeInvalidateReuse(t *testing.T) {
	child := newEvidenceRepo(t)
	writeEvidenceFile(t, child, "child.txt", "one\n")
	mustEvidenceGit(t, child, "add", "child.txt")
	mustEvidenceGit(t, child, "commit", "-m", "child one")

	parent := newEvidenceRepo(t)
	mustEvidenceGit(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "deps/child")
	mustEvidenceGit(t, parent, "commit", "-am", "add child")
	store, _, original := evidenceFixture(t, parent, TargetHead)
	putFixture(t, store, original)

	writeEvidenceFile(t, filepath.Join(parent, "deps", "child"), "child.txt", "dirty\n")
	dirty, err := store.Capture(TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	if dirty.Reusable || !strings.Contains(dirty.ReusableReason, "dirty submodule") {
		t.Fatalf("dirty submodule subject = %#v", dirty)
	}
	mustEvidenceGit(t, filepath.Join(parent, "deps", "child"), "reset", "--hard", "HEAD")

	writeEvidenceFile(t, child, "child.txt", "two\n")
	mustEvidenceGit(t, child, "add", "child.txt")
	mustEvidenceGit(t, child, "commit", "-m", "child two")
	newChildCommit := mustEvidenceGit(t, child, "rev-parse", "HEAD")
	mustEvidenceGit(t, filepath.Join(parent, "deps", "child"), "fetch", child, newChildCommit)
	mustEvidenceGit(t, filepath.Join(parent, "deps", "child"), "checkout", "--detach", newChildCommit)
	mustEvidenceGit(t, parent, "add", "deps/child")
	indexSubject, err := store.Capture(TargetIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !indexSubject.Reusable || indexSubject.TreeOID == original.SubjectTreeOID {
		t.Fatalf("staged gitlink did not produce a reusable new tree: %#v", indexSubject)
	}
	changed, err := store.Fingerprint(indexSubject, fixtureSpec())
	if err != nil {
		t.Fatal(err)
	}
	if changed.Identity == original.Identity || store.Check(indexSubject, changed).Hit {
		t.Fatal("changed submodule gitlink reused the old Receipt")
	}
}

func TestSemanticAndConfigInputsChangeIdentity(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	writeEvidenceFile(t, repo, "config.json", "{\"value\":1}\n")
	mustEvidenceGit(t, repo, "add", ".")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, err := Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := store.Capture(TargetHead)
	if err != nil {
		t.Fatal(err)
	}
	baseSpec := fixtureSpec()
	baseSpec.ConfigPaths = []string{"config.json"}
	base, err := store.Fingerprint(subject, baseSpec)
	if err != nil {
		t.Fatal(err)
	}

	cases := []FingerprintSpec{
		{Profile: "smoke", ValidationPlanDigest: fixtureDigest("changed-plan"), EngineSemanticDigest: baseSpec.EngineSemanticDigest, OptionsDigest: baseSpec.OptionsDigest, ConfigPaths: baseSpec.ConfigPaths},
		{Profile: "smoke", ValidationPlanDigest: baseSpec.ValidationPlanDigest, EngineSemanticDigest: fixtureDigest("changed-engine"), OptionsDigest: baseSpec.OptionsDigest, ConfigPaths: baseSpec.ConfigPaths},
		{Profile: "smoke", ValidationPlanDigest: baseSpec.ValidationPlanDigest, EngineSemanticDigest: baseSpec.EngineSemanticDigest, OptionsDigest: fixtureDigest("changed-options"), ConfigPaths: baseSpec.ConfigPaths},
	}
	for _, spec := range cases {
		changed, err := store.Fingerprint(subject, spec)
		if err != nil {
			t.Fatal(err)
		}
		if changed.Identity == base.Identity {
			t.Fatalf("semantic change did not invalidate identity: %#v", spec)
		}
	}
	writeEvidenceFile(t, repo, "config.json", "{\"value\":2}\n")
	changedConfig, err := store.Fingerprint(subject, baseSpec)
	if err != nil {
		t.Fatal(err)
	}
	if changedConfig.ConfigDigest == base.ConfigDigest || changedConfig.Identity == base.Identity {
		t.Fatal("config content change did not invalidate identity")
	}
}

func TestTamperedReportAndReceiptFailClosed(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, subject, fingerprint := evidenceFixture(t, repo, TargetHead)
	putFixture(t, store, fingerprint)

	if err := os.WriteFile(filepath.Join(store.reportDir(fingerprint.Identity), "report.md"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if decision := store.Check(subject, fingerprint); decision.Hit || decision.Code != CodeReceiptInvalid {
		t.Fatalf("tampered report did not fail closed: %#v", decision)
	}

	if _, err := store.Clean(fingerprint.Profile); err != nil {
		t.Fatal(err)
	}
	putFixture(t, store, fingerprint)
	path := store.receiptPath(fingerprint.Profile, fingerprint.Identity)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	receipt["conclusion"] = "FAIL"
	raw, _ = json.Marshal(receipt)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if decision := store.Check(subject, fingerprint); decision.Hit || decision.Code != CodeReceiptInvalid {
		t.Fatalf("tampered Receipt did not fail closed: %#v", decision)
	}
}

func TestConcurrentPutIsIdempotentOnWindowsRenameSemantics(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, subject, fingerprint := evidenceFixture(t, repo, TargetHead)

	const writers = 8
	var wait sync.WaitGroup
	errs := make(chan error, writers)
	for index := 0; index < writers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			receipt := Receipt{ValidationIdentity: fingerprint.Identity, Fingerprint: fingerprint, Conclusion: "PASS", ResultsDigest: fixtureDigest("results"), Reusable: true, Scope: Scope{IgnoredFilesOutOfScope: true}}
			_, err := store.Put(receipt, fixtureReports())
			errs <- err
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if decision := store.Check(subject, fingerprint); !decision.Hit {
		t.Fatalf("concurrent Put left invalid evidence: %#v", decision)
	}
}

func TestToolchainCacheHitDoesNotRewriteFile(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, subject, _ := evidenceFixture(t, repo, TargetHead)
	path := filepath.Join(store.root, "toolchain.json")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if _, err := store.Fingerprint(subject, fixtureSpec()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("toolchain cache hit rewrote toolchain.json")
	}
}

func TestCorruptToolchainCacheIsRebuilt(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, subject, fingerprint := evidenceFixture(t, repo, TargetHead)
	path := filepath.Join(store.root, "toolchain.json")
	if err := os.WriteFile(path, []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := store.Fingerprint(subject, fixtureSpec())
	if err != nil {
		t.Fatal(err)
	}
	if rebuilt.ToolchainDigest != fingerprint.ToolchainDigest || rebuilt.Identity != fingerprint.Identity {
		t.Fatal("rebuilding a corrupt toolchain cache changed the actual toolchain identity")
	}
	if _, err := readToolchainCache(path); err != nil {
		t.Fatalf("toolchain cache was not repaired: %v", err)
	}
}

func TestCheckUsesExactReceiptPath(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, subject, fingerprint := evidenceFixture(t, repo, TargetHead)
	putFixture(t, store, fingerprint)
	otherDir := filepath.Join(store.root, "receipts", "full")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDir, strings.Repeat("0", 64)+".json"), []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if decision := store.Check(subject, fingerprint); !decision.Hit {
		t.Fatalf("exact smoke lookup was affected by an unrelated profile directory: %#v", decision)
	}
}

func TestPutRejectsFailAndPathsCannotEscapeStore(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "initial")
	store, _, fingerprint := evidenceFixture(t, repo, TargetHead)
	missingDigest := Receipt{ValidationIdentity: fingerprint.Identity, Fingerprint: fingerprint, Conclusion: "PASS", Reusable: true, Scope: Scope{IgnoredFilesOutOfScope: true}}
	if _, err := store.Put(missingDigest, fixtureReports()); err == nil {
		t.Fatal("Receipt without a results digest was accepted")
	}
	bad := Receipt{ValidationIdentity: fingerprint.Identity, Fingerprint: fingerprint, Conclusion: "FAIL", ResultsDigest: fixtureDigest("results"), Reusable: true, Scope: Scope{IgnoredFilesOutOfScope: true}}
	if _, err := store.Put(bad, fixtureReports()); err == nil {
		t.Fatal("FAIL produced a Receipt")
	}
	if _, err := store.List("../escape"); err == nil {
		t.Fatal("profile path escape was accepted")
	}
	if _, err := store.Fingerprint(Subject{TreeOID: fingerprint.SubjectTreeOID}, FingerprintSpec{Profile: "smoke", ValidationPlanDigest: fixtureDigest("plan"), EngineSemanticDigest: fixtureDigest("engine"), OptionsDigest: fixtureDigest("options"), ConfigPaths: []string{"../escape"}}); err == nil {
		t.Fatal("config path escape was accepted")
	}
}

func TestListReturnsNewestReceiptByFileMtime(t *testing.T) {
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "one\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "one")
	store, _, firstFingerprint := evidenceFixture(t, repo, TargetHead)
	first := putFixture(t, store, firstFingerprint)

	writeEvidenceFile(t, repo, "tracked.txt", "two\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "two")
	_, _, secondFingerprint := evidenceFixture(t, repo, TargetHead)
	second := putFixture(t, store, secondFingerprint)

	older := time.Unix(100, 0)
	newer := time.Unix(200, 0)
	if err := os.Chtimes(store.receiptPath("smoke", second.ValidationIdentity), older, older); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(store.receiptPath("smoke", first.ValidationIdentity), newer, newer); err != nil {
		t.Fatal(err)
	}
	receipts, err := store.List("smoke")
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 2 || receipts[0].ValidationIdentity != first.ValidationIdentity || receipts[1].ValidationIdentity != second.ValidationIdentity {
		t.Fatalf("newest-first Receipt order = %#v", receipts)
	}
}

func evidenceFixture(t *testing.T, repo string, target Target) (Repository, Subject, Fingerprint) {
	t.Helper()
	store, err := Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := store.Capture(target)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := store.Fingerprint(subject, fixtureSpec())
	if err != nil {
		t.Fatal(err)
	}
	return store, subject, fingerprint
}

func fixtureSpec() FingerprintSpec {
	return FingerprintSpec{Profile: "smoke", ValidationPlanDigest: fixtureDigest("plan"), EngineSemanticDigest: fixtureDigest("engine"), OptionsDigest: fixtureDigest("options")}
}

func fixtureDigest(value string) string {
	return digestBytes([]byte(value))
}

func fixtureReports() ReportBundle {
	return ReportBundle{ResultsJSON: []byte("{\"results\":[]}"), SummaryJSON: []byte("{\"conclusion\":\"PASS\"}"), ReportMarkdown: []byte("# PASS\n")}
}

func putFixture(t *testing.T, store Repository, fingerprint Fingerprint) Receipt {
	t.Helper()
	receipt, err := store.Put(Receipt{ValidationIdentity: fingerprint.Identity, Fingerprint: fingerprint, Conclusion: "PASS", ResultsDigest: fixtureDigest("results"), Reusable: true, Scope: Scope{IgnoredFilesOutOfScope: true}}, fixtureReports())
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func putNodeFixture(t *testing.T, store Repository, fingerprint Fingerprint) Receipt {
	t.Helper()
	receipt, err := store.PutNode(Receipt{ValidationIdentity: fingerprint.Identity, Fingerprint: fingerprint, Conclusion: "PASS", ResultsDigest: fixtureDigest("node-results"), Reusable: true, Scope: Scope{IgnoredFilesOutOfScope: true}}, fixtureReports())
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func newEvidenceRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	mustEvidenceGit(t, repo, "init", "-q")
	mustEvidenceGit(t, repo, "config", "user.email", "test@example.com")
	mustEvidenceGit(t, repo, "config", "user.name", "Test User")
	return repo
}

func writeEvidenceFile(t *testing.T, repo, name, content string) {
	t.Helper()
	path := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustEvidenceGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	out, err := gitx.Run(repo, args...)
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(out)
}
