package checks

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

const archOSRelease = `NAME="Arch Linux"
PRETTY_NAME="Arch Linux"
ID=arch
BUILD_ID=rolling
`

const sampleCPUInfo = `processor	: 0
model name	: AMD Ryzen 7 5800X 8-Core Processor
cpu cores	: 8
processor	: 1
model name	: AMD Ryzen 7 5800X 8-Core Processor
cpu cores	: 8
`

const meminfo80 = `MemTotal:       16384000 kB
MemAvailable:    3276800 kB
MemFree:         1000000 kB
`

const meminfo95 = `MemTotal:       16384000 kB
MemAvailable:     819200 kB
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fixture")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseOSRelease_Arch(t *testing.T) {
	if got := parseOSRelease(strings.NewReader(archOSRelease)); got != "Arch Linux" {
		t.Fatalf("got %q", got)
	}
}

func TestParseOSRelease_FallbackAndUnknown(t *testing.T) {
	if got := parseOSRelease(strings.NewReader("PRETTY_NAME=\"Foo OS\"\n")); got != "Foo OS" {
		t.Fatalf("fallback got %q", got)
	}
	if got := parseOSRelease(strings.NewReader("ID=foo\n")); got != "Unknown" {
		t.Fatalf("empty got %q", got)
	}
}

func TestOSCheck_Unreadable(t *testing.T) {
	c := OSCheck{OSReleasePath: filepath.Join(t.TempDir(), "missing")}
	rs := c.Run(context.Background())
	if rs[0].Severity != model.SeverityUnknown || rs[0].Message != "os-release unreadable" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestKernelCheck_OK(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"uname": func(args []string) ([]byte, error) {
			if args[0] == "-r" {
				return []byte("6.17.1-arch1-1\n"), nil
			}
			return []byte("x86_64\n"), nil
		},
	}}
	rs := KernelCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityPass || rs[0].Message != "6.17.1-arch1-1 x86_64" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestKernelCheck_Fail(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"uname": func([]string) ([]byte, error) { return nil, errors.New("no uname") },
	}}
	rs := KernelCheck{Runner: m}.Run(context.Background())
	if rs[0].Severity != model.SeverityUnknown {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestCPUCheck_NATemp(t *testing.T) {
	c := CPUCheck{
		CPUInfoPath: writeTemp(t, sampleCPUInfo),
		TempPath:    filepath.Join(t.TempDir(), "missing-temp"),
	}
	rs := c.Run(context.Background())
	if rs[0].Severity != model.SeverityPass {
		t.Fatalf("got %+v", rs[0])
	}
	if !strings.Contains(rs[0].Message, "AMD Ryzen 7 5800X") || !strings.Contains(rs[0].Message, "N/A") {
		t.Fatalf("got %q", rs[0].Message)
	}
	if !strings.Contains(rs[0].Message, "8 cores/2 threads") {
		t.Fatalf("topology got %q", rs[0].Message)
	}
}

func TestMemoryCheck_Warning(t *testing.T) {
	c := MemoryCheck{
		MemInfoPath: writeTemp(t, meminfo80),
		Cfg:         config.Config{MemWarn: 80, MemCrit: 95},
	}
	rs := c.Run(context.Background())
	if rs[0].Severity != model.SeverityWarning {
		t.Fatalf("got %+v", rs[0])
	}
	if !strings.Contains(rs[0].Message, "% used") {
		t.Fatalf("got %q", rs[0].Message)
	}
}

func TestMemoryCheck_Critical(t *testing.T) {
	c := MemoryCheck{
		MemInfoPath: writeTemp(t, meminfo95),
		Cfg:         config.Config{MemWarn: 80, MemCrit: 95},
	}
	rs := c.Run(context.Background())
	if rs[0].Severity != model.SeverityCritical {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestParseMemInfo_FallbackMemFree(t *testing.T) {
	total, avail := parseMemInfo(strings.NewReader("MemTotal: 8000 kB\nMemFree: 2000 kB\n"))
	if total != 8000 || avail != 2000 {
		t.Fatalf("got %d %d", total, avail)
	}
}

func TestUptimeCheck_Format(t *testing.T) {
	c := UptimeCheck{UptimePath: writeTemp(t, "13320.00 50000.00\n")}
	rs := c.Run(context.Background())
	if rs[0].Severity != model.SeverityInfo || rs[0].Message != "3h 42m" {
		t.Fatalf("got %+v", rs[0])
	}
	if got := formatUptime(30 * 60); got != "30m" {
		t.Fatalf("mins got %q", got)
	}
	if got := formatUptime(25*3600 + 600); got != "1d 1h" {
		t.Fatalf("days got %q", got)
	}
}

func TestUptimeCheck_Unreadable(t *testing.T) {
	c := UptimeCheck{UptimePath: filepath.Join(t.TempDir(), "missing")}
	rs := c.Run(context.Background())
	if rs[0].Severity != model.SeverityUnknown {
		t.Fatalf("got %+v", rs[0])
	}
}
