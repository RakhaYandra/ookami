package fixer

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

func res(sev model.Severity, cmd string, args ...string) model.Result {
	var rem *model.Remediation
	if cmd != "" {
		rem = &model.Remediation{Command: cmd, Args: args}
	}
	return model.Result{Severity: sev, Title: "T", Remediation: rem}
}

func TestPlan_AllowsStart(t *testing.T) {
	f := &Fixer{GetEuid: func() int { return 1000 }}
	items := f.Plan([]model.Result{res(model.SeverityWarning, "systemctl", "start", "nginx.service")})
	if len(items) != 1 || !items[0].Allowed {
		t.Fatalf("want 1 allowed item, got %+v", items)
	}
}

func TestPlan_AllowsRestartCritical(t *testing.T) {
	f := &Fixer{GetEuid: func() int { return 1000 }}
	items := f.Plan([]model.Result{res(model.SeverityCritical, "systemctl", "restart", "x")})
	if len(items) != 1 || !items[0].Allowed {
		t.Fatalf("want allowed restart, got %+v", items)
	}
}

func TestPlan_RejectsStop(t *testing.T) {
	f := &Fixer{}
	items := f.Plan([]model.Result{res(model.SeverityWarning, "systemctl", "stop", "x")})
	if len(items) != 1 || items[0].Allowed {
		t.Fatalf("stop must not be allowed: %+v", items)
	}
	if !strings.HasPrefix(items[0].SkipReason, "not an allowed fix: ") {
		t.Fatalf("bad SkipReason: %q", items[0].SkipReason)
	}
}

func TestPlan_RejectsRmRf(t *testing.T) {
	f := &Fixer{}
	items := f.Plan([]model.Result{res(model.SeverityCritical, "rm", "-rf", "/")})
	if len(items) != 1 || items[0].Allowed {
		t.Fatalf("rm -rf must be skipped: %+v", items)
	}
}

func TestPlan_RejectsBadUnit(t *testing.T) {
	f := &Fixer{}
	items := f.Plan([]model.Result{res(model.SeverityWarning, "systemctl", "start", "a;b")})
	if len(items) != 1 || items[0].Allowed {
		t.Fatalf("bad unit must be skipped: %+v", items)
	}
}

func TestPlan_RejectsArgCount(t *testing.T) {
	f := &Fixer{}
	for _, args := range [][]string{{"start"}, {"start", "x", "extra"}} {
		items := f.Plan([]model.Result{res(model.SeverityWarning, "systemctl", args...)})
		if len(items) != 1 || items[0].Allowed {
			t.Fatalf("args %v must be skipped: %+v", args, items)
		}
	}
}

func TestPlan_SkipsNilRemediation(t *testing.T) {
	f := &Fixer{}
	rs := []model.Result{{Severity: model.SeverityWarning, Title: "NoRem"}}
	if items := f.Plan(rs); len(items) != 0 {
		t.Fatalf("nil remediation must be excluded, got %+v", items)
	}
}

func TestPlan_SkipsPassInfo(t *testing.T) {
	f := &Fixer{}
	rs := []model.Result{
		res(model.SeverityPass, "systemctl", "start", "x"),
		res(model.SeverityInfo, "systemctl", "start", "x"),
	}
	if items := f.Plan(rs); len(items) != 0 {
		t.Fatalf("pass/info must be excluded, got %+v", items)
	}
}

func TestConfirm_AcceptsYesVariants(t *testing.T) {
	for _, in := range []string{"y\n", "Y\n", "yes\n", "YES\n", " Yes \n", "y"} {
		var buf bytes.Buffer
		f := &Fixer{Stdin: strings.NewReader(in), Stdout: &buf, IsTTY: true, GetEuid: func() int { return 1000 }}
		it := FixItem{Result: model.Result{Title: "Svc"}, Rem: model.Remediation{Command: "systemctl", Args: []string{"start", "x"}}}
		if !f.Confirm(it) {
			t.Fatalf("input %q must confirm", in)
		}
		if !strings.Contains(buf.String(), "Proposed fix for Svc: sudo systemctl start x") {
			t.Fatalf("bad prompt: %q", buf.String())
		}
	}
}

func TestConfirm_RejectsNoEmptyEOF(t *testing.T) {
	for _, in := range []string{"n\n", "no\n", "\n", "", "maybe\n"} {
		var buf bytes.Buffer
		f := &Fixer{Stdin: strings.NewReader(in), Stdout: &buf, IsTTY: true, GetEuid: func() int { return 1000 }}
		it := FixItem{Result: model.Result{Title: "Svc"}, Rem: model.Remediation{Command: "systemctl", Args: []string{"start", "x"}}}
		if f.Confirm(it) {
			t.Fatalf("input %q must not confirm", in)
		}
	}
}

type readFailer struct{ t *testing.T }

func (r readFailer) Read([]byte) (int, error) {
	r.t.Error("Confirm must not read stdin when not a TTY")
	return 0, errors.New("no read")
}

func TestConfirm_NonTTYNoRead(t *testing.T) {
	var buf bytes.Buffer
	f := &Fixer{Stdin: readFailer{t}, Stdout: &buf, IsTTY: false, GetEuid: func() int { return 1000 }}
	it := FixItem{Result: model.Result{Title: "Svc"}, Rem: model.Remediation{Command: "systemctl", Args: []string{"start", "x"}}}
	if f.Confirm(it) {
		t.Fatal("non-TTY must return false")
	}
	if buf.Len() != 0 {
		t.Fatalf("non-TTY must not print, got %q", buf.String())
	}
}

