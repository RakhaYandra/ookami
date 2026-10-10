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
	if err := RenderHuman(&buf, nil, 100, "OK"); err != nil {
		t.Fatal(err)
	}
}
