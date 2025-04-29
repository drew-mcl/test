// versioner_test.go
package versioner

import (
	"testing"
	"time"
)

var now = time.Date(2025, 4, 28, 15, 0, 0, 0, time.UTC)

func ctx(branch string, cfg Config, tags []string) BuildContext {
	return BuildContext{
		Branch:     branch,
		PipelineID: "321",
		Time:       now,
		Config:     cfg,
		LookupTags: func() ([]string, error) { return tags, nil },
	}
}

func TestDefaultBranch(t *testing.T) {
	got, _ := ctx("main", Config{DefaultBranch: "main"}, nil).Version()
	want := "20250428.321"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestFeatureBranch(t *testing.T) {
	cfg := Config{DefaultBranch: "main", Prefix: "cli"}
	got, _ := ctx("feature/foo", cfg, []string{}).Version()
	want := "cli-20250428.321"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestReleaseFirstPatch(t *testing.T) {
	cfg := Config{DefaultBranch: "main"}
	got, _ := ctx("release/v20250428.100", cfg, []string{}).Version()
	want := "20250428.100.1"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestReleaseNextPatch(t *testing.T) {
	tags := []string{"20250428.100", "20250428.100.1", "20250428.100.2"}
	got, _ := ctx("release/v20250428.100", Config{DefaultBranch: "main"}, tags).Version()
	want := "20250428.100.3"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
