package main

// TODO: The big ticket items missing are:
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
	"time"
)

const url1 = "https://warnung.bund.de/bbk.mowas/gefahrendurchsagen.json"
const url2 = "https://warnung.bund.de/bbk.biwapp/warnmeldungen.json"
const url3 = "https://warnung.bund.de/bbk.dwd/unwetter.json"

// TODO: Handle these as well, need to see them in action
const url4 = "https://warnung.bund.de/bbk.lhp/hochwassermeldungen.json"

type Proxy struct {
	// Maps URLs to active messages, keyed by message ID
	activeAlerts map[URL]map[MessageID]alertMessage
}

func newProxy() Proxy {
	return Proxy{
		activeAlerts: make(map[URL]map[MessageID]alertMessage),
	}
}

func (p Proxy) updateData(url URL) error {
	if _, ok := p.activeAlerts[url]; !ok {
		p.activeAlerts[url] = make(map[MessageID]alertMessage)
	}

	resp, err := http.Get(string(url))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var alerts []alertMessage
	err = json.Unmarshal(content, &alerts)
	if err != nil {
		return err
	}

	newAlerts := make([]alertMessage, 0)

	for _, alert := range alerts {
		// Update messages for this url
		if _, ok := p.activeAlerts[url][alert.Identifier]; !ok {
			// This message is new, track it.
			newAlerts = append(newAlerts, alert)
		}
	}

	// TODO: Handle new messages
	for _, alerts := range newAlerts {
		p.handleNewMessage(alerts)
	}

	// Set new active message map
	p.activeAlerts[url] = make(map[MessageID]alertMessage)
	for _, message := range alerts {
		p.activeAlerts[url][message.Identifier] = message
	}

	return nil
}

func (p Proxy) handleNewMessage(alert alertMessage) {
	log.Println("Would handle new message", alert.Identifier, "now")
}

func (p Proxy) run() {
	// Periodically poll all URLs and update proxy state. On update, check subscribed customers for area containment and notify them.
	urls := []URL{url1, url2, url3, url4}

	ticker := time.NewTicker(60 * time.Second)

	for range ticker.C {
		for _, url := range urls {
			err := p.updateData(url)
			if err != nil {
				log.Fatalln("updating", url, "failed:", err)
			}
			log.Println(url, "refreshed")
		}
		log.Println("waiting 60 seconds for next update")
	}
}

func main() {
	log.Println("Here we go")

	proxy := newProxy()

	proxy.run()
}
