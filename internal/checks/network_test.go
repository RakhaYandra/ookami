package checks

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/RakhaYandra/ookami/internal/config"
	"github.com/RakhaYandra/ookami/internal/model"
)

const netRouteSample = "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
	"wlo1\t00000000\t0100A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n" +
	"wlo1\t0000A8C0\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0\n"

const netRouteNoDefault = "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
	"wlo1\t0000A8C0\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0\n"

func passNetSuite() NetworkSuite {
	return NetworkSuite{
		Cfg:         config.Config{NetworkTimeoutSec: 5},
		Ifaces:      func() ([]net.Interface, error) { return []net.Interface{{Name: "wlo1", Flags: net.FlagUp}}, nil },
		RouteReader: func() (io.Reader, error) { return strings.NewReader(netRouteSample), nil },
		Resolve:     func(context.Context, string) error { return nil },
		Dial:        func(context.Context, string, string) error { return nil },
		Head:        func(string, time.Duration) (int, error) { return 200, nil },
	}
}

func checkIDs(rs []model.Result) []string {
	ids := make([]string, len(rs))
	for i, r := range rs {
		ids[i] = r.ID
	}
	return ids
}

func TestNetwork_Metadata(t *testing.T) {
	md := NetworkSuite{}.Metadata()
	if md.ID != "network-suite" || md.Name != "Network" || md.Category != model.CategoryNetwork || md.Optional {
		t.Fatalf("bad metadata: %+v", md)
	}
}

func TestNetwork_FullPass(t *testing.T) {
	rs := passNetSuite().Run(context.Background())
	if len(rs) != 6 {
		t.Fatalf("want 6 results, got %d", len(rs))
	}
	wantIDs := []string{"network-interface", "network-gateway", "network-dns", "network-internet", "network-github", "network-hub"}
	for i, id := range wantIDs {
		if rs[i].ID != id {
			t.Fatalf("result %d: want ID %s, got %s (%v)", i, id, rs[i].ID, checkIDs(rs))
		}
		if rs[i].Category != model.CategoryNetwork {
			t.Fatalf("result %d: wrong category %+v", i, rs[i])
		}
		if rs[i].Severity != model.SeverityPass {
			t.Fatalf("result %d: want pass, got %+v", i, rs[i])
		}
		if _, ok := rs[i].Details["latency_ms"]; !ok {
			t.Fatalf("result %d: missing latency_ms: %+v", i, rs[i])
		}
	}
	wantTitles := []string{"Interface", "Gateway", "DNS", "Internet", "GitHub", "Docker Hub"}
	for i, title := range wantTitles {
		if rs[i].Title != title {
			t.Fatalf("result %d: want title %q, got %q", i, title, rs[i].Title)
		}
	}
	if rs[0].Message != "wlo1 up" {
		t.Fatalf("iface msg: %q", rs[0].Message)
	}
	if rs[1].Message != "default route present" || rs[2].Message != "github.com resolves" ||
		rs[3].Message != "connected" || rs[4].Message != "reachable" || rs[5].Message != "reachable" {
		t.Fatalf("bad messages: %q %q %q %q %q", rs[1].Message, rs[2].Message, rs[3].Message, rs[4].Message, rs[5].Message)
	}
}

func TestNetwork_IfaceDown(t *testing.T) {
	s := passNetSuite()
	s.Ifaces = func() ([]net.Interface, error) {
		return []net.Interface{{Name: "lo", Flags: net.FlagUp | net.FlagLoopback}}, nil
	}
	rs := s.Run(context.Background())
	if len(rs) != 6 {
		t.Fatalf("want 6 results, got %d", len(rs))
	}
	if rs[0].Severity != model.SeverityCritical || rs[0].Message != "no active network interface" {
		t.Fatalf("got %+v", rs[0])
	}
	for i := 1; i < 6; i++ {
		if rs[i].Severity != model.SeverityInfo || rs[i].Message != "skipped (no interface)" {
			t.Fatalf("result %d: want skip, got %+v", i, rs[i])
		}
	}
}

func TestNetwork_IfaceError(t *testing.T) {
	s := passNetSuite()
	s.Ifaces = func() ([]net.Interface, error) { return nil, errors.New("boom") }
	rs := s.Run(context.Background())
	if rs[0].Severity != model.SeverityCritical {
		t.Fatalf("got %+v", rs[0])
	}
	for i := 1; i < 6; i++ {
		if rs[i].Severity != model.SeverityInfo {
			t.Fatalf("result %d: want info skip, got %+v", i, rs[i])
		}
	}
}

