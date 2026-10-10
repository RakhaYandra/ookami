package scoring

import (
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
)

func TestScoreEmpty(t *testing.T) {
	s, st := Score(nil)
	if s != 100 || st != "healthy" {
		t.Fatalf("empty = (%d,%s), want (100,healthy)", s, st)
	}
	g, gst, cats := Breakdown(nil)
	if g != 100 || gst != "healthy" || len(cats) != 0 {
		t.Fatalf("breakdown empty = (%d,%s,%v), want (100,healthy,empty)", g, gst, cats)
	}
}

// Kasus flat lama: dulu 1 warning (tanpa kategori) -> 95.
// Sekarang single-category: 1 warning di system -> cat 95,
// global = 95 karena satu-satunya kategori aktif (bobot ternormalisasi).
func TestScoreSingleWarningSystem(t *testing.T) {
	rs := []model.Result{{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityWarning}}
	s, st := Score(rs)
	if s != 95 || st != "warning" {
		t.Fatalf("single warning system = (%d,%s), want (95,warning)", s, st)
	}
	_, _, cats := Breakdown(rs)
	if cats[model.CategorySystem].Score != 95 || cats[model.CategorySystem].Status != "warning" {
		t.Fatalf("cat system = %+v, want {95 warning}", cats[model.CategorySystem])
	}
}

func TestScoreSingleCriticalSystem(t *testing.T) {
	rs := []model.Result{{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityCritical}}
	s, st := Score(rs)
	if s != 75 || st != "critical" {
		t.Fatalf("single critical system = (%d,%s), want (75,critical)", s, st)
	}
}

func TestScoreUnknownPenaltyAndStatus(t *testing.T) {
	rs := []model.Result{{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityUnknown}}
	s, st := Score(rs)
	if s != 98 || st != "warning" {
		t.Fatalf("single unknown = (%d,%s), want (98,warning)", s, st)
	}
}

func TestScorePassInfoNoEffect(t *testing.T) {
	rs := []model.Result{
		{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityPass},
		{ID: "sys-2", Category: model.CategorySystem, Severity: model.SeverityInfo},
	}
	s, st := Score(rs)
	if s != 100 || st != "healthy" {
		t.Fatalf("pass+info = (%d,%s), want (100,healthy)", s, st)
	}
}

func TestCategoryScoreMixedPenalties(t *testing.T) {
	rs := []model.Result{
		{Category: model.CategoryStorage, Severity: model.SeverityCritical},
		{Category: model.CategoryStorage, Severity: model.SeverityWarning},
		{Category: model.CategoryStorage, Severity: model.SeverityUnknown},
	}
	cs := categoryScore(rs)
	if cs.Score != 68 || cs.Status != "critical" {
		t.Fatalf("mixed = %+v, want {68 critical}", cs)
	}
}

func TestCategoryScoreClampZero(t *testing.T) {
	rs := make([]model.Result, 5)
	for i := range rs {
		rs[i] = model.Result{ID: "x", Category: model.CategoryNetwork, Severity: model.SeverityCritical}
	}
	cs := categoryScore(rs)
	if cs.Score != 0 || cs.Status != "critical" {
		t.Fatalf("clamp = %+v, want {0 critical}", cs)
	}
	s, _ := Score(rs)
	if s != 0 {
		t.Fatalf("global clamp = %d, want 0", s)
	}
}

func TestBreakdownMultiCategoryWeighted99(t *testing.T) {
	rs := []model.Result{
		{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityPass},
		{ID: "dev-1", Category: model.CategoryDevelopment, Severity: model.SeverityPass},
		{ID: "sto-1", Category: model.CategoryStorage, Severity: model.SeverityPass},
		{ID: "net-1", Category: model.CategoryNetwork, Severity: model.SeverityPass},
		{ID: "svc-1", Category: model.CategoryServices, Severity: model.SeverityWarning},
		{ID: "gpu-1", Category: model.CategoryGPU, Severity: model.SeverityPass},
	}
	g, st, cats := Breakdown(rs)
	if g != 99 {
		t.Fatalf("weighted = %d, want 99 (services 95, lain 100)", g)
	}
	if st != "warning" {
		t.Fatalf("status = %s, want warning", st)
	}
	if cats[model.CategoryServices].Score != 95 {
		t.Fatalf("services cat = %+v, want score 95", cats[model.CategoryServices])
	}
}

func TestBreakdownRenormalizeSingleCategory(t *testing.T) {
	rs := []model.Result{{ID: "net-1", Category: model.CategoryNetwork, Severity: model.SeverityPass}}
	g, st, cats := Breakdown(rs)
	if g != 100 || st != "healthy" {
		t.Fatalf("renorm = (%d,%s), want (100,healthy)", g, st)
	}
	if len(cats) != 1 || cats[model.CategoryNetwork].Score != 100 {
		t.Fatalf("cats = %v, want single network 100", cats)
	}
}

