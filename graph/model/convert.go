package model

import (
	"fmt"

	"github.com/cccac/OpenWarn-Proxy/proxy"
)

func FromGeocode(code proxy.Geocode) Geocode {
	return Geocode{
		Name:  code.ValueName,
		Value: code.Value,
	}
}

func FromSegment(seg proxy.LineSegment) LineSegment {
	p1 := FromLocation(seg.P1)
	p2 := FromLocation(seg.P2)
	return LineSegment{
		P1: &p1,
		P2: &p2,
	}
}

func FromProxyArea(area proxy.Area) Polygon {
	poly := Polygon{}

	for _, seg := range area.Segments {
		poly.Segments = append(poly.Segments, FromSegment(seg))
	}

	return poly
}

func FromAreaDescription(desc proxy.AreaDescription) (Area, error) {
	// log.Printf("Polygon: %#v", desc.Polygon)

	area := Area{
		Description: desc.Description,
	}

	for _, code := range desc.Geocode {
		area.Geocode = append(area.Geocode, FromGeocode(code))
	}

	for _, polygon := range desc.Polygon {
		// Extract polygon from string representation
		proxyArea, err := proxy.NewAreaFromString(polygon)
		if err != nil {
			return area, err
		}

		// Convert line segments to model representation
		modelPolygon := FromProxyArea(proxyArea)
		area.Polygons = append(area.Polygons, modelPolygon)
	}

	return area, nil
}

func FromAlert(activeAlert proxy.Alert) (Alert, error) {
	alert := Alert{}

	meta := AlertMetadata{
		ID:     string(activeAlert.Identifier),
		Sender: activeAlert.Sender,
		SentAt: activeAlert.Sent,
		Status: Status(activeAlert.Status),
		Type:   Type(activeAlert.MsgType),
		Scope:  Scope(activeAlert.Scope),
	}
	if !meta.Status.IsValid() {
		return alert, fmt.Errorf("invalid status `%s`", meta.Status)
	}
	if !meta.Type.IsValid() {
		return alert, fmt.Errorf("invalid type `%s`", meta.Type)
	}
	if !meta.Scope.IsValid() {
		return alert, fmt.Errorf("invalid scope `%s`", meta.Scope)
	}

	var payloadEntries []AlertPayload
	for _, info := range activeAlert.Info {
		if len(info.Category) != 1 {
			return alert, fmt.Errorf("unexpected category length %d", len(info.Category))
		}
		if len(info.ResponseType) > 1 {
			return alert, fmt.Errorf("unexpected response type length %d", len(info.ResponseType))
		}

		var areas []Area
		for _, desc := range info.Area {
			area, err := FromAreaDescription(desc)
			if err != nil {
				return alert, err
			}
			areas = append(areas, area)
		}

		p := AlertPayload{
			Headline:  info.Headline,
			Message:   info.Description,
			Area:      areas,
			Expires:   info.Expires,
			Urgency:   info.Urgency,
			Severity:  info.Severity,
			Certainty: info.Certainty,
		}
		if len(info.ResponseType) == 1 {
			p.Response = info.ResponseType[0]
		}
		if info.Instructions != "" {
			p.Instructions = &info.Instructions
		}
		if info.ContactInformation != "" {
			p.Contact = &info.ContactInformation
		}
		if info.URL != "" {
			url := string(info.URL)
			p.URL = &url
		}

		payloadEntries = append(payloadEntries, p)
	}

	alert.Metadata = &meta
	alert.Payload = payloadEntries

	return alert, nil
}

func ToLocation(loc *LocationInput) proxy.Location {
	return proxy.Location{
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
	}
}

func FromLocation(loc proxy.Location) Location {
	return Location{
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
	}
}
