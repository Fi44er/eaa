package peer

import (
	"time"

	"github.com/pion/webrtc/v4"
)

func (p *Peer) attemptSync() (tryAgain bool) {
	for i := range *p.peerConnections {
		if (*p.peerConnections)[i].PeerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
			*p.peerConnections = append((*p.peerConnections)[:i], (*p.peerConnections)[i+1:]...)
			return true
		}
		existingSenders := map[string]bool{}
		if err := p.manageTracks((*p.peerConnections)[i].PeerConnection, existingSenders); err != nil {
			return true
		}
		if err := p.createAndSendOffer((*p.peerConnections)[i].PeerConnection, (*p.peerConnections)[i].Websocket); err != nil {
			return true
		}
	}
	return false
}

func (p *Peer) SignalPeerConnections() {
	p.listLock.Lock()
	defer func() {
		p.listLock.Unlock()
		p.DispatchKeyFrame()
	}()

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			go func() {
				time.Sleep(time.Second * 3)
				p.SignalPeerConnections()
			}()
		}

		// p.cleanupClosedConnections()
		if !p.attemptSync() {
			break
		}
	}
}
