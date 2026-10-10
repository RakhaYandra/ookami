package output

import "github.com/RakhaYandra/ookami/internal/model"

func ExitCode(rs []model.Result, execErr bool, invalidArgs bool) int {
	if invalidArgs {
		return 4
	}
	if execErr {
		return 3
	}
	for _, r := range rs {
		if r.Severity == model.SeverityCritical {
			return 2
		}
	}
	for _, r := range rs {
		if r.Severity == model.SeverityWarning || r.Severity == model.SeverityUnknown {
			return 1
		}
	}
	return 0
}
