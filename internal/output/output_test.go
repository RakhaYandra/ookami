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
