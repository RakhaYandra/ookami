package scoring

import "github.com/RakhaYandra/ookami/internal/model"

var Weights = map[model.Category]int{
	model.CategorySystem:      20,
	model.CategoryDevelopment: 30,
	model.CategoryStorage:     15,
	model.CategoryNetwork:     15,
	model.CategoryServices:    10,
	model.CategoryGPU:         10,
}

var CategoryOrder = []model.Category{
	model.CategorySystem,
	model.CategoryDevelopment,
	model.CategoryStorage,
	model.CategoryNetwork,
	model.CategoryServices,
	model.CategoryGPU,
}

type CategoryScore struct {
	Score  int
	Status string
}

func worstStatus(rs []model.Result) string {
	hasWarning := false
	for _, r := range rs {
		switch r.Severity {
		case model.SeverityCritical:
			return "critical"
		case model.SeverityWarning, model.SeverityUnknown:
			hasWarning = true
		}
	}
	if hasWarning {
		return "warning"
	}
	return "healthy"
}

func categoryScore(rs []model.Result) CategoryScore {
	score := 100
	for _, r := range rs {
		switch r.Severity {
		case model.SeverityCritical:
			score -= 25
		case model.SeverityWarning:
			score -= 5
		case model.SeverityUnknown:
			score -= 2
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return CategoryScore{Score: score, Status: worstStatus(rs)}
}

func activeCategories(rs []model.Result) []model.Category {
	present := map[model.Category]bool{}
	hasRealGPU := false
	for _, r := range rs {
		if r.Category == model.CategoryGPU {
			if r.ID != "gpu-none" {
				hasRealGPU = true
			}
			continue
		}
		present[r.Category] = true
	}
	out := []model.Category{}
	for _, c := range CategoryOrder {
		if c == model.CategoryGPU {
			if hasRealGPU {
				out = append(out, c)
			}
			continue
		}
		if present[c] {
			out = append(out, c)
		}
	}
	return out
}

func Breakdown(rs []model.Result) (int, string, map[model.Category]CategoryScore) {
	active := activeCategories(rs)
	if len(active) == 0 {
		return 100, "healthy", map[model.Category]CategoryScore{}
	}
	grouped := map[model.Category][]model.Result{}
	for _, r := range rs {
		if r.Category == model.CategoryGPU && r.ID == "gpu-none" {
			continue
		}
		grouped[r.Category] = append(grouped[r.Category], r)
	}
	cats := make(map[model.Category]CategoryScore, len(active))
	weighted, total := 0, 0
	for _, c := range active {
		cs := categoryScore(grouped[c])
		cats[c] = cs
		w := Weights[c]
		weighted += w * cs.Score
		total += w
	}
	if total == 0 {
		return 100, "healthy", cats
	}
	return weighted / total, worstStatus(rs), cats
}

func Score(rs []model.Result) (int, string) {
	g, s, _ := Breakdown(rs)
	return g, s
}
