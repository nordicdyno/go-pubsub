Description
---------------

PubSub websocket server based on NSQ queue transport
So it can work more than one server and must be a pretty fast 


Motivation
---------

Attempt make prototype of distributed websocket PubSub server baked with Go (golang) and NSQ


Usage
---------------

## Install

```
go get github.com/nordicdyno/go-pubsub
```

## Quick start

Start nsqd:
```
nsqd
```

start go-pubsub
```
NSQ_VERBOSE=1 go-pubsub -nsqd=localhost:4150
```

Open test page on http://your-host:8080
(better try it in multiple tabs)

* button "send" - send message from first text field in channel from second field
* button "subscribe" - subscribes on channel from second field
* you can subscribe on more than one channel
* you can send only in one channel on test page (in one moment of time)
* you can't unsubscribe without disconnecting (close page) yet


Status
---------------

Under construction, that means this code is not heavly tested, it with a lot of debug messages and
without any advanced features but it pretty simple and works.
