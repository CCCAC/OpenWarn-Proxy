package main

// TODO: The big ticket items missing are:
// - Persist data somehow?
// - Expiry of messages with an expiry attribute

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
	"flag"

	"golang.org/x/net/websocket"
)

const (
	url1 = "https://warnung.bund.de/bbk.mowas/gefahrendurchsagen.json"
	url2 = "https://warnung.bund.de/bbk.biwapp/warnmeldungen.json"
	url3 = "https://warnung.bund.de/bbk.dwd/unwetter.json"

	// TODO: Handle these as well, need to see them in action
	url4 = "https://warnung.bund.de/bbk.lhp/hochwassermeldungen.json"
)

var (
	_updateDelay time.Duration
	_socketPath string
	_socketAddr string
)

func init() {
	flag.DurationVar(&_updateDelay, "updateDelay", 30 * time.Second, "Intervall between polling for new data")
	flag.StringVar(&_socketPath, "socketPath", "/coords", "Path to websocket")
	flag.StringVar(&_socketAddr, "socketAddr", ":8080", "Address to listen on for websocket connections")

	flag.Parse()
}

type Proxy struct {
	sync.Mutex
	// Maps URLs to active messages, keyed by message ID
	activeAlerts map[URL]map[MessageID]alertMessage
	updateChans  map[chan bool]bool
	areas        map[MessageID][]Area
}

func newProxy() Proxy {
	return Proxy{
		activeAlerts: make(map[URL]map[MessageID]alertMessage),
		updateChans:  make(map[chan bool]bool),
		areas:        make(map[MessageID][]Area),
	}
}

// getMatchingAlerts returns all alerts that have areas affecting the provided coordinate
// This function is really fucking ugly.
func (p *Proxy) getMatchingAlerts(c Coordinate) []alertMessage {
	p.Lock()
	defer p.Unlock()

	var messageIDs []MessageID
	for id, areas := range p.areas {
		for _, area := range areas {
			if area.Contains(c) {
				messageIDs = append(messageIDs, id)
			}
		}
	}

	// Create empty list. This doesn't use the `nil` pattern for new slices because those encode to `null` values in JSON.
	matchingAlerts := make([]alertMessage, 0)
	for _, id := range messageIDs {
		// Gather all messages
		for _, alerts := range p.activeAlerts {
			if alert, ok := alerts[id]; ok {
				matchingAlerts = append(matchingAlerts, alert)
			}
		}
	}

	return matchingAlerts
}

func (p *Proxy) registerUpdateChan(ch chan bool) {
	p.Lock()
	defer p.Unlock()

	p.updateChans[ch] = true
}

func (p *Proxy) unregisterUpdateChan(ch chan bool) {
	p.Lock()
	defer p.Unlock()

	delete(p.updateChans, ch)
}

// socketHandler runs a client connection
func (p *Proxy) socketHandler(conn *websocket.Conn) {
	// When a user sends a Lat/Lon pair, check active alerts on p
	// Otherwise, watch the active alerts for new things
	quit := make(chan interface{})
	done := make(chan interface{})
	coords := make(chan Coordinate)

	updateChan := make(chan bool)
	p.registerUpdateChan(updateChan)
	defer p.unregisterUpdateChan(updateChan)

	// First part, run a goroutine to wait changes on the active alerts
	go func() {
		var currentCoords Coordinate
		coordsSet := false // True if coordinates have been received
		running := true
		enc := json.NewEncoder(conn)

		log.Println("Waiting for changes in active alerts or coordinates")
		for running {
			select {
			case c := <-coords:
				currentCoords = c
				coordsSet = true
				log.Println("Received new coordinate:", c)
				alerts := p.getMatchingAlerts(c)
				enc.Encode(&alerts)
			case <-updateChan:
				if coordsSet {
					alerts := p.getMatchingAlerts(currentCoords)
					enc.Encode(&alerts)
				} else {
					log.Println("not checking update, no coords set")
				}
			case <-quit:
				running = false
			}
		}
		log.Println("Active alert watcher quitting")
		close(done)
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

	// Signal goroutine that it's time to go
	close(quit)
	// ... and wait for it to exit
	<-done
	conn.Close()
}

// socketHandshake is a dumb handshake handler for the websocket. This is required because the websocket library checks the origin
// by default and rejects requests with an unexpected origin.
func (p *Proxy) socketHandshake(conf *websocket.Config, req *http.Request) error {
	log.Println("ws handshake. conf:", conf, "req:", req)
	return nil
}

// updateData requests new data from url and updates the stored alert messages. It returns true if an update was performed, and
// false if no new data arrived
func (p *Proxy) updateData(url URL) (bool, error) {
	if _, ok := p.activeAlerts[url]; !ok {
		p.activeAlerts[url] = make(map[MessageID]alertMessage)
	}

	resp, err := http.Get(string(url))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var alerts []alertMessage
	err = json.Unmarshal(content, &alerts)
	if err != nil {
		return false, err
	}

	// Track new alerts
	newAlerts := 0

	for _, alert := range alerts {
		// Update messages for this url
		if _, ok := p.activeAlerts[url][alert.Identifier]; !ok {
			// This message is new, track it.
			newAlerts++
		}
	}

	// Set new active message map
	p.activeAlerts[url] = make(map[MessageID]alertMessage)

	for _, message := range alerts {
		p.activeAlerts[url][message.Identifier] = message
		// Collect all areas for this message
		var areas []Area
		for _, info := range message.Info {
			for _, area := range info.Area {
				for _, poly := range area.Polygon {
					a, err := NewAreaFromString(poly)
					if err != nil {
						return false, err
					}
					areas = append(areas, a)
				}
			}
		}
		p.areas[message.Identifier] = areas
	}

	return newAlerts != 0, nil
}

func (p *Proxy) run() {
	// Periodically poll all URLs and update proxy state. On update, check subscribed customers for area containment and notify them.
	urls := []URL{url1, url2, url3, url4}

	ticker := time.NewTicker(_updateDelay)

	for {
		needUpdate := false
		p.Lock()
		p.areas = make(map[MessageID][]Area)
		for _, url := range urls {
			newData, err := p.updateData(url)
			if err != nil {
				log.Println("updating", url, "failed:", err)
				continue
			}
			log.Println(url, "refreshed")
			needUpdate = needUpdate || newData
		}
		if needUpdate {
			log.Println("Notifying connected clients of updates")
			// Non-blocking notify to make sure slow clients don't block us
			for ch := range p.updateChans {
				select {
				case ch <- true:
				default:
				}
			}
		}
		p.Unlock()
		log.Println("waiting", _updateDelay, "for next update")
		<-ticker.C
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
	http.Handle(_socketPath, srv)

	err := http.ListenAndServe(_socketAddr, nil)
	if err != nil {
		log.Fatalln("failed to start web socket server:", err)
	}
}
