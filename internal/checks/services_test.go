package checks

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

func testSvcCheck(m *runner.MockRunner, lp func(string) (string, error)) SystemdServiceCheck {
	return SystemdServiceCheck{Runner: m, LookPath: lp, Service: "TestDB",
		ID: "services-testdb", UnitNames: []string{"testdb.service"}, Binaries: []string{"testdbd"}}
}

func okPath(string) (string, error) { return "/usr/bin/testdbd", nil }
func noPath(string) (string, error) { return "", errors.New("not found") }

func TestService_ActivePass(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"systemctl": func([]string) ([]byte, error) {
			return []byte("LoadState=loaded\nActiveState=active\n"), nil
		},
	}}
	rs := testSvcCheck(m, noPath).Run(context.Background())
	r := rs[0]
	if r.Severity != model.SeverityPass || r.Message != "running" || r.Title != "TestDB" {
		t.Fatalf("got %+v", r)
	}
	if r.Remediation != nil {
		t.Fatalf("running should have no remediation: %+v", r)
	}
}

func TestService_StoppedWarningRemediation(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"systemctl": func([]string) ([]byte, error) {
			return []byte("LoadState=loaded\nActiveState=inactive\n"), nil
		},
	}}
	rs := testSvcCheck(m, noPath).Run(context.Background())
	r := rs[0]
	if r.Severity != model.SeverityWarning || r.Message != "stopped" {
		t.Fatalf("got %+v", r)
	}
	rem := r.Remediation
	if rem == nil {
		t.Fatal("stopped needs remediation")
	}
	if rem.Command != "systemctl" || strings.Contains(rem.Command, " ") ||
		rem.Command == "sh" || rem.Command == "bash" {
		t.Fatalf("Command must be single systemctl binary: %q", rem.Command)
	}
	if len(rem.Args) != 2 || rem.Args[0] != "start" || rem.Args[1] != "testdb.service" {
		t.Fatalf("bad args: %q", rem.Args)
	}
	joined := strings.Join(append([]string{rem.Command}, rem.Args...), " ")
	for _, s := range []string{"|", ";", "&", "$", "`", "&&", "||"} {
		if strings.Contains(joined, s) {
			t.Fatalf("shell metachar in remediation: %q", joined)
		}
	}
	if !rem.Safe || !rem.RequiresSudo {
		t.Fatalf("remediation flags: %+v", rem)
	}
}

func TestService_NoUnitBinaryPresent(t *testing.T) {
	m := &runner.MockRunner{} // no handler -> exec.ErrNotFound
	rs := testSvcCheck(m, okPath).Run(context.Background())
	r := rs[0]
	if r.Severity != model.SeverityWarning || r.Message != "installed, no systemd unit" {
		t.Fatalf("got %+v", r)
	}
	if r.Remediation != nil {
		t.Fatalf("no-unit should have nil remediation: %+v", r)
	}
}

func TestService_NotInstalled(t *testing.T) {
	m := &runner.MockRunner{}
	rs := testSvcCheck(m, noPath).Run(context.Background())
	r := rs[0]
	if r.Severity != model.SeverityInfo || r.Message != "not installed" {
		t.Fatalf("got %+v", r)
	}
}

func TestService_MultiCandidateFallback(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"systemctl": func(args []string) ([]byte, error) {
			if args[len(args)-1] == "a.service" {
				return nil, exec.ErrNotFound
			}
			return []byte("LoadState=loaded\nActiveState=active\n"), nil
		},
	}}
	c := SystemdServiceCheck{Runner: m, LookPath: noPath, Service: "TestDB",
		ID: "services-testdb", UnitNames: []string{"a.service", "b.service"}, Binaries: []string{"testdbd"}}
	rs := c.Run(context.Background())
	if rs[0].Severity != model.SeverityPass || rs[0].Message != "running" {
		t.Fatalf("got %+v", rs[0])
	}
	if len(m.Calls) != 2 {
		t.Fatalf("expected 2 systemctl calls, got %d", len(m.Calls))
	}
}

func TestService_GarbledShowTreatedMissing(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"systemctl": func([]string) ([]byte, error) { return []byte("garbage $$$\nfoo=bar\n"), nil },
	}}
	rs := testSvcCheck(m, noPath).Run(context.Background())
	r := rs[0]
	if r.Severity != model.SeverityInfo || r.Message != "not installed" {
		t.Fatalf("garbled show should be missing, got %+v", r)
	}
}

func TestPostgreSQLConstructor(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"systemctl": func([]string) ([]byte, error) {
			return []byte("LoadState=loaded\nActiveState=active\n"), nil
		},
	}}
	c := PostgreSQLCheck(m, noPath)
	if len(c.UnitNames) != 1 || c.UnitNames[0] != "postgresql.service" {
		t.Fatalf("bad PG units: %q", c.UnitNames)
	}
	for _, b := range []string{"psql", "pg_ctl", "postgres"} {
		found := false
		for _, x := range c.Binaries {
			if x == b {
				found = true
			}
		}
		if !found {
			t.Fatalf("PG missing bin %s: %q", b, c.Binaries)
		}
	}
	md := c.Metadata()
	if md.ID != "services-postgresql" || md.Category != model.CategoryServices || !md.Optional || md.Name != "PostgreSQL" {
		t.Fatalf("bad PG metadata: %+v", md)
	}
	if rs := c.Run(context.Background()); rs[0].Severity != model.SeverityPass {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestRedisConstructor(t *testing.T) {
	c := RedisCheck(&runner.MockRunner{}, noPath)
	if len(c.UnitNames) != 2 || c.UnitNames[0] != "redis.service" || c.UnitNames[1] != "redis-server.service" {
		t.Fatalf("bad redis units: %q", c.UnitNames)
	}
	if len(c.Binaries) != 1 || c.Binaries[0] != "redis-server" {
		t.Fatalf("bad redis bins: %q", c.Binaries)
	}
	md := c.Metadata()
	if md.ID != "services-redis" || md.Category != model.CategoryServices || !md.Optional {
		t.Fatalf("bad redis metadata: %+v", md)
	}
}

func TestMySQLConstructor(t *testing.T) {
	c := MySQLCheck(&runner.MockRunner{}, noPath)
	want := []string{"mysqld.service", "mariadb.service", "mysql.service"}
	if len(c.UnitNames) != len(want) {
		t.Fatalf("bad mysql units: %q", c.UnitNames)
	}
	for i := range want {
		if c.UnitNames[i] != want[i] {
			t.Fatalf("bad mysql units: %q", c.UnitNames)
		}
	}
	if len(c.Binaries) != 2 || c.Binaries[0] != "mysqld" || c.Binaries[1] != "mariadbd" {
		t.Fatalf("bad mysql bins: %q", c.Binaries)
	}
	md := c.Metadata()
	if md.ID != "services-mysql" || md.Name != "MySQL" {
		t.Fatalf("bad mysql metadata: %+v", md)
	}
	ids := map[string]bool{
		PostgreSQLCheck(nil, nil).Metadata().ID: true,
		RedisCheck(nil, nil).Metadata().ID:      true,
		MySQLCheck(nil, nil).Metadata().ID:      true,
	}
	if len(ids) != 3 {
		t.Fatal("service IDs not unique")
	}
}
