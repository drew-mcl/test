package versioner

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BuildType enumerates the flow we’re in.
type BuildType int

const (
	feature BuildType = iota
	master
	release
)

// BuildContext is injected by the caller — easy to stub in tests.
type BuildContext struct {
	Branch              string    // CI_COMMIT_BRANCH
	PipelineID          string    // CI_PIPELINE_IID
	Time                time.Time // usually time.Now()
	LookupTagsFn        func() ([]string, error)
	LookupMergesTodayFn func(t time.Time) (int, error)
}

// Version returns the deterministic version string.
func (c BuildContext) Version() (string, error) {
	bt := detectType(c.Branch)
	tags, err := c.LookupTagsFn()
	if err != nil {
		return "", err
	}

	switch bt {
	case feature:
		latest := latestReleaseTag(tags)
		return fmt.Sprintf("%s.%s-SNAPSHOT", latest, c.PipelineID), nil

	case master:
		date := c.Time.Format("2006.01.02") // YYYY.MM.DD
		seq, err := c.LookupMergesTodayFn(c.Time)
		if err != nil {
			return "", err
		}
		rc := nextRC(tags, date, seq)
		return fmt.Sprintf("%s.%d-RC%d", date, seq, rc), nil

	case release:
		base, patch, err := parseReleaseBranch(c.Branch)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%d", base, patch+1), nil

	default:
		return "", fmt.Errorf("unhandled build type")
	}
}

/* ---------- helpers ---------- */

func detectType(branch string) BuildType {
	switch {
	case branch == "master":
		return master
	case strings.HasPrefix(branch, "release/"):
		return release
	default:
		return feature
	}
}

// latestReleaseTag finds the lexicographically-last CalVer tag.
func latestReleaseTag(tags []string) string {
	var last string
	for _, t := range tags {
		if calverRegexp.MatchString(t) && t > last {
			last = t
		}
	}
	if last == "" {
		last = time.Now().Format("2006.01.02") + ".0"
	}
	return last
}

// nextRC ⇒ max(existing RC for the same base)+1
func nextRC(tags []string, date string, seq int) int {
	base := fmt.Sprintf("%s.%d", date, seq)
	re := regexp.MustCompile(fmt.Sprintf(`^%s-RC(\d+)$`, regexp.QuoteMeta(base)))
	max := 0
	for _, t := range tags {
		m := re.FindStringSubmatch(t)
		if len(m) == 2 {
			n, _ := strconv.Atoi(m[1])
			if n > max {
				max = n
			}
		}
	}
	return max + 1
}

var calverRegexp = regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}\.\d+(\.\d+)?$`)

// parseReleaseBranch parses release/vYYYY.MM.DD.seq -> base, patch
func parseReleaseBranch(br string) (base string, patch int, err error) {
	re := regexp.MustCompile(`^release/v(\d{4}\.\d{2}\.\d{2}\.\d+)\.(\d+)$`)
	m := re.FindStringSubmatch(br)
	if len(m) != 3 {
		return "", 0, fmt.Errorf("invalid release branch: %s", br)
	}
	base = m[1]
	patch, _ = strconv.Atoi(m[2])
	return
}

/* ---------- default git helpers (can be replaced in tests) ---------- */

// GitTags returns all tags - lightweight + annotated.
func GitTags() ([]string, error) {
	out, err := exec.Command("git", "tag").CombinedOutput()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines, nil
}

// MergesToday counts merges into master since midnight local time.
func MergesToday(t time.Time) (int, error) {
	since := t.Format("2006-01-02") + " 00:00"
	out, err := exec.Command("git", "rev-list", "--count", "--merges",
		"--since", since, "master").CombinedOutput()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}
