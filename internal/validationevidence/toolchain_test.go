package validationevidence

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestToolchainSemanticDigestChangesWithToolVersions(t *testing.T) {
	tests := []struct {
		name       string
		goVersion  string
		gitVersion string
	}{
		{name: "git", goVersion: "go version go1.26.5 windows/amd64", gitVersion: "git version 2.56.0.windows.1"},
		{name: "go", goVersion: "go version go1.27.0 windows/amd64", gitVersion: "git version 2.55.0.windows.1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := Repository{root: t.TempDir()}
			baseDir := t.TempDir()
			baseGo := writeProbeFile(t, baseDir, "go-base")
			baseGit := writeProbeFile(t, baseDir, "git-base")
			base, err := store.toolchainDigestWith(fakeToolchainProbe(baseGo, baseGit,
				"go version go1.26.5 windows/amd64", "git version 2.55.0.windows.1", "windows", "amd64", nil))
			if err != nil {
				t.Fatal(err)
			}
			changedDir := t.TempDir()
			changed, err := store.toolchainDigestWith(fakeToolchainProbe(
				writeProbeFile(t, changedDir, "go-changed"), writeProbeFile(t, changedDir, "git-changed"),
				test.goVersion, test.gitVersion, "windows", "amd64", nil))
			if err != nil {
				t.Fatal(err)
			}
			if changed == base {
				t.Fatalf("%s version change kept semantic digest %s", test.name, base)
			}
			t.Logf("version-change tool=%s before=%s after=%s", test.name, base, changed)
		})
	}
}

func TestToolchainPathAndMtimeReprobeWithoutSemanticDrift(t *testing.T) {
	store := Repository{root: t.TempDir()}
	firstDir := t.TempDir()
	firstGo := writeProbeFile(t, firstDir, "go")
	firstGit := writeProbeFile(t, firstDir, "git")
	calls := 0
	first, err := store.toolchainDigestWith(fakeToolchainProbe(firstGo, firstGit,
		"go   version go1.26.5 windows/amd64\n", "git version   2.55.0.windows.1\n", "windows", "amd64", &calls))
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("initial probe calls=%d, want 2", calls)
	}

	secondDir := t.TempDir()
	secondGo := writeProbeFile(t, secondDir, "go")
	secondGit := writeProbeFile(t, secondDir, "git")
	second, err := store.toolchainDigestWith(fakeToolchainProbe(secondGo, secondGit,
		"go version go1.26.5 windows/amd64", "git version 2.55.0.windows.1", "windows", "amd64", &calls))
	if err != nil {
		t.Fatal(err)
	}
	if second != first || calls != 4 {
		t.Fatalf("path move digest=%s/%s probeCalls=%d, want same digest and 4 calls", first, second, calls)
	}

	touched := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(secondGit, touched, touched); err != nil {
		t.Fatal(err)
	}
	third, err := store.toolchainDigestWith(fakeToolchainProbe(secondGo, secondGit,
		"go version go1.26.5 windows/amd64", "git version 2.55.0.windows.1", "windows", "amd64", &calls))
	if err != nil {
		t.Fatal(err)
	}
	if third != first || calls != 6 {
		t.Fatalf("mtime touch digest=%s/%s probeCalls=%d, want same digest and 6 calls", first, third, calls)
	}
	cache, err := readToolchainCache(filepath.Join(store.root, "toolchain.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cache.Semantic.Domain != toolchainDigestDomain || cache.Semantic.Version != toolchainDigestVersion {
		t.Fatalf("cache semantic domain/version = %#v", cache.Semantic)
	}
	t.Logf("path-move-and-touch digest=%s probeCalls=%d domain=%s.v%d", third, calls, cache.Semantic.Domain, cache.Semantic.Version)
}

func TestToolchainPlatformArchitectureInjectionChangesDigest(t *testing.T) {
	store := Repository{root: t.TempDir()}
	dir := t.TempDir()
	goPath := writeProbeFile(t, dir, "go")
	gitPath := writeProbeFile(t, dir, "git")
	base, err := store.toolchainDigestWith(fakeToolchainProbe(goPath, gitPath,
		"go version go1.26.5 windows/amd64", "git version 2.55.0.windows.1", "windows", "amd64", nil))
	if err != nil {
		t.Fatal(err)
	}
	platform, err := store.toolchainDigestWith(fakeToolchainProbe(goPath, gitPath,
		"go version go1.26.5 windows/amd64", "git version 2.55.0.windows.1", "linux", "amd64", nil))
	if err != nil {
		t.Fatal(err)
	}
	architecture, err := store.toolchainDigestWith(fakeToolchainProbe(goPath, gitPath,
		"go version go1.26.5 windows/amd64", "git version 2.55.0.windows.1", "linux", "arm64", nil))
	if err != nil {
		t.Fatal(err)
	}
	if base == platform || platform == architecture || base == architecture {
		t.Fatalf("platform/architecture injection did not change digest: %s %s %s", base, platform, architecture)
	}
	repo := newEvidenceRepo(t)
	writeEvidenceFile(t, repo, "tracked.txt", "platform receipt\n")
	mustEvidenceGit(t, repo, "add", "tracked.txt")
	mustEvidenceGit(t, repo, "commit", "-m", "platform receipt")
	evidenceStore, subject, fingerprint := evidenceFixture(t, repo, TargetHead)
	fingerprint = fingerprintWithToolchainDigest(fingerprint, base)
	putFixture(t, evidenceStore, fingerprint)
	injected := fingerprintWithToolchainDigest(fingerprint, platform)
	decision := evidenceStore.Check(subject, injected)
	if decision.Hit || decision.Code != CodeReceiptMiss {
		t.Fatalf("platform injection reuse decision=%#v, want ordinary miss", decision)
	}
	t.Logf("platform-architecture-injection windows/amd64=%s linux/amd64=%s linux/arm64=%s reuse=%s", base, platform, architecture, decision.Code)
}

