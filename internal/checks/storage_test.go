package checks

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

func dfOut(pct int, mount string) string {
	return fmt.Sprintf("Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/sda1 100000 50000 50000 %d%% %s\n", pct, mount)
}

func dfRunner(pct int) *runner.MockRunner {
	return &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"df": func([]string) ([]byte, error) { return []byte(dfOut(pct, "/")), nil },
	}}
}

func TestFilesystemCheck_Pass(t *testing.T) {
	c := FilesystemCheck{Runner: dfRunner(61), Cfg: config.Default(), Mounts: []string{"/"}}
	rs := c.Run(context.Background())
	if len(rs) != 1 {
		t.Fatalf("want 1 result, got %d", len(rs))
	}
	if rs[0].Severity != model.SeverityPass {
		t.Fatalf("got %+v", rs[0])
	}
	if rs[0].ID != "storage-fs-root" || rs[0].Title != "/" || rs[0].Message != "61% used" {
		t.Fatalf("got %+v", rs[0])
	}
	if rs[0].Category != model.CategoryStorage {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestFilesystemCheck_Warning(t *testing.T) {
	c := FilesystemCheck{Runner: dfRunner(82), Cfg: config.Default(), Mounts: []string{"/home"}}
	rs := c.Run(context.Background())
	if len(rs) != 1 || rs[0].Severity != model.SeverityWarning {
		t.Fatalf("got %+v", rs)
	}
	if rs[0].ID != "storage-fs-home" || rs[0].Message != "82% used" {
		t.Fatalf("got %+v", rs[0])
	}
}

func TestFilesystemCheck_Critical(t *testing.T) {
	c := FilesystemCheck{Runner: dfRunner(95), Cfg: config.Default(), Mounts: []string{"/var"}}
	rs := c.Run(context.Background())
	if len(rs) != 1 || rs[0].Severity != model.SeverityCritical {
		t.Fatalf("got %+v", rs)
	}
}

func TestFilesystemCheck_DFErrorExplicit(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"df": func([]string) ([]byte, error) { return nil, errors.New("no df") },
	}}
	rs := FilesystemCheck{Runner: m, Cfg: config.Default(), Mounts: []string{"/", "/home"}}.Run(context.Background())
	if len(rs) != 2 {
		t.Fatalf("want 2 results, got %+v", rs)
	}
	for i, wantID := range []string{"storage-fs-root", "storage-fs-home"} {
		if rs[i].ID != wantID || rs[i].Severity != model.SeverityUnknown || rs[i].Message != "unreadable" {
			t.Fatalf("idx %d got %+v", i, rs[i])
		}
		if rs[i].Category != model.CategoryStorage {
			t.Fatalf("idx %d wrong category: %+v", i, rs[i])
		}
	}
}

func TestFilesystemCheck_MultiMountOrdered(t *testing.T) {
	pcts := map[string]int{"/": 61, "/home": 82, "/var": 95}
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"df": func(args []string) ([]byte, error) { return []byte(dfOut(pcts[args[1]], args[1])), nil },
	}}
	rs := FilesystemCheck{Runner: m, Cfg: config.Default(), Mounts: []string{"/", "/home", "/var"}}.Run(context.Background())
	if len(rs) != 3 {
		t.Fatalf("want 3 results, got %+v", rs)
	}
	want := []struct {
		id  string
		sev model.Severity
	}{
		{"storage-fs-root", model.SeverityPass},
		{"storage-fs-home", model.SeverityWarning},
		{"storage-fs-var", model.SeverityCritical},
	}
	for i, w := range want {
		if rs[i].ID != w.id || rs[i].Severity != w.sev {
			t.Fatalf("idx %d got %+v want %v", i, rs[i], w)
		}
	}
}

func TestFilesystemCheck_DefaultMounts(t *testing.T) {
	m := &runner.MockRunner{Handlers: map[string]func([]string) ([]byte, error){
		"df": func(args []string) ([]byte, error) { return []byte(dfOut(10, args[1])), nil },
	}}
	rs := FilesystemCheck{Runner: m, Cfg: config.Config{}}.Run(context.Background())
	if len(rs) != 3 {
		t.Fatalf("want 3 default mounts, got %+v", rs)
	}
}

func TestParseDF_OK(t *testing.T) {
	n, err := parseDF(dfOut(61, "/"))
	if err != nil || n != 61 {
		t.Fatalf("got %d %v", n, err)
	}
}

func TestParseDF_Broken(t *testing.T) {
	for _, in := range []string{
		"Filesystem 1024-blocks Used Available Capacity Mounted on\n",
		"Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/sda1 100 50\n",
		"Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/sda1 100 50 50 xx% /\n",
	} {
		if _, err := parseDF(in); err == nil {
			t.Fatalf("want error for %q", in)
		}
	}
}
