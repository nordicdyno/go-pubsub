package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bitly/go-nsq"
)

// TODO : rename non-NSQ Topics to Rooms (to avoid misunderstanding)

type ConnectionsMap map[*connection]bool
type TopicsMap map[string]bool

type NsqTopicReader struct {
	topicRealName   string
	channelRealName string
	r               *nsq.Reader
	muMapsLock      sync.RWMutex
	topic2conns     map[string]ConnectionsMap
	conn2topics     map[*connection]TopicsMap
}

const TopicMaxLen = 32
const ChannelMaxLen = TopicMaxLen - len("#ephemeral")

type Subscriber struct {
	Conn  *connection
	Topic string
}

func NewNsqTopicReader(topic string) (*NsqTopicReader, error) {

	topicName := GenNSQtopicName(topic)

	// FIXME: gen channel name to 32 bytes token
	log.Printf("try to subscribe on topic %s (nsq real name %s)", topic, topicName)
	// FIXME: use unique channel name
	// WorkerID("PID" + "hostName") + "connId" -> 32 - 10 => 22 bytes
	// + add http://golang.org/pkg/crypto/sha1/
	// http://www.quora.com/Cryptography/What-is-the-smallest-prefix-length-of-an-SHA1-hash-that-would-guarantee-uniqueness-in-a-reasonable-object-space

	realChannelName := uinqId + "#ephemeral"
	nsqReader, err := nsq.NewReader(topicName, realChannelName)
	log.Println("NSQ channel name is " + realChannelName)

	if err != nil {
		log.Println("nsq.NewReader error: " + err.Error())
		return nil, err
	}

	// TODO : refactor me :)
	/* create topic */
	httpclient := &http.Client{}
	url := fmt.Sprintf(addrNsqdHTTP+"/create_topic?topic=%s", topicName)
	nsqReq, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	nsqResp, err := httpclient.Do(nsqReq)
	// FIXME : use timeouts or other http client
	if err != nil {
		log.Println("NSQ create topic error: " + err.Error())
		return nil, err
	}
	defer nsqResp.Body.Close()
	log.Println("NSQ create topic is probably ok :)")
	/* -------------------------- */

	nReader := &NsqTopicReader{
		topicRealName:   topicName,
		channelRealName: realChannelName,
		topic2conns:     make(map[string]ConnectionsMap),
		conn2topics:     make(map[*connection]TopicsMap),
	}

	// TODO: move to Environment var
	if os.Getenv("NSQ_VERBOSE") != "" {
		nsqReader.VerboseLogging = true
	}
	nsqReader.AddHandler(nReader)
	// TODO: move magic numbers to flags or global vars/consts
	// duration between polling lookupd for new connections (default60 * time.Second)
	nsqReader.LookupdPollInterval = 30 * time.Second
	nsqReader.SetMaxInFlight(10)

	//addr := "tank01:4150"
	var nsqErr error
	if *addrNsqdTCP != "" {
		nsqErr = nsqReader.ConnectToNSQ(*addrNsqdTCP)
	} else {
		nsqErr = nsqReader.ConnectToLookupd(*addrNsqlookupd)
	}
	if nsqErr != nil {
		log.Println("NSQ connection error: " + nsqErr.Error())
		return nil, err
	}
	nReader.r = nsqReader

	return nReader, nil
}

func (ntr *NsqTopicReader) AddConnection(c *connection, topic string) {
	log.Println("add connection for topic " + topic)

	ntr.muMapsLock.Lock()
	if ntr.topic2conns[topic] == nil {
		ntr.topic2conns[topic] = make(ConnectionsMap)
	}
	if ntr.conn2topics[c] == nil {
		ntr.conn2topics[c] = make(TopicsMap)
	}
	ntr.topic2conns[topic][c] = true
	ntr.conn2topics[c][topic] = true
	ntr.muMapsLock.Unlock()
}

func (ntr *NsqTopicReader) RemoveConnection(c *connection) {
	ntr.muMapsLock.Lock()
	for topic, _ := range ntr.conn2topics[c] {
		delete(ntr.topic2conns[topic], c)
		log.Println("  channel: " + topic)
	}
	delete(ntr.conn2topics, c)
	ntr.muMapsLock.Unlock()
}

func (ntr *NsqTopicReader) HandleMessage(msg *nsq.Message) error {
	// TODO : add to log msg.Id msg.Timestamp
	log.Printf("Handled message\n")
	log.Printf("body: %v\n", string(msg.Body))
	// FIXME: check size of message body

	message := SubMessage{}
	err := json.Unmarshal(msg.Body, &message)
	if err != nil {
		log.Println("NSQ HandleMessage ERROR: invalid JSON subscribe data")
		return err
	}
	//log.Printf("message+: %v\n", message)

	// data race (a)
	// TODO : slurp all keys before sending ?
	ntr.muMapsLock.RLock()
	for c, _ := range ntr.topic2conns[message.Channel] {
		log.Println("...send to consumers")
		c.send <- msg.Body
	}
	ntr.muMapsLock.RUnlock()
	return nil
}
