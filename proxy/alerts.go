package proxy

import (
	"fmt"
	"time"
)

type URL string
type MessageID string

type Geocode struct {
	ValueName string `json:"valueName"`
	Value     string `json:"value"`
}

type AreaDescription struct {
	Description string    `json:"areaDesc"`
	Polygon     []string  `json:"polygon"` // TODO: Add proper extraction
	Geocode     []Geocode `json:"geocode"`
}

type infoItem struct {
	Language           string            `json:"language"`
	Category           []string          `json:"category"`     // e.g. []{"Safety"} or []{"Met"} for meterological messages
	Event              string            `json:"event"`        // e.g. "Gefahrenmitteilung", "Gefahreninformation", sometimes just a code
	ResponseType       []string          `json:"responseType"` // suggested response e.g. []{"Monitor", "Prepare"}
	Urgency            string            `json:"urgency"`      // e.g. "Immediate"
	Severity           string            `json:"severity"`     // e.g. "Minor"
	Certainty          string            `json:"certainty"`    // How certain is this message? "Observed"
	Headline           string            `json:"headline"`
	Description        string            `json:"description"`
	Instructions       string            `json:"instruction"`
	ContactInformation string            `json:"contact"`
	URL                URL               `json:"web"`
	Area               []AreaDescription `json:"area"` // List of affected areas
	Expires            time.Time         `json:"expires"`

	// TODO: parameter: list of key-value metadata items
}

func (i infoItem) String() string {
	return fmt.Sprintf("[Category: %v, Event: %s, ResponseType: %v, Urgency: %s, Severity: %s, Headline: %s, Description: %s, Instructions: %s, Contact: %s, URL: %s, Expires: %s]",
		i.Category, i.Event, i.ResponseType, i.Urgency, i.Severity, i.Headline, i.Description, i.Instructions, i.ContactInformation, i.URL, i.Expires)
}

type Alert struct {
	Identifier MessageID `json:"identifier"`
	Sender     string    `json:"sender"`
	Sent       time.Time `json:"sent"`    // Timestamp
	Status     string    `json:"status"`  // Is this a current message?
	MsgType    string    `json:"msgType"` // "Cancel", "Alert"
	Scope      string    `json:"scope"`   // e.g. "Public"

	// `code` is ignored here
	Info []infoItem `json:"info"`
}

func (m Alert) String() string {
	return fmt.Sprintf("[Identifier: %s, Sender: %s, Sent: %s, Status: %s, MsgType: %s, Scope: %s, Info: %s]",
		m.Identifier, m.Sender, m.Sent, m.Status, m.MsgType, m.Scope, m.Info)
}
