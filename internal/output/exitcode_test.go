package output

import (
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
)

func TestExitCode(t *testing.T) {
	cases := []struct {
		name string
		rs   []model.Result
		exec bool
		inv  bool
		want int
	}{
		{"empty", nil, false, false, 0},
		{"warning", []model.Result{{Severity: model.SeverityWarning}}, false, false, 1},
		{"unknown", []model.Result{{Severity: model.SeverityUnknown}}, false, false, 1},
		{"critical", []model.Result{{Severity: model.SeverityCritical}}, false, false, 2},
		{"execErr overrides", []model.Result{{Severity: model.SeverityCritical}}, true, false, 3},
		{"invalidArgs overrides", []model.Result{{Severity: model.SeverityCritical}}, true, true, 4},
	}
	for _, c := range cases {
		if got := ExitCode(c.rs, c.exec, c.inv); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}
