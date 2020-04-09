console.log("setting up web socket");

let ws = new WebSocket("ws://localhost:8080/coords");
ws.onmessage = (msg) => { console.log(msg.data) };
ws.onopen = (event) => {
	ws.send(JSON.stringify({Latitude: 48.8345, Longitude: 8.3819}));
};