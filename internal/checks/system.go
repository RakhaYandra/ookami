package checks

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

var (
	_ model.Check = OSCheck{}
	_ model.Check = KernelCheck{}
	_ model.Check = CPUCheck{}
	_ model.Check = MemoryCheck{}
	_ model.Check = UptimeCheck{}
)

// OSCheck reports the distro name from /etc/os-release.
type OSCheck struct {
	OSReleasePath string
}

func (c OSCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "system-os", Name: "OS",
		Description: "Operating system name from /etc/os-release",
		Category:    model.CategorySystem,
	}
}

func parseOSRelease(r io.Reader) (name string) {
	var pretty string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"`)
		switch strings.TrimSpace(k) {
		case "NAME":
			if name == "" {
				name = v
			}
		case "PRETTY_NAME":
			if pretty == "" {
				pretty = v
			}
		}
	}
	if name != "" {
		return name
	}
	if pretty != "" {
		return pretty
	}
	return "Unknown"
}

func (c OSCheck) Run(_ context.Context) []Result {
	return model.Single(c.run())
}

type Result = model.Result

func (c OSCheck) run() model.Result {
	base := model.Result{ID: "system-os", Category: model.CategorySystem, Title: "OS"}
	path := c.OSReleasePath
	if path == "" {
		path = "/etc/os-release"
	}
	f, err := os.Open(path)
	if err != nil {
		base.Severity = model.SeverityUnknown
		base.Message = "os-release unreadable"
		return base
	}
	defer f.Close()
	name := parseOSRelease(f)
	base.Severity = model.SeverityPass
	base.Message = name
	base.Details = map[string]any{"name": name}
	return base
}

// KernelCheck reports `uname -r` + `-m`.
type KernelCheck struct {
	Runner runner.Runner
}

func (c KernelCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "system-kernel", Name: "Kernel",
		Description: "Kernel release and machine architecture via uname",
		Category:    model.CategorySystem,
	}
}

func (c KernelCheck) Run(ctx context.Context) []Result {
	base := model.Result{ID: "system-kernel", Category: model.CategorySystem, Title: "Kernel"}
	if c.Runner == nil {
		base.Severity = model.SeverityUnknown
		base.Message = "uname unavailable"
		return model.Single(base)
	}
	rel, err1 := c.Runner.Run(ctx, "uname", "-r")
	machine, err2 := c.Runner.Run(ctx, "uname", "-m")
	if err1 != nil || err2 != nil {
		base.Severity = model.SeverityUnknown
		base.Message = "uname unreadable"
		return model.Single(base)
	}
	base.Severity = model.SeverityPass
	base.Message = fmt.Sprintf("%s %s",
		strings.TrimSpace(string(rel)), strings.TrimSpace(string(machine)))
	return model.Single(base)
}

// CPUCheck reports model name, core/thread counts, and temperature if available.
type CPUCheck struct {
	CPUInfoPath string
	TempPath    string
}

func (c CPUCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "system-cpu", Name: "CPU",
		Description: "CPU model, topology, and temperature",
		Category:    model.CategorySystem,
	}
}

func parseCPUInfo(r io.Reader) (name string, cores, threads int) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "model name":
			if name == "" {
				name = v
			}
		case "processor":
			threads++
		case "cpu cores":
			if cores == 0 {
				if n, err := strconv.Atoi(strings.Fields(v)[0]); err == nil {
					cores = n
				}
			}
		}
	}
	if name == "" {
		name = "Unknown"
	}
	if cores == 0 {
		cores = threads
	}
	return name, cores, threads
}

func readTemp(path string) string {
	if path == "" {
		path = "/sys/class/thermal/thermal_zone0/temp"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "N/A"
	}
	milli, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return "N/A"
	}
	return fmt.Sprintf("%d°C", milli/1000)
}

