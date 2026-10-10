package checks

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

var _ model.Check = FilesystemCheck{}

var defaultMounts = []string{"/", "/home", "/var"}

// FilesystemCheck reports disk usage per mount via `df -kP`.
type FilesystemCheck struct {
	Runner runner.Runner
	Cfg    config.Config
	Mounts []string
}

func (c FilesystemCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "storage-fs", Name: "Filesystem",
		Description: "Filesystem usage against warn/crit thresholds via df",
		Category:    model.CategoryStorage,
	}
}

func sanitizeMount(m string) string {
	if m == "/" {
		return "root"
	}
	s := strings.Trim(m, "/")
	s = strings.ReplaceAll(s, "/", "-")
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

// parseDF parses POSIX `df -kP` output: second line, 5th column "61%" -> 61.
func parseDF(out string) (int, error) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("df: want header+row, got %d lines", len(lines))
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return 0, fmt.Errorf("df: want >=5 columns, got %d", len(fields))
	}
	pct := strings.TrimSuffix(fields[4], "%")
	n, err := strconv.Atoi(pct)
	if err != nil {
		return 0, fmt.Errorf("df: bad Use%% %q", fields[4])
	}
	if n < 0 || n > 100 {
		return 0, fmt.Errorf("df: Use%% out of range: %d", n)
	}
	return n, nil
}

func (c FilesystemCheck) Run(ctx context.Context) []Result {
	mounts := c.Mounts
	if len(mounts) == 0 {
		mounts = defaultMounts
	}
	if c.Runner == nil {
		return nil
	}
	warn, crit := c.Cfg.StorageWarn, c.Cfg.StorageCrit
	if warn <= 0 || crit <= 0 {
		d := config.Default()
		warn, crit = d.StorageWarn, d.StorageCrit
	}
	var out []Result
	for _, m := range mounts {
		raw, err := c.Runner.Run(ctx, "df", "-kP", m)
		if err != nil {
			continue
		}
		usePct, err := parseDF(string(raw))
		if err != nil {
			continue
		}
		r := model.Result{
			ID:       "storage-fs-" + sanitizeMount(m),
			Category: model.CategoryStorage,
			Title:    m,
			Message:  fmt.Sprintf("%d%% used", usePct),
			Details:  map[string]any{"mount": m, "use_pct": usePct},
		}
		switch {
		case usePct >= crit:
			r.Severity = model.SeverityCritical
		case usePct >= warn:
			r.Severity = model.SeverityWarning
		default:
			r.Severity = model.SeverityPass
		}
		out = append(out, r)
	}
	return out
}
