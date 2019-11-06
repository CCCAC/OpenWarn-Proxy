package main

// TODO: The big ticket items missing are:
// - Handle area information, extract lat/lon polygon
// - Expose websocket, allow clients to request data for particular location
// - Check existing data for matches
// - Poll for new data, check matches
// - Persist data somehow?
// - Expiry of messages with an expiry attribute

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type geocodeDescription struct {
	ValueName string `json:"valueName"`
	Value     string `json:"value"`
}

type areaDescription struct {
	Description string               `json:"areaDesc"`
	Polygon     []string             `json:"polygon"` // TODO: Add proper extraction
	Geocode     []geocodeDescription `json:"geocode"`
}

type infoItem struct {
	Language           string            `json:"language"`
	Category           []string          `json:"category"`     // e.g. []{"Safety"}
	Event              string            `json:"event"`        // e.g. "Gefahrenmitteilung", "Gefahreninformation", sometimes just a code
	ResponseType       []string          `json:"responseType"` // e.g. []{"Monitor"}
	Urgency            string            `json:"urgency"`      // e.g. "Immediate"
	Severity           string            `json:"severity"`     // e.g. "Minor"
	Certainty          string            `json:"certainty"`    // How certain is this message? "Observed"
	Headline           string            `json:"headline"`
	Description        string            `json:"description"`
	Instructions       string            `json:"instruction"`
	ContactInformation string            `json:"contact"`
	URL                string            `json:"web"`
	Area               []areaDescription `json:"area"` // List of affected areas

	// TODO: parameter: list of key-value metadata items
	// TODO: expires  : time stamp of expiry, e.g. "2019-11-30T15:37:00+01:00"
}

type messagePayload struct {
	Identifier string `json:"identifier"`
	Sender     string `json:"sender"`
	Sent       string `json:"sent"`   // Timestamp
	Status     string `json:"status"` // Is this a current message?
	MsgType    string `json:"msgType"`
	Scope      string `json:"scope"`  // e.g. "Public"

	// `code` is ignored here
	Info []infoItem `json:"info"`
}

const url1 = "https://warnung.bund.de/bbk.mowas/gefahrendurchsagen.json"
const url2 = "https://warnung.bund.de/bbk.biwapp/warnmeldungen.json"
// TODO: Handle these as well, need to see them in action
const url3 = "https://warnung.bund.de/bbk.dwd/unwetter.json"
const url4 = "https://warnung.bund.de/bbk.lhp/hochwassermeldungen.json"

func getData(url string) ([]messagePayload, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload []messagePayload
	err = json.Unmarshal(content, &payload)
	if err != nil {
		return nil, err
	}

	return payload, err
}

func main() {
	log.Println("Here we go")

	p, err := getData(url1)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(url1, p)

	p, err = getData(url2)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(url2, p)
}
