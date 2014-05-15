package main

// TODO :   add statistic on /stat url
//          cleanup code (grep TODO/FIXME/XXX)
//          add flags
//          Add tests
// MAYBE: change topic/channels naming schema (add prefix, or allow real names)

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/gorilla/mux"
)

var (
	addr           = flag.String("addr", ":8080", "http service address")
	assets         = flag.String("assets", defaultAssetPath(), "path to assets")
	addrNsqd       = flag.String("nsqd-http", "localhost:4151", "nsqd HTTP address")
	addrNsqlookupd = flag.String("lookupd-http", "localhost:14161", "lookupd HTTP address")

	homeTempl *template.Template
	pid       = os.Getpid()
	uinqId    = ""
	// nsqd instances HTTP-balancer (or concrete nsqd HTTP-endpoint)
	// nsqlookupd instances HTTP-balancer
	addrNsqlookupdHTTP = ""
	addrNsqdHTTP       = ""
)

func init() {
	hostName, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	uinqId = (hostName + "_" + strconv.Itoa(pid))
	if len(uinqId) > ChannelMaxLen {
		uinqId = uinqId[0 : ChannelMaxLen-1]
	}
}

func defaultAssetPath() string {
	p, err := build.Default.Import("github.com/nordicdyno/go-pubsub", "", build.FindOnly)
	if err != nil {
		return "."
	}
	return filepath.Join(p.Dir, "resources")
}

func homeHandler(c http.ResponseWriter, req *http.Request) {
	homeTempl.Execute(c, req.Host)
}

type PostMessage struct {
	Channel string `json:"channel"`
	//Data    json.RawMessage  `json:"data"`
	Data map[string]interface{}
}

/*
type PostMessageData struct {
	Content        string
	Timestamp      uint64
	Type           string
	UserEmail      string `json:"user_email"`
	UserId         string `json:"user_id"`
	UserProfileUrl string `json:"user_profile_url"`
}
*/

func main() {
	flag.Parse()
	homePath := filepath.Join(*assets, "home.html")
	log.Println("homePath: " + homePath)
	homeTempl = template.Must(template.ParseFiles(homePath))

	addrNsqlookupdHTTP = "http://" + *addrNsqlookupd
	addrNsqdHTTP = "http://" + *addrNsqd
	log.Printf("vars: %s %s\n", addrNsqdHTTP, addrNsqlookupdHTTP)

	go h.run()

	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/channel/{channel}/event/chat_message/", postHandler).
		Methods("POST")

	http.HandleFunc("/socket/websocket", wsHandler)
	http.Handle("/", r)

	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func postHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println("ERROR: can't read http body")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	message := PostMessage{}
	err = json.Unmarshal(requestBody, &message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("ERROR: invalid JSON data: " + string(requestBody))
		return
	}

	vars := mux.Vars(req)
	log.Printf("postHandler/channel => %s\n", vars["channel"])
	log.Printf("postHandler/message: %+v\n", message)

	/*
		bJSON, err := json.Marshal(message.Data)
		if err != nil {
			log.Println("json marshaling error: " + err.Error())
			return
		}
	*/

	httpclient := &http.Client{}
	url := fmt.Sprintf(addrNsqdHTTP+"/put?topic=%s", GenNSQtopicName(vars["channel"]))

	//log.Printf("POST to %s; bJSON => «%s»\n", url, string(bJSON))
	//nsqReq, err := http.NewRequest("POST", url, bytes.NewBuffer(bJSON))
	log.Printf("POST to %s; bJSON => «%s»\n", url, string(requestBody))
	nsqReq, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	nsqResp, err := httpclient.Do(nsqReq)
	defer nsqResp.Body.Close()

	// FIXME : use timeouts or other http client
	if err != nil {
		log.Println("NSQ publish error: " + err.Error())
		return
	}
	log.Println("NSQ publish probably ok :)")
}
