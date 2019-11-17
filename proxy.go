package main

// TODO: The big ticket items missing are:
// - Persist data somehow?
// - Expiry of messages with an expiry attribute

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
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
	_socketPath  string
	_socketAddr  string
	_logLevel    string
	_logCallers  bool
)

func init() {
	flag.DurationVar(&_updateDelay, "updateDelay", 30*time.Second, "Intervall between polling for new data")
	flag.StringVar(&_socketPath, "socketPath", "/coords", "Path to websocket")
	flag.StringVar(&_socketAddr, "socketAddr", ":8080", "Address to listen on for websocket connections")
	flag.StringVar(&_logLevel, "logLevel", "info", "Log level to use")
	flag.BoolVar(&_logCallers, "logCallers", false, "Whether to log callers")

	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
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
func (cl *Client) getMatchingAlerts(c Coordinate) []alertMessage {
	p := cl.p
	p.Lock()
	defer p.Unlock()

	messageIDs := make(map[MessageID]bool)
	cl.Log().WithField("numareas", len(p.areas)).Debug("checking area collections")
	for id, areas := range p.areas {
		cl.Log().WithField("numareas", len(areas)).Println("checking area(s)")
		for _, area := range areas {
			if area.Contains(c) {
				messageIDs[id] = true
			}
		}
	}
	cl.Log().WithFields(logrus.Fields{
		"count": len(messageIDs),
		"ids":   messageIDs,
	}).Debug("got matching message IDs")

	// Create empty list. This doesn't use the `nil` pattern for new slices because those encode to `null` values in JSON.
	matchingAlerts := make([]alertMessage, 0)
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

type Client struct {
	p   *Proxy
	log atomic.Value
}

func (c *Client) Log() *logrus.Entry {
	return c.log.Load().(*logrus.Entry)
}

func (c *Client) SetLog(l *logrus.Entry) {
	c.log.Store(l)
}

// socketHandler runs a client connection
func (p *Proxy) socketHandler(w http.ResponseWriter, r *http.Request) {
	client := Client{
		p: p,
	}
	client.SetLog(logrus.WithFields(logrus.Fields{
		"component": "client",
		"remote":    r.RemoteAddr,
	}))

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		client.Log().Error("Failed to set up websocket:", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "internal server error")

	// When a user sends a Lat/Lon pair, check active alerts on p
	// Otherwise, watch the active alerts for new things
	quit := make(chan interface{})
	done := make(chan interface{})
	coords := make(chan Coordinate)

	updateChan := make(chan bool)
	p.registerUpdateChan(updateChan)
	defer p.unregisterUpdateChan(updateChan)

	// First part, run a goroutine to await changes on the active alerts
	go func() {
		var currentCoords Coordinate
		coordsSet := false // True if coordinates have been received
		running := true

		client.Log().Info("Waiting for changes in active alerts or coordinates")
		for running {
			writer, err := conn.Writer(r.Context(), websocket.MessageText)
			if err != nil {
				client.Log().Error("can't set up writer:", err)
				return
			}
			enc := json.NewEncoder(writer)

			select {
			case c := <-coords:
				currentCoords = c
				coordsSet = true
				client.SetLog(client.Log().WithField("coordinate", c))
				client.Log().Info("Received new coordinate")
				alerts := client.getMatchingAlerts(c)
				enc.Encode(&alerts)
			case <-updateChan:
				if coordsSet {
					alerts := client.getMatchingAlerts(currentCoords)
					enc.Encode(&alerts)
				} else {
					client.Log().Info("not checking update, no coords set")
				}
			case <-quit:
				running = false
			}
			writer.Close()
		}
		client.Log().Info("Active alert watcher quitting")
		close(done)
	}()

	// Consume from the websocket tow gather new lat/lon pairs, exit on first error
	for {
		mt, reader, err := conn.Reader(r.Context())
		if err != nil {
			client.Log().Error("Failed to read from websocket:", err)
			break
		}
		if mt != websocket.MessageText {
			// Consume all message and them
			for {
				buf := make([]byte, 32)
				_, err := reader.Read(buf)
				if err != nil {
					client.Log().Error("Failed to read from websocket:", err)
					break
				}
			}
			continue
		}

		for {
			dec := json.NewDecoder(reader)
			var coord Coordinate
			err = dec.Decode(&coord)
			if err != nil {
				if errors.Is(err, io.EOF) {
					client.Log().Error("Error while decoding:", err)
				}
				break
			}

			// Update coordinates and re-check active alerts
			coords <- coord
		}
	}

	// Signal goroutine that it's time to go
	close(quit)
	// ... and wait for it to exit
	<-done
	conn.Close(websocket.StatusNormalClosure, "")
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
func (p *Proxy) updateLoop() {
	log := logrus.WithField("component", "updater")

	urls := []URL{url1, url2, url3, url4}

	ticker := time.NewTicker(_updateDelay)

	for {
		needUpdate := false
		p.Lock()
		p.areas = make(map[MessageID][]Area)
		for _, url := range urls {
			newData, err := p.updateData(url)
			if err != nil {
				log.WithFields(logrus.Fields{
					"url": url,
					"err": err}).Error("update failed")
				continue
			}
			log.WithField("url", url).Debug("data refreshed")
			needUpdate = needUpdate || newData
		}
		if needUpdate {
			log.Info("Notifying connected clients of updates")
			// Non-blocking notify to make sure slow clients don't block us
			for ch := range p.updateChans {
				select {
				case ch <- true:
				default:
				}
			}
		}
		p.Unlock()
		log.WithField("delay", _updateDelay).Debug("waiting for next update")
		<-ticker.C
	}
}

func main() {
	flag.Parse()

	lvl, err := logrus.ParseLevel(_logLevel)
	if err != nil {
		logrus.Fatalln("Can't parse log level:", err)
	}
	logrus.SetReportCaller(_logCallers)
	logrus.SetLevel(lvl)

	logrus.Info("Starting up")

	proxy := newProxy()

	go proxy.updateLoop()

	http.HandleFunc(_socketPath, proxy.socketHandler)

	logrus.Info("Handlers configured, app started")

	err = http.ListenAndServe(_socketAddr, nil)
	if err != nil {
		logrus.Fatalln("failed to start web socket server:", err)
	}
}
