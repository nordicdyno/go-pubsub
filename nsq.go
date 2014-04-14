package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	//"strings"

	"github.com/bitly/go-nsq"
)

type NsqTopicReader struct {
	topicRealName   string
	channelRealName string
	r               *nsq.Reader
	connections     map[*connection]bool
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
	nsqResp, err := httpclient.Do(nsqReq)
	_ = nsqResp // TODO : process me ?
	// FIXME : use timeouts or other http client
	if err != nil {
		log.Println("NSQ create topic error: " + err.Error())
		return nil, err
	}
	log.Println("NSQ create topic is probably ok :)")
	/* -------------------------- */

	nReader := &NsqTopicReader{
		topicRealName:   topicName,
		channelRealName: realChannelName,
		//r:               *nsq.Reader,
		connections: make(map[*connection]bool),
	}

	nsqReader.VerboseLogging = true
	nsqReader.AddHandler(nReader)
	// duration between polling lookupd for new connections (default60 * time.Second)
	nsqReader.LookupdPollInterval = 30 * time.Second
	nsqReader.SetMaxInFlight(10)

	//addr := "tank01:4150"
	err = nsqReader.ConnectToLookupd(*addrNsqlookupd)
	if err != nil {
		log.Println("NSQ connection error: " + err.Error())
		return nil, err
	}
	nReader.r = nsqReader

	return nReader, nil
}

func (ntr *NsqTopicReader) HandleMessage(msg *nsq.Message) error {
	// TODO : add to log msg.Id msg.Timestamp
	log.Printf("Handled message\n")
	log.Printf("body: %v\n", string(msg.Body))
	// FIXME: check size of message body
	for c := range ntr.connections {
		log.Println("...send to consumer")
		c.send <- msg.Body
	}
	return nil
}
