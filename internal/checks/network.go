package checks

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/model"
)

var _ model.Check = NetworkSuite{}

// NetworkSuite cascades 6 network layers, stopping at the first failure.
type NetworkSuite struct {
	Cfg         config.Config
	Ifaces      func() ([]net.Interface, error)
	RouteReader func() (io.Reader, error)
	Resolve     func(ctx context.Context, host string) error
	Dial        func(ctx context.Context, network, addr string) error
	Head        func(url string, timeout time.Duration) (statusCode int, err error)
}

func (NetworkSuite) Metadata() model.CheckMetadata {
	return model.CheckMetadata{ID: "network-suite", Name: "Network",
		Description: "network connectivity cascade", Category: model.CategoryNetwork, Optional: false}
}

// hasDefaultRoute reports whether /proc/net/route content has a default route.
// Skips the header line; matches Destination=="00000000" with RTF_GATEWAY (0x2).
func hasDefaultRoute(r io.Reader) bool {
	if r == nil {
		return false
	}
	sc := bufio.NewScanner(r)
	first := true
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		f := strings.Fields(sc.Text())
		if len(f) < 4 || f[1] != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(f[3], 16, 32)
		if err != nil {
			continue
		}
		if flags&0x2 != 0 {
			return true
		}
	}
	return false
}

func netSkip(id, title, reason string) model.Result {
	return model.Result{ID: id, Category: model.CategoryNetwork,
		Severity: model.SeverityInfo, Title: title, Message: "skipped (" + reason + ")"}
}

func (s NetworkSuite) Run(ctx context.Context) []model.Result {
	timeout := time.Duration(s.Cfg.NetworkTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	ifaces := s.Ifaces
	if ifaces == nil {
		ifaces = net.Interfaces
	}
	routeReader := s.RouteReader
	if routeReader == nil {
		routeReader = func() (io.Reader, error) { return os.Open("/proc/net/route") }
	}
	resolve := s.Resolve
	if resolve == nil {
		resolve = func(ctx context.Context, host string) error {
			addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return err
			}
			if len(addrs) == 0 {
				return errors.New("no addresses for " + host)
			}
			return nil
		}
	}
	dial := s.Dial
	if dial == nil {
		dial = func(ctx context.Context, network, addr string) error {
			var d net.Dialer
			conn, err := d.DialContext(ctx, network, addr)
			if err != nil {
				return err
			}
			return conn.Close()
		}
	}
	head := s.Head
	if head == nil {
		head = func(url string, timeout time.Duration) (int, error) {
			c := &http.Client{Timeout: timeout}
			req, err := http.NewRequest("HEAD", url, nil)
			if err != nil {
				return 0, err
			}
			resp, err := c.Do(req)
			if err != nil {
				return 0, err
			}
			defer func() { _ = resp.Body.Close() }()
			return resp.StatusCode, nil
		}
	}

	ids := []string{"network-interface", "network-gateway", "network-dns",
		"network-internet", "network-github", "network-hub"}
	titles := []string{"Interface", "Gateway", "DNS", "Internet", "GitHub", "Docker Hub"}
	skipReasons := []string{"no interface", "no gateway", "no dns", "no internet", "github unreachable"}
	fail := func(idx int, sev model.Severity, msg string, start time.Time) model.Result {
		return model.Result{ID: ids[idx], Category: model.CategoryNetwork,
			Severity: sev, Title: titles[idx], Message: msg,
			Details: map[string]any{"latency_ms": time.Since(start).Milliseconds()}}
	}
	skipRest := func(from, failedAt int, out []model.Result) []model.Result {
		for i := from; i < 6; i++ {
			out = append(out, netSkip(ids[i], titles[i], skipReasons[failedAt]))
		}
		return out
	}
	out := make([]model.Result, 0, 6)

	// 1. interface: first Up && !Loopback.
	start := time.Now()
	ifaceName := ""
	if list, err := ifaces(); err == nil {
		for _, iface := range list {
			if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
				ifaceName = iface.Name
				break
			}
		}
	}
	if ifaceName == "" {
		out = append(out, fail(0, model.SeverityCritical, "no active network interface", start))
		return skipRest(1, 0, out)
	}
	out = append(out, model.Result{ID: ids[0], Category: model.CategoryNetwork,
		Severity: model.SeverityPass, Title: titles[0], Message: ifaceName + " up",
		Details: map[string]any{"latency_ms": time.Since(start).Milliseconds()}})

	// 2. gateway.
	start = time.Now()
	gwOK := false
	if rr, err := routeReader(); err == nil && rr != nil {
		gwOK = hasDefaultRoute(rr)
		if c, ok := rr.(io.Closer); ok {
			_ = c.Close()
		}
	}
	if !gwOK {
		out = append(out, fail(1, model.SeverityCritical, "no default route", start))
		return skipRest(2, 1, out)
	}
	out = append(out, model.Result{ID: ids[1], Category: model.CategoryNetwork,
		Severity: model.SeverityPass, Title: titles[1], Message: "default route present",
		Details: map[string]any{"latency_ms": time.Since(start).Milliseconds()}})

	// 3. DNS.
	start = time.Now()
	func() {
		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := resolve(cctx, "github.com"); err != nil {
			out = append(out, fail(2, model.SeverityCritical, "DNS resolution failed", start))
		} else {
			out = append(out, model.Result{ID: ids[2], Category: model.CategoryNetwork,
				Severity: model.SeverityPass, Title: titles[2], Message: "github.com resolves",
				Details: map[string]any{"latency_ms": time.Since(start).Milliseconds()}})
		}
	}()
	if out[len(out)-1].Severity != model.SeverityPass {
		return skipRest(3, 2, out)
	}

	// 4. internet dial.
	start = time.Now()
	func() {
		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := dial(cctx, "tcp", "1.1.1.1:443"); err != nil {
			out = append(out, fail(3, model.SeverityWarning, "no internet connectivity", start))
		} else {
			out = append(out, model.Result{ID: ids[3], Category: model.CategoryNetwork,
				Severity: model.SeverityPass, Title: titles[3], Message: "connected",
				Details: map[string]any{"latency_ms": time.Since(start).Milliseconds()}})
		}
	}()
	if out[len(out)-1].Severity != model.SeverityPass {
		return skipRest(4, 3, out)
	}

	// 5. GitHub HEAD (2xx/3xx).
	start = time.Now()
	code, err := head("https://github.com", timeout)
	if err != nil || code < 200 || code >= 400 {
		out = append(out, fail(4, model.SeverityWarning, "GitHub unreachable", start))
		return skipRest(5, 4, out)
	}
	out = append(out, model.Result{ID: ids[4], Category: model.CategoryNetwork,
		Severity: model.SeverityPass, Title: titles[4], Message: "reachable",
		Details: map[string]any{"latency_ms": time.Since(start).Milliseconds()}})

	// 6. Docker Hub HEAD (2xx/3xx/401; registry returns 401 without auth).
	start = time.Now()
	code, err = head("https://registry-1.docker.io/v2/", timeout)
	if err != nil || ((code < 200 || code >= 400) && code != 401) {
		out = append(out, fail(5, model.SeverityWarning, "Docker Hub unreachable", start))
		return out
	}
	out = append(out, model.Result{ID: ids[5], Category: model.CategoryNetwork,
		Severity: model.SeverityPass, Title: titles[5], Message: "reachable",
		Details: map[string]any{"latency_ms": time.Since(start).Milliseconds()}})
	return out
}
