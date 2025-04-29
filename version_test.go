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

func TestFeatureBranchWithSuffixAndPrefix(t *testing.T) {
	cfg := Config{
		DefaultBranch: "main",
		FeatureSuffix: "SNAPSHOT",
		Prefix:        "cli",
	}
	got, _ := ctx("feat/payments", cfg, nil).Version()
	want := "cli-20250428.321-SNAPSHOT"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestFeatureBranchNoSuffix(t *testing.T) {
	got, _ := ctx("feat/clean", Config{DefaultBranch: "main"}, nil).Version()
	want := "20250428.321"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestReleaseBranchFirstPatch(t *testing.T) {
	// no existing patch tags
	tags := []string{"20250428.100"}
	got, _ := ctx("release/v20250428.100", Config{DefaultBranch: "main"}, tags).Version()
	want := "20250428.100.1"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestReleaseBranchSubsequentPatch(t *testing.T) {
	tags := []string{"20250428.100", "20250428.100.1"}
	got, _ := ctx("release/v20250428.100", Config{DefaultBranch: "main"}, tags).Version()
	want := "20250428.100.2"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
