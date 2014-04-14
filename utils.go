package main

import (
	"crypto/sha1"
	"fmt"
)

type SubMessage struct {
	Event   string
	Channel string `json:"channel"`
	// Data    SubMessageData `json:"data"` // just ignore it
}

func NameToSHA1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

func GenNSQtopicName(topic string) string {
	return (NameToSHA1(topic))[0 : TopicMaxLen-1]
}
