package checks

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

func TestToolVersion(t *testing.T) {
	cases := map[string]string{
		"git version 2.51.0":              "2.51.0",
		"go version go1.25.1 linux/amd64": "1.25.1",
		"v24.9.0":                         "24.9.0",
		"11.6.0":                          "11.6.0",
		"Python 3.13.7":                   "3.13.7",
		"PHP 8.4.10 (cli)":                "8.4.10",
		"Docker version 28.5.0, build x":  "28.5.0",
		"":                                "",
		"no-version-here":                 "",
	}
	for in, want := range cases {
		if got := toolVersion(in); got != want {
			t.Errorf("toolVersion(%q)=%q want %q", in, got, want)
		}
	}
}

func TestGitCheck_OK(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"git": func([]string) ([]byte, error) { return []byte("git version 2.51.0\n"), nil },
	}}
	rs := GitCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass || rs[0].Message != "2.51.0" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestGitCheck_Missing(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"git": func([]string) ([]byte, error) { return nil, errors.New("not found") },
	}}
	rs := GitCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityInfo || rs[0].Message != "Git not installed" {
		t.Fatalf("got %+v", rs[0])
	}
	md := (GitCheck{}).Metadata()
	if !md.Optional {
		t.Fatal("git should be optional")
	}
}

func TestGoCheck_OK(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"go": func([]string) ([]byte, error) { return []byte("go version go1.25.1 linux/amd64\n"), nil },
	}}
	rs := GoCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass || rs[0].Message != "1.25.1" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestGoCheck_Missing(t *testing.T) {
	m := &runner.MockRunner{}
	rs := GoCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityInfo {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestNodeCheck_OK(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"node": func([]string) ([]byte, error) { return []byte("v24.9.0\n"), nil },
		"npm":  func([]string) ([]byte, error) { return []byte("11.6.0\n"), nil },
	}}
	rs := NodeCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass {
		t.Fatalf("got %+v", rs[0])
	}
	if rs[0].Message != "24.9.0 (npm 11.6.0)" {
		t.Fatalf("got %q", rs[0].Message)
	}
}

func TestNodeCheck_NpmMissing(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"node": func([]string) ([]byte, error) { return []byte("v24.9.0\n"), nil },
	}}
	rs := NodeCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass {
		t.Fatalf("got %+v", rs[0])
	}
	if rs[0].Message != "24.9.0 (npm N/A)" {
		t.Fatalf("got %q", rs[0].Message)
	}
}

func TestNodeCheck_Missing(t *testing.T) {
	m := &runner.MockRunner{}
	rs := NodeCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityInfo {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestPythonCheck_Python3(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"python3": func([]string) ([]byte, error) { return []byte("Python 3.13.7\n"), nil },
	}}
	rs := PythonCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass || rs[0].Message != "3.13.7" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestPythonCheck_Fallback(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"python3": func([]string) ([]byte, error) { return nil, errors.New("no python3") },
		"python":  func([]string) ([]byte, error) { return []byte("Python 3.12.0\n"), nil },
	}}
	rs := PythonCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass || rs[0].Message != "3.12.0" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestPythonCheck_Missing(t *testing.T) {
	m := &runner.MockRunner{}
	rs := PythonCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityInfo {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestPHPCheck_OK(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"php": func([]string) ([]byte, error) { return []byte("PHP 8.4.10 (cli) built\n"), nil },
	}}
	rs := PHPCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass || rs[0].Message != "8.4.10" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestPHPCheck_Missing(t *testing.T) {
	m := &runner.MockRunner{}
	rs := PHPCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityInfo || rs[0].Message != "PHP not installed" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestDockerCheck_OK(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"docker": func([]string) ([]byte, error) { return []byte("Docker version 28.5.0, build xyz\n"), nil },
	}}
	rs := DockerCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass || rs[0].Message != "28.5.0" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestDockerCheck_Missing(t *testing.T) {
	m := &runner.MockRunner{}
	rs := DockerCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityInfo {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestDockerDaemonCheck_Running(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"docker": func(args []string) ([]byte, error) {
			if len(args) > 0 && args[0] == "info" {
				return []byte("Server Version: 28.5.0\n"), nil
			}
			return []byte("Docker version 28.5.0\n"), nil
		},
	}}
	rs := DockerDaemonCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass || rs[0].Message != "running" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestDockerDaemonCheck_Stopped(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"docker": func(args []string) ([]byte, error) {
			if len(args) > 0 && args[0] == "info" {
				return nil, errors.New("Cannot connect to the Docker daemon")
			}
			return []byte("Docker version 28.5.0\n"), nil
		},
	}}
	rs := DockerDaemonCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityWarning || rs[0].Message != "stopped" {
		t.Fatalf("got %+v", rs[0])
	}
	r := rs[0].Remediation
	if r == nil {
		t.Fatal("missing remediation")
	}
	if r.Command != "systemctl" {
		t.Fatalf("command got %q", r.Command)
	}
	if strings.Contains(r.Command, " ") || strings.Contains(r.Command, "sh") {
		t.Fatalf("must not be shell string: %q", r.Command)
	}
	if len(r.Args) != 2 || r.Args[0] != "start" || r.Args[1] != "docker" {
		t.Fatalf("args got %v", r.Args)
	}
	if !r.Safe || !r.RequiresSudo {
		t.Fatalf("flags got %+v", r)
	}
}

func TestDockerDaemonCheck_Skipped(t *testing.T) {
	m := &runner.MockRunner{}
	rs := DockerDaemonCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityInfo || rs[0].Message != "skipped (docker not installed)" {
		t.Fatalf("got %+v", rs[0])
	}
	md := (DockerDaemonCheck{}).Metadata()
	if md.Optional {
		t.Fatal("daemon must not be optional")
	}
}

func TestDevelopment_Metadata(t *testing.T) {
	checks := []model.Check{
		GitCheck{}, GoCheck{}, NodeCheck{}, PythonCheck{}, PHPCheck{}, DockerCheck{}, DockerDaemonCheck{},
	}
	for _, c := range checks {
		md := c.Metadata()
		if md.Category != model.CategoryDevelopment {
			t.Errorf("%s category=%q", md.ID, md.Category)
		}
		wantOptional := md.ID != "development-docker-daemon"
		if md.Optional != wantOptional {
			t.Errorf("%s optional=%v want %v", md.ID, md.Optional, wantOptional)
		}
	}
}
