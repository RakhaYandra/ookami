package fixer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/RakhaYandra/ookami/internal/model"
	"github.com/RakhaYandra/ookami/internal/runner"
)

type Fixer struct {
	Runner  runner.Runner
	Stdin   io.Reader
	Stdout  io.Writer
	IsTTY   bool
	GetEuid func() int
	br      *bufio.Reader
}

type FixItem struct {
	Result     model.Result
	Rem        model.Remediation
	Allowed    bool
	SkipReason string
}

var unitRe = regexp.MustCompile(`^[a-zA-Z0-9@._:\-]+$`)

func (f *Fixer) euid() int {
	if f != nil && f.GetEuid != nil {
		return f.GetEuid()
	}
	return os.Geteuid()
}

func (f *Fixer) out() io.Writer {
	if f != nil && f.Stdout != nil {
		return f.Stdout
	}
	return io.Discard
}

func (f *Fixer) Plan(rs []model.Result) []FixItem {
	var items []FixItem
	for _, r := range rs {
		if r.Severity != model.SeverityWarning && r.Severity != model.SeverityCritical {
			continue
		}
		if r.Remediation == nil {
			continue
		}
		rem := *r.Remediation
		it := FixItem{Result: r, Rem: rem}
		if rem.Command == "systemctl" && len(rem.Args) == 2 &&
			(rem.Args[0] == "start" || rem.Args[0] == "restart") &&
			unitRe.MatchString(rem.Args[1]) {
			it.Allowed = true
		} else {
			it.Allowed = false
			it.SkipReason = "not an allowed fix: " + strings.Join(append([]string{rem.Command}, rem.Args...), " ")
		}
		items = append(items, it)
	}
	return items
}

func (f *Fixer) Confirm(it FixItem) bool {
	if f == nil || !f.IsTTY {
		return false
	}
	cmd := strings.Join(append([]string{it.Rem.Command}, it.Rem.Args...), " ")
	prefix := ""
	if f.euid() != 0 {
		prefix = "sudo "
	}
	_, _ = fmt.Fprintf(f.out(), "Proposed fix for %s: %s%s\nExecute? [y/N] ", it.Result.Title, prefix, cmd)
	if f.Stdin == nil {
		return false
	}
	if f.br == nil {
		f.br = bufio.NewReader(f.Stdin)
	}
	line, err := f.br.ReadString('\n')
	if err != nil && len(line) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func (f *Fixer) Apply(ctx context.Context, items []FixItem) (fixed, failed, skipped int) {
	for _, it := range items {
		if !it.Allowed {
			_, _ = fmt.Fprintln(f.out(), "skipped: "+it.SkipReason)
			skipped++
			continue
		}
		if !f.Confirm(it) {
			_, _ = fmt.Fprintln(f.out(), "skipped.")
			skipped++
			continue
		}
		var argv []string
		if f.euid() == 0 {
			argv = append([]string{it.Rem.Command}, it.Rem.Args...)
		} else {
			argv = append([]string{"sudo", it.Rem.Command}, it.Rem.Args...)
		}
		name, rest := argv[0], argv[1:]
		var err error
		if f.Runner != nil {
			_, err = f.Runner.Run(ctx, name, rest...)
		} else {
			err = fmt.Errorf("no runner")
		}
		if err != nil {
			first := strings.SplitN(err.Error(), "\n", 2)[0]
			_, _ = fmt.Fprintf(f.out(), "✗ %s: %s\n", it.Result.Title, first)
			failed++
			continue
		}
		_, _ = fmt.Fprintf(f.out(), "✓ %s fixed\n", it.Result.Title)
		fixed++
	}
	return fixed, failed, skipped
}
