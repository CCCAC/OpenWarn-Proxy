package main

import "testing"

func TestPolygonInArea(t *testing.T) {
	t.Fatal("Build me!")
}

func TestAreaFromString(t *testing.T) {
	a, err := NewAreaFromString("")
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(a.Coordinates) != 0 {
		t.Error("Unexpected coordinates:", a.Coordinates)
	}

	a, err = NewAreaFromString("1,2 3,4 1.5,3")
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(a.Coordinates) != 3 {
		t.Errorf("Unexpected number of coordinates: want 3, have %d", len(a.Coordinates))
	}
	if a.IsDegenerate() {
		t.Error("Area is degenerate!")
	}
}
