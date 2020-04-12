package proxy

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// This file contains the code for handling arbitrary lat/lon polygons.

type InvalidCoordinateError struct {
	s string
	v []string
}

func (e InvalidCoordinateError) Error() string {
	return fmt.Sprintf("Invalid coordinate string '%s', splits into %#v", e.s, e.v)
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// NewCoordinateFromString returns a Coordinate extracted from s. s has the following layout:
//
//    s := "7.8,50.1" // Longitude, Latitude
//
// This layout is somewhat unusual, since usually the latitude comes first.
func NewCoordinateFromString(s string) (Location, error) {
	var c Location
	v := strings.Split(strings.TrimSpace(s), ",")
	if len(v) != 2 {
		return c, InvalidCoordinateError{s, v}
	}

	var err error
	c.Longitude, err = strconv.ParseFloat(v[0], 64)
	if err != nil {
		return c, fmt.Errorf("parsing latitude: %w", err)
	}
	c.Latitude, err = strconv.ParseFloat(v[1], 64)
	if err != nil {
		return c, fmt.Errorf("parsing longitude: %w", err)
	}

	return c, nil
}

func (c Location) String() string {
	return fmt.Sprintf("[Lat:% 3.3f, Lon:% 3.3f]", c.Latitude, c.Longitude)
}

type LineSegment struct {
	P1, P2 Location
}

func (l LineSegment) String() string {
	return fmt.Sprintf("[%s->%s]", l.P1, l.P2)
}

type Area struct {
	Segments []LineSegment
}

// NewAreaFromString loads coordinates from the string representation and returns an area. The returned area may not
// be a single polygon and it may not be convex.
func NewAreaFromString(coords string) (Area, error) {
	var a Area

	coords = strings.TrimSpace(coords)

	if coords == "" {
		// No coordinates
		return a, nil
	}

	chunks := strings.Split(coords, " ")
	if len(chunks) == 1 {
		return a, errors.New("malformed polygon")
	}
	if chunks[0] != chunks[len(chunks)-1] {
		// Polygon is not closed. Do that manually.
		chunks = append(chunks, chunks[0])
	}

	// Construct line segments of the polygon
	lastCoord := chunks[0]

	for _, chunk := range chunks[1:] {
		p1, err := NewCoordinateFromString(lastCoord)
		if err != nil {
			return a, err
		}
		p2, err := NewCoordinateFromString(chunk)
		if err != nil {
			return a, err
		}
		seg := LineSegment{P1: p1, P2: p2}
		a.Segments = append(a.Segments, seg)
		lastCoord = chunk
	}

	return a, nil
}

// Contains returns true if c is inside the polygon described by a
func (a Area) Contains(c Location) bool {
	// Cast a ray from c to the right. Count crossings with line segments of a. If the number of crossings is even, c is outside
	// of a.

	intersections := 0

	for _, seg := range a.Segments {
		// Check if a is to the left of the rightmost part of seg and between the end points
		// Check latitudes (Y coords)
		minLat := math.Min(seg.P1.Latitude, seg.P2.Latitude)
		maxLat := math.Max(seg.P1.Latitude, seg.P2.Latitude)

		if c.Latitude < minLat || c.Latitude > maxLat {
			// Above or below line segment
			continue
		}

		// Calculate x coordinate of the intersection of a line through seg.p1 and sec.p2 (l1) and a line to the right through c (l2)
		if seg.P1.Longitude == seg.P2.Longitude {
			// Deal with seg.p1 and seg.p2 on one vertical line
			if c.Longitude <= seg.P1.Longitude {
				intersections++
			}
			continue
		}

		// Get slope of l1
		// Determine which of p1, p2 is leftmost
		leftP := seg.P1
		rightP := seg.P2
		if leftP.Longitude > rightP.Longitude {
			leftP = seg.P2
			rightP = seg.P1
		}
		// Calculate slope
		slope := (rightP.Latitude - leftP.Latitude) / (rightP.Longitude - leftP.Longitude)
		if slope == 0 {
			// This segment is a horizontal line. Just compare its latitude with the latitude of the point we're testing.
			if c.Latitude == leftP.Latitude && c.Longitude <= leftP.Longitude {
				intersections++
			}
			continue
		}

		// Longitude of collision point is: Long = (Lat_c)/d
		lonColl := c.Latitude / slope

		if lonColl <= c.Longitude {
			intersections++
		}
	}

	return intersections%2 != 0
}
