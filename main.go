package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/widget"
	"github.com/pion/obs-wormhole/internal/rtmp"
	"github.com/pion/obs-wormhole/internal/ui"
	"github.com/pion/webrtc/v3"
)

func main() {
	peerConnection, videoTrack, audioTrack := createPeerConnection()
	go rtmp.StartServer(peerConnection, videoTrack, audioTrack)

	// Global PeerConnection and RTMP Server
	// Pull stats from each and populate Wormhole Status Form
	wormholeStatusForm := ui.CreateWormholeStatusForm()

	peerConnection.OnICEConnectionStateChange(func(i webrtc.ICEConnectionState) {
		fmt.Println(i)
	})

	// Form to Get/Set SDP
	isOffer := false
	wormholeEstablishForm := &widget.Form{
		BaseWidget: widget.BaseWidget{Hidden: true},
	}
	wormholeEstablishForm.OnSubmit = func() {
		for _, i := range wormholeEstablishForm.Items {
			entry, ok := i.Widget.(*widget.Entry)
			if ok && !entry.Disabled() {
				decoded, err := base64.StdEncoding.DecodeString(entry.Text)
				panicIfErr(err)

				remoteDescription := &webrtc.SessionDescription{}
				panicIfErr(json.Unmarshal(decoded, remoteDescription))

				panicIfErr(peerConnection.SetRemoteDescription(*remoteDescription))
			}
		}

		if !isOffer {
			fmt.Println("CreateAnswer")
			fmt.Println("SetLocalDescription")
		}
	}

	formLabel := widget.NewLabel("")
	wormholeTypeForm := ui.CreateWormholeTypeForm(formLabel, func(typ, role string, self *widget.Form) {
		// wormholeType = typ
		isOffer = role == "Offer"
		localSDP := []byte{}

		if isOffer {
			offer, err := peerConnection.CreateOffer(nil)
			panicIfErr(err)

			gatherPromise := webrtc.GatheringCompletePromise(peerConnection)
			panicIfErr(peerConnection.SetLocalDescription(offer))
			<-gatherPromise

			raw, err := json.Marshal(peerConnection.LocalDescription())
			panicIfErr(err)
			localSDP = raw
		}

		ui.PopulateWormholeEstablishForm(localSDP, wormholeEstablishForm)

		formLabel.SetText("")
		self.Hidden = true
		wormholeEstablishForm.Hidden = false
		wormholeStatusForm.Hidden = false
	})

	myApp := app.New()
	myWindow := myApp.NewWindow("OBS Wormhole")
	myWindow.Resize(fyne.Size{Width: 1280, Height: 1024})

	myWindow.SetContent(widget.NewScrollContainer(widget.NewVBox(
		formLabel,
		wormholeTypeForm,
		wormholeEstablishForm,
		wormholeStatusForm,
	)))
	myWindow.ShowAndRun()
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
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
