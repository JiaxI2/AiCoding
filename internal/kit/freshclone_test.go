package kit

import (
	"reflect"
	"testing"
)

func TestFreshCloneChecksAreLeafCommands(t *testing.T) {
	bin := `C:\repo\bin\aicoding.exe`
	for _, tc := range []struct {
		profile string
		want    [][]string
	}{
		{"Smoke", [][]string{{bin, "version"}}},
		{"Full", [][]string{{"go", "test", "./..."}}},
		{"Release", [][]string{{bin, "release", "verify", "--json"}}},
	} {
		got, err := freshCloneChecks(bin, tc.profile)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("freshCloneChecks(%q) = %#v, %v; want %#v", tc.profile, got, err, tc.want)
		}
	}
	if _, err := freshCloneChecks(bin, "Nightly"); err == nil {
		t.Fatal("unsupported fresh-clone profile must fail")
	}
}
