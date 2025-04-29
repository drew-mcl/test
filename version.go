// versioner.go
//
// Package versioner produces deterministic CalVer strings for GitLab pipelines.
//
// ─  Default-branch  → YYYYMMDD.<PipelineID>.0
// ─  Feature branch  → [<Prefix>-]LATEST.<PipelineID>[-<Suffix>]
// ─  Release branch  → [<Prefix>-]<BasePrefix>.<NextPatch>
//
//   - BasePrefix is   YYYYMMDD.<PipelineID>         (⚠ no “.0”)
//   - Branch name is  release/v<basePrefix>
//   - Patch numbers therefore start at 1 and auto-increment.
//
// All final tags (those cut from default or release branches) therefore obey
//
//	YYYYMMDD.<build>.<patch>
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
	Prefix        string // optional; prepended with "<prefix>-"
	FeatureSuffix string // optional; appended "-<suffix>" on *feature* builds only
}

type BuildContext struct {
	Branch     string    // CI_COMMIT_BRANCH
	PipelineID string    // CI_PIPELINE_IID   (used as <build>)
	Time       time.Time // normally time.Now()
	Config     Config
	LookupTags func() ([]string, error) // overridable for tests
}

// Version returns the canonical version string or an error.
func (c BuildContext) Version() (string, error) {
	switch classify(c.Config.DefaultBranch, c.Branch) {

	case typeDefault:
		v := fmt.Sprintf("%s.%s.0",
			c.Time.Format(dateLayout),
			c.PipelineID,
		)
		return addPrefix(v, c.Config.Prefix), nil

	case typeRelease:
		basePrefix, next, err := nextPatch(c.Branch, c.LookupTags)
		if err != nil {
			return "", err
		}
		v := fmt.Sprintf("%s.%d", basePrefix, next)
		return addPrefix(v, c.Config.Prefix), nil

	default: // feature / hot-fix
		base, err := latestFinal(c.LookupTags)
		if err != nil {
			return "", err
		}
		// first ever build → seed with YYYYMMDD.0.0
		if base == "" {
			base = fmt.Sprintf("%s.0.0", c.Time.Format(dateLayout))
		}

		v := fmt.Sprintf("%s.%s", base, c.PipelineID)
		if suf := strings.TrimPrefix(c.Config.FeatureSuffix, "-"); suf != "" {
			v += "-" + suf
		}
		return addPrefix(v, c.Config.Prefix), nil
	}
}

// ---------------- Internals ------------------------------------------------------------------------------------------

const dateLayout = "20060102"

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

var finalTagRE = regexp.MustCompile(`^\d{8}\.\d+\.\d+$`)

// latestFinal returns the lexicographically-last *valid* final tag.
// If there are tags but none match the expected pattern an error is returned.
// If there are no tags at all it returns "" and no error (caller decides).
func latestFinal(lookup func() ([]string, error)) (string, error) {
	ts, _ := lookup()
	latest := ""
	for _, t := range ts {
		if finalTagRE.MatchString(t) && t > latest {
			latest = t
		}
	}
	if latest != "" {
		return latest, nil
	}
	if len(ts) == 0 {
		return "", nil // first ever build
	}
	return "", fmt.Errorf("no tags match expected format YYYYMMDD.<build>.<patch>")
}

/* ---------- helpers for release branches ------------------------------------ */

// release/v20250428.123   →  basePrefix = 20250428.123
var relBranchRE = regexp.MustCompile(`^release/v(\d{8}\.\d+)$`)

func nextPatch(br string, lookup func() ([]string, error)) (basePrefix string, nextPatch int, err error) {
	m := relBranchRE.FindStringSubmatch(br)
	if len(m) != 2 {
		err = fmt.Errorf("invalid release branch: %s", br)
		return
	}
	basePrefix = m[1] // YYYYMMDD.<build>

	ts, _ := lookup()
	max := 0
	re := regexp.MustCompile(fmt.Sprintf(`^%s\.(\d+)$`, regexp.QuoteMeta(basePrefix)))
	for _, t := range ts {
		if mm := re.FindStringSubmatch(t); len(mm) == 2 {
			n, _ := strconv.Atoi(mm[1])
			if n > max {
				max = n
			}
		}
	}
	nextPatch = max + 1
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
