package main

import (
	"fmt"
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
		go func(conn *websocket.Conn) {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println("write error: ", err)
			}
		}(conn)
	}
}

func main() {
	upgrader := websocket.Upgrader{}
	chatServer := &ChatServer{
		clients: make(map[*websocket.Conn]struct{}),
	}
	http.HandleFunc("/html", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `
<html>
	<head>
		<title>WebSocket</title>
		<link rel="icon" href="data:,">
		<style>
			#msgbox {
				width: 600px;
				height: 300px;
				border: 1px solid;
			}
			#msgbox > p {
				margin: 0px;
			}
			#textbox {
				width: 600px;
				border: 1px solid;
			}
		</style>
	</head>
	<body>
		<h1>WebSocket</h1>
		<form id="chatform">
			<div id="msgbox">
			</div>
			<div>
				<label for="textbox">MSG: </label>
			</div>
			<div>
				<input name="textbox" id="textbox" />
			</div>
		</form>
		<script>
			function appendMsg(data) {
				let msgbox = document.querySelector("#msgbox");
				let p = document.createElement("p");
				p.append(data);
				msgbox.append(p);
			}
			let ws = new WebSocket("ws://" + location.host + "/ws");
			ws.onmessage = function(ev) {
				appendMsg(ev.data);
			};
			let chatform = document.querySelector("#chatform");
			chatform.onsubmit = function(ev) {
				ev.preventDefault();
				let textbox = document.querySelector("#textbox");
				ws.send(textbox.value);
				textbox.value = "";
				return false;
			}
		</script>
	</body>
</html>`)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		chatServer.AddClient(conn)
		defer chatServer.RemoveClient(conn)
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				log.Printf("read error: %s, %T", err, err)
				return
			}
			fmt.Println(string(msg))
			chatServer.Broadcast(msg)
		}
	})

	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal()
	}
}
