package scoring

import "github.com/RakhaYandra/ookami/internal/model"

func Score(rs []model.Result) (int, string) {
	score := 100
	hasCritical := false
	hasWarning := false
	for _, r := range rs {
		switch r.Severity {
		case model.SeverityCritical:
			score -= 25
			hasCritical = true
		case model.SeverityWarning:
			score -= 5
			hasWarning = true
		case model.SeverityUnknown:
			score -= 2
			hasWarning = true
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	switch {
	case hasCritical:
		return score, "critical"
	case hasWarning:
		return score, "warning"
	default:
		return score, "healthy"
	}
}
