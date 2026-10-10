package runner

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMockRunner(t *testing.T) {
	m := &MockRunner{
		Handlers: map[string]func(args []string) ([]byte, error){
			"git": func(args []string) ([]byte, error) {
				return []byte("git version 2.51.0"), nil
			},
		},
	}
	out, err := m.Run(context.Background(), "git", "--version")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(string(out), "git version 2.51.0") {
		t.Fatalf("Run() = %q, want contains %q", out, "git version 2.51.0")
	}
	if len(m.Calls) != 1 || m.Calls[0].Name != "git" {
		t.Fatalf("Calls = %v, want [{git}]", m.Calls)
	}
	if len(m.Calls[0].Args) != 1 || m.Calls[0].Args[0] != "--version" {
		t.Fatalf("Calls[0].Args = %v, want [--version]", m.Calls[0].Args)
	}
}

func TestMockRunnerNotFound(t *testing.T) {
	m := &MockRunner{}
	if _, err := m.Run(context.Background(), "git", "--version"); err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
}

func TestOSRunnerTimeout(t *testing.T) {
	r := NewOSRunner(50 * time.Millisecond)
	ctx := context.Background()
	_, err := r.Run(ctx, "sleep", "2")
	if err == nil {
		t.Fatal("Run() error = nil, want timeout/deadline")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "killed") {
		t.Fatalf("Run() error = %v, want timeout/deadline", err)
	}
}

func TestOSRunnerEcho(t *testing.T) {
	r := NewOSRunner(5 * time.Second)
	out, err := r.Run(context.Background(), "echo", "hi")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.TrimSpace(string(out)) != "hi" {
		t.Fatalf("Run() = %q, want %q", out, "hi")
	}
}

func TestMockRunnerParallel(t *testing.T) {
	m := &MockRunner{
		Handlers: map[string]func(args []string) ([]byte, error){
			"git": func(args []string) ([]byte, error) { return []byte("ok"), nil },
		},
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := m.Run(context.Background(), "git", "--version"); err != nil {
				t.Errorf("Run() error = %v", err)
			}
		}()
	}
	wg.Wait()
	if len(m.Calls) != 10 {
		t.Fatalf("len(Calls) = %d, want 10", len(m.Calls))
	}
}
