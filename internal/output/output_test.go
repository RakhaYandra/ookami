package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
)

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	rs := []model.Result{{ID: "a", Severity: model.SeverityPass, Title: "ok"}}
	if err := RenderJSON(&buf, rs, 100, "OK"); err != nil {
		t.Fatal(err)
	}
	var v struct {
		Score int `json:"score"`
	}
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if v.Score != 100 {
		t.Fatalf("score=%d", v.Score)
	}
}

func TestRenderJSONBreakdownOrdered(t *testing.T) {
	var buf bytes.Buffer
	rs := []model.Result{
		{ID: "svc", Category: model.CategoryServices, Severity: model.SeverityWarning, Title: "svc"},
		{ID: "net", Category: model.CategoryNetwork, Severity: model.SeverityPass, Title: "net"},
		{ID: "sys", Category: model.CategorySystem, Severity: model.SeverityPass, Title: "sys"},
		{ID: "dev", Category: model.CategoryDevelopment, Severity: model.SeverityPass, Title: "dev"},
	}
	if err := RenderJSONBreakdown(&buf, rs); err != nil {
		t.Fatal(err)
	}
	var v struct {
		Score      int    `json:"score"`
		Status     string `json:"status"`
		Categories []struct {
			Category string `json:"category"`
			Status   string `json:"status"`
			Score    int    `json:"score"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	want := []string{"system", "development", "network", "services"}
	if len(v.Categories) != len(want) {
		t.Fatalf("categories=%v, want %v", v.Categories, want)
	}
	for i, w := range want {
		if v.Categories[i].Category != w {
			t.Fatalf("order[%d]=%q, want %q (full=%v)", i, v.Categories[i].Category, w, v.Categories)
		}
	}
	for _, c := range v.Categories {
		if c.Category == "services" && c.Score != 95 {
			t.Fatalf("services score=%d, want 95", c.Score)
		}
	}
}

func TestRenderJSONBreakdownGPUNoneExcluded(t *testing.T) {
	var buf bytes.Buffer
	rs := []model.Result{
		{ID: "sys", Category: model.CategorySystem, Severity: model.SeverityPass, Title: "sys"},
		{ID: "gpu-none", Category: model.CategoryGPU, Severity: model.SeverityInfo, Title: "no gpu"},
	}
	if err := RenderJSONBreakdown(&buf, rs); err != nil {
		t.Fatal(err)
	}
	var v struct {
		Categories []struct {
			Category string `json:"category"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	for _, c := range v.Categories {
		if c.Category == "gpu" {
			t.Fatalf("gpu-none must be excluded, got %v", v.Categories)
		}
	}
	if len(v.Categories) != 1 || v.Categories[0].Category != "system" {
		t.Fatalf("want [system], got %v", v.Categories)
	}
}
func TestRenderQuiet(t *testing.T) {
	var buf bytes.Buffer
	rs := []model.Result{
		{Severity: model.SeverityPass, Title: "pass"},
		{Severity: model.SeverityInfo, Title: "info"},
		{Severity: model.SeverityWarning, Title: "warn"},
		{Severity: model.SeverityCritical, Title: "crit"},
		{Severity: model.SeverityUnknown, Title: "unk"},
	}
	if err := RenderQuiet(&buf, rs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"warn", "crit", "unk"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, no := range []string{"pass", "info"} {
		if strings.Contains(out, no) {
			t.Errorf("should not contain %q", no)
		}
	}
}

func TestRenderQuietSymbolAndMessage(t *testing.T) {
	var buf bytes.Buffer
	rs := []model.Result{
		{Severity: model.SeverityWarning, Title: "PostgreSQL", Message: "installed, no systemd unit"},
		{Severity: model.SeverityCritical, Title: "Disk", Message: "full"},
		{Severity: model.SeverityUnknown, Title: "Uptime", Message: "n/a"},
	}
	if err := RenderQuiet(&buf, rs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"⚠ PostgreSQL installed, no systemd unit", "✗ Disk full", "? Uptime n/a"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestRenderHumanEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHuman(&buf, nil, 100, "healthy"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No results.") {
		t.Errorf("empty output should contain %q, got %q", "No results.", buf.String())
	}
}

func TestRenderHumanGrouped(t *testing.T) {
	var buf bytes.Buffer
	rs := []model.Result{
		{Category: model.CategoryDevelopment, Severity: model.SeverityWarning, Title: "Go", Message: "old"},
		{Category: model.CategorySystem, Severity: model.SeverityPass, Title: "OS", Message: "ok"},
		{Category: model.CategoryDevelopment, Severity: model.SeverityCritical, Title: "Docker", Message: "down"},
		{Category: model.CategorySystem, Severity: model.SeverityUnknown, Title: "Uptime", Message: "n/a"},
	}
	if err := RenderHuman(&buf, rs, 70, "critical"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "OOKAMI") {
		t.Errorf("missing header:\n%s", out)
	}
	sys := strings.Index(out, "System")
	dev := strings.Index(out, "Development")
	if sys < 0 || dev < 0 || sys > dev {
		t.Errorf("categories not in order (System before Development):\n%s", out)
	}
	for _, want := range []string{"✓ OS ok", "⚠ Go old", "✗ Docker down", "? Uptime"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	// unknown counts as warning: 1 warning + 1 unknown = 2 warnings, 1 critical
	for _, want := range []string{"Score: 70/100", "2 warnings, 1 critical issue", "has critical issues"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing footer %q:\n%s", want, out)
		}
	}
}

func TestRenderHumanSuggestedAction(t *testing.T) {
	var buf bytes.Buffer
	rs := []model.Result{
		{
			Category: model.CategoryServices, Severity: model.SeverityWarning,
			Title: "Docker", Message: "down",
			Remediation: &model.Remediation{
				Description: "Start Docker daemon", Command: "systemctl",
				Args:         []string{"start", "docker"},
				RequiresSudo: true,
			},
		},
		{
			Category: model.CategoryDevelopment, Severity: model.SeverityCritical,
			Title: "Go", Message: "missing",
			Remediation: &model.Remediation{
				Description: "Install Go", Command: "apt",
				Args: []string{"install", "golang"},
			},
		},
	}
	if err := RenderHuman(&buf, rs, 50, "critical"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	wantSudo := "    → Suggested action: systemctl start docker (Start Docker daemon, requires sudo)"
	if !strings.Contains(out, wantSudo) {
		t.Errorf("missing exact %q:\n%s", wantSudo, out)
	}
	wantPlain := "    → Suggested action: apt install golang (Install Go)"
	if !strings.Contains(out, wantPlain) {
		t.Errorf("missing exact %q:\n%s", wantPlain, out)
	}
}

func TestRenderHumanNoSuggestedAction(t *testing.T) {
	var buf bytes.Buffer
	rs := []model.Result{
		{Category: model.CategorySystem, Severity: model.SeverityPass, Title: "OS", Remediation: &model.Remediation{Description: "d", Command: "cmd"}},
		{Category: model.CategorySystem, Severity: model.SeverityInfo, Title: "Info", Remediation: &model.Remediation{Description: "d", Command: "cmd"}},
		{Category: model.CategorySystem, Severity: model.SeverityWarning, Title: "WarnNoRem"},
		{Category: model.CategorySystem, Severity: model.SeverityCritical, Title: "CritNoRem"},
	}
	if err := RenderHuman(&buf, rs, 90, "warning"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "Suggested action") {
		t.Errorf("should not contain Suggested action:\n%s", buf.String())
	}
}

func TestRenderHumanVerboseDetailsSorted(t *testing.T) {
	var buf bytes.Buffer
	rs := []model.Result{
		{
			Category: model.CategorySystem, Severity: model.SeverityPass,
			Title: "Disk", Message: "ok",
			Details: map[string]any{"b": 2, "a": 1},
		},
	}
	if err := RenderHumanVerbose(&buf, rs, 100, "healthy"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	ia := strings.Index(out, "      a=1")
	ib := strings.Index(out, "      b=2")
	if ia < 0 || ib < 0 || ia > ib {
		t.Errorf("details not sorted (a before b):\n%s", out)
	}
	for _, want := range []string{"Score: 100/100", "OOKAMI"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing footer %q:\n%s", want, out)
		}
	}
	var plain bytes.Buffer
	if err := RenderHuman(&plain, rs, 100, "healthy"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain.String(), "a=1") {
		t.Errorf("non-verbose should not contain details:\n%s", plain.String())
	}
}

func TestNoANSI(t *testing.T) {
	rs := []model.Result{
		{
			Category: model.CategoryServices, Severity: model.SeverityWarning,
			Title: "Docker", Message: "down",
			Details:     map[string]any{"a": 1},
			Remediation: &model.Remediation{Description: "Start Docker daemon", Command: "systemctl", Args: []string{"start", "docker"}, RequiresSudo: true},
		},
		{Category: model.CategorySystem, Severity: model.SeverityPass, Title: "OS"},
	}
	var h, v, q bytes.Buffer
	if err := RenderHuman(&h, rs, 80, "warning"); err != nil {
		t.Fatal(err)
	}
	if err := RenderHumanVerbose(&v, rs, 80, "warning"); err != nil {
		t.Fatal(err)
	}
	if err := RenderQuiet(&q, rs); err != nil {
		t.Fatal(err)
	}
	for name, out := range map[string]string{"human": h.String(), "verbose": v.String(), "quiet": q.String()} {
		if strings.Contains(out, "\x1b") {
			t.Errorf("%s contains ANSI escape:\n%q", name, out)
		}
	}
}
