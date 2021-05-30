package main

import (
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type ChatServer struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func (cs *ChatServer) AddClient(conn *websocket.Conn) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.clients[conn] = struct{}{}
}

func (cs *ChatServer) RemoveClient(conn *websocket.Conn) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.clients, conn)
}

func (cs *ChatServer) Broadcast(msg []byte) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for conn := range cs.clients {
		err := conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func main() {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	chatServer := &ChatServer{
		clients: make(map[*websocket.Conn]struct{}),
	}

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<!doctype html>
<html>
	<head>
		<title>Chat</title>
		<meta charset="UTF-8" />
		<style>
			#textbox {
				width: 600px;
				height: 300px;
				border: 1px solid;
			}
			#textbox p {
				margin: 0px;
			}
		</style>
	</head>
	<body>
		<h1>Chat</h1>
		<div id="textbox">
		</div>
		<form id="chatform">
			<div><label for="msgbox">MSG</label></div>
			<div>
				<input name="msgbox" id="msgbox">
			</div>
		</form>
		<script>
		let ws = new WebSocket("ws://" + location.host + "/ws");
		ws.onmessage = function(ev) {
			let p = document.createElement("p");
			p.append(ev.data);
			let textbox = document.querySelector("#textbox");
			textbox.append(p);
		};
		let chatform = document.querySelector("#chatform");
		chatform.onsubmit = function(ev) {
			ev.preventDefault();
			let msgbox = document.querySelector("#msgbox");
			ws.send(msgbox.value);
			msgbox.value = "";
			return false;
		}
		</script>
	</body>
</html>
`)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		chatServer.AddClient(conn)
		defer chatServer.RemoveClient(conn)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				return
			}
			log.Println(string(msg))
			chatServer.Broadcast(msg)
		}
	})
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}
