// Package versioner produces deterministic CalVer strings for GitLab pipelines.
//
// ─  Default-branch  → YYYY.MM.DD.<PipelineID>
// ─  Feature branch  → [<Prefix>-]LATEST.<PipelineID>[-<Suffix>]
// ─  Release branch  → [<Prefix>-]<BaseTag>.<NextPatch>
//
// BaseTag syntax must be YYYY.MM.DD.<PipelineID>; release branch name must be
// release/v<baseTag>.  NextPatch starts at 1 and auto-increments.
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
	DefaultBranch string //   "main", "master", "trunk" …
	Prefix        string //   optional; prepended with '<prefix>-'
	FeatureSuffix string //   optional; appended as '-<suffix>' on *feature* builds only
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
		v := fmt.Sprintf("%s.%s", c.Time.Format("2006.01.02"), c.PipelineID)
		return addPrefix(v, c.Config.Prefix), nil

	case typeRelease:
		base, next, err := nextPatch(c.Branch, c.LookupTags)
		if err != nil {
			return "", err
		}
		v := fmt.Sprintf("%s.%d", base, next)
		return addPrefix(v, c.Config.Prefix), nil

	default: // feature / hot-fix
		base := latestFinal(c.LookupTags)
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

/* ---------- helpers for feature branches ------------------------------------ */

var finalTagRE = regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}\.\d+$`)

func latestFinal(lookup func() ([]string, error)) string {
	ts, _ := lookup()
	last := ""
	for _, t := range ts {
		if finalTagRE.MatchString(t) && t > last {
			last = t
		}
	}
	if last == "" {
		last = time.Now().Format("2006.01.02") + ".0"
	}
	return last
}

/* ---------- helpers for release branches ------------------------------------ */

var relBranchRE = regexp.MustCompile(`^release/v(\d{4}\.\d{2}\.\d{2}\.\d+)$`)

func nextPatch(br string, lookup func() ([]string, error)) (base string, patch int, err error) {
	m := relBranchRE.FindStringSubmatch(br)
	if len(m) != 2 {
		err = fmt.Errorf("invalid release branch: %s", br)
		return
	}
	base = m[1]

	ts, _ := lookup()
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
