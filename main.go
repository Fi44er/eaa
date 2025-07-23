package main

import (
	"flag"
	"net/http"
	"os"
	"root/service/peer"
	"root/service/track"
	"root/service/ws"
	"root/utils"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/logging"
	"github.com/pion/webrtc/v4"
)

var (
	addr     = flag.String("addr", ":8080", "http service address")
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	indexTemplate = &template.Template{}

	// lock for peerConnections and trackLocals
	listLock        sync.RWMutex
	peerConnections []utils.PeerConnectionState
	trackLocals     map[string]*webrtc.TrackLocalStaticRTP

	log = logging.NewDefaultLoggerFactory().NewLogger("sfu-ws")
)

func main() {
	// Parse the flags passed to program
	flag.Parse()

	// Init other state
	trackLocals = map[string]*webrtc.TrackLocalStaticRTP{}

	// Read index.html from disk into memory, serve whenever anyone requests /
	indexHTML, err := os.ReadFile("index.html")
	if err != nil {
		panic(err)
	}
	indexTemplate = template.Must(template.New("").Parse(string(indexHTML)))

	peer := peer.NewPeer(&listLock, &peerConnections, trackLocals, log)
	track := track.NewTrack(&listLock, trackLocals, peer)
	wsHandler := ws.NewW(log, &listLock, &peerConnections, peer, track)

	// websocket handler
	http.HandleFunc("/websocket", wsHandler.WebsocketHandler)

	// index.html handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err = indexTemplate.Execute(w, "wss://"+r.Host+"/websocket"); err != nil {
			log.Errorf("Failed to parse index template: %v", err)
		}
	})

	// request a keyframe every 3 seconds
	go func() {
		for range time.NewTicker(time.Second * 3).C {
			peer.DispatchKeyFrame()
		}
	}()

	// start HTTP server
	if err = http.ListenAndServeTLS(*addr, "./cert.pem", "./key.pem", nil); err != nil { //nolint: gosec
		log.Errorf("Failed to start http server: %v", err)
	}
}
