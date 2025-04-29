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
	want := "20250428.321.0"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestFeatureWithSuffixAndPrefix(t *testing.T) {
	tags := []string{"20250427.17.0"}
	cfg := Config{DefaultBranch: "main", FeatureSuffix: "SNAPSHOT", Prefix: "cli"}
	got, _ := ctx("feat/pay", cfg, tags).Version()
	want := "cli-20250427.17.0.321-SNAPSHOT"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestReleasePatch(t *testing.T) {
	tags := []string{"20250428.100.0", "20250428.100.1"}
	got, _ := ctx("release/v20250428.100", Config{DefaultBranch: "main"}, tags).Version()
	want := "20250428.100.2"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestFeatureNoValidTags(t *testing.T) {
	tags := []string{"not-a-calver"}
	_, err := ctx("feat/broken", Config{DefaultBranch: "main"}, tags).Version()
	if err == nil {
		t.Fatalf("expected error when latest tag is invalid")
	}
}
