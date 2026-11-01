package aoc

import "testing"

func TestCourses(t *testing.T) {
	p := New()
	courses := p.Courses()
	if len(courses) != len(Years) {
		t.Fatalf("got %d courses, want %d", len(courses), len(Years))
	}
	first := courses[0]
	if first.ID != "aoc-2024" || first.ProviderID != "aoc" {
		t.Fatalf("first course = %+v", first)
	}
	if len(first.Stages) != 25 {
		t.Fatalf("stages = %d", len(first.Stages))
	}
	if first.Stages[0].Number != 1 || first.Stages[24].Number != 25 {
		t.Fatalf("stage numbering = %d..%d", first.Stages[0].Number, first.Stages[24].Number)
	}
	if !first.Ready() {
		t.Fatal("course should be ready")
	}
	if first.Tester.Available {
		t.Fatal("no tester should be reported for aoc")
	}
	if first.Note == "" {
		t.Fatal("placeholder note should be set")
	}
}

func TestIdentity(t *testing.T) {
	p := New()
	if p.ID() != "aoc" || p.Planned() || p.Note() == "" {
		t.Fatalf("provider identity = %q planned=%v note=%q", p.ID(), p.Planned(), p.Note())
	}
}
