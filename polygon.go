package main

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

type Coordinate struct {
	Latitude, Longitude float64
}

func NewCoordinateFromString(s string) (Coordinate, error) {
	var c Coordinate
	v := strings.Split(strings.TrimSpace(s), ",")
	if len(v) != 2 {
		return c, InvalidCoordinateError{s, v}
	}

	var err error
	c.Latitude, err = strconv.ParseFloat(v[0], 64)
	if err != nil {
		return c, fmt.Errorf("parsing latitude: %w", err)
	}
	c.Longitude, err = strconv.ParseFloat(v[1], 64)
	if err != nil {
		return c, fmt.Errorf("parsing longitude: %w", err)
	}

	return c, nil
}

func (c Coordinate) String() string {
	return fmt.Sprintf("%.3fx%.3f", c.Latitude, c.Longitude)
}

type LineSegment struct {
	p1, p2 Coordinate
}

func (l LineSegment) String() string {
	return fmt.Sprintf("[%s->%s]", l.p1, l.p2)
}

type Area struct {
	Segments []LineSegment
}

// NewAreaFromString loads coordinates from the string representation and returns an area. The returned area may not be a single
// polygon and it may not be convex.
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
		// First and last coordinate of the list need to be equal, otherwise the polygon is not closed
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
		seg := LineSegment{p1: p1, p2: p2}
		a.Segments = append(a.Segments, seg)
		lastCoord = chunk
	}

	return a, nil
}

// Contains returns true if c is inside the polygon described by a
func (a Area) Contains(c Coordinate) bool {
	// Cast a ray from c to the right. Count crossings with line segments of a. If the number of crossings is even, c is outside
	// of a.

	intersections := 0

	for _, seg := range a.Segments {
		// Check if a is to the left of the rightmost part of seg and between the end points
		// Check latitudes (Y coords)
		minLat := math.Min(seg.p1.Latitude, seg.p2.Latitude)
		maxLat := math.Max(seg.p1.Latitude, seg.p2.Latitude)

		if c.Latitude < minLat || c.Latitude > maxLat {
			// Above or below line segment
			continue
		}

		// Now check if c is to the left of the rightmost point
		maxLong := math.Max(seg.p1.Longitude, seg.p2.Longitude)
		if c.Longitude <= maxLong {
			intersections++
		}
	}

	return intersections%2 != 0
}
