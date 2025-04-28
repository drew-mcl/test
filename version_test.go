package versioner

import (
	"testing"
	"time"
)

func fixedCtx(branch string, tags []string) BuildContext {
	return BuildContext{
		Branch:              branch,
		PipelineID:          "123",
		Time:                time.Date(2025, 4, 28, 10, 0, 0, 0, time.UTC),
		LookupTagsFn:        func() ([]string, error) { return tags, nil },
		LookupMergesTodayFn: func(_ time.Time) (int, error) { return 5, nil },
	}
}

func TestFeatureSnapshot(t *testing.T) {
	ctx := fixedCtx("feat/awesome", []string{"2025.04.27.3"})
	got, _ := ctx.Version()
	want := "2025.04.27.3.123-SNAPSHOT"
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}

func TestMasterRCIncrement(t *testing.T) {
	ctx := fixedCtx("master",
		[]string{"2025.04.28.5-RC1"})
	got, _ := ctx.Version()
	want := "2025.04.28.5-RC2"
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}

func TestReleasePatch(t *testing.T) {
	ctx := fixedCtx("release/v2025.04.28.5.1", nil)
	got, _ := ctx.Version()
	want := "2025.04.28.5.2"
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}
