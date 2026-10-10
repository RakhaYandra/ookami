package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

var (
	_ model.Check = GitCheck{}
	_ model.Check = GoCheck{}
	_ model.Check = NodeCheck{}
	_ model.Check = PythonCheck{}
	_ model.Check = PHPCheck{}
	_ model.Check = DockerCheck{}
	_ model.Check = DockerDaemonCheck{}
)

// toolVersion extracts the first version-like token from command output.
// "git version 2.51.0" -> "2.51.0", "go version go1.25.1 linux/amd64" -> "1.25.1",
// "v24.9.0" -> "24.9.0".
func toolVersion(out string) string {
	for _, f := range strings.Fields(out) {
		t := strings.Trim(f, ",()[]\"'")
		s := t
		if strings.HasPrefix(s, "go") && len(s) > 2 && s[2] >= '0' && s[2] <= '9' {
			s = s[2:]
		} else if len(s) > 1 && (s[0] == 'v' || s[0] == 'V') && s[1] >= '0' && s[1] <= '9' {
			s = s[1:]
		}
		s = strings.TrimRight(s, ",;:")
		if s == "" || s[0] < '0' || s[0] > '9' {
			continue
		}
		return s
	}
	return ""
}

func versionOrRaw(out string) string {
	if v := toolVersion(out); v != "" {
		return v
	}
	return strings.TrimSpace(out)
}

// GitCheck reports `git --version`.
type GitCheck struct {
	Runner runner.Runner
}

func (c GitCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "development-git", Name: "Git",
		Description: "Git version via git --version",
		Category:    model.CategoryDevelopment,
		Optional:    true,
	}
}

func (c GitCheck) Run(ctx context.Context) []Result {
	base := model.Result{ID: "development-git", Category: model.CategoryDevelopment, Title: "Git"}
	if c.Runner == nil {
		base.Severity = model.SeverityInfo
		base.Message = "Git not installed"
		return model.Single(base)
	}
	out, err := c.Runner.Run(ctx, "git", "--version")
	if err != nil {
		base.Severity = model.SeverityInfo
		base.Message = "Git not installed"
		return model.Single(base)
	}
	v := versionOrRaw(string(out))
	base.Severity = model.SeverityPass
	base.Message = v
	base.Details = map[string]any{"version": v}
	return model.Single(base)
}

// GoCheck reports `go version`.
type GoCheck struct {
	Runner runner.Runner
}

func (c GoCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "development-go", Name: "Go",
		Description: "Go toolchain version via go version",
		Category:    model.CategoryDevelopment,
		Optional:    true,
	}
}

func (c GoCheck) Run(ctx context.Context) []Result {
	base := model.Result{ID: "development-go", Category: model.CategoryDevelopment, Title: "Go"}
	if c.Runner == nil {
		base.Severity = model.SeverityInfo
		base.Message = "Go not installed"
		return model.Single(base)
	}
	out, err := c.Runner.Run(ctx, "go", "version")
	if err != nil {
		base.Severity = model.SeverityInfo
		base.Message = "Go not installed"
		return model.Single(base)
	}
	v := versionOrRaw(string(out))
	base.Severity = model.SeverityPass
	base.Message = v
	base.Details = map[string]any{"version": v}
	return model.Single(base)
}

// NodeCheck reports `node --version` and `npm --version`.
type NodeCheck struct {
	Runner runner.Runner
}

func (c NodeCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "development-node", Name: "Node",
		Description: "Node.js and npm versions via node/npm --version",
		Category:    model.CategoryDevelopment,
		Optional:    true,
	}
}

func (c NodeCheck) Run(ctx context.Context) []Result {
	base := model.Result{ID: "development-node", Category: model.CategoryDevelopment, Title: "Node"}
	if c.Runner == nil {
		base.Severity = model.SeverityInfo
		base.Message = "Node not installed"
		return model.Single(base)
	}
	out, err := c.Runner.Run(ctx, "node", "--version")
	if err != nil {
		base.Severity = model.SeverityInfo
		base.Message = "Node not installed"
		return model.Single(base)
	}
	nodeV := versionOrRaw(string(out))
	npmV := "N/A"
	if npmOut, npmErr := c.Runner.Run(ctx, "npm", "--version"); npmErr == nil {
		npmV = versionOrRaw(string(npmOut))
	}
	base.Severity = model.SeverityPass
	base.Message = fmt.Sprintf("%s (npm %s)", nodeV, npmV)
	base.Details = map[string]any{"node": nodeV, "npm": npmV}
	return model.Single(base)
}

