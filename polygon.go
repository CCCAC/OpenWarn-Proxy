package main

import (
	"strings"
	"strconv"
	"fmt"
)

// This file contains the code for handling arbitrary lat/lon polygons.

type Coordinate struct {
	Latitude, Longitude float64
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

type Area struct {
	Coordinates []Coordinate
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

	for _, chunk := range strings.Split(coords, " ") {
		coord, err := NewCoordinateFromString(strings.TrimSpace(chunk))
		if err != nil {
			return a, err
		}

		a.Coordinates = append(a.Coordinates, coord)
	}

	return a, nil
}

// IsDegenerate returns true if the area represented as a is degenerate, that is, it contains non-convex polygons.
func (a Area) IsDegenerate() bool {
	return len(a.Coordinates) < 3
}
