package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Names of subscribed channels
	subscribed map[string]*NsqTopicReader
}

func (c *connection) reader() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		c.processMessage(message)
		//h.broadcast <- message
	}
	c.ws.Close()
}

func (c *connection) processMessage(msg []byte) {
	message := SubMessage{}
	err := json.Unmarshal(msg, &message)
	if err != nil {
		log.Println("ERROR: invalid JSON subscribe data")
		return
	}
	log.Printf("message+: %v\n", message)

	s := Subscriber{Conn: c, Topic: message.Channel}
	log.Printf("send to channel: %v\n", s)
	h.addSubscriber <- &s
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}

	log.Println("Create ws connection")
	// TODO: Add Id generation
	// FIXME: move magic number to const
	c := &connection{
		send:       make(chan []byte, 4096),
		ws:         ws,
		subscribed: make(map[string]*NsqTopicReader),
	}
	h.register <- c
	defer func() { h.unregister <- c }()
	go c.writer()
	c.reader()
}
