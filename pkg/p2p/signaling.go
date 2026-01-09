package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/1amkhush/torrentium/pkg/logger"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

const SignalingProtocolID = "/torrentium/webrtc-signaling/1.0"

type SignalingMessage struct {
	Type string `json:"type"` // "offer", "answer", "ice-candidate", "close"
	Data string `json:"data"`
}

// RegisterSignalingProtocol sets up WebRTC signaling protocol handler
func RegisterSignalingProtocol(h host.Host, onOffer func(offer, remotePeerID string, s network.Stream) (string, error)) {
	log := logger.WithComponent("signaling")

	h.SetStreamHandler(SignalingProtocolID, func(s network.Stream) {
		remotePeer := s.Conn().RemotePeer().String()
		log.Debug().Str("remote_peer", remotePeer).Msg("Received incoming signaling connection")

		decoder := json.NewDecoder(s)
		encoder := json.NewEncoder(s)

		var msg SignalingMessage
		if err := decoder.Decode(&msg); err != nil {
			log.Error().Err(err).Str("remote_peer", remotePeer).Msg("Error decoding signaling message")
			_ = s.Reset()
			return
		}

		if msg.Type != "offer" {
			log.Warn().Str("type", msg.Type).Str("remote_peer", remotePeer).Msg("Expected offer, got different message type")
			_ = s.Reset()
			return
		}

		answer, err := onOffer(msg.Data, remotePeer, s)
		if err != nil {
			log.Error().Err(err).Str("remote_peer", remotePeer).Msg("Error handling offer")
			errorMsg := SignalingMessage{
				Type: "error",
				Data: fmt.Sprintf("ERROR:%s", err.Error()),
			}
			_ = encoder.Encode(errorMsg)
			_ = s.Reset()
			return
		}

		answerMsg := SignalingMessage{
			Type: "answer",
			Data: answer,
		}

		if err := encoder.Encode(answerMsg); err != nil {
			log.Error().Err(err).Str("remote_peer", remotePeer).Msg("Error encoding answer")
			_ = s.Reset()
			return
		}

		// Keep stream open for ICE candidate exchange
		log.Debug().Str("remote_peer", remotePeer).Msg("Signaling stream established")

		// Handle additional signaling messages (ICE candidates, etc.)
		for {
			var additionalMsg SignalingMessage
			if err := decoder.Decode(&additionalMsg); err != nil {
				log.Debug().Err(err).Str("remote_peer", remotePeer).Msg("Signaling stream closed or error")
				break
			}

			switch additionalMsg.Type {
			case "close":
				log.Debug().Str("remote_peer", remotePeer).Msg("Peer requested signaling stream close")
				return
			case "ice-candidate":
				log.Debug().Str("remote_peer", remotePeer).Msg("Received ICE candidate from peer")
			default:
				log.Warn().Str("type", additionalMsg.Type).Str("remote_peer", remotePeer).Msg("Unknown signaling message type")
			}
		}
	})
}
