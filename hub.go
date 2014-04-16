package main

import (
	"log"
)

type SubMessage struct {
	Event   string
	Channel string `json:"channel"`
	// Data    SubMessageData `json:"data"` // just ignore it
}

type hub struct {
	// Registered connections.
	connections map[*connection]bool

	// register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection

	// Register Nsq Reader
	addSubscriber chan *Subscriber

	// Registered Nsq Readers
	nsqReaders map[string]*NsqTopicReader // NSQ topic -> nsqReader
}

var h = hub{
	register:      make(chan *connection),
	unregister:    make(chan *connection),
	addSubscriber: make(chan *Subscriber),
	connections:   make(map[*connection]bool),
	nsqReaders:    make(map[string]*NsqTopicReader),
}

func (h *hub) run() {
	log.Printf("hub start of life\n")
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
		case c := <-h.unregister:
			log.Println("h.unregister fired")

			delete(h.connections, c)
			log.Println("connection deleted from hub's poll OK")

			for topic, reader := range c.subscribed {
				log.Println("delete connection on topic:" + topic)
				reader.RemoveConnection(c)
			}
			close(c.send)
			//c.subscribed = nil
		case sub := <-h.addSubscriber:
			log.Println("h.addSubscriber fired")
			nsqTopicName := GenNSQtopicName(sub.Topic)
			reader, is_reader_exists := h.nsqReaders[nsqTopicName]

			c := sub.Conn
			if !is_reader_exists {
				var err error
				reader, err = NewNsqTopicReader(sub.Topic)
				if err != nil {
					log.Printf("failed to subscribe to topic '%s'", sub.Topic)
					break
				}
			}
			c.subscribed[nsqTopicName] = reader
			// reader.connections[c] = true
			h.nsqReaders[nsqTopicName] = reader
			reader.AddConnection(c, sub.Topic)
		}
	}

	panic("hub end of life")
}