func TestActiveCategoriesOrderStable(t *testing.T) {
	rs := []model.Result{
		{ID: "gpu-1", Category: model.CategoryGPU, Severity: model.SeverityPass},
		{ID: "svc-1", Category: model.CategoryServices, Severity: model.SeverityPass},
		{ID: "net-1", Category: model.CategoryNetwork, Severity: model.SeverityPass},
		{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityPass},
	}
	got := activeCategories(rs)
	want := []model.Category{model.CategorySystem, model.CategoryNetwork, model.CategoryGPU, model.CategoryServices}
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestGpuNoneAloneExcluded(t *testing.T) {
	rs := []model.Result{{ID: "gpu-none", Category: model.CategoryGPU, Severity: model.SeverityInfo}}
	if got := activeCategories(rs); len(got) != 0 {
		t.Fatalf("active = %v, want empty", got)
	}
	g, st, cats := Breakdown(rs)
	if g != 100 || st != "healthy" || len(cats) != 0 {
		t.Fatalf("gpu-none alone = (%d,%s,%v), want (100,healthy,empty)", g, st, cats)
	}
}

func TestGpuNoneExcludedWithOthers(t *testing.T) {
	rs := []model.Result{
		{ID: "gpu-none", Category: model.CategoryGPU, Severity: model.SeverityInfo},
		{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityPass},
	}
	got := activeCategories(rs)
	if len(got) != 1 || got[0] != model.CategorySystem {
		t.Fatalf("active = %v, want [system]", got)
	}
	gs, gst, _ := Breakdown(rs)
	if gs != 100 || gst != "healthy" {
		t.Fatalf("global = (%d,%s), want (100,healthy)", gs, gst)
	}
}

func TestGpuRealIncluded(t *testing.T) {
	rs := []model.Result{
		{ID: "gpu-driver", Category: model.CategoryGPU, Severity: model.SeverityPass},
		{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityPass},
	}
	got := activeCategories(rs)
	if len(got) != 2 || got[0] != model.CategorySystem || got[1] != model.CategoryGPU {
		t.Fatalf("active = %v, want [system gpu]", got)
	}
}

func TestBreakdownStatusesPerCategory(t *testing.T) {
	rs := []model.Result{
		{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityCritical},
		{ID: "net-1", Category: model.CategoryNetwork, Severity: model.SeverityWarning},
		{ID: "sto-1", Category: model.CategoryStorage, Severity: model.SeverityUnknown},
		{ID: "dev-1", Category: model.CategoryDevelopment, Severity: model.SeverityPass},
	}
	_, _, cats := Breakdown(rs)
	if cats[model.CategorySystem].Status != "critical" {
		t.Fatalf("system status = %s, want critical", cats[model.CategorySystem].Status)
	}
	if cats[model.CategoryNetwork].Status != "warning" {
		t.Fatalf("network status = %s, want warning", cats[model.CategoryNetwork].Status)
	}
	if cats[model.CategoryStorage].Status != "warning" {
		t.Fatalf("storage(unknown) status = %s, want warning", cats[model.CategoryStorage].Status)
	}
	if cats[model.CategoryDevelopment].Status != "healthy" {
		t.Fatalf("dev status = %s, want healthy", cats[model.CategoryDevelopment].Status)
	}
}

func TestScoreGlobalStatusWorstSeverity(t *testing.T) {
	rs := []model.Result{
		{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityPass},
		{ID: "net-1", Category: model.CategoryNetwork, Severity: model.SeverityWarning},
		{ID: "sto-1", Category: model.CategoryStorage, Severity: model.SeverityCritical},
	}
	_, st := Score(rs)
	if st != "critical" {
		t.Fatalf("status = %s, want critical", st)
	}
}

func TestScoreDeterministic(t *testing.T) {
	rs := []model.Result{
		{ID: "sys-1", Category: model.CategorySystem, Severity: model.SeverityWarning},
		{ID: "net-1", Category: model.CategoryNetwork, Severity: model.SeverityCritical},
		{ID: "sto-1", Category: model.CategoryStorage, Severity: model.SeverityPass},
	}
	a1, s1 := Score(rs)
	a2, s2 := Score(rs)
	if a1 != a2 || s1 != s2 {
		t.Fatalf("nondeterministic: (%d,%s) vs (%d,%s)", a1, s1, a2, s2)
	}
	g1, _, c1 := Breakdown(rs)
	g2, _, c2 := Breakdown(rs)
	if g1 != g2 || len(c1) != len(c2) {
		t.Fatalf("breakdown nondeterministic: %d vs %d", g1, g2)
	}
	for k, v := range c1 {
		if c2[k] != v {
			t.Fatalf("breakdown cats differ at %s: %+v vs %+v", k, v, c2[k])
		}
	}
}
