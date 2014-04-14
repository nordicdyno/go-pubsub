package main

import (
	"fmt"
	"log"
	"net/http"
)

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
	nsqReaders map[string]*NsqTopicReader // topic -> nsqReader
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
				_ = reader
				conns := h.nsqReaders[topic].connections
				log.Printf("found reader for topic '%s', (real names is '%s':'%s')\n",
					topic, reader.topicRealName, reader.channelRealName,
				)
				log.Printf("topic conns count: %v\n", len(conns))
				delete(conns, c)
				if len(conns) == 0 {
					log.Printf("stop & cleanup reader for topic '%s'", topic)
					reader.r.Stop()
					delete(h.nsqReaders, topic)
					// TODO : refactor me :)
					/* delete topic */
					httpclient := &http.Client{}
					url := fmt.Sprintf(addrNsqlookupdHTTP+"/delete_topic?topic=%s", reader.topicRealName)
					log.Println("REQUEST on " + url)
					nsqReq, err := http.NewRequest("GET", url, nil)
					nsqResp, err := httpclient.Do(nsqReq)
					_ = nsqResp // TODO : process me ?
					// FIXME : use timeouts or other http client
					if err != nil {
						log.Println("NSQ delete topic error: " + err.Error())
						break
					}
					log.Println("NSQ delete topic is probably ok :)")
					/* -------------------------- */
					break // break loop (cleanup is over)
				}
				/* */
			}
			close(c.send)
			//c.subscribed = nil
		case sub := <-h.addSubscriber:
			log.Println("h.addSubscriber fired")
			reader, ok := h.nsqReaders[sub.Topic]
			c := sub.Conn
			if !ok {
				var err error
				reader, err = NewNsqTopicReader(sub.Topic)
				if err != nil {
					log.Printf("failed to subscribe to topic '%s'", sub.Topic)
					break
				}
			}
			c.subscribed[sub.Topic] = reader
			reader.connections[c] = true
			h.nsqReaders[sub.Topic] = reader
		}
	}

	panic("hub end of life")
}
