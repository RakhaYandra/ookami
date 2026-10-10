package runner

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type OSRunner struct {
	Timeout time.Duration
}

func NewOSRunner(timeout time.Duration) *OSRunner {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &OSRunner{Timeout: timeout}
}

func (r *OSRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

type Call struct {
	Name string
	Args []string
}

type MockRunner struct {
	Handlers map[string]func(args []string) ([]byte, error)
	mu       sync.Mutex
	Calls    []Call
}

func (m *MockRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.Calls = append(m.Calls, Call{Name: name, Args: append([]string(nil), args...)})
	m.mu.Unlock()
	h := m.Handlers[name]
	if h == nil {
		return nil, exec.ErrNotFound
	}
	return h(args)
}
