// Package versioner produces deterministic CalVer strings for GitLab pipelines.
//
// ─  Default-branch  → YYYYMMDD.<PipelineID>
// ─  Feature branch  → [<Prefix>-]YYYYMMDD.<PipelineID>[-<Suffix>]
// ─  Release branch  → [<Prefix>-]<BaseTag>.<NextPatch>
//
//   - BaseTag syntax: YYYYMMDD.<PipelineID>
//   - Release branch name:  release/v<baseTag>
//   - NextPatch starts at 1 and auto-increments.
package versioner

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------------- Public ---------------------------------------------------------------------------------------------

type Config struct {
	DefaultBranch string // "main", "master", "trunk" …
	Prefix        string // optional; prepended with '<prefix>-'
	FeatureSuffix string // optional; appended as '-<suffix>' on *feature* builds only
}

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

	case typeDefault:
		v := fmt.Sprintf("%s.%s", c.Time.Format("20060102"), c.PipelineID)
		return addPrefix(v, c.Config.Prefix), nil

	case typeRelease:
		base, next, err := nextPatch(c.Branch, c.LookupTags)
		if err != nil {
			return "", err
		}
		v := fmt.Sprintf("%s.%d", base, next)
		return addPrefix(v, c.Config.Prefix), nil

	default: // feature / hot-fix
		base := c.Time.Format("20060102")
		v := fmt.Sprintf("%s.%s", base, c.PipelineID)
		if suf := strings.TrimPrefix(c.Config.FeatureSuffix, "-"); suf != "" {
			v += "-" + suf
		}
		return addPrefix(v, c.Config.Prefix), nil
	}
}

// ---------------- Internals ------------------------------------------------------------------------------------------

type branchKind int

const (
	typeFeature branchKind = iota
	typeDefault
	typeRelease
)

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

func addPrefix(v, p string) string {
	if p == "" {
		return v
	}
	return strings.TrimSuffix(p, "-") + "-" + v
}

/* ---------- helpers for release branches ------------------------------------ */

var relBranchRE = regexp.MustCompile(`^release/v(\d{8}\.\d+)$`)

func nextPatch(br string, lookup func() ([]string, error)) (base string, patch int, err error) {
	m := relBranchRE.FindStringSubmatch(br)
	if len(m) != 2 {
		err = fmt.Errorf("invalid release branch: %s", br)
		return
	}
	base = m[1]

	// graceful degradation if lookup is nil
	var ts []string
	if lookup != nil {
		ts, _ = lookup()
	}

	max := 0
	re := regexp.MustCompile(fmt.Sprintf(`^%s\.(\d+)$`, regexp.QuoteMeta(base)))
	for _, t := range ts {
		if mm := re.FindStringSubmatch(t); len(mm) == 2 {
			n, _ := strconv.Atoi(mm[1])
			if n > max {
				max = n
			}
		}
	}
	patch = max + 1
	return
}

/* ---------- default Git helpers (may be stubbed in tests) -------------------- */

func GitTags() ([]string, error) {
	out, err := exec.Command("git", "tag").CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}
