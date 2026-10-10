package checks

import (
	"context"
	"os/exec"
	"strings"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

var _ model.Check = SystemdServiceCheck{}

// SystemdServiceCheck reports systemd unit status with binary fallback.
type SystemdServiceCheck struct {
	Runner    runner.Runner
	LookPath  func(string) (string, error)
	Service   string
	ID        string
	UnitNames []string
	Binaries  []string
}

func PostgreSQLCheck(r runner.Runner, lp func(string) (string, error)) SystemdServiceCheck {
	return SystemdServiceCheck{Runner: r, LookPath: lp, Service: "PostgreSQL",
		ID: "services-postgresql", UnitNames: []string{"postgresql.service"},
		Binaries: []string{"psql", "pg_ctl", "postgres"}}
}

func RedisCheck(r runner.Runner, lp func(string) (string, error)) SystemdServiceCheck {
	return SystemdServiceCheck{Runner: r, LookPath: lp, Service: "Redis",
		ID: "services-redis", UnitNames: []string{"redis.service", "redis-server.service"},
		Binaries: []string{"redis-server"}}
}

func MySQLCheck(r runner.Runner, lp func(string) (string, error)) SystemdServiceCheck {
	return SystemdServiceCheck{Runner: r, LookPath: lp, Service: "MySQL",
		ID: "services-mysql", UnitNames: []string{"mysqld.service", "mariadb.service", "mysql.service"},
		Binaries: []string{"mysqld", "mariadbd"}}
}

func (c SystemdServiceCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{ID: c.ID, Name: c.Service,
		Description: "systemd service status", Category: model.CategoryServices, Optional: true}
}

// parseSystemdShow reports whether systemctl show output means loaded+active.
func parseSystemdShow(out string) (loaded, active bool) {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "LoadState=loaded") {
			loaded = true
		}
		if strings.Contains(line, "ActiveState=active") {
			active = true
		}
	}
	return loaded, active
}

func (c SystemdServiceCheck) findLoadedUnit(ctx context.Context) (string, bool) {
	if c.Runner == nil {
		return "", false
	}
	for _, unit := range c.UnitNames {
		out, err := c.Runner.Run(ctx, "systemctl", "show", "-p", "LoadState", "-p", "ActiveState", unit)
		if err != nil {
			continue
		}
		if loaded, active := parseSystemdShow(string(out)); loaded {
			return unit, active
		}
	}
	return "", false
}

func (c SystemdServiceCheck) anyBinary() bool {
	lp := c.LookPath
	if lp == nil {
		lp = exec.LookPath
	}
	for _, b := range c.Binaries {
		if _, err := lp(b); err == nil {
			return true
		}
	}
	return false
}

func (c SystemdServiceCheck) Run(ctx context.Context) []Result {
	base := model.Result{ID: c.ID, Category: model.CategoryServices, Title: c.Service}
	if unit, active := c.findLoadedUnit(ctx); unit != "" {
		if active {
			base.Severity = model.SeverityPass
			base.Message = "running"
			return model.Single(base)
		}
		base.Severity = model.SeverityWarning
		base.Message = "stopped"
		base.Remediation = &model.Remediation{
			Description:  "Start " + c.Service,
			Command:      "systemctl",
			Args:         []string{"start", unit},
			Safe:         true,
			RequiresSudo: true,
		}
		return model.Single(base)
	}
	if c.anyBinary() {
		base.Severity = model.SeverityWarning
		base.Message = "installed, no systemd unit"
		return model.Single(base)
	}
	base.Severity = model.SeverityInfo
	base.Message = "not installed"
	return model.Single(base)
}
