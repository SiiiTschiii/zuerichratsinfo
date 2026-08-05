package votes_test

import (
	"os/exec"
	"strings"
	"testing"
)

// forbidden lists packages that must never appear in pkg/votes' transitive
// dependency set. The whole point of the package is that sources depend on it
// and it depends on none of them; an accidental import here would reintroduce
// the coupling the source-neutral model exists to remove — and would make an
// import cycle the first symptom rather than a design review.
var forbidden = []string{
	"github.com/siiitschiii/zuerichratsinfo/pkg/zurichapi",
	"github.com/siiitschiii/zuerichratsinfo/pkg/openparldata",
	"github.com/siiitschiii/zuerichratsinfo/pkg/voteposting",
	"github.com/siiitschiii/zuerichratsinfo/pkg/imagegen",
}

func TestPackageHasNoSourceDependencies(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "github.com/siiitschiii/zuerichratsinfo/pkg/votes").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}

	deps := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		deps[line] = true
	}

	for _, pkg := range forbidden {
		if deps[pkg] {
			t.Errorf("pkg/votes must not depend on %s", pkg)
		}
	}

	// Guard against the list above going stale: nothing from this module
	// belongs in the dependency set at all.
	for dep := range deps {
		if strings.HasPrefix(dep, "github.com/siiitschiii/zuerichratsinfo/") &&
			dep != "github.com/siiitschiii/zuerichratsinfo/pkg/votes" {
			t.Errorf("pkg/votes must not depend on module package %s", dep)
		}
	}
}