func (c CPUCheck) Run(_ context.Context) []Result {
	base := model.Result{ID: "system-cpu", Category: model.CategorySystem, Title: "CPU"}
	path := c.CPUInfoPath
	if path == "" {
		path = "/proc/cpuinfo"
	}
	f, err := os.Open(path)
	if err != nil {
		base.Severity = model.SeverityUnknown
		base.Message = "cpuinfo unreadable"
		return model.Single(base)
	}
	defer f.Close()
	name, cores, threads := parseCPUInfo(f)
	base.Severity = model.SeverityPass
	base.Message = fmt.Sprintf("%s (%d cores/%d threads, %s)", name, cores, threads, readTemp(c.TempPath))
	return model.Single(base)
}

// MemoryCheck compares used% against MemWarn/MemCrit.
type MemoryCheck struct {
	MemInfoPath string
	Cfg         config.Config
}

func (c MemoryCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "system-memory", Name: "Memory",
		Description: "Memory usage against warn/crit thresholds",
		Category:    model.CategorySystem,
	}
}

func parseMemInfo(r io.Reader) (totalKB, availKB uint64) {
	var memFree uint64
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(v))
		if len(fields) == 0 {
			continue
		}
		n, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(k) {
		case "MemTotal":
			totalKB = n
		case "MemAvailable":
			availKB = n
		case "MemFree":
			memFree = n
		}
	}
	if availKB == 0 {
		availKB = memFree
	}
	return totalKB, availKB
}

func (c MemoryCheck) Run(_ context.Context) []Result {
	base := model.Result{ID: "system-memory", Category: model.CategorySystem, Title: "Memory"}
	path := c.MemInfoPath
	if path == "" {
		path = "/proc/meminfo"
	}
	f, err := os.Open(path)
	if err != nil {
		base.Severity = model.SeverityUnknown
		base.Message = "meminfo unreadable"
		return model.Single(base)
	}
	defer f.Close()
	total, avail := parseMemInfo(f)
	if total == 0 {
		base.Severity = model.SeverityUnknown
		base.Message = "meminfo unreadable"
		return model.Single(base)
	}
	warn, crit := c.Cfg.MemWarn, c.Cfg.MemCrit
	if warn <= 0 || crit <= 0 {
		d := config.Default()
		warn, crit = d.MemWarn, d.MemCrit
	}
	usedPct := int((total - avail) * 100 / total)
	base.Message = fmt.Sprintf("%.1f / %.0f GB available (%d%% used)",
		float64(avail)/1048576, float64(total)/1048576, usedPct)
	base.Details = map[string]any{"total_kb": total, "avail_kb": avail, "used_pct": usedPct}
	switch {
	case usedPct >= crit:
		base.Severity = model.SeverityCritical
	case usedPct >= warn:
		base.Severity = model.SeverityWarning
	default:
		base.Severity = model.SeverityPass
	}
	return model.Single(base)
}

// UptimeCheck is informational only (SeverityInfo, never cuts score).
type UptimeCheck struct {
	UptimePath string
}

func (c UptimeCheck) Metadata() model.CheckMetadata {
	return model.CheckMetadata{
		ID: "system-uptime", Name: "Uptime",
		Description: "System uptime from /proc/uptime",
		Category:    model.CategorySystem,
		Optional:    true,
	}
}

func formatUptime(sec float64) string {
	mins := int(sec / 60)
	if mins < 60 {
		return fmt.Sprintf("%dm", mins)
	}
	h, m := mins/60, mins%60
	if h < 24 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dd %dh", h/24, h%24)
}

func (c UptimeCheck) Run(_ context.Context) []Result {
	base := model.Result{ID: "system-uptime", Category: model.CategorySystem, Title: "Uptime"}
	path := c.UptimePath
	if path == "" {
		path = "/proc/uptime"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		base.Severity = model.SeverityInfo
		base.Message = "uptime unreadable"
		return model.Single(base)
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		base.Severity = model.SeverityInfo
		base.Message = "uptime unreadable"
		return model.Single(base)
	}
	sec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		base.Severity = model.SeverityInfo
		base.Message = "uptime unreadable"
		return model.Single(base)
	}
	base.Severity = model.SeverityInfo
	base.Message = formatUptime(sec)
	base.Details = map[string]any{"seconds": sec}
	return model.Single(base)
}