func TestConfirm_RootNoSudoPrefix(t *testing.T) {
	var buf bytes.Buffer
	f := &Fixer{Stdin: strings.NewReader("y\n"), Stdout: &buf, IsTTY: true, GetEuid: func() int { return 0 }}
	it := FixItem{Result: model.Result{Title: "Svc"}, Rem: model.Remediation{Command: "systemctl", Args: []string{"start", "x"}}}
	if !f.Confirm(it) {
		t.Fatal("want true")
	}
	if strings.Contains(buf.String(), "sudo") {
		t.Fatalf("root prompt must not contain sudo: %q", buf.String())
	}
}

func TestApply_SudoArgv(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"sudo": func(args []string) ([]byte, error) { return []byte("ok"), nil },
	}}
	var buf bytes.Buffer
	f := &Fixer{Runner: m, Stdin: strings.NewReader("y\n"), Stdout: &buf, IsTTY: true, GetEuid: func() int { return 1000 }}
	items := []FixItem{{Result: model.Result{Title: "Svc"}, Rem: model.Remediation{Command: "systemctl", Args: []string{"start", "x"}}, Allowed: true}}
	fixed, failed, skipped := f.Apply(context.Background(), items)
	if fixed != 1 || failed != 0 || skipped != 0 {
		t.Fatalf("got %d/%d/%d", fixed, failed, skipped)
	}
	if len(m.Calls) != 1 || m.Calls[0].Name != "sudo" || strings.Join(m.Calls[0].Args, " ") != "systemctl start x" {
		t.Fatalf("bad argv: %+v", m.Calls)
	}
	if !strings.Contains(buf.String(), "✓ Svc fixed") {
		t.Fatalf("bad output: %q", buf.String())
	}
}

func TestApply_RootDirect(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"systemctl": func(args []string) ([]byte, error) { return []byte("ok"), nil },
	}}
	var buf bytes.Buffer
	f := &Fixer{Runner: m, Stdin: strings.NewReader("yes\n"), Stdout: &buf, IsTTY: true, GetEuid: func() int { return 0 }}
	items := []FixItem{{Result: model.Result{Title: "Svc"}, Rem: model.Remediation{Command: "systemctl", Args: []string{"restart", "x"}}, Allowed: true}}
	fixed, _, _ := f.Apply(context.Background(), items)
	if fixed != 1 {
		t.Fatalf("want fixed=1, out=%q", buf.String())
	}
	if len(m.Calls) != 1 || m.Calls[0].Name != "systemctl" {
		t.Fatalf("root must exec directly: %+v", m.Calls)
	}
}

func TestApply_RunnerErrorContinues(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"sudo": func(args []string) ([]byte, error) {
			if len(args) > 0 && args[len(args)-1] == "bad" {
				return nil, errors.New("boom\nsecond line")
			}
			return []byte("ok"), nil
		},
	}}
	var buf bytes.Buffer
	f := &Fixer{Runner: m, Stdin: strings.NewReader("y\ny\n"), Stdout: &buf, IsTTY: true, GetEuid: func() int { return 1000 }}
	items := []FixItem{
		{Result: model.Result{Title: "Bad"}, Rem: model.Remediation{Command: "systemctl", Args: []string{"start", "bad"}}, Allowed: true},
		{Result: model.Result{Title: "Good"}, Rem: model.Remediation{Command: "systemctl", Args: []string{"start", "good"}}, Allowed: true},
	}
	fixed, failed, skipped := f.Apply(context.Background(), items)
	if fixed != 1 || failed != 1 || skipped != 0 {
		t.Fatalf("got %d/%d/%d out=%q", fixed, failed, skipped, buf.String())
	}
	if !strings.Contains(buf.String(), "✗ Bad: boom") || strings.Contains(buf.String(), "second line") {
		t.Fatalf("must show first err line only: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "✓ Good fixed") {
		t.Fatalf("second item must still run: %q", buf.String())
	}
}

func TestApply_SkippedNotExecuted(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"sudo": func(args []string) ([]byte, error) { return []byte("ok"), nil },
	}}
	var buf bytes.Buffer
	f := &Fixer{Runner: m, Stdin: strings.NewReader("y\n"), Stdout: &buf, IsTTY: true, GetEuid: func() int { return 1000 }}
	items := []FixItem{{Result: model.Result{Title: "T"}, Rem: model.Remediation{Command: "rm", Args: []string{"-rf", "/"}}, Allowed: false, SkipReason: "not an allowed fix: rm -rf /"}}
	fixed, failed, skipped := f.Apply(context.Background(), items)
	if fixed != 0 || failed != 0 || skipped != 1 {
		t.Fatalf("got %d/%d/%d", fixed, failed, skipped)
	}
	if len(m.Calls) != 0 {
		t.Fatalf("skipped must not exec: %+v", m.Calls)
	}
	if !strings.Contains(buf.String(), "skipped: not an allowed fix:") {
		t.Fatalf("bad output: %q", buf.String())
	}
}

func TestApply_DeclinedCountsSkipped(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"sudo": func(args []string) ([]byte, error) { return []byte("ok"), nil },
	}}
	var buf bytes.Buffer
	f := &Fixer{Runner: m, Stdin: strings.NewReader("n\n"), Stdout: &buf, IsTTY: true, GetEuid: func() int { return 1000 }}
	items := []FixItem{{Result: model.Result{Title: "Svc"}, Rem: model.Remediation{Command: "systemctl", Args: []string{"start", "x"}}, Allowed: true}}
	_, _, skipped := f.Apply(context.Background(), items)
	if skipped != 1 || len(m.Calls) != 0 {
		t.Fatalf("declined must skip without exec: skipped=%d calls=%+v", skipped, m.Calls)
	}
	if !strings.Contains(buf.String(), "skipped.") {
		t.Fatalf("bad output: %q", buf.String())
	}
}
