package cli

import (
	"testing"
	"time"
)

func TestReuseGovernanceRouteAndSkillVerify(t *testing.T) {
	repo := t.TempDir()
	writeGoControlFixture(t, repo)

	res, err := runGovernance([]string{"reuse", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !res.OK || res.Command != "governance reuse" {
		t.Fatalf("reuse governance route failed: res=%#v err=%v", res, err)
	}

	res, err = runSkill([]string{"verify", "--all", "--profile", "Smoke", "--repo-root", repo, "--json"}, time.Now())
	if err != nil || !res.OK || res.Command != "skill verify" {
		t.Fatalf("skill verify reuse integration failed: res=%#v err=%v", res, err)
	}
}