func TestToolchainProbeFailuresAreFailClosed(t *testing.T) {
	t.Run("locate", func(t *testing.T) {
		probe := fakeToolchainProbe("", "", "", "", "windows", "amd64", nil)
		probe.Locate = func(name string) (executableFingerprint, error) {
			return executableFingerprint{}, errors.New("executable is unreadable: " + name)
		}
		assertFingerprintFailure(t, Repository{root: t.TempDir()}, probe, "unreadable executable")
	})
	t.Run("version-exit", func(t *testing.T) {
		dir := t.TempDir()
		probe := fakeToolchainProbe(writeProbeFile(t, dir, "go"), writeProbeFile(t, dir, "git"),
			"", "", "windows", "amd64", nil)
		probe.Version = func(name string, _ executableFingerprint) (string, error) {
			return "", errors.New(name + " version exited 1")
		}
		assertFingerprintFailure(t, Repository{root: t.TempDir()}, probe, "version command failure")
	})
	t.Run("unparseable", func(t *testing.T) {
		dir := t.TempDir()
		probe := fakeToolchainProbe(writeProbeFile(t, dir, "go"), writeProbeFile(t, dir, "git"),
			"\xff\xfe乱码", "git version 2.55.0.windows.1", "windows", "amd64", nil)
		assertFingerprintFailure(t, Repository{root: t.TempDir()}, probe, "unparseable version output")
	})
}

func TestCorruptToolchainCacheIsRejectedAndRebuiltFromProbe(t *testing.T) {
	store := Repository{root: t.TempDir()}
	dir := t.TempDir()
	goPath := writeProbeFile(t, dir, "go")
	gitPath := writeProbeFile(t, dir, "git")
	calls := 0
	probe := fakeToolchainProbe(goPath, gitPath,
		"go version go1.26.5 windows/amd64", "git version 2.55.0.windows.1", "windows", "amd64", &calls)
	before, err := store.toolchainDigestWith(probe)
	if err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(store.root, "toolchain.json")
	if err := os.WriteFile(cachePath, []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := store.toolchainDigestWith(probe)
	if err != nil {
		t.Fatal(err)
	}
	if before != after || calls != 4 {
		t.Fatalf("corrupt cache rebuild digest=%s/%s probeCalls=%d", before, after, calls)
	}
	if _, err := readToolchainCache(cachePath); err != nil {
		t.Fatalf("rebuilt cache is invalid: %v", err)
	}
	t.Logf("corrupt-cache rejected=true rebuilt=true digest=%s probeCalls=%d", after, calls)
}

func fakeToolchainProbe(goPath, gitPath, goVersion, gitVersion, platform, architecture string, calls *int) toolchainProbe {
	return toolchainProbe{
		Platform: platform, Architecture: architecture,
		Locate: func(name string) (executableFingerprint, error) {
			switch name {
			case "go":
				return fingerprintExecutablePath(goPath)
			case "git":
				return fingerprintExecutablePath(gitPath)
			default:
				return executableFingerprint{}, errors.New("unexpected tool " + name)
			}
		},
		Version: func(name string, _ executableFingerprint) (string, error) {
			if calls != nil {
				*calls++
			}
			if name == "go" {
				return goVersion, nil
			}
			if name == "git" {
				return gitVersion, nil
			}
			return "", errors.New("unexpected tool " + name)
		},
	}
}

func writeProbeFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(name+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(abs)
}

func assertFingerprintFailure(t *testing.T, store Repository, probe toolchainProbe, label string) {
	t.Helper()
	_, err := store.toolchainDigestWith(probe)
	var evidenceErr *Error
	if !errors.As(err, &evidenceErr) || evidenceErr.Code != CodeFingerprintInvalid {
		t.Fatalf("%s error=%v, want %s", label, err, CodeFingerprintInvalid)
	}
	t.Logf("%s exit=fail code=%s message=%s", label, evidenceErr.Code, evidenceErr.Message)
}

func fingerprintWithToolchainDigest(fingerprint Fingerprint, toolchainDigest string) Fingerprint {
	fingerprint.Identity = ""
	fingerprint.ToolchainDigest = toolchainDigest
	payload, _ := json.Marshal(fingerprint)
	fingerprint.Identity = digestBytes(payload)
	return fingerprint
}
