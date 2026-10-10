package scoring

import (
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
)

func TestScore(t *testing.T) {
	s, st := Score(nil)
	if s != 100 || st != "healthy" {
		t.Fatalf("empty = (%d,%s), want (100,healthy)", s, st)
	}
	s, st = Score([]model.Result{{Severity: model.SeverityWarning}})
	if s != 95 || st != "warning" {
		t.Fatalf("warning = (%d,%s), want (95,warning)", s, st)
	}
	s, st = Score([]model.Result{{Severity: model.SeverityCritical}})
	if s != 75 || st != "critical" {
		t.Fatalf("critical = (%d,%s), want (75,critical)", s, st)
	}
}

func TestScoreDeterministic(t *testing.T) {
	rs := []model.Result{{Severity: model.SeverityWarning}, {Severity: model.SeverityCritical}}
	a1, s1 := Score(rs)
	a2, s2 := Score(rs)
	if a1 != a2 || s1 != s2 {
		t.Fatalf("nondeterministic: (%d,%s) vs (%d,%s)", a1, s1, a2, s2)
	}
}
