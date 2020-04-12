package model

import (
	"fmt"
	"github.com/cccac/OpenWarn-Proxy/proxy"
)

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

		// TODO: Fill with useful info
		var areas []Area

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
