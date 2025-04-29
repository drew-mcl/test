// versioner.go
// Package versioner produces deterministic CalVer strings for GitLab pipelines.
//
// ─ Default branch  → YYYYMMDD.<PipelineID>
// ─ Feature branch  → [<Prefix>-]YYYYMMDD.<PipelineID>
// ─ Release branch  → [<Prefix>-]<BaseTag>.<NextPatch>
//
// BaseTag syntax must be YYYYMMDD.<PipelineID>; release branch name must be
// release/v<BaseTag>. NextPatch starts at 1 and auto-increments.
package versioner

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Config holds versioning settings.
type Config struct {
	DefaultBranch string // "main", "master", …
	Prefix        string // optional; prepended as '<prefix>-'
}

// BuildContext carries CI context for version generation.
type BuildContext struct {
	Branch     string    // CI_COMMIT_BRANCH
	PipelineID string    // CI_PIPELINE_IID
	Time       time.Time // generally time.Now()
	Config     Config
	LookupTags func() ([]string, error) // overridable for tests
}

// Version returns the canonical version string or an error.
func (c BuildContext) Version() (string, error) {
	switch classify(c.Config.DefaultBranch, c.Branch) {
	case typeDefault, typeFeature:
		// Default & Feature share DATE.BUILD format
		base := c.Time.Format("20060102") + "." + c.PipelineID
		return addPrefix(base, c.Config.Prefix), nil

	case typeRelease:
		baseTag, next, err := nextPatch(c.Branch, c.LookupTags)
		if err != nil {
			return "", err
		}
		v := fmt.Sprintf("%s.%d", baseTag, next)
		return addPrefix(v, c.Config.Prefix), nil

	default:
		// fallback
		base := c.Time.Format("20060102") + "." + c.PipelineID
		return addPrefix(base, c.Config.Prefix), nil
	}
}

// branchKind defines types of branches.
type branchKind int

const (
	typeFeature branchKind = iota
	typeDefault
	typeRelease
)

// classify categorizes the branch.
func classify(def, br string) branchKind {
	switch {
	case br == def:
		return typeDefault
	case strings.HasPrefix(br, "release/"):
		return typeRelease
	default:
		return typeFeature
	}
}

// addPrefix applies the optional prefix.
func addPrefix(v, p string) string {
	if p == "" {
		return v
	}
	return strings.TrimSuffix(p, "-") + "-" + v
}

// relBranchRE matches release branches: release/vYYYYMMDD.<PipelineID>
var relBranchRE = regexp.MustCompile(`^release/v(\d{8}\.\d+)$`)

// nextPatch finds the next patch number for a given release branch.
func nextPatch(br string, lookup func() ([]string, error)) (base string, patch int, err error) {
	m := relBranchRE.FindStringSubmatch(br)
	if len(m) != 2 {
		err = fmt.Errorf("invalid release branch: %s", br)
		return
	}
	base = m[1]

	ts, _ := lookup()
	max := 0
	// match tags like BaseTag.N
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `\.(\d+)$`)
	for _, t := range ts {
		if mm := re.FindStringSubmatch(t); len(mm) == 2 {
			if n, errA := strconv.Atoi(mm[1]); errA == nil && n > max {
				max = n
			}
		}
	}
	patch = max + 1
	return
}

// GitTags returns git tags (stubbed in tests).
func GitTags() ([]string, error) {
	out, err := exec.Command("git", "tag").CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}
