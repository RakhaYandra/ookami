package registry

import "github.com/RakhaYandra/ookami/internal/model"

var checks []model.Check

func Register(c model.Check) { checks = append(checks, c) }

func Ordered() []model.Check {
	out := make([]model.Check, len(checks))
	copy(out, checks)
	return out
}

func ByCategory(cat model.Category) []model.Check {
	var r []model.Check
	for _, c := range checks {
		if c.Metadata().Category == cat {
			r = append(r, c)
		}
	}
	return r
}

func Clear() { checks = nil }
