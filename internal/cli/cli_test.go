package cli

import (
	"bytes"
	"strings"
	"testing"
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

func TestDoctorFlags(t *testing.T) {
	cmd := NewDoctorCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetArgs([]string{"--json", "--quiet"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("doctor flags execute: %v", err)
	}
}