// PythonCheck tries `python3 --version` then `python --version`.
type PythonCheck struct {
	Runner runner.Runner
}

func (c PythonCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "development-python", Name: "Python",
		Description: "Python version via python3/python --version",
		Category:    model.CategoryDevelopment,
		Optional:    true,
	}
}

func (c PythonCheck) Run(ctx context.Context) []Result {
	base := model.Result{ID: "development-python", Category: model.CategoryDevelopment, Title: "Python"}
	if c.Runner == nil {
		base.Severity = model.SeverityInfo
		base.Message = "Python not installed"
		return model.Single(base)
	}
	if out, err := c.Runner.Run(ctx, "python3", "--version"); err == nil {
		v := versionOrRaw(string(out))
		base.Severity = model.SeverityPass
		base.Message = v
		base.Details = map[string]any{"version": v, "binary": "python3"}
		return model.Single(base)
	}
	if out, err := c.Runner.Run(ctx, "python", "--version"); err == nil {
		v := versionOrRaw(string(out))
		base.Severity = model.SeverityPass
		base.Message = v
		base.Details = map[string]any{"version": v, "binary": "python"}
		return model.Single(base)
	}
	base.Severity = model.SeverityInfo
	base.Message = "Python not installed"
	return model.Single(base)
}

// PHPCheck reports `php --version`.
type PHPCheck struct {
	Runner runner.Runner
}

func (c PHPCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "development-php", Name: "PHP",
		Description: "PHP version via php --version",
		Category:    model.CategoryDevelopment,
		Optional:    true,
	}
}

func (c PHPCheck) Run(ctx context.Context) []Result {
	base := model.Result{ID: "development-php", Category: model.CategoryDevelopment, Title: "PHP"}
	if c.Runner == nil {
		base.Severity = model.SeverityInfo
		base.Message = "PHP not installed"
		return model.Single(base)
	}
	out, err := c.Runner.Run(ctx, "php", "--version")
	if err != nil {
		base.Severity = model.SeverityInfo
		base.Message = "PHP not installed"
		return model.Single(base)
	}
	v := versionOrRaw(string(out))
	base.Severity = model.SeverityPass
	base.Message = v
	base.Details = map[string]any{"version": v}
	return model.Single(base)
}

// DockerCheck reports `docker --version`.
type DockerCheck struct {
	Runner runner.Runner
}

func (c DockerCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "development-docker", Name: "Docker",
		Description: "Docker version via docker --version",
		Category:    model.CategoryDevelopment,
		Optional:    true,
	}
}

func (c DockerCheck) Run(ctx context.Context) []Result {
	base := model.Result{ID: "development-docker", Category: model.CategoryDevelopment, Title: "Docker"}
	if c.Runner == nil {
		base.Severity = model.SeverityInfo
		base.Message = "Docker not installed"
		return model.Single(base)
	}
	out, err := c.Runner.Run(ctx, "docker", "--version")
	if err != nil {
		base.Severity = model.SeverityInfo
		base.Message = "Docker not installed"
		return model.Single(base)
	}
	v := versionOrRaw(string(out))
	base.Severity = model.SeverityPass
	base.Message = v
	base.Details = map[string]any{"version": v}
	return model.Single(base)
}

// DockerDaemonCheck reports `docker info`; skipped when docker is absent.
type DockerDaemonCheck struct {
	Runner runner.Runner
}

func (c DockerDaemonCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "development-docker-daemon", Name: "Docker Daemon",
		Description: "Docker daemon status via docker info",
		Category:    model.CategoryDevelopment,
		Optional:    false,
	}
}

func (c DockerDaemonCheck) Run(ctx context.Context) []Result {
	base := model.Result{ID: "development-docker-daemon", Category: model.CategoryDevelopment, Title: "Docker Daemon"}
	if c.Runner == nil {
		base.Severity = model.SeverityInfo
		base.Message = "skipped (docker not installed)"
		return model.Single(base)
	}
	if _, err := c.Runner.Run(ctx, "docker", "--version"); err != nil {
		base.Severity = model.SeverityInfo
		base.Message = "skipped (docker not installed)"
		return model.Single(base)
	}
	if _, err := c.Runner.Run(ctx, "docker", "info"); err != nil {
		base.Severity = model.SeverityWarning
		base.Message = "stopped"
		base.Remediation = &model.Remediation{
			Description:  "Start Docker daemon",
			Command:      "systemctl",
			Args:         []string{"start", "docker"},
			Safe:         true,
			RequiresSudo: true,
		}
		return model.Single(base)
	}
	base.Severity = model.SeverityPass
	base.Message = "running"
	return model.Single(base)
}
