package navtree

import (
	"testing"

	"github.com/shrsv/AgentLaws/internal/model"
)

func TestBuild_Nesting(t *testing.T) {
	sections := []model.Section{
		{ID: "a", Level: 1, Title: "A"},
		{ID: "a.1", Level: 2, Title: "A.1"},
		{ID: "a.2", Level: 2, Title: "A.2"},
		{ID: "a.2.1", Level: 3, Title: "A.2.1"},
		{ID: "b", Level: 1, Title: "B"},
	}

	roots := Build(sections)
	if len(roots) != 2 {
		t.Fatalf("roots: got %d, want 2", len(roots))
	}

	a := roots[0]
	if a.Section.ID != "a" || len(a.Children) != 2 {
		t.Fatalf("root[0]: got ID=%s children=%d, want a/2", a.Section.ID, len(a.Children))
	}
	if a.Children[0].Section.ID != "a.1" || len(a.Children[0].Children) != 0 {
		t.Errorf("a.Children[0]: got %+v", a.Children[0].Section)
	}
	a2 := a.Children[1]
	if a2.Section.ID != "a.2" || len(a2.Children) != 1 {
		t.Fatalf("a.Children[1]: got ID=%s children=%d, want a.2/1", a2.Section.ID, len(a2.Children))
	}
	if a2.Children[0].Section.ID != "a.2.1" {
		t.Errorf("a.2.Children[0]: got %s, want a.2.1", a2.Children[0].Section.ID)
	}

	b := roots[1]
	if b.Section.ID != "b" || len(b.Children) != 0 {
		t.Fatalf("root[1]: got ID=%s children=%d, want b/0", b.Section.ID, len(b.Children))
	}
}
