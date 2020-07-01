package ui

import (
	"encoding/base64"

	"fyne.io/fyne"
	"fyne.io/fyne/widget"
)

func CreateWormholeTypeForm(formLabel *widget.Label, onSuccess func(string, string, *widget.Form)) *widget.Form {
	wormholeType := widget.NewSelect([]string{"WebRTC Source", "P2P Streaming"}, nil)
	webrtcRole := widget.NewSelect([]string{"Offer", "Answer"}, nil)

	wormholeTypeForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Wormhole Type", Widget: wormholeType},
			{Text: "WebRTC Role", Widget: webrtcRole},
		},
	}

	wormholeTypeForm.OnSubmit = func() {
		if wormholeType.Selected == "" || webrtcRole.Selected == "" {
			formLabel.SetText("Type and Role must be set")
		} else {
			onSuccess(wormholeType.Selected, webrtcRole.Selected, wormholeTypeForm)
		}
	}

	return wormholeTypeForm
}

func PopulateWormholeEstablishForm(offer []byte, form *widget.Form) {
	remoteSDP := widget.NewMultiLineEntry()
	remoteSDP.Wrapping = fyne.TextWrapBreak

	localSDP := widget.NewMultiLineEntry()
	localSDP.Wrapping = fyne.TextWrapBreak
	localSDP.Disable()
	localSDP.Text = base64.StdEncoding.EncodeToString(offer)

	form.Append("Local SDP", localSDP)
	form.Append("Remote SDP", remoteSDP)
}

func CreateWormholeStatusForm() *widget.Form {
	wormholeStatusForm := &widget.Form{
		BaseWidget: widget.BaseWidget{Hidden: true},
	}

	return wormholeStatusForm
}
