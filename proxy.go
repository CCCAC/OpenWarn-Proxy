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

	"golang.org/x/net/websocket"
)

const (
	url1 = "https://warnung.bund.de/bbk.mowas/gefahrendurchsagen.json"
	url2 = "https://warnung.bund.de/bbk.biwapp/warnmeldungen.json"
	url3 = "https://warnung.bund.de/bbk.dwd/unwetter.json"

	// TODO: Handle these as well, need to see them in action
	url4 = "https://warnung.bund.de/bbk.lhp/hochwassermeldungen.json"
)

type Proxy struct {
	// Maps URLs to active messages, keyed by message ID
	activeAlerts map[URL]map[MessageID]alertMessage
}

func newProxy() Proxy {
	return Proxy{
		activeAlerts: make(map[URL]map[MessageID]alertMessage),
	}
}

// getMatchingAlerts returns all alerts that have areas affecting the provided coordinate
// This function is really fucking ugly.
func (p Proxy) getMatchingAlerts(c Coordinate) []alertMessage {
	var alerts []alertMessage

loop:
	for _, alert := range p.activeAlerts {
		for _, msg := range alert {
			for _, info := range msg.Info {
				for _, area := range info.Area {
					for _, poly := range area.Polygon {
						a, err := NewAreaFromString(poly)
						if err != nil {
							log.Println("can't parse polygon", err)
							continue
						}
						if a.Contains(c) {
							alerts = append(alerts, msg)
							continue loop
						}
					}
				}
			}
		}
	}

	return alerts
}

// socketHandler runs a client connection
func (p Proxy) socketHandler(conn *websocket.Conn) {
	// When a user sends a Lat/Lon pair, check active alerts on p
	// Otherwise, watch the active alerts for new things

	coords := make(chan Coordinate)
	quit := make(chan interface{})

	// First part, run a goroutine to wait changes on the active alerts
	go func() {
		enc := json.NewEncoder(conn)

		log.Println("Waiting for changes in active alerts or coordinates")
	loop:
		for {
			select {
			case c := <-coords:
				log.Println("Received new coordinate:", c)
				alerts := p.getMatchingAlerts(c)
				enc.Encode(&alerts)
			case <-quit:
				break loop
			}
		}
		log.Println("Active alert watcher quitting")
	}()

	// Consume from the websocket to gather new lat/lon pairs, exit on first error
	dec := json.NewDecoder(conn)
	for dec.More() {
		var coord Coordinate
		err := dec.Decode(&coord)
		if err != nil {
			log.Println("Error while decoding:", err)
			break
		}

		// Update coordinates and re-check active alerts
		coords <- coord
	}

	close(quit)
	conn.Close()
}

// socketHandshake is a dumb handshake handler for the websocket. This is required because the websocket library checks the origin
// by default and rejects requests with an unexpected origin.
func (p Proxy) socketHandshake(conf *websocket.Config, req *http.Request) error {
	log.Println("ws handshake. conf:", conf, "req:", req)
	return nil
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

	go proxy.run()

	srv := websocket.Server{
		Handshake: proxy.socketHandshake,
		Handler:   websocket.Handler(proxy.socketHandler),
	}
	http.Handle("/coords", srv)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalln("failed to start web socket server:", err)
	}
}
