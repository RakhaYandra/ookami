package doctor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

type panicCheck struct{}

func (panicCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{ID: "test-panic", Name: "Panic", Category: model.CategorySystem}
}

func (panicCheck) Run(context.Context) []model.Result { panic("boom") }

// stubCheck is a hermetic stand-in: no host paths, no runner.
type stubCheck struct {
	id     string
	delay  time.Duration
	record func(string)
}

func (s stubCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{ID: s.id, Name: s.id, Category: model.CategorySystem}
}

func (s stubCheck) Run(context.Context) []model.Result {
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	if s.record != nil {
		s.record(s.id)
	}
	return model.Single(model.Result{ID: s.id, Category: model.CategorySystem, Title: s.id, Severity: model.SeverityPass, Message: "ok"})
}

func TestDefaultChecksOrder(t *testing.T) {
	cs := DefaultChecks(config.Default(), runner.NewOSRunner(0))
	if len(cs) != 17 {
		t.Fatalf("want 17 checks, got %d", len(cs))
	}
	want := []string{
		"system-os", "system-kernel", "system-cpu", "system-memory", "system-uptime",
		"development-git", "development-go", "development-node", "development-python",
		"development-php", "development-docker", "development-docker-daemon",
		"storage-fs",
		"services-postgresql", "services-redis", "services-mysql",
		"network-suite",
	}
	for i, w := range want {
		if got := cs[i].Metadata().ID; got != w {
			t.Fatalf("index %d: want %s, got %s", i, w, got)
		}
	}
	// Category order: system → development → storage → services → network (network last).
	var cats []model.Category
	for _, c := range cs {
		cats = append(cats, c.Metadata().Category)
	}
	for i, wantCat := range []model.Category{
		model.CategorySystem, model.CategorySystem, model.CategorySystem,
		model.CategorySystem, model.CategorySystem,
		model.CategoryDevelopment, model.CategoryDevelopment, model.CategoryDevelopment,
		model.CategoryDevelopment, model.CategoryDevelopment,
		model.CategoryDevelopment, model.CategoryDevelopment,
		model.CategoryStorage,
		model.CategoryServices, model.CategoryServices, model.CategoryServices,
		model.CategoryNetwork,
	} {
		if cats[i] != wantCat {
			t.Fatalf("index %d: want category %s, got %s", i, wantCat, cats[i])
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
	if got := len(FilterByCategory(cs, model.CategoryServices)); got != 3 {
		t.Fatalf("services: want 3, got %d", got)
	}
	if got := len(FilterByCategory(cs, model.CategoryNetwork)); got != 1 {
		t.Fatalf("network: want 1, got %d", got)
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
	var mu sync.Mutex
	var ran []string
	stub := func(id string, delay time.Duration) model.Check {
		return stubCheck{id: id, delay: delay, record: func(id string) {
			mu.Lock()
			ran = append(ran, id)
			mu.Unlock()
		}}
	}
	// First check sleeps longest: input order must win over completion order.
	cs := []model.Check{
		stub("test-a", 60*time.Millisecond),
		stub("test-b", 20*time.Millisecond),
		stub("test-c", 0),
	}
	rs := RunAll(context.Background(), cs)
	if len(rs) != len(cs) {
		t.Fatalf("want %d results, got %d", len(cs), len(rs))
	}
	for i, w := range []string{"test-a", "test-b", "test-c"} {
		if rs[i].ID != w {
			t.Fatalf("index %d: want %s, got %s", i, w, rs[i].ID)
		}
	}
}