func TestNetwork_RouteMissing(t *testing.T) {
	s := passNetSuite()
	s.RouteReader = func() (io.Reader, error) { return strings.NewReader(netRouteNoDefault), nil }
	rs := s.Run(context.Background())
	if rs[0].Severity != model.SeverityPass {
		t.Fatalf("iface should pass: %+v", rs[0])
	}
	if rs[1].Severity != model.SeverityCritical || rs[1].Message != "no default route" {
		t.Fatalf("got %+v", rs[1])
	}
	for i := 2; i < 6; i++ {
		if rs[i].Severity != model.SeverityInfo || rs[i].Message != "skipped (no gateway)" {
			t.Fatalf("result %d: want skip, got %+v", i, rs[i])
		}
	}
}

func TestNetwork_DNSFail(t *testing.T) {
	s := passNetSuite()
	s.Resolve = func(context.Context, string) error { return errors.New("no such host") }
	rs := s.Run(context.Background())
	if rs[2].Severity != model.SeverityCritical || rs[2].Message != "DNS resolution failed" {
		t.Fatalf("got %+v", rs[2])
	}
	for i := 3; i < 6; i++ {
		if rs[i].Severity != model.SeverityInfo || rs[i].Message != "skipped (no dns)" {
			t.Fatalf("result %d: want skip, got %+v", i, rs[i])
		}
	}
}

func TestNetwork_DialFail(t *testing.T) {
	s := passNetSuite()
	s.Dial = func(context.Context, string, string) error { return errors.New("timeout") }
	rs := s.Run(context.Background())
	if rs[3].Severity != model.SeverityWarning || rs[3].Message != "no internet connectivity" {
		t.Fatalf("got %+v", rs[3])
	}
	for i := 4; i < 6; i++ {
		if rs[i].Severity != model.SeverityInfo || rs[i].Message != "skipped (no internet)" {
			t.Fatalf("result %d: want skip, got %+v", i, rs[i])
		}
	}
}

func TestNetwork_GitHub500(t *testing.T) {
	s := passNetSuite()
	s.Head = func(url string, _ time.Duration) (int, error) {
		if strings.Contains(url, "github.com") {
			return 500, nil
		}
		return 200, nil
	}
	rs := s.Run(context.Background())
	if rs[4].Severity != model.SeverityWarning || rs[4].Message != "GitHub unreachable" {
		t.Fatalf("got %+v", rs[4])
	}
	if rs[5].Severity != model.SeverityInfo || rs[5].Message != "skipped (github unreachable)" {
		t.Fatalf("hub should skip: %+v", rs[5])
	}
}

func TestNetwork_HubTimeout(t *testing.T) {
	s := passNetSuite()
	s.Head = func(url string, _ time.Duration) (int, error) {
		if strings.Contains(url, "docker.io") {
			return 0, errors.New("timeout")
		}
		return 200, nil
	}
	rs := s.Run(context.Background())
	for i := 0; i < 5; i++ {
		if rs[i].Severity != model.SeverityPass {
			t.Fatalf("result %d: want pass, got %+v", i, rs[i])
		}
	}
	if rs[5].Severity != model.SeverityWarning || rs[5].Message != "Docker Hub unreachable" {
		t.Fatalf("got %+v", rs[5])
	}
}

func TestNetwork_Hub401Reachable(t *testing.T) {
	s := passNetSuite()
	s.Head = func(url string, _ time.Duration) (int, error) {
		if strings.Contains(url, "docker.io") {
			return 401, nil
		}
		return 200, nil
	}
	rs := s.Run(context.Background())
	if rs[5].Severity != model.SeverityPass || rs[5].Message != "reachable" {
		t.Fatalf("401 must count as reachable: %+v", rs[5])
	}
}

func TestHasDefaultRoute_Present(t *testing.T) {
	if !hasDefaultRoute(strings.NewReader(netRouteSample)) {
		t.Fatal("sample should have default route")
	}
}

func TestHasDefaultRoute_Missing(t *testing.T) {
	if hasDefaultRoute(strings.NewReader(netRouteNoDefault)) {
		t.Fatal("should report no default route")
	}
	if hasDefaultRoute(strings.NewReader("Iface\tDestination\tGateway\tFlags\n")) {
		t.Fatal("header-only should be false")
	}
	if hasDefaultRoute(nil) {
		t.Fatal("nil reader should be false")
	}
}
