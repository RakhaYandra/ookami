package registry

import (
	"context"
	"testing"

	"github.com/RakhaYandra/ookami/internal/model"
)

type fakeCheck struct {
	id  string
	cat model.Category
}

func (f fakeCheck) ID() string                        { return f.id }
func (f fakeCheck) Category() model.Category          { return f.cat }
func (f fakeCheck) Metadata() model.CheckMetadata     { return model.CheckMetadata{ID: f.id, Category: f.cat} }
func (f fakeCheck) Run(ctx context.Context) model.Result { return model.Result{ID: f.id} }

func TestRegisterOrderedByCategory(t *testing.T) {
	Clear()
	Register(fakeCheck{id: "a", cat: model.CategorySystem})
	Register(fakeCheck{id: "b", cat: model.CategoryNetwork})

	got := Ordered()
	if len(got) != 2 {
		t.Fatalf("Ordered len = %d, want 2", len(got))
	}
	if got[0].ID() != "a" || got[1].ID() != "b" {
		t.Fatalf("Ordered order = %v, want [a b]", []string{got[0].ID(), got[1].ID()})
	}

	sys := ByCategory(model.CategorySystem)
	if len(sys) != 1 || sys[0].ID() != "a" {
		t.Fatalf("ByCategory system = %v, want [a]", sys)
	}

	Clear()
	if len(Ordered()) != 0 {
		t.Fatalf("Ordered after Clear != 0")
	}
}
