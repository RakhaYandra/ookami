package doctor

import (
	"context"
	"testing"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

type panicCheck struct{}

func (panicCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{ID: "test-panic", Name: "Panic", Category: model.CategorySystem}
}

func (panicCheck) Run(context.Context) []model.Result { panic("boom") }

func TestDefaultChecksOrder(t *testing.T) {
	cs := DefaultChecks(config.Default(), runner.NewOSRunner(0))
	if len(cs) != 13 {
		t.Fatalf("want 13 checks, got %d", len(cs))
	}
	want := []string{
		"system-os", "system-kernel", "system-cpu", "system-memory", "system-uptime",
		"development-git", "development-go", "development-node", "development-python",
		"development-php", "development-docker", "development-docker-daemon",
		"storage-fs",
	}
	for i, w := range want {
		if got := cs[i].Metadata().ID; got != w {
			t.Fatalf("index %d: want %s, got %s", i, w, got)
		}
	}
}

func TestFilterByCategory(t *testing.T) {
	cs := DefaultChecks(config.Default(), runner.NewOSRunner(0))
	if got := len(FilterByCategory(cs, model.CategorySystem)); got != 5 {
		t.Fatalf("system: want 5, got %d", got)
	}
	if got := len(FilterByCategory(cs, model.CategoryDevelopment)); got != 7 {
		t.Fatalf("development: want 7, got %d", got)
	}
	if got := len(FilterByCategory(cs, model.CategoryStorage)); got != 1 {
		t.Fatalf("storage: want 1, got %d", got)
	}
	if got := len(FilterByCategory(cs, model.CategoryNetwork)); got != 0 {
		t.Fatalf("network: want 0, got %d", got)
	}
}

func TestRunAllRecoversPanic(t *testing.T) {
	rs := RunAll(context.Background(), []model.Check{panicCheck{}})
	if len(rs) != 1 {
		t.Fatalf("want 1 result, got %d", len(rs))
	}
	if rs[0].Severity != model.SeverityUnknown {
		t.Fatalf("want unknown, got %s", rs[0].Severity)
	}
}

func TestRunAllDeterministicOrder(t *testing.T) {
	cs := DefaultChecks(config.Default(), runner.NewOSRunner(0))
	rs := RunAll(context.Background(), FilterByCategory(cs, model.CategorySystem))
	if len(rs) == 0 {
		t.Fatal("want non-empty results")
	}
	if rs[0].ID != "system-os" {
		t.Fatalf("want first result system-os, got %s", rs[0].ID)
	}
}
