package main

import (
	//"fmt"
	"strings"
)

func GenNSQtopicName(topic string) string {
	parts := strings.SplitN(topic, "_", 2)
	return parts[0]
}

/*
func NameToSHA1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

func GenNSQtopicName(topic string) string {
	return (NameToSHA1(topic))[0 : TopicMaxLen-1]
}
*/
