package model

import (
	"strings"
	"testing"
)

func TestCategoryConstants(t *testing.T) {
	got := []Category{CategorySystem, CategoryDevelopment, CategoryStorage, CategoryNetwork, CategoryGPU, CategoryServices}
	want := []Category{"system", "development", "storage", "network", "gpu", "services"}
	if len(got) != len(want) {
		t.Fatalf("got %d categories, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("category[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSeverityConstants(t *testing.T) {
	got := []Severity{SeverityPass, SeverityInfo, SeverityWarning, SeverityCritical, SeverityUnknown}
	want := []Severity{"pass", "info", "warning", "critical", "unknown"}
	if len(got) != len(want) {
		t.Fatalf("got %d severities, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("severity[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRemediationNoShell(t *testing.T) {
	r := Remediation{Description: "restart service", Command: "systemctl", Args: []string{"restart", "docker"}, Safe: false, RequiresSudo: true}
	if strings.Contains(r.Command, " ") {
		t.Errorf("Command %q must be single binary, no space", r.Command)
	}
}
