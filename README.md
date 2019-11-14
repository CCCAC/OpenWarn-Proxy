# OpenWarn-Proxy

OpenWarn-Proxy gathers civil protection alerts from the german office for
civilan protection at https://warnung.bund.de and proxies them to a websocket.
Clients can open a websocket connection and provide a location as
longitude/latitude. If an alert matching that location shows up (or is already
known to the proxy), it is sent to the client.

Client messages are JSON encoded and look like this:

    {
        "Latitude": 48.8345,
        "Longitude": 8.3819
    }

The response is a JSON encoded array of alerts. To get an idea of how the
response may look, take a look at the source for the alert messages at one of
the following URLs:

    - https://warnung.bund.de/bbk.mowas/gefahrendurchsagen.json
    - https://warnung.bund.de/bbk.biwapp/warnmeldungen.json
    - https://warnung.bund.de/bbk.dwd/unwetter.json
    - https://warnung.bund.de/bbk.lhp/hochwassermeldungen.json

To run the proxy, simply run the binary like this:

    ./OpenWarn-Proxy -updateDelay=1m30s -socketPath=/coords -sockerAddr=:8080

This will poll for updates ever one-and-a-half minutes and provide a websocket
at path "/coords" on port 8080. To connect from a locally running browser, open
its Javascript console and run the following:

    ws = new WebSocket("ws://localhost:8080/coords")
    ws.onmessage = (msg) => { console.log(msg.data) }
    ws.send(JSON.stringify({Latitude: 48.8345, Longitude: 8.3819})

You will immediately get a list of active alerts for that location, and if a new
one comes in, it will be provided to the websocket and the callback function
will run again.
