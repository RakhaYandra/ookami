package doctor

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/RakhaYandra/ookami/internal/checks"
	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

// perCheckTimeout bounds a single check; total budget (60s) is set by callers.
// 45s covers NetworkSuite worst-case: 6 layers x 5s sequential in one check.
const perCheckTimeout = 45 * time.Second

// DefaultChecks returns 18 checks ordered system → development → storage → services → network → gpu.
func DefaultChecks(cfg config.Config, r runner.Runner) []model.Check {
	return []model.Check{
		// system (5)
		checks.OSCheck{},
		checks.KernelCheck{Runner: r},
		checks.CPUCheck{},
		checks.MemoryCheck{Cfg: cfg},
		checks.UptimeCheck{},
		// development (7)
		checks.GitCheck{Runner: r},
		checks.GoCheck{Runner: r},
		checks.NodeCheck{Runner: r},
		checks.PythonCheck{Runner: r},
		checks.PHPCheck{Runner: r},
		checks.DockerCheck{Runner: r},
		checks.DockerDaemonCheck{Runner: r},
		// storage (1)
		checks.FilesystemCheck{Runner: r, Cfg: cfg},
		// services (3)
		checks.PostgreSQLCheck(r, exec.LookPath),
		checks.RedisCheck(r, exec.LookPath),
		checks.MySQLCheck(r, exec.LookPath),
		// network (1)
		checks.NetworkSuite{Cfg: cfg},
		// gpu (1)
		checks.GPUSuite{Runner: r},
	}
}

// FilterByCategory keeps checks whose Metadata().Category == cat.
func FilterByCategory(cs []model.Check, cat model.Category) []model.Check {
	var out []model.Check
	for _, c := range cs {
		if c.Metadata().Category == cat {
			out = append(out, c)
		}
	}
	return out
}

// RunAll executes checks concurrently and returns flattened results in
// deterministic (input) order. A panicking check yields one UNKNOWN result.
func RunAll(ctx context.Context, cs []model.Check) []model.Result {
	bucket := make([][]model.Result, len(cs))
	var wg sync.WaitGroup
	for i, c := range cs {
		wg.Add(1)
		go func(i int, c model.Check) {
			defer wg.Done()
			bucket[i] = runOne(ctx, c)
		}(i, c)
	}
	wg.Wait()
	var out []model.Result
	for _, rs := range bucket {
		out = append(out, rs...)
	}
	return out
}

func runOne(ctx context.Context, c model.Check) (rs []model.Result) {
	meta := model.CheckMetadata{Category: "unknown", ID: "unknown", Name: "unknown"}
	func() {
		defer func() { _ = recover() }()
		meta = c.Metadata()
	}()
	defer func() {
		if v := recover(); v != nil {
			rs = model.Single(model.Result{
				ID:       meta.ID,
				Category: meta.Category,
				Title:    meta.Name,
				Severity: model.SeverityUnknown,
				Message:  "check panicked",
			})
		}
		if rs == nil {
			rs = model.Single(model.Result{
				ID:       meta.ID,
				Category: meta.Category,
				Title:    meta.Name,
				Severity: model.SeverityUnknown,
				Message:  "no results",
			})
		}
	}()
	cctx, cancel := context.WithTimeout(ctx, perCheckTimeout)
	defer cancel()
	rs = c.Run(cctx)
	return rs
}
