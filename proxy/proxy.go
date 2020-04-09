package proxy

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
)

const (
	url1 = "https://warnung.bund.de/bbk.mowas/gefahrendurchsagen.json"
	url2 = "https://warnung.bund.de/bbk.biwapp/warnmeldungen.json"
	url3 = "https://warnung.bund.de/bbk.dwd/unwetter.json"

	// TODO: Handle these as well, need to see them in action
	url4 = "https://warnung.bund.de/bbk.lhp/hochwassermeldungen.json"
)

type Proxy struct {
	sync.Mutex
	// Maps URLs to active messages, keyed by message ID
	activeAlerts map[URL]map[MessageID]alertMessage
	updateChans  map[chan bool]bool
	areas        map[MessageID][]Area
}

func (p *Proxy) GetAllAlerts() []alertMessage {
	p.Lock()
	defer p.Unlock()

	var alerts []alertMessage
	for _, messages := range p.activeAlerts {
		for _, alert := range messages {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// getMatchingAlerts returns all alerts that have areas affecting the provided coordinate
// This function is really fucking ugly.
func (p *Proxy) GetMatchingAlerts(c Coordinate) []alertMessage {
	p.Lock()
	defer p.Unlock()

	messageIDs := make(map[MessageID]bool)
	for id, areas := range p.areas {
		for _, area := range areas {
			if area.Contains(c) {
				messageIDs[id] = true
			}
		}
	}

	var matchingAlerts []alertMessage
	for id := range messageIDs {
		// Gather all messages
		for _, alerts := range p.activeAlerts {
			if alert, ok := alerts[id]; ok {
				matchingAlerts = append(matchingAlerts, alert)
			}
		}
	}

	return matchingAlerts
}

func New() Proxy {
	return Proxy{
		activeAlerts: make(map[URL]map[MessageID]alertMessage),
		areas:        make(map[MessageID][]Area),
	}
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

// updateData requests new data from url and updates the stored alert messages. It returns true if an update was performed, and
// false if no new data arrived
//
// It requires p to be locked.
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

// updateLoop polls all URLs and updates the proxy state. On update, it checks subscribed customers for area containment and notify
// them.
func (p *Proxy) UpdateLoop() {
	urls := []URL{url1, url2, url3, url4}

	ticker := time.NewTicker(90 * time.Second)

	for {
		needUpdate := false
		p.Lock()
		p.areas = make(map[MessageID][]Area)
		for _, url := range urls {
			newData, err := p.updateData(url)
			if err != nil {
				log.Println("update failed:", err)
				continue
			}
			needUpdate = needUpdate || newData
		}
		if needUpdate {
			// Non-blocking notify to make sure slow clients don't block us
			for ch := range p.updateChans {
				select {
				case ch <- true:
				default:
				}
			}
		}
		p.Unlock()
		<-ticker.C
	}
}
