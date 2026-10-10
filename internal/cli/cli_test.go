package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/output"
)

func TestVersionOutput(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version execute: %v", err)
	}
	if !strings.Contains(buf.String(), Version) {
		t.Fatalf("output %q does not contain version %q", buf.String(), Version)
	}
}

func TestCheckInvalidCategory(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetArgs([]string{"check", "bogus"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid category, got nil")
	}
}

func TestCheckValidCategory(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetArgs([]string{"check", "network"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("check network execute: %v", err)
	}
}

func TestCheckServicesCategory(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"check", "services"})
	err := cmd.Execute()
	if err == nil {
		return
	}
	var ee *ExitError
	if errors.As(err, &ee) && (ee.Code == 0 || ee.Code == 1 || ee.Code == 2) {
		return
	}
	t.Fatalf("check services execute: want nil or ExitError 0/1/2, got %v", err)
}

func TestDoctorFlagParsing(t *testing.T) {
	cmd := NewDoctorCmd()
	if err := cmd.ParseFlags([]string{"--json", "--quiet", "--fix"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	for flag := range map[string]bool{"json": true, "quiet": true, "fix": true} {
		got, err := cmd.Flags().GetBool(flag)
		if err != nil {
			t.Fatalf("get --%s: %v", flag, err)
		}
		if !got {
			t.Fatalf("want --%s=true after parsing", flag)
		}
	}

	def := NewDoctorCmd()
	for _, flag := range []string{"json", "quiet", "fix"} {
		got, err := def.Flags().GetBool(flag)
		if err != nil {
			t.Fatalf("get default --%s: %v", flag, err)
		}
		if got {
			t.Fatalf("want default --%s=false", flag)
		}
	}
}

func TestExitCodeMapsToExitError(t *testing.T) {
	rs := []model.Result{{Severity: model.SeverityWarning}}
	if got := output.ExitCode(rs, false, false); got != 1 {
		t.Fatalf("warning ExitCode: want 1, got %d", got)
	}
	ee := &ExitError{Code: 1}
	var target *ExitError
	if !errors.As(ee, &target) || target.Code != 1 {
		t.Fatal("errors.As(ExitError) failed")
	}
}
