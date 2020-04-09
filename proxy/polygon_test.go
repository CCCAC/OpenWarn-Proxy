package proxy

import "testing"

const (
	_testArea1 = `13.09,50.783 13.109,50.79 13.139,50.77 13.17,50.784 13.196,50.784 13.242,50.772 13.277,50.763 13.284,50.74 13.333,50.746 13.358,50.753 13.364,50.753 13.378,50.735 13.37,50.716 13.406,50.685 13.433,50.676 13.431,50.665 13.485,50.65 13.485,50.641 13.501,50.633 13.464,50.603 13.428,50.611 13.425,50.616 13.42,50.615 13.391,50.645 13.373,50.644 13.369,50.618 13.321,50.601 13.325,50.583 13.291,50.575 13.279,50.593 13.256,50.595 13.235,50.578 13.229,50.554 13.195,50.503 13.175,50.504 13.137,50.506 13.127,50.517 13.103,50.503 13.061,50.501 13.044,50.511 13.033,50.509 13.022,50.477 13.024,50.451 12.966,50.415 12.936,50.411 12.894,50.43 12.836,50.454 12.807,50.442 12.795,50.449 12.754,50.438 12.696,50.401 12.666,50.414 12.628,50.415 12.583,50.407 12.584,50.424 12.533,50.445 12.494,50.469 12.46,50.497 12.467,50.514 12.471,50.518 12.477,50.523 12.512,50.53 12.529,50.546 12.548,50.561 12.582,50.552 12.613,50.565 12.584,50.575 12.589,50.609 12.631,50.62 12.64,50.636 12.648,50.64 12.687,50.629 12.698,50.636 12.711,50.643 12.708,50.666 12.718,50.687 12.705,50.7 12.683,50.7 12.653,50.71 12.642,50.722 12.666,50.732 12.65,50.754 12.686,50.755 12.695,50.739 12.714,50.738 12.727,50.752 12.713,50.77 12.74,50.78 12.755,50.771 12.778,50.789 12.796,50.785 12.817,50.789 12.84,50.801 12.852,50.797 12.889,50.784 12.895,50.754 12.906,50.748 12.938,50.744 12.953,50.759 12.967,50.755 13.005,50.771 13.019,50.771 13.047,50.807 13.072,50.785 13.09,50.783`
	_testArea2 = `-1,-1 1,-1 1,1 -1,1 -1,-1` // Square
)

type testCase struct {
	expected bool
	c        Coordinate
}

func TestPolygonInArea(t *testing.T) {
	a, err := NewAreaFromString(_testArea1)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if a.Contains(Coordinate{0.0, 0.0}) {
		t.Error("a should not contain (0.0, 0.0) but it does")
	}

	a, err = NewAreaFromString(_testArea2)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	testCases := []testCase{
		{true, Coordinate{0, 0}},
		{false, Coordinate{0, -3}},
		{true, Coordinate{-1, -1}},
		{true, Coordinate{0.3, 0.3}},
	}

	for _, testCase := range testCases {
		is := a.Contains(testCase.c)
		if is != testCase.expected {
			t.Error("expected:", testCase.c, "=", testCase.expected, "got:", is)
		}
	}
}

func TestAreaFromString(t *testing.T) {
	a, err := NewAreaFromString("")
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(a.Segments) != 0 {
		t.Error("Unexpected line segments:", a.Segments)
	}

	a, err = NewAreaFromString("1,2 3,4 1.5,3 1,2")
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(a.Segments) != 3 {
		t.Errorf("Unexpected number of coordinates: want 3, have %d", len(a.Segments))
	}

	_, err = NewAreaFromString(_testArea1)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
}
