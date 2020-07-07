package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"math/rand"

	"fyne.io/fyne"
	"fyne.io/fyne/widget"
	inRtmp "github.com/pion/obs-wormhole/internal/rtmp"
	"github.com/pion/webrtc/v3"
	"github.com/yutopp/go-rtmp"
)

type SignalingPageContext struct {
	IsOffer bool
}

type SignalingPage struct {
	fyne.Widget
	statusChan     chan webrtc.ICEConnectionState
	rtmpServer     *rtmp.Server
	peerConnection *webrtc.PeerConnection
}

func NewSignalingPage(navigator Navigator, ctx SignalingPageContext) (Page, error) {
	remoteSDPInput := widget.NewMultiLineEntry()
	remoteSDPInput.Wrapping = fyne.TextWrapBreak

	localSDPInput := widget.NewMultiLineEntry()
	localSDPInput.Wrapping = fyne.TextWrapBreak
	localSDPInput.Disable()

	form := widget.NewAccordionContainer(
		widget.NewAccordionItem("Local SDP", localSDPInput),
		widget.NewAccordionItem("Remote SDP", remoteSDPInput),
	)

	errLabel := widget.NewLabel("")

	peerConnection, videoTrack, audioTrack := createPeerConnection()
	rtmpServer := inRtmp.StartServer(peerConnection, videoTrack, audioTrack)

	statusChan := make(chan webrtc.ICEConnectionState, 1)
	peerConnection.OnICEConnectionStateChange(func(status webrtc.ICEConnectionState) {
		log.Println(status)
		select {
		case statusChan <- status:
		default:
			// in case of slow reader, just throw away the state.
			// Since it's a buffered channel, there should be always an initial state for the UI
			<-statusChan
		}
	})

	submitButton := widget.NewButton("Submit", func() {
		sdp := remoteSDPInput.Text
		if sdp == "" {
			errLabel.SetText("SDP can't be empty")
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(sdp)
		panicIfErr(err)

		remoteDescription := &webrtc.SessionDescription{}
		panicIfErr(json.Unmarshal(decoded, remoteDescription))

		panicIfErr(peerConnection.SetRemoteDescription(*remoteDescription))

		if !ctx.IsOffer {
			answer, err := peerConnection.CreateAnswer(nil)
			panicIfErr(err)

			peerConnection.SetLocalDescription(answer)
		}

		navigator.Push(RouteStatus, StatusPageContext{
			StatusChan: statusChan,
		})
	})

	backButton := widget.NewButton("Back", func() {
		navigator.Reset()
	})

	if ctx.IsOffer {
		offer, err := peerConnection.CreateOffer(nil)
		panicIfErr(err)

		gatherPromise := webrtc.GatheringCompletePromise(peerConnection)
		panicIfErr(peerConnection.SetLocalDescription(offer))
		<-gatherPromise

		raw, err := json.Marshal(peerConnection.LocalDescription())
		panicIfErr(err)
		localSDPInput.SetText(base64.StdEncoding.EncodeToString(raw))
	}

	content := widget.NewScrollContainer(widget.NewVBox(
		form,
		submitButton,
		backButton,
		errLabel,
	))
	return &SignalingPage{
		Widget:         content,
		statusChan:     statusChan,
		peerConnection: peerConnection,
		rtmpServer:     rtmpServer,
	}, nil
}

func (page *SignalingPage) BeforeDestroy() {
	close(page.statusChan)
	page.rtmpServer.Close()
	page.peerConnection.Close()
}

func createPeerConnection() (*webrtc.PeerConnection, *webrtc.Track, *webrtc.Track) {
	// Only support PCMA + H264
	m := webrtc.MediaEngine{}
	m.RegisterCodec(webrtc.NewRTPH264Codec(webrtc.DefaultPayloadTypeH264, 90000))
	m.RegisterCodec(webrtc.NewRTPPCMACodec(webrtc.DefaultPayloadTypePCMA, 8000))

	// Create a PeerConnection
	peerConnection, err := webrtc.NewAPI(webrtc.WithMediaEngine(m)).NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	panicIfErr(err)

	videoTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeH264, rand.Uint32(), "video", "pion")
	panicIfErr(err)

	_, err = peerConnection.AddTrack(videoTrack)
	panicIfErr(err)

	audioTrack, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypePCMA, rand.Uint32(), "audio", "pion")
	panicIfErr(err)

	_, err = peerConnection.AddTrack(audioTrack)
	panicIfErr(err)

	return peerConnection, videoTrack, audioTrack
}
