package main

import (
	"fmt"
	"strconv"
	"strings"
)

// This file contains the code for handling arbitrary lat/lon polygons.

type Coordinate struct {
	Latitude, Longitude float64
}
func (c Coordinate) String() string {
	return fmt.Sprintf("%.3fx%.3f", c.Latitude, c.Longitude)
}

type InvalidCoordinateError struct {
	s string
	v []string
}

func (e InvalidCoordinateError) Error() string {
	return fmt.Sprintf("Invalid coordinate string '%s', splits into %#v", e.s, e.v)
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
	if chunks[0] != chunks[len(chunks)-1] {
		// First and last coordinate of the list need to be equal, otherwise the polygon is not closed
		return a, fmt.Errorf("polygon not closed")
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
